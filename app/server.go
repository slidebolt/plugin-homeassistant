package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"syscall"

	"github.com/gorilla/websocket"
	"github.com/grandcat/zeroconf"
	domain "github.com/slidebolt/sb-domain"
	messenger "github.com/slidebolt/sb-messenger-sdk"
	translate "github.com/slidebolt/plugin-homeassistant/internal/translate"
	storage "github.com/slidebolt/sb-storage-sdk"
)

// --- Wire types (mirror the mock server protocol exactly) ---

type wireMessage struct {
	Type      string         `json:"type"`
	Auth      *bool          `json:"auth,omitempty"`
	ClientID  string         `json:"client_id,omitempty"`
	Snapshot  *wireSnapshot  `json:"snapshot,omitempty"`
	ID        string         `json:"id,omitempty"`
	ServerID  string         `json:"server_id,omitempty"`
	EntityID string         `json:"entity_id,omitempty"`
	Command  string         `json:"command,omitempty"`
	Params   map[string]any `json:"params,omitempty"`
	Success  *bool          `json:"success,omitempty"`
	Entity   *wireEntity    `json:"entity,omitempty"`
}

type wireSnapshot struct {
	Devices []wireDevice `json:"devices"`
}

type wireDevice struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Entities []wireEntity `json:"entities"`
}

type wireEntity struct {
	UniqueID   string         `json:"unique_id"`
	EntityID   string         `json:"entity_id"`
	Platform   string         `json:"platform"`
	Name       string         `json:"name"`
	Available  bool           `json:"available"`
	State      map[string]any `json:"state"`
	Attributes map[string]any `json:"attributes"`
}

// Broadcaster defines the interface for sending messages to connected HA clients.
type Broadcaster interface {
	Broadcast(msg wireMessage)
}

// --- haServer ---

type haServer struct {
	cfg        Config
	store      storage.Storage
	cmds       *messenger.Commands
	instanceID string

	mu      sync.RWMutex
	clients []*websocket.Conn

	srv  *http.Server
	ln   net.Listener
	mdns *zeroconf.Server

	// onSnapshotSent is called after the first successful handshake (test hook).
	onSnapshotSent func()

	// testEntities is an optional in-memory entity map used in tests when no
	// real storage is wired up. Keyed by entity_id (e.g. "light.testlight").
	testEntities map[string]domain.Entity

	// mockBroadcaster is used in tests to intercept outgoing messages.
	mockBroadcaster Broadcaster

	trustedClientID string
}

type pairingKey struct{}

func (pairingKey) Key() string { return "plugin-homeassistant.pairing" }

type pairingRecord struct {
	TrustedClientID string `json:"trusted_client_id"`
}

func newServer(cfg Config, store storage.Storage, cmds *messenger.Commands) *haServer {
	return &haServer{
		cfg:   cfg,
		store: store,
		cmds:  cmds,
	}
}

// Start begins listening for WebSocket connections and registers mDNS.
// Returns the actual port the server is listening on.
func (s *haServer) Start() (int, error) {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1) //nolint:errcheck
			})
		},
	}
	ln, err := lc.Listen(context.Background(), "tcp", ":"+s.cfg.Port)
	if err != nil {
		return 0, fmt.Errorf("listen: %w", err)
	}
	s.ln = ln
	port := ln.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWS)
	s.srv = &http.Server{Handler: mux}

	go func() {
		if err := s.srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("plugin-homeassistant: server error: %v", err)
		}
	}()

	iface := outboundInterface()
	s.instanceID = s.cfg.SystemUUID
	mdns, err := zeroconf.Register(
		"SlideBolt",
		"_slidebolt._tcp",
		"local.",
		port,
		[]string{"version=1.0.0", "id=" + s.cfg.SystemUUID, "mac=" + s.cfg.SystemMAC},
		iface,
	)
	if err != nil {
		log.Printf("plugin-homeassistant: mDNS registration failed: %v", err)
	} else {
		s.mdns = mdns
		log.Printf("plugin-homeassistant: mDNS advertising _slidebolt._tcp on port %d", port)
	}

	return port, nil
}

// Stop shuts down the HTTP server and unregisters mDNS.
func (s *haServer) Stop() {
	// Close all active WebSocket connections so HA detects the disconnect immediately.
	s.mu.Lock()
	for _, c := range s.clients {
		c.Close()
	}
	s.clients = nil
	s.mu.Unlock()

	if s.mdns != nil {
		s.mdns.Shutdown()
	}
	if s.srv != nil {
		s.srv.Shutdown(context.Background()) //nolint:errcheck
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (s *haServer) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("plugin-homeassistant: upgrade error: %v", err)
		return
	}
	defer func() {
		s.removeClient(conn)
		conn.Close()
	}()

	log.Printf("plugin-homeassistant: HA connected from %s", r.RemoteAddr)

	// 1. Read hello from HA
	var hello wireMessage
	if err := conn.ReadJSON(&hello); err != nil {
		log.Printf("plugin-homeassistant: read hello: %v", err)
		return
	}
	if hello.Type != "hello" {
		log.Printf("plugin-homeassistant: expected hello, got %q", hello.Type)
		return
	}

	// 2. Respond with hello + auth
	authOK := s.authorizeClientID(hello.ClientID, r.RemoteAddr)
	if !authOK {
		_ = conn.WriteJSON(wireMessage{Type: "hello", Auth: &authOK, ServerID: s.instanceID})
		return
	}

	if err := conn.WriteJSON(wireMessage{Type: "hello", Auth: &authOK, ServerID: s.instanceID}); err != nil {
		log.Printf("plugin-homeassistant: write hello: %v", err)
		return
	}

	// 3. Send snapshot
	snap, err := s.buildSnapshot()
	if err != nil {
		log.Printf("plugin-homeassistant: build snapshot: %v", err)
		return
	}
	if err := conn.WriteJSON(wireMessage{Type: "snapshot", Snapshot: snap}); err != nil {
		log.Printf("plugin-homeassistant: write snapshot: %v", err)
		return
	}
	total := 0
	for _, d := range snap.Devices {
		total += len(d.Entities)
	}
	log.Printf("plugin-homeassistant: snapshot sent (%d entities)", total)

	if s.onSnapshotSent != nil {
		s.onSnapshotSent()
	}

	// 4. Register client and enter command loop
	s.addClient(conn)
	for {
		var msg wireMessage
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("plugin-homeassistant: HA disconnected: %v", err)
			return
		}

		if msg.Type == "command" {
			s.handleInboundCommand(conn, msg)
		} else {
			log.Printf("plugin-homeassistant: unknown message type %q", msg.Type)
		}
	}
}

func (s *haServer) authorizeClientID(clientID, remoteAddr string) bool {
	if clientID == "" {
		log.Printf("plugin-homeassistant: missing client_id from %s", remoteAddr)
		return false
	}

	trustedClientID, err := s.getTrustedClientID()
	if err != nil {
		log.Printf("plugin-homeassistant: load pairing state: %v", err)
		return false
	}
	if trustedClientID == "" {
		if err := s.setTrustedClientID(clientID); err != nil {
			log.Printf("plugin-homeassistant: store pairing state: %v", err)
			return false
		}
		log.Printf("plugin-homeassistant: pairing established for client_id=%s", clientID)
		return true
	}
	if trustedClientID != clientID {
		log.Printf("plugin-homeassistant: pairing rejected for client_id=%s", clientID)
		log.Printf("plugin-homeassistant: client_id mismatch from %s", remoteAddr)
		return false
	}
	return true
}

func (s *haServer) getTrustedClientID() (string, error) {
	s.mu.RLock()
	trustedClientID := s.trustedClientID
	s.mu.RUnlock()
	if trustedClientID != "" {
		return trustedClientID, nil
	}
	if s.store == nil {
		return "", nil
	}

	data, err := s.store.GetInternal(pairingKey{})
	if err != nil {
		return "", nil
	}
	var record pairingRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return "", fmt.Errorf("parse pairing state: %w", err)
	}

	s.mu.Lock()
	s.trustedClientID = record.TrustedClientID
	s.mu.Unlock()
	return record.TrustedClientID, nil
}

func (s *haServer) setTrustedClientID(clientID string) error {
	s.mu.Lock()
	s.trustedClientID = clientID
	s.mu.Unlock()
	if s.store == nil {
		return nil
	}

	body, err := json.Marshal(pairingRecord{TrustedClientID: clientID})
	if err != nil {
		return fmt.Errorf("marshal pairing state: %w", err)
	}
	if err := s.store.SetInternal(pairingKey{}, body); err != nil {
		return fmt.Errorf("persist pairing state: %w", err)
	}
	return nil
}

// handleInboundCommand processes a command arriving from HA via WebSocket.
// It routes the command to the entity's owning plugin via messenger, or
// applies it locally if the entity belongs to this plugin (or in test mode).
func (s *haServer) handleInboundCommand(conn *websocket.Conn, msg wireMessage) {
	entity, err := s.findEntityByWireID(msg.EntityID)
	if err != nil {
		log.Printf("plugin-homeassistant: entity not found for %q: %v", msg.EntityID, err)
		fail := false
		conn.WriteJSON(wireMessage{Type: "command_result", ID: msg.ID, EntityID: msg.EntityID, Success: &fail}) //nolint:errcheck
		return
	}

	entityType := entity.Type

	cmd, err := translate.FromHA(entityType, msg.Command, msg.Params)
	if err != nil {
		log.Printf("plugin-homeassistant: fromHA %s/%s: %v", entityType, msg.Command, err)
		fail := false
		conn.WriteJSON(wireMessage{Type: "command_result", ID: msg.ID, EntityID: msg.EntityID, Success: &fail}) //nolint:errcheck
		return
	}

	if s.testEntities != nil {
		// Test mode: apply locally and broadcast.
		updated := applyCommand(entity, cmd)
		s.testEntities[msg.EntityID] = updated
		success := true
		conn.WriteJSON(wireMessage{Type: "command_result", ID: msg.ID, EntityID: msg.EntityID, Success: &success}) //nolint:errcheck
		we := entityToWire(updated)
		s.Broadcast(wireMessage{Type: "entity_updated", Entity: &we})
		return
	}

	// Production: route the command to the entity's owning plugin.
	if actionCmd, ok := cmd.(messenger.Action); ok && s.cmds != nil {
		target := domain.EntityKey{Plugin: entity.Plugin, DeviceID: entity.DeviceID, ID: entity.ID}
		if err := s.cmds.Send(target, actionCmd); err != nil {
			log.Printf("plugin-homeassistant: route command to %s: %v", target.Key(), err)
			fail := false
			conn.WriteJSON(wireMessage{Type: "command_result", ID: msg.ID, EntityID: msg.EntityID, Success: &fail}) //nolint:errcheck
			return
		}
	}

	success := true
	conn.WriteJSON(wireMessage{Type: "command_result", ID: msg.ID, EntityID: msg.EntityID, Success: &success}) //nolint:errcheck
}

// findEntityByWireID looks up an entity by its HA entity_id (e.g.
// "light.plugin_kasa_living_room_lamp1"). The entity_id is built from
// WireID() so we match against that.
func (s *haServer) findEntityByWireID(entityID string) (domain.Entity, error) {
	if s.testEntities != nil {
		if e, ok := s.testEntities[entityID]; ok {
			return e, nil
		}
		return domain.Entity{}, fmt.Errorf("entity %q not found", entityID)
	}
	entries, err := s.store.Query(storage.Query{
		Where: []storage.Filter{{Field: "labels.PluginHomeassistant", Op: storage.Exists}},
	})
	if err != nil {
		return domain.Entity{}, fmt.Errorf("query: %w", err)
	}
	for _, entry := range entries {
		var entity domain.Entity
		if err := json.Unmarshal(entry.Data, &entity); err != nil {
			continue
		}
		if WireID(entity) == entityID {
			return entity, nil
		}
	}
	return domain.Entity{}, fmt.Errorf("entity %q not found", entityID)
}

// Broadcast sends a message to all connected HA clients.
func (s *haServer) Broadcast(msg wireMessage) {
	if s.mockBroadcaster != nil {
		s.mockBroadcaster.Broadcast(msg)
		return
	}

	s.mu.RLock()
	clients := make([]*websocket.Conn, len(s.clients))
	copy(clients, s.clients)
	s.mu.RUnlock()

	for _, conn := range clients {
		if err := conn.WriteJSON(msg); err != nil {
			log.Printf("plugin-homeassistant: broadcast error: %v", err)
		}
	}
}

func (s *haServer) addClient(conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients = append(s.clients, conn)
}

func (s *haServer) removeClient(conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.clients {
		if c == conn {
			s.clients = append(s.clients[:i], s.clients[i+1:]...)
			return
		}
	}
}

// buildSnapshot reads all plugin-homeassistant entities and returns them
// grouped by device in wire format. Uses testEntities if set (no storage needed).
func (s *haServer) buildSnapshot() (*wireSnapshot, error) {
	if s.testEntities != nil {
		deviceMap := make(map[string]*wireDevice)
		for _, entity := range s.testEntities {
			dev, ok := deviceMap[entity.DeviceID]
			if !ok {
				dev = &wireDevice{ID: entity.DeviceID, Name: entity.DeviceID}
				deviceMap[entity.DeviceID] = dev
			}
			dev.Entities = append(dev.Entities, entityToWire(entity))
		}
		snap := &wireSnapshot{}
		for _, dev := range deviceMap {
			snap.Devices = append(snap.Devices, *dev)
		}
		return snap, nil
	}
	if s.store == nil {
		return &wireSnapshot{Devices: []wireDevice{}}, nil
	}
	entries, err := s.store.Query(storage.Query{
		Where: []storage.Filter{{Field: "labels.PluginHomeassistant", Op: storage.Exists}},
	})
	if err != nil {
		return nil, fmt.Errorf("query entities: %w", err)
	}

	deviceMap := make(map[string]*wireDevice)
	for _, entry := range entries {
		var entity domain.Entity
		if err := json.Unmarshal(entry.Data, &entity); err != nil {
			continue
		}
		devKey := entity.Plugin + "." + entity.DeviceID
		dev, ok := deviceMap[devKey]
		if !ok {
			dev = &wireDevice{ID: devKey, Name: entity.DeviceID}
			deviceMap[devKey] = dev
		}
		dev.Entities = append(dev.Entities, entityToWire(entity))
	}

	snap := &wireSnapshot{}
	for _, dev := range deviceMap {
		snap.Devices = append(snap.Devices, *dev)
	}
	return snap, nil
}

// haPlatform maps a domain entity type to the corresponding Home Assistant
// platform name. Most types pass through unchanged; "alarm" is the exception.
func haPlatform(domainType string) string {
	if domainType == "alarm" {
		return "alarm_control_panel"
	}
	return domainType
}

// slugify converts a string to a HA-safe object_id: lowercase, non-alphanum
// characters replaced with underscores, leading/trailing underscores stripped,
// consecutive underscores collapsed.
func slugify(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prev := byte('_')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			b.WriteByte(c)
			prev = c
		case c >= 'A' && c <= 'Z':
			b.WriteByte(c + 32) // tolower
			prev = c
		default:
			if prev != '_' {
				b.WriteByte('_')
				prev = '_'
			}
		}
	}
	out := b.String()
	return strings.Trim(out, "_")
}

// WireID returns the HA entity_id for a domain entity, e.g.
// "light.plugin_kasa_living_room_lamp1". The object_id portion is the
// slugified entity key so it is globally unique across plugins.
func WireID(entity domain.Entity) string {
	return haPlatform(entity.Type) + "." + slugify(entity.Key())
}

// entityToWire converts a domain.Entity to the HA wire format.
func entityToWire(entity domain.Entity) wireEntity {
	platform := haPlatform(entity.Type)
	stateMap, attrs := translate.ToHA(entity)
	return wireEntity{
		UniqueID:   entity.Key(),
		EntityID:   WireID(entity),
		Platform:   platform,
		Name:       entity.Name,
		Available:  true,
		State:      stateMap,
		Attributes: attrs,
	}
}

// outboundInterface returns the network interface used for the default route
// so mDNS is advertised on the same interface HA's zeroconf listens on.
func outboundInterface() []net.Interface {
	conn, err := net.Dial("udp", "1.1.1.1:53")
	if err != nil {
		return nil
	}
	defer conn.Close()
	localIP := conn.LocalAddr().(*net.UDPAddr).IP

	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && ip.Equal(localIP) {
				return []net.Interface{iface}
			}
		}
	}
	return nil
}

// --- Test helpers (used by manual/integration tests in cmd/) ---

// ServerRunner is returned by NewServerForTest for testing.
type ServerRunner interface {
	Start() (int, error)
	Stop()
	// Connected returns a channel that receives once the first HA client
	// completes the hello handshake (snapshot sent).
	Connected() <-chan struct{}
}

// testHAServer wraps haServer and signals on first successful handshake.
type testHAServer struct {
	*haServer
	connectedCh chan struct{}
	once        sync.Once
}

func (t *testHAServer) Connected() <-chan struct{} { return t.connectedCh }

func (t *testHAServer) Start() (int, error) {
	t.haServer.onSnapshotSent = func() {
		t.once.Do(func() { close(t.connectedCh) })
	}
	return t.haServer.Start()
}

// NewServerForTest creates a standalone haServer for testing.
// Provide entities to serve in the snapshot; commands will mutate the in-memory map.
func NewServerForTest(cfg Config, entities ...domain.Entity) ServerRunner {
	s := newServer(cfg, nil, nil)
	if len(entities) > 0 {
		s.testEntities = make(map[string]domain.Entity, len(entities))
		for _, e := range entities {
			s.testEntities[WireID(e)] = e
		}
	}
	return &testHAServer{
		haServer:    s,
		connectedCh: make(chan struct{}),
	}
}

//go:build !integration
// +build !integration

package main

import (
	"encoding/json"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	testkit "github.com/slidebolt/sb-testkit"

	domain "github.com/slidebolt/sb-domain"
	messenger "github.com/slidebolt/sb-messenger-sdk"
	storage "github.com/slidebolt/sb-storage-sdk"

	"github.com/slidebolt/plugin-homeassistant/app"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// multiEnv creates a TestEnv with messenger + storage, wires up the plugin-ha
// app, and returns everything needed to drive tests.
type multiEnv struct {
	t     *testing.T
	env   *testkit.TestEnv
	store storage.Storage
	app   *app.App
	port  string
}

func newMultiEnv(t *testing.T) *multiEnv {
	t.Helper()

	env := testkit.NewTestEnv(t)
	env.Start("messenger")
	env.Start("storage")

	// Use port 0 so each test gets a random free port.
	t.Setenv("HA_PORT", "0")
	t.Setenv("SYSTEM_UUID", "test-system-uuid")
	t.Setenv("SYSTEM_MAC", "02:00:00:00:00:01")

	haApp := app.New()
	deps := map[string]json.RawMessage{
		"messenger": env.MessengerPayload(),
	}
	if _, err := haApp.OnStart(deps); err != nil {
		t.Fatalf("app.OnStart: %v", err)
	}
	t.Cleanup(func() { haApp.OnShutdown() })

	// Read the actual port from the server's listener.
	port := haApp.Port()

	return &multiEnv{
		t:     t,
		env:   env,
		store: env.Storage(),
		app:   haApp,
		port:  port,
	}
}

func (m *multiEnv) saveEntity(entity domain.Entity) {
	m.t.Helper()
	if err := m.store.Save(entity); err != nil {
		m.t.Fatalf("save entity %s: %v", entity.Key(), err)
	}
	// Persist labels/meta in sidecar so they survive Save() stripping.
	profile := make(map[string]any)
	if len(entity.Labels) > 0 {
		profile["labels"] = entity.Labels
	}
	if len(entity.Meta) > 0 {
		profile["meta"] = entity.Meta
	}
	if len(profile) > 0 {
		data, _ := json.Marshal(profile)
		if err := m.store.SetProfile(entity, json.RawMessage(data)); err != nil {
			m.t.Fatalf("setprofile %s: %v", entity.Key(), err)
		}
	}
}

func labeledEntity(plugin, device, id, typ, name string, state any) domain.Entity {
	return domain.Entity{
		Plugin:   plugin,
		DeviceID: device,
		ID:       id,
		Type:     typ,
		Name:     name,
		Labels:   map[string][]string{"PluginHomeassistant": {"true"}},
		State:    state,
	}
}

func unlabeledEntity(plugin, device, id, typ, name string, state any) domain.Entity {
	return domain.Entity{
		Plugin:   plugin,
		DeviceID: device,
		ID:       id,
		Type:     typ,
		Name:     name,
		State:    state,
	}
}

// wsCollector connects to the HA WebSocket server, completes the handshake,
// and collects messages by type.
type wsCollector struct {
	mu   sync.Mutex
	msgs map[string][]json.RawMessage
	conn *websocket.Conn
}

func (m *multiEnv) connectWS(t *testing.T) *wsCollector {
	t.Helper()

	u := url.URL{Scheme: "ws", Host: "127.0.0.1:" + m.port, Path: "/ws"}

	var conn *websocket.Conn
	var err error
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}

	// Hello handshake
	conn.WriteJSON(map[string]string{"type": "hello", "client_id": "test-ha-client"})
	var hello map[string]any
	conn.ReadJSON(&hello)
	if hello["type"] != "hello" {
		t.Fatalf("expected hello, got %v", hello)
	}
	if auth, ok := hello["auth"].(bool); !ok || !auth {
		t.Fatalf("expected authorized hello, got %v", hello)
	}

	// Read snapshot (discard)
	var snap map[string]any
	conn.ReadJSON(&snap)

	wc := &wsCollector{
		msgs: make(map[string][]json.RawMessage),
		conn: conn,
	}
	t.Cleanup(func() { conn.Close() })

	go wc.readLoop()
	return wc
}

func (wc *wsCollector) readLoop() {
	for {
		_, data, err := wc.conn.ReadMessage()
		if err != nil {
			return
		}
		var msg struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(data, &msg) != nil {
			continue
		}
		wc.mu.Lock()
		wc.msgs[msg.Type] = append(wc.msgs[msg.Type], data)
		wc.mu.Unlock()
	}
}

func (wc *wsCollector) waitForType(t *testing.T, msgType string, count int) []json.RawMessage {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		wc.mu.Lock()
		n := len(wc.msgs[msgType])
		wc.mu.Unlock()
		if n >= count {
			wc.mu.Lock()
			defer wc.mu.Unlock()
			return wc.msgs[msgType][:count]
		}
		time.Sleep(50 * time.Millisecond)
	}
	wc.mu.Lock()
	defer wc.mu.Unlock()
	t.Fatalf("timed out waiting for %d %q messages, got %d", count, msgType, len(wc.msgs[msgType]))
	return nil
}

func (wc *wsCollector) countType(msgType string) int {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	return len(wc.msgs[msgType])
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestMulti_LabeledEntityAppearsInHA(t *testing.T) {
	m := newMultiEnv(t)
	ws := m.connectWS(t)

	// Save a labeled light entity from an external plugin.
	entity := labeledEntity("plugin-kasa", "living-room", "lamp1", "light", "Living Room Lamp", domain.Light{
		Power:      true,
		Brightness: 200,
	})
	m.saveEntity(entity)

	// Watch should fire OnAdd → entity_added broadcast.
	msgs := ws.waitForType(t, "entity_added", 1)

	var added struct {
		Entity struct {
			UniqueID string `json:"unique_id"`
			Platform string `json:"platform"`
			Name     string `json:"name"`
		} `json:"entity"`
	}
	if err := json.Unmarshal(msgs[0], &added); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if added.Entity.UniqueID != "plugin-kasa.living-room.lamp1" {
		t.Errorf("unique_id = %q, want plugin-kasa.living-room.lamp1", added.Entity.UniqueID)
	}
	if added.Entity.Platform != "light" {
		t.Errorf("platform = %q, want light", added.Entity.Platform)
	}
	if added.Entity.Name != "Living Room Lamp" {
		t.Errorf("name = %q, want Living Room Lamp", added.Entity.Name)
	}
}

func TestMulti_UnlabeledEntityExcluded(t *testing.T) {
	m := newMultiEnv(t)
	ws := m.connectWS(t)

	// Save an entity WITHOUT the PluginHomeassistant label.
	entity := unlabeledEntity("plugin-kasa", "bedroom", "lamp2", "light", "Bedroom Lamp", domain.Light{
		Power: true,
	})
	m.saveEntity(entity)

	// Give time for potential (incorrect) delivery.
	time.Sleep(500 * time.Millisecond)

	if n := ws.countType("entity_added"); n != 0 {
		t.Errorf("got %d entity_added messages for unlabeled entity, want 0", n)
	}
}

func TestMulti_CrossPluginEntity(t *testing.T) {
	m := newMultiEnv(t)
	ws := m.connectWS(t)

	// Entity from a completely different plugin, but labeled for HA.
	entity := labeledEntity("plugin-esphome", "garage", "door-sensor", "binary_sensor", "Garage Door", domain.BinarySensor{
		On:          true,
		DeviceClass: "door",
	})
	m.saveEntity(entity)

	msgs := ws.waitForType(t, "entity_added", 1)

	var added struct {
		Entity struct {
			UniqueID string         `json:"unique_id"`
			Platform string         `json:"platform"`
			State    map[string]any `json:"state"`
		} `json:"entity"`
	}
	json.Unmarshal(msgs[0], &added)

	if added.Entity.UniqueID != "plugin-esphome.garage.door-sensor" {
		t.Errorf("unique_id = %q, want plugin-esphome.garage.door-sensor", added.Entity.UniqueID)
	}
	if added.Entity.Platform != "binary_sensor" {
		t.Errorf("platform = %q, want binary_sensor", added.Entity.Platform)
	}
}

func TestMulti_UpdatePushesToHA(t *testing.T) {
	m := newMultiEnv(t)
	ws := m.connectWS(t)

	entity := labeledEntity("plugin-kasa", "office", "desk-lamp", "switch", "Desk Lamp", domain.Switch{
		Power: false,
	})
	m.saveEntity(entity)
	ws.waitForType(t, "entity_added", 1)

	// Update state — same entity, still labeled.
	entity.State = domain.Switch{Power: true}
	m.saveEntity(entity)

	msgs := ws.waitForType(t, "entity_updated", 1)

	var updated struct {
		Entity struct {
			State map[string]any `json:"state"`
		} `json:"entity"`
	}
	if err := json.Unmarshal(msgs[0], &updated); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	state, _ := updated.Entity.State["is_on"].(bool)
	if !state {
		t.Errorf("is_on = %v (full: %v), want true", state, updated.Entity.State)
	}
}

func TestMulti_LabelRemovalRemovesFromHA(t *testing.T) {
	m := newMultiEnv(t)
	ws := m.connectWS(t)

	entity := labeledEntity("plugin-kasa", "hall", "motion", "binary_sensor", "Hall Motion", domain.BinarySensor{
		On:          false,
		DeviceClass: "motion",
	})
	m.saveEntity(entity)
	ws.waitForType(t, "entity_added", 1)

	// Remove the label via sidecar and save again → OnRemove fires.
	entity.Labels = map[string][]string{}
	if err := m.store.Save(entity); err != nil {
		t.Fatalf("save entity: %v", err)
	}
	// Clear labels in sidecar.
	clearProfile, _ := json.Marshal(map[string]any{"labels": map[string][]string{}})
	if err := m.store.SetProfile(entity, json.RawMessage(clearProfile)); err != nil {
		t.Fatalf("clear profile: %v", err)
	}

	ws.waitForType(t, "entity_removed", 1)
}

func TestMulti_CommandRoutesToOwningPlugin(t *testing.T) {
	m := newMultiEnv(t)

	// Subscribe as the owning plugin to receive commands.
	cmds := messenger.NewCommands(m.env.Messenger(), domain.LookupCommand)
	var received struct {
		mu   sync.Mutex
		addr messenger.Address
		cmd  any
	}

	sub, err := cmds.Receive("plugin-kasa.>", func(addr messenger.Address, cmd any) {
		received.mu.Lock()
		received.addr = addr
		received.cmd = cmd
		received.mu.Unlock()
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	m.env.Messenger().Flush()
	defer sub.Unsubscribe()

	// Save entity so the server knows about it (via snapshot query).
	entity := labeledEntity("plugin-kasa", "living-room", "lamp1", "switch", "Lamp", domain.Switch{
		Power: false,
	})
	m.saveEntity(entity)
	// Wait for Watch to register it.
	time.Sleep(500 * time.Millisecond)

	// Connect WS and send command.
	ws := m.connectWS(t)
	_ = ws // consume snapshot

	wireID := app.WireID(entity)
	cmdMsg := map[string]any{
		"type":      "command",
		"id":        "test-cmd-1",
		"entity_id": wireID,
		"command":   "turn_on",
		"params":    map[string]any{},
	}
	if err := ws.conn.WriteJSON(cmdMsg); err != nil {
		t.Fatalf("write command: %v", err)
	}

	// Wait for command to arrive at the owning plugin.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		received.mu.Lock()
		got := received.cmd
		received.mu.Unlock()
		if got != nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	received.mu.Lock()
	defer received.mu.Unlock()
	if received.cmd == nil {
		t.Fatal("owning plugin did not receive the command")
	}
	if received.addr.Plugin != "plugin-kasa" {
		t.Errorf("addr.Plugin = %q, want plugin-kasa", received.addr.Plugin)
	}
	if received.addr.DeviceID != "living-room" {
		t.Errorf("addr.DeviceID = %q, want living-room", received.addr.DeviceID)
	}
	if received.addr.EntityID != "lamp1" {
		t.Errorf("addr.EntityID = %q, want lamp1", received.addr.EntityID)
	}
	if _, ok := received.cmd.(domain.SwitchTurnOn); !ok {
		t.Errorf("cmd type = %T, want SwitchTurnOn", received.cmd)
	}
}

func TestMulti_AllEntityTypesWithLabel(t *testing.T) {
	m := newMultiEnv(t)
	ws := m.connectWS(t)

	entities := []domain.Entity{
		labeledEntity("plugin-test", "dev", "sw", "switch", "SW", domain.Switch{Power: true}),
		labeledEntity("plugin-test", "dev", "lt", "light", "LT", domain.Light{Power: true, Brightness: 128}),
		labeledEntity("plugin-test", "dev", "cv", "cover", "CV", domain.Cover{Position: 50}),
		labeledEntity("plugin-test", "dev", "lk", "lock", "LK", domain.Lock{Locked: true}),
		labeledEntity("plugin-test", "dev", "fn", "fan", "FN", domain.Fan{Power: true}),
		labeledEntity("plugin-test", "dev", "sn", "sensor", "SN", domain.Sensor{Value: "42"}),
		labeledEntity("plugin-test", "dev", "bs", "binary_sensor", "BS", domain.BinarySensor{On: true}),
		labeledEntity("plugin-test", "dev", "cl", "climate", "CL", domain.Climate{HVACMode: "heat", Temperature: 22}),
		labeledEntity("plugin-test", "dev", "bt", "button", "BT", domain.Button{}),
		labeledEntity("plugin-test", "dev", "nm", "number", "NM", domain.Number{Value: 50, Min: 0, Max: 100, Step: 1}),
		labeledEntity("plugin-test", "dev", "sl", "select", "SL", domain.Select{Option: "home", Options: []string{"home", "away"}}),
		labeledEntity("plugin-test", "dev", "tx", "text", "TX", domain.Text{Value: "hi", Max: 100}),
		labeledEntity("plugin-test", "dev", "al", "alarm", "AL", domain.Alarm{AlarmState: "disarmed"}),
		labeledEntity("plugin-test", "dev", "cm", "camera", "CM", domain.Camera{}),
		labeledEntity("plugin-test", "dev", "vl", "valve", "VL", domain.Valve{Position: 100, ReportsPosition: true}),
		labeledEntity("plugin-test", "dev", "si", "siren", "SI", domain.Siren{IsOn: false}),
		labeledEntity("plugin-test", "dev", "hm", "humidifier", "HM", domain.Humidifier{IsOn: true, TargetHumidity: 50}),
		labeledEntity("plugin-test", "dev", "mp", "media_player", "MP", domain.MediaPlayer{State: "playing"}),
		labeledEntity("plugin-test", "dev", "rm", "remote", "RM", domain.Remote{IsOn: true}),
		labeledEntity("plugin-test", "dev", "ev", "event", "EV", domain.Event{EventTypes: []string{"press"}}),
	}

	for _, e := range entities {
		m.saveEntity(e)
	}

	// All 20 should arrive as entity_added.
	msgs := ws.waitForType(t, "entity_added", len(entities))

	seen := make(map[string]bool)
	for _, raw := range msgs {
		var msg struct {
			Entity struct {
				UniqueID string `json:"unique_id"`
			} `json:"entity"`
		}
		json.Unmarshal(raw, &msg)
		seen[msg.Entity.UniqueID] = true
	}

	for _, e := range entities {
		if !seen[e.Key()] {
			t.Errorf("entity %s not received via entity_added", e.Key())
		}
	}
}

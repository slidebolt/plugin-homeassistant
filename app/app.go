package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"

	contract "github.com/slidebolt/sb-contract"
	domain "github.com/slidebolt/sb-domain"
	messenger "github.com/slidebolt/sb-messenger-sdk"
	storage "github.com/slidebolt/sb-storage-sdk"
)

const PluginID = "plugin-homeassistant"

// App is the importable runtime for the plugin-homeassistant binary.
// Keep production behavior here so tests can exercise it without importing cmd/.
type App struct {
	msg    messenger.Messenger
	store  storage.Storage
	cmds   *messenger.Commands
	subs   []messenger.Subscription
	watch  *storage.Watcher
	cfg    Config
	ctx    context.Context
	cancel context.CancelFunc
	srv    *haServer
}

func New() *App {
	return &App{}
}

func (a *App) Hello() contract.HelloResponse {
	return contract.HelloResponse{
		ID:              PluginID,
		Kind:            contract.KindPlugin,
		ContractVersion: contract.ContractVersion,
		DependsOn:       []string{"messenger", "storage"},
	}
}

func (a *App) OnStart(deps map[string]json.RawMessage) (json.RawMessage, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	a.cfg = cfg
	a.ctx, a.cancel = context.WithCancel(context.Background())

	msg, err := messenger.Connect(deps)
	if err != nil {
		return nil, fmt.Errorf("connect messenger: %w", err)
	}
	a.msg = msg

	store, err := storage.Connect(deps)
	if err != nil {
		return nil, fmt.Errorf("connect storage: %w", err)
	}
	a.store = store

	a.cmds = messenger.NewCommands(msg, domain.LookupCommand)

	// Subscribe to commands for entities owned by this plugin.
	sub, err := a.cmds.Receive(PluginID+".>", a.handleCommand)
	if err != nil {
		return nil, fmt.Errorf("subscribe commands: %w", err)
	}
	a.subs = append(a.subs, sub)

	a.srv = newServer(a.cfg, a.store, a.cmds)
	port, err := a.srv.Start()
	if err != nil {
		return nil, fmt.Errorf("start server: %w", err)
	}

	// haFingerprint returns a JSON string of the entity's Home Assistant wire
	// representation, but with the dynamic state field zeroed out. This
	// captures changes to configuration (names, labels, supported features,
	// units) while ignoring simple state updates.
	haFingerprint := func(data json.RawMessage) string {
		var entity domain.Entity
		if err := json.Unmarshal(data, &entity); err != nil {
			return ""
		}
		we := entityToWire(entity)
		we.State = nil // Ignore dynamic state for the fingerprint
		b, _ := json.Marshal(we)
		return string(b)
	}

	// Watch for entities labeled PluginHomeassistant across all plugins.
	labelQuery := storage.Query{
		Where: []storage.Filter{{Field: "labels.PluginHomeassistant", Op: storage.Exists}},
	}
	a.watch, err = storage.Watch(msg, labelQuery, storage.WatchHandlers{
		OnAdd: func(key string, data json.RawMessage) {
			var entity domain.Entity
			if err := json.Unmarshal(data, &entity); err != nil {
				return
			}
			we := entityToWire(entity)
			a.srv.Broadcast(wireMessage{Type: "entity_added", Entity: &we})
			log.Printf("plugin-homeassistant: entity added %s", key)
		},
		OnRemove: func(key string, data json.RawMessage) {
			var entity domain.Entity
			if err := json.Unmarshal(data, &entity); err != nil {
				return
			}
			we := entityToWire(entity)
			a.srv.Broadcast(wireMessage{Type: "entity_removed", Entity: &we})
			log.Printf("plugin-homeassistant: entity removed %s", key)
		},
		OnCapabilityUpdate: func(key string, data json.RawMessage) {
			var entity domain.Entity
			if err := json.Unmarshal(data, &entity); err != nil {
				return
			}
			// Configuration changed (names, units, features, etc).
			// Signal HA to re-discover the entity by broadcasting entity_added.
			we := entityToWire(entity)
			a.srv.Broadcast(wireMessage{Type: "entity_added", Entity: &we})
			log.Printf("plugin-homeassistant: entity configuration updated %s", key)
		},
		OnStateUpdate: func(key string, data json.RawMessage) {
			var entity domain.Entity
			if err := json.Unmarshal(data, &entity); err != nil {
				return
			}
			// Only state changed. Standard update.
			we := entityToWire(entity)
			a.srv.Broadcast(wireMessage{Type: "entity_updated", Entity: &we})
		},
		Fingerprint: haFingerprint,
	})
	if err != nil {
		return nil, fmt.Errorf("watch labeled entities: %w", err)
	}

	// Populate the watcher with pre-existing entities and send initial discovery.
	entries, err := store.Query(labelQuery)
	if err != nil {
		return nil, fmt.Errorf("initial entity query: %w", err)
	}
	for _, entry := range entries {
		var entity domain.Entity
		if err := json.Unmarshal(entry.Data, &entity); err != nil {
			continue
		}
		a.watch.Populate(entry.Key, entry.Data)
		we := entityToWire(entity)
		a.srv.Broadcast(wireMessage{Type: "entity_added", Entity: &we})
	}

	log.Printf("plugin-homeassistant: started on port %d, advertising via mDNS", port)
	return nil, nil
}

// Port returns the port the WebSocket server is listening on.
func (a *App) Port() string {
	if a.srv != nil && a.srv.ln != nil {
		return fmt.Sprintf("%d", a.srv.ln.Addr().(*net.TCPAddr).Port)
	}
	return "0"
}

func (a *App) OnShutdown() error {
	if a.watch != nil {
		a.watch.Stop()
	}
	if a.srv != nil {
		a.srv.Stop()
	}
	for _, sub := range a.subs {
		sub.Unsubscribe()
	}
	if a.store != nil {
		a.store.Close()
	}
	if a.msg != nil {
		a.msg.Close()
	}
	if a.cancel != nil {
		a.cancel()
	}
	return nil
}

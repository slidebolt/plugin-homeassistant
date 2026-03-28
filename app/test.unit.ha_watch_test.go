package app

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	domain "github.com/slidebolt/sb-domain"
	testkit "github.com/slidebolt/sb-testkit"
	storage "github.com/slidebolt/sb-storage-sdk"
)

// collector gathers Watch callbacks in a thread-safe way.
type msgCollector struct {
	mu   sync.Mutex
	msgs []wireMessage
}

func (c *msgCollector) Broadcast(msg wireMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgs = append(c.msgs, msg)
}

func (c *msgCollector) wait(t *testing.T, count int) []wireMessage {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		n := len(c.msgs)
		c.mu.Unlock()
		if n >= count {
			c.mu.Lock()
			out := append([]wireMessage(nil), c.msgs...)
			c.mu.Unlock()
			return out
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d messages, got %d", count, len(c.msgs))
	return nil
}

func TestHAWatch_CapabilityUpdate(t *testing.T) {
	e := testkit.NewTestEnv(t)
	e.Start("messenger")
	e.Start("storage")
	msg := e.Messenger()
	store := e.Storage()

	// 1. Pre-existing entity with PluginHomeassistant label
	ent := domain.Entity{
		ID: "light1", Plugin: "test", DeviceID: "dev1",
		Type: "light", Name: "Living Room",
		State: domain.Light{Power: false, Brightness: 0},
	}
	if err := store.Save(ent); err != nil {
		t.Fatal(err)
	}
	profile := json.RawMessage(`{"labels":{"PluginHomeassistant":["true"]}}`)
	if err := store.SetProfile(ent, profile); err != nil {
		t.Fatal(err)
	}
	// Re-fetch to get merged entity for initial population
	entRaw, _ := store.Get(ent)
	json.Unmarshal(entRaw, &ent)

	app := New()
	app.store = store
	app.msg = msg
	app.cfg = Config{Port: "0"}
	// We need to trigger OnStart or manually set up the watch
	// Since OnStart does a lot, let's just manually setup what we need.
	app.srv = newServer(app.cfg, store, nil)
	collector := &msgCollector{}
	app.srv.mockBroadcaster = collector

	// Mirror the setup in app.go
	haFingerprint := func(data json.RawMessage) string {
		var entity domain.Entity
		if err := json.Unmarshal(data, &entity); err != nil {
			return ""
		}
		we := entityToWire(entity)
		we.State = nil
		b, _ := json.Marshal(we)
		return string(b)
	}

	labelQuery := storage.Query{
		Where: []storage.Filter{{Field: "labels.PluginHomeassistant", Op: storage.Exists}},
	}
	var err error
	app.watch, err = storage.Watch(msg, labelQuery, storage.WatchHandlers{
		OnCapabilityUpdate: func(key string, data json.RawMessage) {
			var entity domain.Entity
			json.Unmarshal(data, &entity)
			we := entityToWire(entity)
			app.srv.Broadcast(wireMessage{Type: "entity_added", Entity: &we})
		},
		OnStateUpdate: func(key string, data json.RawMessage) {
			var entity domain.Entity
			json.Unmarshal(data, &entity)
			we := entityToWire(entity)
			app.srv.Broadcast(wireMessage{Type: "entity_updated", Entity: &we})
		},
		Fingerprint: haFingerprint,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer app.watch.Stop()

	// Initial population (simulates what OnStart does)
	app.watch.Populate(ent.Key(), mustMarshal(ent))
	// Initial discovery broadcast
	we := entityToWire(ent)
	app.srv.Broadcast(wireMessage{Type: "entity_added", Entity: &we})

	// Wait for initial broadcast
	collector.wait(t, 1)

	// 2. State-only update
	ent.State = domain.Light{Power: true, Brightness: 254}
	if err := store.Save(ent); err != nil {
		t.Fatal(err)
	}

	msgs := collector.wait(t, 2)
	if msgs[1].Type != "entity_updated" {
		t.Errorf("state-only change: got type %q, want \"entity_updated\"", msgs[1].Type)
	}

	// 3. Capability change (name change)
	ent.Name = "New Name"
	if err := store.Save(ent); err != nil {
		t.Fatal(err)
	}

	msgs = collector.wait(t, 3)
	if msgs[2].Type != "entity_added" {
		t.Errorf("capability change: got type %q, want \"entity_added\"", msgs[2].Type)
	}
	if msgs[2].Entity.Name != "New Name" {
		t.Errorf("name: got %q, want \"New Name\"", msgs[2].Entity.Name)
	}
}

func mustMarshal(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

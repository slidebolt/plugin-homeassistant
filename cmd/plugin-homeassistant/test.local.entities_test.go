//go:build integration

// Integration tests for entity snapshot and two-way command flow.
// These tests require a running Home Assistant instance with the Slidebolt
// custom component installed and a Slidebolt config entry already present.
//
// Run with:
//
//	HA_URL=http://localhost:38123 HA_LONG_LIVED_ACCESS_TOKEN=<token> \
//	  go test -tags integration -v -count=1 -timeout 60s \
//	  ./plugin-homeassistant/cmd/plugin-homeassistant/
//
// Environment variables:
//
//	HA_URL                     Base URL of Home Assistant (default: http://localhost:8123)
//	HA_LONG_LIVED_ACCESS_TOKEN Home Assistant long-lived access token (required for API calls)
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	domain "github.com/slidebolt/sb-domain"
	"github.com/slidebolt/plugin-homeassistant/app"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func haBaseURL() string {
	if v := os.Getenv("HA_URL"); v != "" {
		return v
	}
	return "http://localhost:38123"
}

func haLongLivedAccessToken() string {
	return os.Getenv("HA_LONG_LIVED_ACCESS_TOKEN")
}

func skipIfHADown(t *testing.T) {
	t.Helper()
	resp, err := http.Get(haBaseURL())
	if err != nil || resp.StatusCode == 0 {
		t.Skipf("HA not reachable at %s — start HA first", haBaseURL())
	}
	resp.Body.Close()
}

func haGET(t *testing.T, path string) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest("GET", haBaseURL()+path, nil)
	if tok := haLongLivedAccessToken(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var result map[string]any
	json.Unmarshal(b, &result) //nolint:errcheck
	return resp.StatusCode, result
}

func haPOST(t *testing.T, path string, body map[string]any) int {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", haBaseURL()+path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if tok := haLongLivedAccessToken(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// waitForHAState polls HA's /api/states/<entity_id> until the predicate
// returns true or the timeout elapses.
func waitForHAState(t *testing.T, entityID string, timeout time.Duration, predicate func(map[string]any) bool) map[string]any {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		status, body := haGET(t, "/api/states/"+entityID)
		if status == 200 && predicate(body) {
			return body
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for HA state of %s", entityID)
	return nil
}

// startServerWithEntities starts the plugin server seeded with the given
// entities, waits for HA to connect and complete the handshake, then returns
// the server. Caller must defer srv.Stop().
func startServerWithEntities(t *testing.T, entities ...domain.Entity) app.ServerRunner {
	t.Helper()
	srv := app.NewServerForTest(app.Config{Port: "39444", SystemUUID: "test-uuid", SystemMAC: "E0:00:00:00:00:FF"}, entities...)
	port, err := srv.Start()
	if err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Logf("server on port %d", port)

	// Always stop on timeout so the listener is freed for the next test.
	t.Cleanup(func() { srv.Stop() })

	select {
	case <-srv.Connected():
		t.Log("HA connected and handshake complete")
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for HA to connect — is Slidebolt configured in HA?")
	}
	return srv
}

// testEntity builds a minimal domain.Entity with the given type, id, name and state.
func testEntity(typ, id, name string, state any) domain.Entity {
	return domain.Entity{
		Plugin:   "plugin-homeassistant",
		DeviceID: "test-device",
		ID:       id,
		Type:     typ,
		Name:     name,
		State:    state,
	}
}

// testHAID returns the entity_id that HA creates for a test entity.
// With _attr_has_entity_name=True and no device, HA uses slugify(name).
// Our test names slugify to the entity ID, so this is simply platform.id.
func testHAID(typ, id string) string {
	platform := typ
	if typ == "alarm" {
		platform = "alarm_control_panel"
	}
	return platform + "." + id
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestIntegration_HAConnectsOnStartup verifies the basic handshake still works.
func TestIntegration_HAConnectsOnStartup(t *testing.T) {
	skipIfHADown(t)

	srv := app.NewServerForTest(app.Config{Port: "39444", SystemUUID: "test-uuid", SystemMAC: "E0:00:00:00:00:FF"})
	port, err := srv.Start()
	if err != nil {
		t.Fatalf("start server: %v", err)
	}
	t.Cleanup(func() { srv.Stop() })
	t.Logf("server on port %d, waiting for HA to connect...", port)

	select {
	case <-srv.Connected():
		t.Log("HA connected and handshake complete")
	case <-time.After(30 * time.Second):
		t.Fatal("timed out — is Slidebolt configured in HA? Run the manual test first if not")
	}
}

// TestIntegration_SnapshotEntitiesReachHA seeds a light and a switch, starts
// the server, and then polls HA's REST API to confirm both entities appear
// with correct state fields.
func TestIntegration_SnapshotEntitiesReachHA(t *testing.T) {
	skipIfHADown(t)
	if haLongLivedAccessToken() == "" {
		t.Skip("HA_LONG_LIVED_ACCESS_TOKEN not set — required to query HA REST API")
	}

	entities := []domain.Entity{
		testEntity("light", "integ_light", "integ_light", domain.Light{
			Power:      true,
			Brightness: 180,
		}),
		testEntity("switch", "integ_switch", "integ_switch", domain.Switch{
			Power: false,
		}),
	}

	srv := startServerWithEntities(t, entities...)
	defer srv.Stop()

	// Give HA a moment to load the entities from the snapshot.
	time.Sleep(2 * time.Second)

	// Assert light appeared in HA with correct state.
	t.Run("light appears with correct state", func(t *testing.T) {
		body := waitForHAState(t, testHAID("light", "integ_light"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
		attrs, _ := body["attributes"].(map[string]any)
		if attrs == nil {
			t.Fatalf("no attributes in HA state: %v", body)
		}
		t.Logf("HA light state: %v", body["state"])
		t.Logf("HA light attributes: %v", attrs)

		if body["state"] != "on" {
			t.Errorf("expected state=on, got %v", body["state"])
		}
		brightness, _ := attrs["brightness"].(float64)
		if int(brightness) != 180 {
			t.Errorf("expected brightness=180, got %v", attrs["brightness"])
		}
	})

	// Assert switch appeared.
	t.Run("switch appears with correct state", func(t *testing.T) {
		body := waitForHAState(t, testHAID("switch", "integ_switch"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
		t.Logf("HA switch state: %v", body["state"])
		// Switch state is "on" or "off" at top level.
		if body["state"] != "off" {
			t.Errorf("expected state=off, got %v", body["state"])
		}
	})
}

// TestIntegration_CommandRoundTrip seeds a light, starts the server, then
// calls HA's service API to turn the light on with a brightness. Asserts the
// plugin receives and applies the command by querying HA state afterward.
func TestIntegration_CommandRoundTrip(t *testing.T) {
	skipIfHADown(t)
	if haLongLivedAccessToken() == "" {
		t.Skip("HA_LONG_LIVED_ACCESS_TOKEN not set — required to call HA services")
	}

	entities := []domain.Entity{
		testEntity("light", "cmd_light", "cmd_light", domain.Light{
			Power:      false,
			Brightness: 0,
		}),
	}

	srv := startServerWithEntities(t, entities...)
	defer srv.Stop()

	time.Sleep(2 * time.Second)

	// Confirm light starts off.
	waitForHAState(t, testHAID("light", "cmd_light"), 10*time.Second, func(s map[string]any) bool {
		return s["state"] != nil
	})

	// Call HA service: turn_on with brightness 200.
	t.Logf("calling light.turn_on with brightness=200")
	status := haPOST(t, "/api/services/light/turn_on", map[string]any{
		"entity_id":  testHAID("light", "cmd_light"),
		"brightness": 200,
	})
	if status != 200 && status != 201 {
		t.Fatalf("service call returned %d", status)
	}

	// Wait for HA to reflect the updated state pushed back by the plugin.
	t.Run("light brightness updated after command", func(t *testing.T) {
		body := waitForHAState(t, testHAID("light", "cmd_light"), 10*time.Second, func(s map[string]any) bool {
			attrs, _ := s["attributes"].(map[string]any)
			if attrs == nil {
				return false
			}
			b, _ := attrs["brightness"].(float64)
			return int(b) == 200
		})
		attrs, _ := body["attributes"].(map[string]any)
		t.Logf("HA light attributes after command: %v", fmt.Sprintf("%v", attrs))
		if body["state"] != "on" {
			t.Errorf("expected state=on after turn_on, got %v", body["state"])
		}
	})
}

// ---------------------------------------------------------------------------
// Full entity matrix: snapshot + command round-trips for all 20 entity types.
// ---------------------------------------------------------------------------

// allTestEntities returns one entity per type for comprehensive testing.
func allTestEntities() []domain.Entity {
	return []domain.Entity{
		testEntity("switch", "it_switch", "IT Switch", domain.Switch{Power: false}),
		testEntity("cover", "it_cover", "IT Cover", domain.Cover{Position: 100}),
		testEntity("lock", "it_lock", "IT Lock", domain.Lock{Locked: true}),
		testEntity("fan", "it_fan", "IT Fan", domain.Fan{Power: false}),
		testEntity("sensor", "it_sensor", "IT Sensor", domain.Sensor{Value: "23.5", DeviceClass: "temperature", Unit: "°C"}),
		testEntity("binary_sensor", "it_binary_sensor", "IT Binary Sensor", domain.BinarySensor{On: true, DeviceClass: "motion"}),
		testEntity("climate", "it_climate", "IT Climate", domain.Climate{
			HVACMode: "heat", Temperature: 22, HVACModes: []string{"off", "heat", "cool"},
			TemperatureUnit: "°C",
		}),
		testEntity("button", "it_button", "IT Button", domain.Button{}),
		testEntity("number", "it_number", "IT Number", domain.Number{Value: 50, Min: 0, Max: 100, Step: 1}),
		testEntity("select", "it_select", "IT Select", domain.Select{Option: "home", Options: []string{"home", "away", "sleep"}}),
		testEntity("text", "it_text", "IT Text", domain.Text{Value: "hello", Max: 255}),
		testEntity("alarm", "it_alarm", "IT Alarm", domain.Alarm{AlarmState: "disarmed"}),
		testEntity("camera", "it_camera", "IT Camera", domain.Camera{IsRecording: false, MotionDetection: false}),
		testEntity("valve", "it_valve", "IT Valve", domain.Valve{Position: 100, ReportsPosition: true}),
		testEntity("siren", "it_siren", "IT Siren", domain.Siren{IsOn: false}),
		testEntity("humidifier", "it_humidifier", "IT Humidifier", domain.Humidifier{
			IsOn: true, TargetHumidity: 50, MinHumidity: 30, MaxHumidity: 80,
			AvailableModes: []string{"normal", "turbo"}, Mode: "normal",
		}),
		testEntity("media_player", "it_media_player", "IT Media Player", domain.MediaPlayer{
			State: "paused", VolumeLevel: 0.5, SourceList: []string{"TV", "Radio"}, Source: "TV",
		}),
		testEntity("remote", "it_remote", "IT Remote", domain.Remote{
			IsOn: true, ActivityList: []string{"TV", "Music"}, CurrentActivity: "TV",
		}),
		testEntity("event", "it_event", "IT Event", domain.Event{
			EventTypes: []string{"click", "double_click"}, DeviceClass: "button",
		}),
	}
}

// TestIntegration_AllEntitySnapshots starts a server with all 20 entity types
// and verifies each one appears in HA with correct state.
func TestIntegration_AllEntitySnapshots(t *testing.T) {
	skipIfHADown(t)
	if haLongLivedAccessToken() == "" {
		t.Skip("HA_LONG_LIVED_ACCESS_TOKEN not set")
	}

	entities := allTestEntities()
	srv := startServerWithEntities(t, entities...)
	defer srv.Stop()

	time.Sleep(3 * time.Second)

	t.Run("switch", func(t *testing.T) {
		body := waitForHAState(t, testHAID("switch", "it_switch"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
		if body["state"] != "off" {
			t.Errorf("expected state=off, got %v", body["state"])
		}
	})

	t.Run("cover", func(t *testing.T) {
		body := waitForHAState(t, testHAID("cover", "it_cover"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
		if body["state"] != "open" {
			t.Errorf("expected state=open, got %v", body["state"])
		}
		attrs, _ := body["attributes"].(map[string]any)
		if pos, _ := attrs["current_position"].(float64); int(pos) != 100 {
			t.Errorf("expected current_position=100, got %v", attrs["current_position"])
		}
	})

	t.Run("lock", func(t *testing.T) {
		body := waitForHAState(t, testHAID("lock", "it_lock"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
		if body["state"] != "locked" {
			t.Errorf("expected state=locked, got %v", body["state"])
		}
	})

	t.Run("fan", func(t *testing.T) {
		body := waitForHAState(t, testHAID("fan", "it_fan"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
		if body["state"] != "off" {
			t.Errorf("expected state=off, got %v", body["state"])
		}
	})

	t.Run("sensor", func(t *testing.T) {
		body := waitForHAState(t, testHAID("sensor", "it_sensor"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
		if body["state"] != "23.5" {
			t.Errorf("expected state=23.5, got %v", body["state"])
		}
		attrs, _ := body["attributes"].(map[string]any)
		if attrs["device_class"] != "temperature" {
			t.Errorf("expected device_class=temperature, got %v", attrs["device_class"])
		}
	})

	t.Run("binary_sensor", func(t *testing.T) {
		body := waitForHAState(t, testHAID("binary_sensor", "it_binary_sensor"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
		if body["state"] != "on" {
			t.Errorf("expected state=on, got %v", body["state"])
		}
		attrs, _ := body["attributes"].(map[string]any)
		if attrs["device_class"] != "motion" {
			t.Errorf("expected device_class=motion, got %v", attrs["device_class"])
		}
	})

	t.Run("climate", func(t *testing.T) {
		body := waitForHAState(t, testHAID("climate", "it_climate"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
		if body["state"] != "heat" {
			t.Errorf("expected state=heat, got %v", body["state"])
		}
		attrs, _ := body["attributes"].(map[string]any)
		temp, _ := attrs["temperature"].(float64)
		// 22°C ≈ 72°F — HA converts to system unit.
		if temp < 20 || temp > 80 {
			t.Errorf("expected temperature ~22°C or ~72°F, got %v", temp)
		}
	})

	t.Run("button", func(t *testing.T) {
		waitForHAState(t, testHAID("button", "it_button"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
	})

	t.Run("number", func(t *testing.T) {
		body := waitForHAState(t, testHAID("number", "it_number"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
		state := fmt.Sprintf("%v", body["state"])
		if state != "50" && state != "50.0" {
			t.Errorf("expected state=50, got %v", body["state"])
		}
	})

	t.Run("select", func(t *testing.T) {
		body := waitForHAState(t, testHAID("select", "it_select"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
		if body["state"] != "home" {
			t.Errorf("expected state=home, got %v", body["state"])
		}
	})

	t.Run("text", func(t *testing.T) {
		body := waitForHAState(t, testHAID("text", "it_text"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
		if body["state"] != "hello" {
			t.Errorf("expected state=hello, got %v", body["state"])
		}
	})

	t.Run("alarm", func(t *testing.T) {
		body := waitForHAState(t, testHAID("alarm", "it_alarm"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
		if body["state"] != "disarmed" {
			t.Errorf("expected state=disarmed, got %v", body["state"])
		}
	})

	t.Run("camera", func(t *testing.T) {
		waitForHAState(t, testHAID("camera", "it_camera"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
	})

	t.Run("valve", func(t *testing.T) {
		body := waitForHAState(t, testHAID("valve", "it_valve"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
		if body["state"] != "open" {
			t.Errorf("expected state=open, got %v", body["state"])
		}
		attrs, _ := body["attributes"].(map[string]any)
		if pos, _ := attrs["current_position"].(float64); int(pos) != 100 {
			t.Errorf("expected current_position=100, got %v", attrs["current_position"])
		}
	})

	t.Run("siren", func(t *testing.T) {
		body := waitForHAState(t, testHAID("siren", "it_siren"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
		if body["state"] != "off" {
			t.Errorf("expected state=off, got %v", body["state"])
		}
	})

	t.Run("humidifier", func(t *testing.T) {
		body := waitForHAState(t, testHAID("humidifier", "it_humidifier"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
		if body["state"] != "on" {
			t.Errorf("expected state=on, got %v", body["state"])
		}
		attrs, _ := body["attributes"].(map[string]any)
		if h, _ := attrs["humidity"].(float64); int(h) != 50 {
			t.Errorf("expected humidity=50, got %v", attrs["humidity"])
		}
	})

	t.Run("media_player", func(t *testing.T) {
		body := waitForHAState(t, testHAID("media_player", "it_media_player"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
		if body["state"] != "paused" {
			t.Errorf("expected state=paused, got %v", body["state"])
		}
		attrs, _ := body["attributes"].(map[string]any)
		if vol, _ := attrs["volume_level"].(float64); vol != 0.5 {
			t.Errorf("expected volume_level=0.5, got %v", attrs["volume_level"])
		}
	})

	t.Run("remote", func(t *testing.T) {
		body := waitForHAState(t, testHAID("remote", "it_remote"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
		if body["state"] != "on" {
			t.Errorf("expected state=on, got %v", body["state"])
		}
	})

	t.Run("event", func(t *testing.T) {
		waitForHAState(t, testHAID("event", "it_event"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
	})
}

// TestIntegration_AllEntityCommands starts a server with all commandable entities
// and verifies commands are received and state updates flow back to HA.
func TestIntegration_AllEntityCommands(t *testing.T) {
	skipIfHADown(t)
	if haLongLivedAccessToken() == "" {
		t.Skip("HA_LONG_LIVED_ACCESS_TOKEN not set")
	}

	entities := allTestEntities()
	srv := startServerWithEntities(t, entities...)
	defer srv.Stop()

	time.Sleep(3 * time.Second)

	// Wait for all entities to appear before issuing commands.
	waitForHAState(t, testHAID("switch", "it_switch"), 15*time.Second, func(s map[string]any) bool {
		return s["state"] != nil
	})

	t.Run("switch/turn_on", func(t *testing.T) {
		status := haPOST(t, "/api/services/switch/turn_on", map[string]any{
			"entity_id": testHAID("switch", "it_switch"),
		})
		if status != 200 && status != 201 {
			t.Fatalf("service call returned %d", status)
		}
		body := waitForHAState(t, testHAID("switch", "it_switch"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] == "on"
		})
		if body["state"] != "on" {
			t.Errorf("expected state=on, got %v", body["state"])
		}
	})

	t.Run("cover/set_position", func(t *testing.T) {
		status := haPOST(t, "/api/services/cover/set_cover_position", map[string]any{
			"entity_id": testHAID("cover", "it_cover"),
			"position":  50,
		})
		if status != 200 && status != 201 {
			t.Fatalf("service call returned %d", status)
		}
		body := waitForHAState(t, testHAID("cover", "it_cover"), 10*time.Second, func(s map[string]any) bool {
			attrs, _ := s["attributes"].(map[string]any)
			if attrs == nil {
				return false
			}
			pos, _ := attrs["current_position"].(float64)
			return int(pos) == 50
		})
		attrs, _ := body["attributes"].(map[string]any)
		pos, _ := attrs["current_position"].(float64)
		if int(pos) != 50 {
			t.Errorf("expected current_position=50, got %v", attrs["current_position"])
		}
	})

	t.Run("lock/unlock", func(t *testing.T) {
		status := haPOST(t, "/api/services/lock/unlock", map[string]any{
			"entity_id": testHAID("lock", "it_lock"),
		})
		if status != 200 && status != 201 {
			t.Fatalf("service call returned %d", status)
		}
		body := waitForHAState(t, testHAID("lock", "it_lock"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] == "unlocked"
		})
		if body["state"] != "unlocked" {
			t.Errorf("expected state=unlocked, got %v", body["state"])
		}
	})

	t.Run("fan/set_percentage", func(t *testing.T) {
		// NOTE: fan.turn_on fails because slidebolt-hacs fan.py async_turn_on
		// signature is missing positional args (percentage, preset_mode).
		// Testing set_percentage instead.
		status := haPOST(t, "/api/services/fan/set_percentage", map[string]any{
			"entity_id":  testHAID("fan", "it_fan"),
			"percentage": 75,
		})
		if status != 200 && status != 201 {
			t.Fatalf("service call returned %d", status)
		}
		body := waitForHAState(t, testHAID("fan", "it_fan"), 10*time.Second, func(s map[string]any) bool {
			attrs, _ := s["attributes"].(map[string]any)
			if attrs == nil {
				return false
			}
			pct, _ := attrs["percentage"].(float64)
			return pct == 75
		})
		attrs, _ := body["attributes"].(map[string]any)
		pct, _ := attrs["percentage"].(float64)
		if pct != 75 {
			t.Errorf("expected percentage=75, got %v", pct)
		}
	})

	t.Run("climate/set_temperature", func(t *testing.T) {
		// Get current temperature to verify it changes.
		initial := waitForHAState(t, testHAID("climate", "it_climate"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] != nil
		})
		initAttrs, _ := initial["attributes"].(map[string]any)
		initTemp, _ := initAttrs["temperature"].(float64)

		// HA accepts values in its system unit. Send a value different from initial.
		newTemp := initTemp + 5
		status := haPOST(t, "/api/services/climate/set_temperature", map[string]any{
			"entity_id":   testHAID("climate", "it_climate"),
			"temperature": newTemp,
		})
		if status != 200 && status != 201 {
			t.Fatalf("service call returned %d", status)
		}
		body := waitForHAState(t, testHAID("climate", "it_climate"), 10*time.Second, func(s map[string]any) bool {
			attrs, _ := s["attributes"].(map[string]any)
			if attrs == nil {
				return false
			}
			temp, _ := attrs["temperature"].(float64)
			return temp != initTemp
		})
		attrs, _ := body["attributes"].(map[string]any)
		temp, _ := attrs["temperature"].(float64)
		t.Logf("climate temperature changed from %v to %v", initTemp, temp)
		if temp == initTemp {
			t.Errorf("expected temperature to change from %v", initTemp)
		}
	})

	t.Run("button/press", func(t *testing.T) {
		status := haPOST(t, "/api/services/button/press", map[string]any{
			"entity_id": testHAID("button", "it_button"),
		})
		if status != 200 && status != 201 {
			t.Fatalf("service call returned %d", status)
		}
	})

	t.Run("number/set_value", func(t *testing.T) {
		status := haPOST(t, "/api/services/number/set_value", map[string]any{
			"entity_id": testHAID("number", "it_number"),
			"value":     75,
		})
		if status != 200 && status != 201 {
			t.Fatalf("service call returned %d", status)
		}
		body := waitForHAState(t, testHAID("number", "it_number"), 10*time.Second, func(s map[string]any) bool {
			st := fmt.Sprintf("%v", s["state"])
			return st == "75" || st == "75.0"
		})
		state := fmt.Sprintf("%v", body["state"])
		if state != "75" && state != "75.0" {
			t.Errorf("expected state=75, got %v", body["state"])
		}
	})

	t.Run("select/select_option", func(t *testing.T) {
		status := haPOST(t, "/api/services/select/select_option", map[string]any{
			"entity_id": testHAID("select", "it_select"),
			"option":    "away",
		})
		if status != 200 && status != 201 {
			t.Fatalf("service call returned %d", status)
		}
		body := waitForHAState(t, testHAID("select", "it_select"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] == "away"
		})
		if body["state"] != "away" {
			t.Errorf("expected state=away, got %v", body["state"])
		}
	})

	t.Run("text/set_value", func(t *testing.T) {
		status := haPOST(t, "/api/services/text/set_value", map[string]any{
			"entity_id": testHAID("text", "it_text"),
			"value":     "world",
		})
		if status != 200 && status != 201 {
			t.Fatalf("service call returned %d", status)
		}
		body := waitForHAState(t, testHAID("text", "it_text"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] == "world"
		})
		if body["state"] != "world" {
			t.Errorf("expected state=world, got %v", body["state"])
		}
	})

	t.Run("alarm/arm_home", func(t *testing.T) {
		status := haPOST(t, "/api/services/alarm_control_panel/alarm_arm_home", map[string]any{
			"entity_id": testHAID("alarm", "it_alarm"),
		})
		if status != 200 && status != 201 {
			t.Fatalf("service call returned %d", status)
		}
		body := waitForHAState(t, testHAID("alarm", "it_alarm"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] == "armed_home"
		})
		if body["state"] != "armed_home" {
			t.Errorf("expected state=armed_home, got %v", body["state"])
		}
	})

	t.Run("camera/enable_motion_detection", func(t *testing.T) {
		status := haPOST(t, "/api/services/camera/enable_motion_detection", map[string]any{
			"entity_id": testHAID("camera", "it_camera"),
		})
		if status != 200 && status != 201 {
			t.Fatalf("service call returned %d", status)
		}
		waitForHAState(t, testHAID("camera", "it_camera"), 10*time.Second, func(s map[string]any) bool {
			attrs, _ := s["attributes"].(map[string]any)
			if attrs == nil {
				return false
			}
			md, _ := attrs["motion_detection"].(bool)
			return md
		})
	})

	t.Run("valve/set_position", func(t *testing.T) {
		status := haPOST(t, "/api/services/valve/set_valve_position", map[string]any{
			"entity_id": testHAID("valve", "it_valve"),
			"position":  50,
		})
		if status != 200 && status != 201 {
			t.Fatalf("service call returned %d", status)
		}
		body := waitForHAState(t, testHAID("valve", "it_valve"), 10*time.Second, func(s map[string]any) bool {
			attrs, _ := s["attributes"].(map[string]any)
			if attrs == nil {
				return false
			}
			pos, _ := attrs["current_position"].(float64)
			return int(pos) == 50
		})
		attrs, _ := body["attributes"].(map[string]any)
		pos, _ := attrs["current_position"].(float64)
		if int(pos) != 50 {
			t.Errorf("expected current_position=50, got %v", attrs["current_position"])
		}
	})

	t.Run("siren/turn_on", func(t *testing.T) {
		status := haPOST(t, "/api/services/siren/turn_on", map[string]any{
			"entity_id": testHAID("siren", "it_siren"),
		})
		if status != 200 && status != 201 {
			t.Fatalf("service call returned %d", status)
		}
		body := waitForHAState(t, testHAID("siren", "it_siren"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] == "on"
		})
		if body["state"] != "on" {
			t.Errorf("expected state=on, got %v", body["state"])
		}
	})

	t.Run("humidifier/set_humidity", func(t *testing.T) {
		status := haPOST(t, "/api/services/humidifier/set_humidity", map[string]any{
			"entity_id": testHAID("humidifier", "it_humidifier"),
			"humidity":  60,
		})
		if status != 200 && status != 201 {
			t.Fatalf("service call returned %d", status)
		}
		body := waitForHAState(t, testHAID("humidifier", "it_humidifier"), 10*time.Second, func(s map[string]any) bool {
			attrs, _ := s["attributes"].(map[string]any)
			if attrs == nil {
				return false
			}
			h, _ := attrs["humidity"].(float64)
			return int(h) == 60
		})
		attrs, _ := body["attributes"].(map[string]any)
		h, _ := attrs["humidity"].(float64)
		if int(h) != 60 {
			t.Errorf("expected humidity=60, got %v", attrs["humidity"])
		}
	})

	t.Run("media_player/media_play", func(t *testing.T) {
		status := haPOST(t, "/api/services/media_player/media_play", map[string]any{
			"entity_id": testHAID("media_player", "it_media_player"),
		})
		if status != 200 && status != 201 {
			t.Fatalf("service call returned %d", status)
		}
		body := waitForHAState(t, testHAID("media_player", "it_media_player"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] == "playing"
		})
		if body["state"] != "playing" {
			t.Errorf("expected state=playing, got %v", body["state"])
		}
	})

	t.Run("remote/turn_off", func(t *testing.T) {
		status := haPOST(t, "/api/services/remote/turn_off", map[string]any{
			"entity_id": testHAID("remote", "it_remote"),
		})
		if status != 200 && status != 201 {
			t.Fatalf("service call returned %d", status)
		}
		body := waitForHAState(t, testHAID("remote", "it_remote"), 10*time.Second, func(s map[string]any) bool {
			return s["state"] == "off"
		})
		if body["state"] != "off" {
			t.Errorf("expected state=off, got %v", body["state"])
		}
	})
}

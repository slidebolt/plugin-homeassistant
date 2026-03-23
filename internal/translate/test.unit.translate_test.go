package translate

import (
"encoding/json"
"reflect"
"testing"

domain "github.com/slidebolt/sb-domain"
)

// ---------------------------------------------------------------------------
// Decode tests (kept for backward compatibility)
// ---------------------------------------------------------------------------

func TestDecode_Light(t *testing.T) {
tests := []struct {
name      string
raw       string
wantOK    bool
wantPower bool
wantBr    int
}{
{"valid on with brightness", `{"power":true,"brightness":200}`, true, true, 200},
{"valid off", `{"power":false,"brightness":0}`, true, false, 0},
{"brightness clamped at max", `{"power":true,"brightness":300}`, true, true, 254},
{"brightness clamped at min", `{"power":true,"brightness":-10}`, true, true, 0},
{"empty payload", ``, false, false, 0},
{"garbage payload", `not json`, false, false, 0},
}
for _, tc := range tests {
t.Run(tc.name, func(t *testing.T) {
got, ok := Decode("light", json.RawMessage(tc.raw))
if ok != tc.wantOK {
t.Fatalf("ok: got %v, want %v", ok, tc.wantOK)
}
if !ok {
return
}
s := got.(domain.Light)
if s.Power != tc.wantPower {
t.Errorf("Power: got %v, want %v", s.Power, tc.wantPower)
}
if s.Brightness != tc.wantBr {
t.Errorf("Brightness: got %d, want %d", s.Brightness, tc.wantBr)
}
})
}
}

func TestDecode_Switch(t *testing.T) {
tests := []struct {
name      string
raw       string
wantOK    bool
wantPower bool
}{
{"on", `{"power":true}`, true, true},
{"off", `{"power":false}`, true, false},
{"empty", ``, false, false},
{"garbage", `!!!`, false, false},
}
for _, tc := range tests {
t.Run(tc.name, func(t *testing.T) {
got, ok := Decode("switch", json.RawMessage(tc.raw))
if ok != tc.wantOK {
t.Fatalf("ok: got %v, want %v", ok, tc.wantOK)
}
if !ok {
return
}
s := got.(domain.Switch)
if s.Power != tc.wantPower {
t.Errorf("Power: got %v, want %v", s.Power, tc.wantPower)
}
})
}
}

func TestDecode_Cover(t *testing.T) {
tests := []struct {
name    string
raw     string
wantOK  bool
wantPos int
}{
{"mid", `{"position":50}`, true, 50},
{"open", `{"position":100}`, true, 100},
{"closed", `{"position":0}`, true, 0},
{"over max clamped", `{"position":150}`, true, 100},
{"under min clamped", `{"position":-5}`, true, 0},
{"empty", ``, false, 0},
}
for _, tc := range tests {
t.Run(tc.name, func(t *testing.T) {
got, ok := Decode("cover", json.RawMessage(tc.raw))
if ok != tc.wantOK {
t.Fatalf("ok: got %v, want %v", ok, tc.wantOK)
}
if !ok {
return
}
s := got.(domain.Cover)
if s.Position != tc.wantPos {
t.Errorf("Position: got %d, want %d", s.Position, tc.wantPos)
}
})
}
}

func TestDecode_Fan(t *testing.T) {
tests := []struct {
name    string
raw     string
wantOK  bool
wantPct int
}{
{"on full", `{"power":true,"percentage":100}`, true, 100},
{"on half", `{"power":true,"percentage":50}`, true, 50},
{"off", `{"power":false,"percentage":0}`, true, 0},
{"over max clamped", `{"power":true,"percentage":120}`, true, 100},
{"under min clamped", `{"power":true,"percentage":-5}`, true, 0},
{"empty", ``, false, 0},
}
for _, tc := range tests {
t.Run(tc.name, func(t *testing.T) {
got, ok := Decode("fan", json.RawMessage(tc.raw))
if ok != tc.wantOK {
t.Fatalf("ok: got %v, want %v", ok, tc.wantOK)
}
if !ok {
return
}
s := got.(domain.Fan)
if s.Percentage != tc.wantPct {
t.Errorf("Percentage: got %d, want %d", s.Percentage, tc.wantPct)
}
})
}
}

func TestDecode_Sensor(t *testing.T) {
tests := []struct {
name     string
raw      string
wantOK   bool
wantUnit string
}{
{"temp with unit", `{"value":22.5,"unit":"°C"}`, true, "°C"},
{"no unit", `{"value":100}`, true, ""},
{"empty", ``, false, ""},
{"garbage", `xyz`, false, ""},
}
for _, tc := range tests {
t.Run(tc.name, func(t *testing.T) {
got, ok := Decode("sensor", json.RawMessage(tc.raw))
if ok != tc.wantOK {
t.Fatalf("ok: got %v, want %v", ok, tc.wantOK)
}
if !ok {
return
}
s := got.(domain.Sensor)
if s.Unit != tc.wantUnit {
t.Errorf("Unit: got %q, want %q", s.Unit, tc.wantUnit)
}
})
}
}

func TestDecode_Climate(t *testing.T) {
tests := []struct {
name     string
raw      string
wantOK   bool
wantMode string
wantTemp float64
}{
{"cool mode", `{"hvacMode":"cool","temperature":21}`, true, "cool", 21},
{"off", `{"hvacMode":"off","temperature":0}`, true, "off", 0},
{"empty", ``, false, "", 0},
}
for _, tc := range tests {
t.Run(tc.name, func(t *testing.T) {
got, ok := Decode("climate", json.RawMessage(tc.raw))
if ok != tc.wantOK {
t.Fatalf("ok: got %v, want %v", ok, tc.wantOK)
}
if !ok {
return
}
s := got.(domain.Climate)
if s.HVACMode != tc.wantMode {
t.Errorf("HVACMode: got %q, want %q", s.HVACMode, tc.wantMode)
}
if s.Temperature != tc.wantTemp {
t.Errorf("Temperature: got %v, want %v", s.Temperature, tc.wantTemp)
}
})
}
}

func TestDecode_UnknownType(t *testing.T) {
_, ok := Decode("thermostat_v2", json.RawMessage(`{"foo":"bar"}`))
if ok {
t.Fatal("expected unknown entity type to return ok=false")
}
}

// ---------------------------------------------------------------------------
// Encode tests (kept for backward compatibility)
// ---------------------------------------------------------------------------

func TestEncode_LightTurnOn(t *testing.T) {
out, err := Encode(domain.LightTurnOn{}, nil)
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
var result map[string]any
json.Unmarshal(out, &result)
if result["state"] != "ON" {
t.Errorf("expected state=ON, got %v", result)
}
}

func TestEncode_LightSetBrightness(t *testing.T) {
tests := []struct {
name    string
cmd     domain.LightSetBrightness
wantErr bool
}{
{"valid 200", domain.LightSetBrightness{Brightness: 200}, false},
{"valid 0", domain.LightSetBrightness{Brightness: 0}, false},
{"max 254", domain.LightSetBrightness{Brightness: 254}, false},
{"255 rejected", domain.LightSetBrightness{Brightness: 255}, true},
{"negative rejected", domain.LightSetBrightness{Brightness: -1}, true},
}
for _, tc := range tests {
t.Run(tc.name, func(t *testing.T) {
out, err := Encode(tc.cmd, nil)
if (err != nil) != tc.wantErr {
t.Fatalf("err: got %v, wantErr %v", err, tc.wantErr)
}
if !tc.wantErr && out == nil {
t.Error("expected non-nil output")
}
})
}
}

func TestEncode_LightSetColorTemp(t *testing.T) {
tests := []struct {
name    string
cmd     domain.LightSetColorTemp
wantErr bool
}{
{"valid 370", domain.LightSetColorTemp{Mireds: 370}, false},
{"min 153", domain.LightSetColorTemp{Mireds: 153}, false},
{"max 500", domain.LightSetColorTemp{Mireds: 500}, false},
{"152 rejected", domain.LightSetColorTemp{Mireds: 152}, true},
{"501 rejected", domain.LightSetColorTemp{Mireds: 501}, true},
}
for _, tc := range tests {
t.Run(tc.name, func(t *testing.T) {
_, err := Encode(tc.cmd, nil)
if (err != nil) != tc.wantErr {
t.Fatalf("err: got %v, wantErr %v", err, tc.wantErr)
}
})
}
}

func TestEncode_LightSetRGB(t *testing.T) {
tests := []struct {
name    string
cmd     domain.LightSetRGB
wantErr bool
}{
{"valid", domain.LightSetRGB{R: 255, G: 128, B: 0}, false},
{"all zero", domain.LightSetRGB{R: 0, G: 0, B: 0}, false},
{"R=256 rejected", domain.LightSetRGB{R: 256, G: 0, B: 0}, true},
{"B negative rejected", domain.LightSetRGB{R: 0, G: 0, B: -1}, true},
}
for _, tc := range tests {
t.Run(tc.name, func(t *testing.T) {
_, err := Encode(tc.cmd, nil)
if (err != nil) != tc.wantErr {
t.Fatalf("err: got %v, wantErr %v", err, tc.wantErr)
}
})
}
}

func TestEncode_LightSetHS(t *testing.T) {
tests := []struct {
name    string
cmd     domain.LightSetHS
wantErr bool
}{
{"valid", domain.LightSetHS{Hue: 180, Saturation: 50}, false},
{"hue 361 rejected", domain.LightSetHS{Hue: 361, Saturation: 50}, true},
{"saturation 101 rejected", domain.LightSetHS{Hue: 90, Saturation: 101}, true},
}
for _, tc := range tests {
t.Run(tc.name, func(t *testing.T) {
_, err := Encode(tc.cmd, nil)
if (err != nil) != tc.wantErr {
t.Fatalf("err: got %v, wantErr %v", err, tc.wantErr)
}
})
}
}

func TestEncode_LightSetEffect(t *testing.T) {
if _, err := Encode(domain.LightSetEffect{Effect: "rainbow"}, nil); err != nil {
t.Fatalf("unexpected error: %v", err)
}
if _, err := Encode(domain.LightSetEffect{Effect: ""}, nil); err == nil {
t.Error("expected error for empty effect")
}
}

func TestEncode_FanSetSpeed(t *testing.T) {
tests := []struct {
name    string
cmd     domain.FanSetSpeed
wantErr bool
}{
{"valid 50%", domain.FanSetSpeed{Percentage: 50}, false},
{"valid 0%", domain.FanSetSpeed{Percentage: 0}, false},
{"valid 100%", domain.FanSetSpeed{Percentage: 100}, false},
{"101% rejected", domain.FanSetSpeed{Percentage: 101}, true},
{"negative rejected", domain.FanSetSpeed{Percentage: -1}, true},
}
for _, tc := range tests {
t.Run(tc.name, func(t *testing.T) {
_, err := Encode(tc.cmd, nil)
if (err != nil) != tc.wantErr {
t.Fatalf("err: got %v, wantErr %v", err, tc.wantErr)
}
})
}
}

func TestEncode_CoverSetPosition(t *testing.T) {
tests := []struct {
name    string
cmd     domain.CoverSetPosition
wantErr bool
}{
{"valid 50", domain.CoverSetPosition{Position: 50}, false},
{"valid 0", domain.CoverSetPosition{Position: 0}, false},
{"valid 100", domain.CoverSetPosition{Position: 100}, false},
{"101 rejected", domain.CoverSetPosition{Position: 101}, true},
{"negative rejected", domain.CoverSetPosition{Position: -1}, true},
}
for _, tc := range tests {
t.Run(tc.name, func(t *testing.T) {
_, err := Encode(tc.cmd, nil)
if (err != nil) != tc.wantErr {
t.Fatalf("err: got %v, wantErr %v", err, tc.wantErr)
}
})
}
}

func TestEncode_SelectOption(t *testing.T) {
tests := []struct {
name    string
cmd     domain.SelectOption
wantErr bool
}{
{"valid option", domain.SelectOption{Option: "eco"}, false},
{"empty option rejected", domain.SelectOption{Option: ""}, true},
}
for _, tc := range tests {
t.Run(tc.name, func(t *testing.T) {
_, err := Encode(tc.cmd, nil)
if (err != nil) != tc.wantErr {
t.Fatalf("err: got %v, wantErr %v", err, tc.wantErr)
}
})
}
}

func TestEncode_ClimateSetMode(t *testing.T) {
tests := []struct {
name    string
cmd     domain.ClimateSetMode
wantErr bool
}{
{"cool", domain.ClimateSetMode{HVACMode: "cool"}, false},
{"off", domain.ClimateSetMode{HVACMode: "off"}, false},
{"empty rejected", domain.ClimateSetMode{HVACMode: ""}, true},
}
for _, tc := range tests {
t.Run(tc.name, func(t *testing.T) {
_, err := Encode(tc.cmd, nil)
if (err != nil) != tc.wantErr {
t.Fatalf("err: got %v, wantErr %v", err, tc.wantErr)
}
})
}
}

func TestEncode_SwitchCommands(t *testing.T) {
for _, cmd := range []any{domain.SwitchTurnOn{}, domain.SwitchTurnOff{}, domain.SwitchToggle{}} {
out, err := Encode(cmd, nil)
if err != nil {
t.Errorf("%T: unexpected error: %v", cmd, err)
}
if len(out) == 0 {
t.Errorf("%T: empty output", cmd)
}
}
}

func TestEncode_UnknownCommand(t *testing.T) {
type unknownCmd struct{}
_, err := Encode(unknownCmd{}, nil)
if err == nil {
t.Fatal("expected error for unknown command type")
}
}

func TestRoundTrip_LightState(t *testing.T) {
raw := json.RawMessage(`{"power":true,"brightness":150}`)
state, ok := Decode("light", raw)
if !ok {
t.Fatal("Decode failed")
}
light := state.(domain.Light)
if !light.Power || light.Brightness != 150 {
t.Errorf("unexpected state: %+v", light)
}
out, err := Encode(domain.LightSetBrightness{Brightness: light.Brightness}, nil)
if err != nil {
t.Fatalf("Encode: %v", err)
}
var result map[string]any
json.Unmarshal(out, &result)
if result["brightness"] == nil {
t.Error("encoded output missing brightness")
}
}

// ---------------------------------------------------------------------------
// ToHA tests
// ---------------------------------------------------------------------------

func TestToHA_Light(t *testing.T) {
entity := domain.Entity{
Type:  "light",
State: domain.Light{Power: true, Brightness: 200},
}
state, _ := ToHA(entity)
if state["is_on"] != true {
t.Errorf("is_on: got %v, want true", state["is_on"])
}
if state["brightness"] != 200 {
t.Errorf("brightness: got %v, want 200", state["brightness"])
}
}

func TestToHA_Switch(t *testing.T) {
entity := domain.Entity{Type: "switch", State: domain.Switch{Power: false}}
state, _ := ToHA(entity)
if state["is_on"] != false {
t.Errorf("is_on: got %v, want false", state["is_on"])
}
}

func TestToHA_Cover(t *testing.T) {
entity := domain.Entity{Type: "cover", State: domain.Cover{Position: 100}}
state, _ := ToHA(entity)
if state["current_position"] != 100 {
t.Errorf("current_position: got %v, want 100", state["current_position"])
}
if state["state"] != "open" {
t.Errorf("state: got %v, want open", state["state"])
}
}

func TestToHA_Climate(t *testing.T) {
entity := domain.Entity{
Type:  "climate",
State: domain.Climate{HVACMode: "cool", Temperature: 22.5},
}
state, _ := ToHA(entity)
if state["hvac_mode"] != "cool" {
t.Errorf("hvac_mode: got %v, want cool", state["hvac_mode"])
}
if state["target_temperature"] != 22.5 {
t.Errorf("target_temperature: got %v, want 22.5", state["target_temperature"])
}
}

func TestToHA_Alarm(t *testing.T) {
entity := domain.Entity{Type: "alarm", State: domain.Alarm{AlarmState: "disarmed"}}
state, _ := ToHA(entity)
if state["alarm_state"] != "disarmed" {
t.Errorf("alarm_state: got %v, want disarmed", state["alarm_state"])
}
}

func TestToHA_Camera(t *testing.T) {
entity := domain.Entity{Type: "camera", State: domain.Camera{IsStreaming: false}}
state, _ := ToHA(entity)
if state["is_streaming"] != false {
t.Errorf("is_streaming: got %v, want false", state["is_streaming"])
}
}

func TestToHA_Valve(t *testing.T) {
entity := domain.Entity{Type: "valve", State: domain.Valve{Position: 100}}
state, _ := ToHA(entity)
if state["current_valve_position"] != 100 {
t.Errorf("current_valve_position: got %v, want 100", state["current_valve_position"])
}
if state["state"] != "open" {
t.Errorf("state: got %v, want open", state["state"])
}
}

func TestToHA_Siren(t *testing.T) {
entity := domain.Entity{Type: "siren", State: domain.Siren{IsOn: false}}
state, _ := ToHA(entity)
if state["is_on"] != false {
t.Errorf("is_on: got %v, want false", state["is_on"])
}
}

func TestToHA_Humidifier(t *testing.T) {
entity := domain.Entity{
Type:  "humidifier",
State: domain.Humidifier{TargetHumidity: 45},
}
state, _ := ToHA(entity)
if state["target_humidity"] != 45 {
t.Errorf("target_humidity: got %v, want 45", state["target_humidity"])
}
}

func TestToHA_MediaPlayer(t *testing.T) {
entity := domain.Entity{
Type:  "media_player",
State: domain.MediaPlayer{State: "paused"},
}
state, _ := ToHA(entity)
if state["state"] != "paused" {
t.Errorf("state: got %v, want paused", state["state"])
}
}

func TestToHA_Remote(t *testing.T) {
entity := domain.Entity{Type: "remote", State: domain.Remote{IsOn: true}}
state, _ := ToHA(entity)
if state["is_on"] != true {
t.Errorf("is_on: got %v, want true", state["is_on"])
}
}

func TestToHA_Event(t *testing.T) {
entity := domain.Entity{
Type:  "event",
State: domain.Event{DeviceClass: "doorbell"},
}
state, _ := ToHA(entity)
if state["device_class"] != "doorbell" {
t.Errorf("device_class: got %v, want doorbell", state["device_class"])
}
}

// ---------------------------------------------------------------------------
// FromHA tests
// ---------------------------------------------------------------------------

func TestFromHA_Light_TurnOn(t *testing.T) {
cmd, err := FromHA("light", "turn_on", map[string]any{})
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if _, ok := cmd.(domain.LightTurnOn); !ok {
t.Errorf("expected LightTurnOn, got %T", cmd)
}
}

func TestFromHA_Light_TurnOn_Brightness(t *testing.T) {
cmd, err := FromHA("light", "turn_on", map[string]any{"brightness": float64(200)})
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
c, ok := cmd.(domain.LightSetBrightness)
if !ok {
t.Fatalf("expected LightSetBrightness, got %T", cmd)
}
if c.Brightness != 200 {
t.Errorf("Brightness: got %d, want 200", c.Brightness)
}
}

func TestFromHA_Switch_TurnOn(t *testing.T) {
cmd, err := FromHA("switch", "turn_on", map[string]any{})
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if _, ok := cmd.(domain.SwitchTurnOn); !ok {
t.Errorf("expected SwitchTurnOn, got %T", cmd)
}
}

func TestFromHA_Cover_Open(t *testing.T) {
cmd, err := FromHA("cover", "open_cover", map[string]any{})
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if _, ok := cmd.(domain.CoverOpen); !ok {
t.Errorf("expected CoverOpen, got %T", cmd)
}
}

func TestFromHA_Alarm_Disarm(t *testing.T) {
cmd, err := FromHA("alarm", "alarm_disarm", map[string]any{})
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if _, ok := cmd.(domain.AlarmDisarm); !ok {
t.Errorf("expected AlarmDisarm, got %T", cmd)
}
}

func TestFromHA_Camera_RecordStart(t *testing.T) {
cmd, err := FromHA("camera", "turn_on", map[string]any{})
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if _, ok := cmd.(domain.CameraRecordStart); !ok {
t.Errorf("expected CameraRecordStart, got %T", cmd)
}
}

func TestFromHA_Valve_Open(t *testing.T) {
cmd, err := FromHA("valve", "open_valve", map[string]any{})
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if _, ok := cmd.(domain.ValveOpen); !ok {
t.Errorf("expected ValveOpen, got %T", cmd)
}
}

func TestFromHA_Siren_TurnOn(t *testing.T) {
cmd, err := FromHA("siren", "turn_on", map[string]any{})
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if _, ok := cmd.(domain.SirenTurnOn); !ok {
t.Errorf("expected SirenTurnOn, got %T", cmd)
}
}

func TestFromHA_Humidifier_TurnOn(t *testing.T) {
cmd, err := FromHA("humidifier", "turn_on", map[string]any{})
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if _, ok := cmd.(domain.HumidifierTurnOn); !ok {
t.Errorf("expected HumidifierTurnOn, got %T", cmd)
}
}

func TestFromHA_MediaPlayer_Play(t *testing.T) {
cmd, err := FromHA("media_player", "media_play", map[string]any{})
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if _, ok := cmd.(domain.MediaPlay); !ok {
t.Errorf("expected MediaPlay, got %T", cmd)
}
}

func TestFromHA_Remote_TurnOff(t *testing.T) {
cmd, err := FromHA("remote", "turn_off", map[string]any{})
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
if _, ok := cmd.(domain.RemoteTurnOff); !ok {
t.Errorf("expected RemoteTurnOff, got %T", cmd)
}
}

func TestFromHA_Unsupported(t *testing.T) {
_, err := FromHA("foo", "bar", map[string]any{})
if err == nil {
t.Fatal("expected error for unsupported entity/action")
}
}

// Verify FromHA returns the correct concrete type using reflect
func TestFromHA_TypeCheck(t *testing.T) {
tests := []struct {
entityType string
action     string
params     map[string]any
wantType   string
}{
{"light", "turn_on", map[string]any{}, "LightTurnOn"},
{"light", "turn_off", map[string]any{}, "LightTurnOff"},
{"switch", "toggle", map[string]any{}, "SwitchToggle"},
{"fan", "turn_on", map[string]any{}, "FanTurnOn"},
{"lock", "lock", map[string]any{}, "LockLock"},
{"alarm", "alarm_arm_home", map[string]any{}, "AlarmArmHome"},
{"alarm", "alarm_arm_away", map[string]any{}, "AlarmArmAway"},
{"alarm", "alarm_arm_night", map[string]any{}, "AlarmArmNight"},
{"camera", "turn_off", map[string]any{}, "CameraRecordStop"},
{"camera", "enable_motion_detection", map[string]any{}, "CameraEnableMotion"},
{"camera", "disable_motion_detection", map[string]any{}, "CameraDisableMotion"},
{"valve", "close_valve", map[string]any{}, "ValveClose"},
{"siren", "turn_off", map[string]any{}, "SirenTurnOff"},
{"humidifier", "turn_off", map[string]any{}, "HumidifierTurnOff"},
{"media_player", "media_pause", map[string]any{}, "MediaPause"},
{"media_player", "media_stop", map[string]any{}, "MediaStop"},
{"media_player", "media_next_track", map[string]any{}, "MediaNextTrack"},
{"media_player", "media_previous_track", map[string]any{}, "MediaPreviousTrack"},
{"remote", "turn_on", map[string]any{}, "RemoteTurnOn"},
{"button", "press", map[string]any{}, "ButtonPress"},
}
for _, tc := range tests {
t.Run(tc.entityType+"/"+tc.action, func(t *testing.T) {
cmd, err := FromHA(tc.entityType, tc.action, tc.params)
if err != nil {
t.Fatalf("unexpected error: %v", err)
}
got := reflect.TypeOf(cmd).Name()
if got != tc.wantType {
t.Errorf("type: got %q, want %q", got, tc.wantType)
}
})
}
}

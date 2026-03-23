package translate

// translate.go — Decode/Encode translation layer.
//
// This file is the core "guts" of every plugin. When copying plugin-homeassistant
// to build a new plugin, replace the per-type decode/encode functions with
// your protocol's actual translation logic.
//
//   Decode: raw protocol bytes → canonical domain state (lenient)
//     - Called on every inbound event from the device
//     - Clamp out-of-range values rather than reject
//     - Return (nil, false) to silently skip unrecognisable payloads
//
//   Encode: canonical domain command → raw protocol bytes (strict)
//     - Called when a command arrives for an entity
//     - Return an error for anything invalid — never send bad data to a device
//     - internal contains the raw discovery payload (ReadFile Internal) with
//       device-specific metadata: topics, scale factors, capability flags

import (
	"encoding/json"
	"fmt"

	domain "github.com/slidebolt/sb-domain"
)

// Decode converts a raw protocol payload into a canonical domain state.
// Returns (state, true) on success, (nil, false) to silently skip.
func Decode(entityType string, raw json.RawMessage) (any, bool) {
	switch entityType {
	case "light":
		return decodeLight(raw)
	case "switch":
		return decodeSwitch(raw)
	case "cover":
		return decodeCover(raw)
	case "lock":
		return decodeLock(raw)
	case "fan":
		return decodeFan(raw)
	case "sensor":
		return decodeSensor(raw)
	case "binary_sensor":
		return decodeBinarySensor(raw)
	case "climate":
		return decodeClimate(raw)
	case "button":
		return decodeButton(raw)
	case "number":
		return decodeNumber(raw)
	case "select":
		return decodeSelect(raw)
	case "text":
		return decodeText(raw)
	case "alarm":
		return decodeAlarm(raw)
	case "camera":
		return decodeCamera(raw)
	case "valve":
		return decodeValve(raw)
	case "siren":
		return decodeSiren(raw)
	case "humidifier":
		return decodeHumidifier(raw)
	case "media_player":
		return decodeMediaPlayer(raw)
	case "remote":
		return decodeRemote(raw)
	case "event":
		return decodeEvent(raw)
	default:
		return nil, false
	}
}

// Encode converts a SlideBolt domain command into a raw protocol payload.
// internal is the raw discovery payload previously stored with WriteFile(Internal).
// Returns an error if the command is invalid or unsupported.
func Encode(cmd any, internal json.RawMessage) (json.RawMessage, error) {
	switch c := cmd.(type) {
	case domain.LightTurnOn:
		return encodeLightTurnOn(c, internal)
	case domain.LightTurnOff:
		return encodeLightTurnOff(c, internal)
	case domain.LightSetBrightness:
		return encodeLightSetBrightness(c, internal)
	case domain.LightSetColorTemp:
		return encodeLightSetColorTemp(c, internal)
	case domain.LightSetRGB:
		return encodeLightSetRGB(c, internal)
	case domain.LightSetRGBW:
		return encodeLightSetRGBW(c, internal)
	case domain.LightSetRGBWW:
		return encodeLightSetRGBWW(c, internal)
	case domain.LightSetHS:
		return encodeLightSetHS(c, internal)
	case domain.LightSetXY:
		return encodeLightSetXY(c, internal)
	case domain.LightSetWhite:
		return encodeLightSetWhite(c, internal)
	case domain.LightSetEffect:
		return encodeLightSetEffect(c, internal)
	case domain.SwitchTurnOn:
		return encodeSwitchTurnOn(c, internal)
	case domain.SwitchTurnOff:
		return encodeSwitchTurnOff(c, internal)
	case domain.SwitchToggle:
		return encodeSwitchToggle(c, internal)
	case domain.FanTurnOn:
		return encodeFanTurnOn(c, internal)
	case domain.FanTurnOff:
		return encodeFanTurnOff(c, internal)
	case domain.FanSetSpeed:
		return encodeFanSetSpeed(c, internal)
	case domain.CoverOpen:
		return encodeCoverOpen(c, internal)
	case domain.CoverClose:
		return encodeCoverClose(c, internal)
	case domain.CoverSetPosition:
		return encodeCoverSetPosition(c, internal)
	case domain.LockLock:
		return encodeLockLock(c, internal)
	case domain.LockUnlock:
		return encodeLockUnlock(c, internal)
	case domain.ButtonPress:
		return encodeButtonPress(c, internal)
	case domain.NumberSetValue:
		return encodeNumberSetValue(c, internal)
	case domain.SelectOption:
		return encodeSelectOption(c, internal)
	case domain.TextSetValue:
		return encodeTextSetValue(c, internal)
	case domain.ClimateSetMode:
		return encodeClimateSetMode(c, internal)
	case domain.ClimateSetTemperature:
		return encodeClimateSetTemperature(c, internal)
	default:
		return nil, fmt.Errorf("translate: unsupported command type %T", cmd)
	}
}

// ---------------------------------------------------------------------------
// Decode: per-type (raw protocol → domain state)
//
// plugin-homeassistant uses identity decode — raw IS already canonical JSON.
// Real plugins replace these with protocol-specific parsing.
// ---------------------------------------------------------------------------

func decodeLight(raw json.RawMessage) (any, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var s domain.Light
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, false
	}
	// Clamp brightness to valid range.
	if s.Brightness < 0 {
		s.Brightness = 0
	}
	if s.Brightness > 254 {
		s.Brightness = 254
	}
	return s, true
}

func decodeSwitch(raw json.RawMessage) (any, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var s domain.Switch
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, false
	}
	return s, true
}

func decodeCover(raw json.RawMessage) (any, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var s domain.Cover
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, false
	}
	if s.Position < 0 {
		s.Position = 0
	}
	if s.Position > 100 {
		s.Position = 100
	}
	return s, true
}

func decodeLock(raw json.RawMessage) (any, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var s domain.Lock
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, false
	}
	return s, true
}

func decodeFan(raw json.RawMessage) (any, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var s domain.Fan
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, false
	}
	if s.Percentage < 0 {
		s.Percentage = 0
	}
	if s.Percentage > 100 {
		s.Percentage = 100
	}
	return s, true
}

func decodeSensor(raw json.RawMessage) (any, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var s domain.Sensor
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, false
	}
	return s, true
}

func decodeBinarySensor(raw json.RawMessage) (any, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var s domain.BinarySensor
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, false
	}
	return s, true
}

func decodeClimate(raw json.RawMessage) (any, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var s domain.Climate
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, false
	}
	return s, true
}

func decodeButton(raw json.RawMessage) (any, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var s domain.Button
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, false
	}
	return s, true
}

func decodeNumber(raw json.RawMessage) (any, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var s domain.Number
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, false
	}
	return s, true
}

func decodeSelect(raw json.RawMessage) (any, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var s domain.Select
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, false
	}
	return s, true
}

func decodeText(raw json.RawMessage) (any, bool) {
	if len(raw) == 0 {
		return nil, false
	}
	var s domain.Text
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, false
	}
	return s, true
}

// ---------------------------------------------------------------------------
// Encode: per-command (domain command → raw protocol payload)
//
// plugin-homeassistant uses identity encode — canonical JSON IS the protocol.
// Real plugins replace these with protocol-specific serialization.
// internal holds the raw discovery payload for device-specific metadata.
// ---------------------------------------------------------------------------

func encodeLightTurnOn(_ domain.LightTurnOn, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(map[string]any{"state": "ON"})
}

func encodeLightTurnOff(c domain.LightTurnOff, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(c)
}

func encodeLightSetBrightness(c domain.LightSetBrightness, _ json.RawMessage) (json.RawMessage, error) {
	if c.Brightness < 0 || c.Brightness > 254 {
		return nil, fmt.Errorf("translate: brightness %d out of range [0,254]", c.Brightness)
	}
	return json.Marshal(c)
}

func encodeLightSetColorTemp(c domain.LightSetColorTemp, _ json.RawMessage) (json.RawMessage, error) {
	if c.Mireds < 153 || c.Mireds > 500 {
		return nil, fmt.Errorf("translate: mireds %d out of range [153,500]", c.Mireds)
	}
	return json.Marshal(c)
}

func encodeLightSetRGB(c domain.LightSetRGB, _ json.RawMessage) (json.RawMessage, error) {
	for name, v := range map[string]int{"r": c.R, "g": c.G, "b": c.B} {
		if v < 0 || v > 255 {
			return nil, fmt.Errorf("translate: %s value %d out of range [0,255]", name, v)
		}
	}
	return json.Marshal(c)
}

func encodeLightSetRGBW(c domain.LightSetRGBW, _ json.RawMessage) (json.RawMessage, error) {
	for name, v := range map[string]int{"r": c.R, "g": c.G, "b": c.B, "w": c.W} {
		if v < 0 || v > 255 {
			return nil, fmt.Errorf("translate: %s value %d out of range [0,255]", name, v)
		}
	}
	return json.Marshal(c)
}

func encodeLightSetRGBWW(c domain.LightSetRGBWW, _ json.RawMessage) (json.RawMessage, error) {
	for name, v := range map[string]int{"r": c.R, "g": c.G, "b": c.B, "cw": c.CW, "ww": c.WW} {
		if v < 0 || v > 255 {
			return nil, fmt.Errorf("translate: %s value %d out of range [0,255]", name, v)
		}
	}
	return json.Marshal(c)
}

func encodeLightSetHS(c domain.LightSetHS, _ json.RawMessage) (json.RawMessage, error) {
	if c.Hue < 0 || c.Hue > 360 {
		return nil, fmt.Errorf("translate: hue %.2f out of range [0,360]", c.Hue)
	}
	if c.Saturation < 0 || c.Saturation > 100 {
		return nil, fmt.Errorf("translate: saturation %.2f out of range [0,100]", c.Saturation)
	}
	return json.Marshal(c)
}

func encodeLightSetXY(c domain.LightSetXY, _ json.RawMessage) (json.RawMessage, error) {
	if c.X < 0 || c.X > 1 {
		return nil, fmt.Errorf("translate: x %.4f out of range [0,1]", c.X)
	}
	if c.Y < 0 || c.Y > 1 {
		return nil, fmt.Errorf("translate: y %.4f out of range [0,1]", c.Y)
	}
	return json.Marshal(c)
}

func encodeLightSetWhite(c domain.LightSetWhite, _ json.RawMessage) (json.RawMessage, error) {
	if c.White < 0 || c.White > 254 {
		return nil, fmt.Errorf("translate: white %d out of range [0,254]", c.White)
	}
	return json.Marshal(c)
}

func encodeLightSetEffect(c domain.LightSetEffect, _ json.RawMessage) (json.RawMessage, error) {
	if c.Effect == "" {
		return nil, fmt.Errorf("translate: effect must not be empty")
	}
	return json.Marshal(c)
}

func encodeSwitchTurnOn(_ domain.SwitchTurnOn, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(map[string]any{"state": "ON"})
}

func encodeSwitchTurnOff(_ domain.SwitchTurnOff, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(map[string]any{"state": "OFF"})
}

func encodeSwitchToggle(_ domain.SwitchToggle, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(map[string]any{"state": "TOGGLE"})
}

func encodeFanTurnOn(_ domain.FanTurnOn, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(map[string]any{"state": "ON"})
}

func encodeFanTurnOff(_ domain.FanTurnOff, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(map[string]any{"state": "OFF"})
}

func encodeFanSetSpeed(c domain.FanSetSpeed, _ json.RawMessage) (json.RawMessage, error) {
	if c.Percentage < 0 || c.Percentage > 100 {
		return nil, fmt.Errorf("translate: fan percentage %d out of range 0-100", c.Percentage)
	}
	return json.Marshal(c)
}

func encodeCoverOpen(_ domain.CoverOpen, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(map[string]any{"state": "OPEN"})
}

func encodeCoverClose(_ domain.CoverClose, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(map[string]any{"state": "CLOSE"})
}

func encodeCoverSetPosition(c domain.CoverSetPosition, _ json.RawMessage) (json.RawMessage, error) {
	if c.Position < 0 || c.Position > 100 {
		return nil, fmt.Errorf("translate: cover position %d out of range 0-100", c.Position)
	}
	return json.Marshal(c)
}

func encodeLockLock(_ domain.LockLock, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(map[string]any{"state": "LOCK"})
}

func encodeLockUnlock(c domain.LockUnlock, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(c)
}

func encodeButtonPress(_ domain.ButtonPress, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(map[string]any{"action": "PRESS"})
}

func encodeNumberSetValue(c domain.NumberSetValue, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(c)
}

func encodeSelectOption(c domain.SelectOption, _ json.RawMessage) (json.RawMessage, error) {
	if c.Option == "" {
		return nil, fmt.Errorf("translate: select option must not be empty")
	}
	return json.Marshal(c)
}

func encodeTextSetValue(c domain.TextSetValue, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(c)
}

func encodeClimateSetMode(c domain.ClimateSetMode, _ json.RawMessage) (json.RawMessage, error) {
	if c.HVACMode == "" {
		return nil, fmt.Errorf("translate: climate hvac_mode must not be empty")
	}
	return json.Marshal(c)
}

func encodeClimateSetTemperature(c domain.ClimateSetTemperature, _ json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(c)
}

// ---------------------------------------------------------------------------
// Decode: new entity types
// ---------------------------------------------------------------------------

func decodeAlarm(raw json.RawMessage) (any, bool) {
if len(raw) == 0 {
return nil, false
}
var s domain.Alarm
if err := json.Unmarshal(raw, &s); err != nil {
return nil, false
}
return s, true
}

func decodeCamera(raw json.RawMessage) (any, bool) {
if len(raw) == 0 {
return nil, false
}
var s domain.Camera
if err := json.Unmarshal(raw, &s); err != nil {
return nil, false
}
return s, true
}

func decodeValve(raw json.RawMessage) (any, bool) {
if len(raw) == 0 {
return nil, false
}
var s domain.Valve
if err := json.Unmarshal(raw, &s); err != nil {
return nil, false
}
if s.Position < 0 {
s.Position = 0
}
if s.Position > 100 {
s.Position = 100
}
return s, true
}

func decodeSiren(raw json.RawMessage) (any, bool) {
if len(raw) == 0 {
return nil, false
}
var s domain.Siren
if err := json.Unmarshal(raw, &s); err != nil {
return nil, false
}
return s, true
}

func decodeHumidifier(raw json.RawMessage) (any, bool) {
if len(raw) == 0 {
return nil, false
}
var s domain.Humidifier
if err := json.Unmarshal(raw, &s); err != nil {
return nil, false
}
return s, true
}

func decodeMediaPlayer(raw json.RawMessage) (any, bool) {
if len(raw) == 0 {
return nil, false
}
var s domain.MediaPlayer
if err := json.Unmarshal(raw, &s); err != nil {
return nil, false
}
return s, true
}

func decodeRemote(raw json.RawMessage) (any, bool) {
if len(raw) == 0 {
return nil, false
}
var s domain.Remote
if err := json.Unmarshal(raw, &s); err != nil {
return nil, false
}
return s, true
}

func decodeEvent(raw json.RawMessage) (any, bool) {
if len(raw) == 0 {
return nil, false
}
var s domain.Event
if err := json.Unmarshal(raw, &s); err != nil {
return nil, false
}
return s, true
}

// ---------------------------------------------------------------------------
// ToHA converts a domain.Entity to HA wire format.
// Returns state fields and attributes maps.
// ---------------------------------------------------------------------------

func ToHA(entity domain.Entity) (map[string]any, map[string]any) {
state := map[string]any{}
attrs := map[string]any{}
switch entity.Type {
case "light":
s, _ := entity.State.(domain.Light)
state["is_on"] = s.Power
state["brightness"] = s.Brightness
state["supported_features"] = 0

// Derive supported color modes from the entity's commands — these
// represent hardware capability and don't change with current state.
modes := []string{}
for _, cmd := range entity.Commands {
	switch cmd {
	case "light_set_rgb":
		modes = append(modes, "rgb")
	case "light_set_color_temp":
		modes = append(modes, "color_temp")
	case "light_set_xy":
		modes = append(modes, "xy")
	}
}
if len(modes) == 0 {
	modes = []string{"brightness"}
}
state["supported_color_modes"] = modes

// Pick active color mode based on which state data is populated.
hasRGB := s.RGB != nil && (s.RGB[0] > 0 || s.RGB[1] > 0 || s.RGB[2] > 0)
hasXY := len(s.XY) == 2 && (s.XY[0] > 0 || s.XY[1] > 0)
hasCT := s.Temperature > 0
if hasCT && !hasRGB && !hasXY {
state["color_mode"] = "color_temp"
} else if hasXY {
state["color_mode"] = "xy"
} else if hasRGB {
state["color_mode"] = "rgb"
} else {
state["color_mode"] = modes[0]
}
if s.RGB != nil {
state["rgb_color"] = s.RGB
}
if hasXY {
state["xy_color"] = s.XY
}
if hasCT {
state["color_temp_kelvin"] = 1_000_000 / s.Temperature
state["min_color_temp_kelvin"] = 1_000_000 / 500 // 500 mireds = 2000K
state["max_color_temp_kelvin"] = 1_000_000 / 153 // 153 mireds ≈ 6536K
}
case "switch":
s, _ := entity.State.(domain.Switch)
state["is_on"] = s.Power
case "cover":
s, _ := entity.State.(domain.Cover)
state["current_position"] = s.Position
if s.Position > 0 {
state["state"] = "open"
} else {
state["state"] = "closed"
}
state["supported_features"] = 15
if s.DeviceClass != "" {
attrs["device_class"] = s.DeviceClass
}
case "lock":
s, _ := entity.State.(domain.Lock)
state["is_locked"] = s.Locked
state["is_locking"] = false
state["is_unlocking"] = false
case "fan":
s, _ := entity.State.(domain.Fan)
state["is_on"] = s.Power
state["percentage"] = s.Percentage
// SET_SPEED=1 | TURN_OFF=16 | TURN_ON=32
state["supported_features"] = 1 | 16 | 32
case "sensor":
s, _ := entity.State.(domain.Sensor)
state["native_value"] = s.Value
if s.Unit != "" {
state["native_unit_of_measurement"] = s.Unit
}
if s.DeviceClass != "" {
state["device_class"] = s.DeviceClass
}
state["state_class"] = "measurement"
case "binary_sensor":
s, _ := entity.State.(domain.BinarySensor)
state["is_on"] = s.On
if s.DeviceClass != "" {
state["device_class"] = s.DeviceClass
}
case "climate":
s, _ := entity.State.(domain.Climate)
state["hvac_mode"] = s.HVACMode
if len(s.HVACModes) > 0 {
state["hvac_modes"] = s.HVACModes
}
state["target_temperature"] = s.Temperature
if s.CurrentTemperature > 0 {
state["current_temperature"] = s.CurrentTemperature
}
if s.TemperatureUnit != "" {
state["temperature_unit"] = s.TemperatureUnit
}
// Always provide min/max so HA doesn't get nil comparisons.
minT := s.MinTemp
if minT == 0 {
minT = 7
}
maxT := s.MaxTemp
if maxT == 0 {
maxT = 35
}
state["min_temp"] = minT
state["max_temp"] = maxT
if s.TargetTempStep > 0 {
state["target_temperature_step"] = s.TargetTempStep
}
state["supported_features"] = 1
case "button":
s, _ := entity.State.(domain.Button)
if s.DeviceClass != "" {
state["device_class"] = s.DeviceClass
}
case "number":
s, _ := entity.State.(domain.Number)
state["native_value"] = s.Value
state["native_min_value"] = s.Min
state["native_max_value"] = s.Max
state["native_step"] = s.Step
if s.Unit != "" {
state["native_unit_of_measurement"] = s.Unit
}
state["mode"] = "slider"
case "select":
s, _ := entity.State.(domain.Select)
state["current_option"] = s.Option
state["options"] = s.Options
case "text":
s, _ := entity.State.(domain.Text)
state["native_value"] = s.Value
state["native_min"] = s.Min
state["native_max"] = s.Max
if s.Mode != "" {
state["mode"] = s.Mode
}
case "alarm":
s, _ := entity.State.(domain.Alarm)
state["alarm_state"] = s.AlarmState
state["code_arm_required"] = s.CodeArmRequired
state["supported_features"] = 15
case "camera":
s, _ := entity.State.(domain.Camera)
state["is_streaming"] = s.IsStreaming
state["is_recording"] = s.IsRecording
state["motion_detection_enabled"] = s.MotionDetection
if s.StreamSource != "" {
state["stream_source"] = s.StreamSource
}
if s.SnapshotURL != "" {
state["snapshot_url"] = s.SnapshotURL
}
state["supported_features"] = 3
case "valve":
s, _ := entity.State.(domain.Valve)
state["current_valve_position"] = s.Position
if s.Position > 0 {
state["state"] = "open"
} else {
state["state"] = "closed"
}
state["reports_position"] = s.ReportsPosition
state["supported_features"] = 7
if s.DeviceClass != "" {
attrs["device_class"] = s.DeviceClass
}
case "siren":
s, _ := entity.State.(domain.Siren)
state["is_on"] = s.IsOn
if len(s.AvailableTones) > 0 {
state["available_tones"] = s.AvailableTones
}
state["supported_features"] = 15
case "humidifier":
s, _ := entity.State.(domain.Humidifier)
state["is_on"] = s.IsOn
state["target_humidity"] = s.TargetHumidity
state["current_humidity"] = s.CurrentHumidity
state["min_humidity"] = s.MinHumidity
state["max_humidity"] = s.MaxHumidity
if s.Mode != "" {
state["mode"] = s.Mode
}
if len(s.AvailableModes) > 0 {
state["available_modes"] = s.AvailableModes
}
state["supported_features"] = 1
case "media_player":
s, _ := entity.State.(domain.MediaPlayer)
state["state"] = s.State
state["volume_level"] = s.VolumeLevel
state["is_volume_muted"] = s.IsVolumeMuted
if s.MediaTitle != "" {
state["media_title"] = s.MediaTitle
}
if s.MediaArtist != "" {
state["media_artist"] = s.MediaArtist
}
if s.Source != "" {
state["source"] = s.Source
}
if len(s.SourceList) > 0 {
state["source_list"] = s.SourceList
}
state["supported_features"] = 63439
case "remote":
s, _ := entity.State.(domain.Remote)
state["is_on"] = s.IsOn
if len(s.ActivityList) > 0 {
state["activity_list"] = s.ActivityList
}
if s.CurrentActivity != "" {
state["current_activity"] = s.CurrentActivity
}
state["supported_features"] = 4
case "event":
s, _ := entity.State.(domain.Event)
state["event_types"] = s.EventTypes
if s.DeviceClass != "" {
state["device_class"] = s.DeviceClass
}
}
return state, attrs
}

// ---------------------------------------------------------------------------
// FromHA converts an HA command to a SlideBolt domain command.
// ---------------------------------------------------------------------------

func FromHA(entityType, action string, params map[string]any) (any, error) {
switch entityType {
case "light":
switch action {
case "turn_on":
if v, ok := params["brightness"]; ok {
b := toInt(v)
if b < 0 || b > 254 {
return nil, fmt.Errorf("brightness %d out of range [0,254]", b)
}
cmd := domain.LightSetBrightness{Brightness: b}
return cmd, nil
}
if v, ok := params["rgb_color"]; ok {
rgb := toIntSlice(v)
if len(rgb) != 3 {
return nil, fmt.Errorf("rgb_color must have 3 elements")
}
cmd := domain.LightSetRGB{R: rgb[0], G: rgb[1], B: rgb[2]}
if b, ok := params["brightness"]; ok {
cmd.Brightness = toInt(b)
}
return cmd, nil
}
if v, ok := params["rgbw_color"]; ok {
rgbw := toIntSlice(v)
if len(rgbw) != 4 {
return nil, fmt.Errorf("rgbw_color must have 4 elements")
}
cmd := domain.LightSetRGBW{R: rgbw[0], G: rgbw[1], B: rgbw[2], W: rgbw[3]}
if b, ok := params["brightness"]; ok {
cmd.Brightness = toInt(b)
}
return cmd, nil
}
if v, ok := params["rgbww_color"]; ok {
rgbww := toIntSlice(v)
if len(rgbww) != 5 {
return nil, fmt.Errorf("rgbww_color must have 5 elements")
}
cmd := domain.LightSetRGBWW{R: rgbww[0], G: rgbww[1], B: rgbww[2], CW: rgbww[3], WW: rgbww[4]}
if b, ok := params["brightness"]; ok {
cmd.Brightness = toInt(b)
}
return cmd, nil
}
if v, ok := params["hs_color"]; ok {
hs := toFloatSlice(v)
if len(hs) != 2 {
return nil, fmt.Errorf("hs_color must have 2 elements")
}
cmd := domain.LightSetHS{Hue: hs[0], Saturation: hs[1]}
if b, ok := params["brightness"]; ok {
cmd.Brightness = toInt(b)
}
return cmd, nil
}
if v, ok := params["xy_color"]; ok {
xy := toFloatSlice(v)
if len(xy) != 2 {
return nil, fmt.Errorf("xy_color must have 2 elements")
}
cmd := domain.LightSetXY{X: xy[0], Y: xy[1]}
if b, ok := params["brightness"]; ok {
cmd.Brightness = toInt(b)
}
return cmd, nil
}
if v, ok := params["white"]; ok {
return domain.LightSetWhite{White: toInt(v)}, nil
}
if v, ok := params["effect"]; ok {
return domain.LightSetEffect{Effect: toString(v)}, nil
}
if v, ok := params["color_temp_kelvin"]; ok {
kelvin := toInt(v)
mireds := 1_000_000 / kelvin
return domain.LightSetColorTemp{Mireds: mireds}, nil
}
return domain.LightTurnOn{}, nil
case "turn_off":
return domain.LightTurnOff{}, nil
}
case "switch":
switch action {
case "turn_on":
return domain.SwitchTurnOn{}, nil
case "turn_off":
return domain.SwitchTurnOff{}, nil
case "toggle":
return domain.SwitchToggle{}, nil
}
case "cover":
switch action {
case "open_cover":
return domain.CoverOpen{}, nil
case "close_cover":
return domain.CoverClose{}, nil
case "set_cover_position":
return domain.CoverSetPosition{Position: toInt(params["position"])}, nil
}
case "lock":
switch action {
case "lock":
return domain.LockLock{}, nil
case "unlock":
return domain.LockUnlock{}, nil
}
case "fan":
switch action {
case "turn_on":
return domain.FanTurnOn{}, nil
case "turn_off":
return domain.FanTurnOff{}, nil
case "set_percentage":
return domain.FanSetSpeed{Percentage: toInt(params["percentage"])}, nil
}
case "button":
if action == "press" {
return domain.ButtonPress{}, nil
}
case "number":
if action == "set_native_value" {
return domain.NumberSetValue{Value: toFloat64(params["value"])}, nil
}
case "select":
if action == "select_option" {
return domain.SelectOption{Option: toString(params["option"])}, nil
}
case "text":
if action == "set_value" {
return domain.TextSetValue{Value: toString(params["value"])}, nil
}
case "climate":
switch action {
case "set_hvac_mode":
return domain.ClimateSetMode{HVACMode: toString(params["hvac_mode"])}, nil
case "set_temperature":
return domain.ClimateSetTemperature{Temperature: toFloat64(params["temperature"])}, nil
}
case "alarm":
code := toOptionalString(params["code"])
switch action {
case "alarm_arm_home":
return domain.AlarmArmHome{Code: code}, nil
case "alarm_arm_away":
return domain.AlarmArmAway{Code: code}, nil
case "alarm_arm_night":
return domain.AlarmArmNight{Code: code}, nil
case "alarm_disarm":
return domain.AlarmDisarm{Code: code}, nil
}
case "camera":
switch action {
case "turn_on":
return domain.CameraRecordStart{}, nil
case "turn_off":
return domain.CameraRecordStop{}, nil
case "enable_motion_detection":
return domain.CameraEnableMotion{}, nil
case "disable_motion_detection":
return domain.CameraDisableMotion{}, nil
}
case "valve":
switch action {
case "open_valve":
return domain.ValveOpen{}, nil
case "close_valve":
return domain.ValveClose{}, nil
case "set_valve_position":
return domain.ValveSetPosition{Position: toInt(params["position"])}, nil
}
case "siren":
switch action {
case "turn_on":
return domain.SirenTurnOn{}, nil
case "turn_off":
return domain.SirenTurnOff{}, nil
case "set_tone":
return domain.SirenSetTone{Tone: toString(params["tone"])}, nil
}
case "humidifier":
switch action {
case "turn_on":
return domain.HumidifierTurnOn{}, nil
case "turn_off":
return domain.HumidifierTurnOff{}, nil
case "set_humidity":
return domain.HumidifierSetHumidity{Humidity: toInt(params["humidity"])}, nil
case "set_mode":
return domain.HumidifierSetMode{Mode: toString(params["mode"])}, nil
}
case "media_player":
switch action {
case "media_play":
return domain.MediaPlay{}, nil
case "media_pause":
return domain.MediaPause{}, nil
case "media_stop":
return domain.MediaStop{}, nil
case "media_next_track":
return domain.MediaNextTrack{}, nil
case "media_previous_track":
return domain.MediaPreviousTrack{}, nil
case "set_volume_level":
return domain.MediaSetVolume{VolumeLevel: toFloat64(params["volume_level"])}, nil
case "mute_volume":
return domain.MediaMute{Mute: toBool(params["is_volume_muted"])}, nil
case "select_source":
return domain.MediaSelectSource{Source: toString(params["source"])}, nil
}
case "remote":
switch action {
case "turn_on":
return domain.RemoteTurnOn{}, nil
case "turn_off":
return domain.RemoteTurnOff{}, nil
case "set_activity":
return domain.RemoteSetActivity{Activity: toString(params["activity"])}, nil
case "send_command":
return domain.RemoteSendCommand{Command: toString(params["command"])}, nil
}
}
return nil, fmt.Errorf("unsupported: entityType=%q action=%q", entityType, action)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func toInt(v any) int {
switch n := v.(type) {
case int:
return n
case int64:
return int(n)
case float64:
return int(n)
case json.Number:
i, _ := n.Int64()
return int(i)
}
return 0
}

func toFloat64(v any) float64 {
switch n := v.(type) {
case float64:
return n
case int:
return float64(n)
case json.Number:
f, _ := n.Float64()
return f
}
return 0
}

func toString(v any) string {
if s, ok := v.(string); ok {
return s
}
return ""
}

func toBool(v any) bool {
if b, ok := v.(bool); ok {
return b
}
return false
}

func toIntSlice(v any) []int {
if arr, ok := v.([]any); ok {
result := make([]int, len(arr))
for i, el := range arr {
result[i] = toInt(el)
}
return result
}
return nil
}

func toFloatSlice(v any) []float64 {
if arr, ok := v.([]any); ok {
result := make([]float64, len(arr))
for i, el := range arr {
result[i] = toFloat64(el)
}
return result
}
return nil
}

func toOptionalString(v any) *string {
if s, ok := v.(string); ok && s != "" {
return &s
}
return nil
}

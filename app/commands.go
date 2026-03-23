package app

import (
	"encoding/json"
	"log"

	domain "github.com/slidebolt/sb-domain"
	messenger "github.com/slidebolt/sb-messenger-sdk"
)

// handleCommand is called when a SlideBolt command arrives via the messenger
// for an entity owned by this plugin. It updates storage state and pushes
// entity_updated to all connected HA clients.
func (a *App) handleCommand(addr messenger.Address, cmd any) {
	key := domain.EntityKey{Plugin: addr.Plugin, DeviceID: addr.DeviceID, ID: addr.EntityID}

	raw, err := a.store.Get(key)
	if err != nil {
		log.Printf("plugin-homeassistant: get entity %s: %v", key.Key(), err)
		return
	}
	var entity domain.Entity
	if err := json.Unmarshal(raw, &entity); err != nil {
		log.Printf("plugin-homeassistant: unmarshal entity %s: %v", key.Key(), err)
		return
	}

	updated := applyCommand(entity, cmd)
	if err := a.store.Save(updated); err != nil {
		log.Printf("plugin-homeassistant: save entity %s: %v", key.Key(), err)
		return
	}

	if a.srv != nil {
		we := entityToWire(updated)
		a.srv.Broadcast(wireMessage{Type: "entity_updated", Entity: &we})
	}
}

// applyCommand applies a typed SlideBolt domain command to an entity's state.
func applyCommand(entity domain.Entity, cmd any) domain.Entity {
	switch c := cmd.(type) {
	case domain.LightTurnOn:
		if s, ok := entity.State.(domain.Light); ok {
			s.Power = true
			entity.State = s
		}
	case domain.LightTurnOff:
		if s, ok := entity.State.(domain.Light); ok {
			s.Power = false
			_ = c.Transition
			entity.State = s
		}
	case domain.LightSetBrightness:
		if s, ok := entity.State.(domain.Light); ok {
			s.Power = true
			s.Brightness = c.Brightness
			entity.State = s
		}
	case domain.LightSetColorTemp:
		if s, ok := entity.State.(domain.Light); ok {
			s.Power = true
			s.Temperature = c.Mireds
			entity.State = s
		}
	case domain.LightSetRGB:
		if s, ok := entity.State.(domain.Light); ok {
			s.Power = true
			s.RGB = []int{c.R, c.G, c.B}
			if c.Brightness != 0 {
				s.Brightness = c.Brightness
			}
			entity.State = s
		}
	case domain.LightSetRGBW:
		if s, ok := entity.State.(domain.Light); ok {
			s.Power = true
			s.RGBW = []int{c.R, c.G, c.B, c.W}
			if c.Brightness != 0 {
				s.Brightness = c.Brightness
			}
			entity.State = s
		}
	case domain.LightSetRGBWW:
		if s, ok := entity.State.(domain.Light); ok {
			s.Power = true
			s.RGBWW = []int{c.R, c.G, c.B, c.CW, c.WW}
			if c.Brightness != 0 {
				s.Brightness = c.Brightness
			}
			entity.State = s
		}
	case domain.LightSetHS:
		if s, ok := entity.State.(domain.Light); ok {
			s.Power = true
			s.HS = []float64{c.Hue, c.Saturation}
			if c.Brightness != 0 {
				s.Brightness = c.Brightness
			}
			entity.State = s
		}
	case domain.LightSetXY:
		if s, ok := entity.State.(domain.Light); ok {
			s.Power = true
			s.XY = []float64{c.X, c.Y}
			if c.Brightness != 0 {
				s.Brightness = c.Brightness
			}
			entity.State = s
		}
	case domain.LightSetWhite:
		if s, ok := entity.State.(domain.Light); ok {
			s.Power = true
			s.White = c.White
			entity.State = s
		}
	case domain.LightSetEffect:
		if s, ok := entity.State.(domain.Light); ok {
			s.Power = true
			s.Effect = c.Effect
			entity.State = s
		}
	case domain.SwitchTurnOn:
		if s, ok := entity.State.(domain.Switch); ok {
			s.Power = true
			entity.State = s
		}
	case domain.SwitchTurnOff:
		if s, ok := entity.State.(domain.Switch); ok {
			s.Power = false
			entity.State = s
		}
	case domain.SwitchToggle:
		if s, ok := entity.State.(domain.Switch); ok {
			s.Power = !s.Power
			entity.State = s
		}
	case domain.FanTurnOn:
		if s, ok := entity.State.(domain.Fan); ok {
			s.Power = true
			if s.Percentage == 0 {
				s.Percentage = 50
			}
			entity.State = s
		}
	case domain.FanTurnOff:
		if s, ok := entity.State.(domain.Fan); ok {
			s.Power = false
			entity.State = s
		}
	case domain.FanSetSpeed:
		if s, ok := entity.State.(domain.Fan); ok {
			s.Percentage = c.Percentage
			s.Power = c.Percentage > 0
			entity.State = s
		}
	case domain.CoverOpen:
		if s, ok := entity.State.(domain.Cover); ok {
			s.Position = 100
			entity.State = s
		}
	case domain.CoverClose:
		if s, ok := entity.State.(domain.Cover); ok {
			s.Position = 0
			entity.State = s
		}
	case domain.CoverSetPosition:
		if s, ok := entity.State.(domain.Cover); ok {
			s.Position = c.Position
			entity.State = s
		}
	case domain.LockLock:
		if s, ok := entity.State.(domain.Lock); ok {
			s.Locked = true
			entity.State = s
		}
	case domain.LockUnlock:
		if s, ok := entity.State.(domain.Lock); ok {
			s.Locked = false
			entity.State = s
		}
	case domain.ButtonPress:
		// Button press is stateless — no state mutation needed.
	case domain.NumberSetValue:
		if s, ok := entity.State.(domain.Number); ok {
			s.Value = c.Value
			entity.State = s
		}
	case domain.SelectOption:
		if s, ok := entity.State.(domain.Select); ok {
			s.Option = c.Option
			entity.State = s
		}
	case domain.TextSetValue:
		if s, ok := entity.State.(domain.Text); ok {
			s.Value = c.Value
			entity.State = s
		}
	case domain.ClimateSetMode:
		if s, ok := entity.State.(domain.Climate); ok {
			s.HVACMode = c.HVACMode
			entity.State = s
		}
	case domain.ClimateSetTemperature:
		if s, ok := entity.State.(domain.Climate); ok {
			s.Temperature = c.Temperature
			entity.State = s
		}
	case domain.AlarmArmHome:
		if s, ok := entity.State.(domain.Alarm); ok {
			s.AlarmState = "armed_home"
			entity.State = s
		}
	case domain.AlarmArmAway:
		if s, ok := entity.State.(domain.Alarm); ok {
			s.AlarmState = "armed_away"
			entity.State = s
		}
	case domain.AlarmArmNight:
		if s, ok := entity.State.(domain.Alarm); ok {
			s.AlarmState = "armed_night"
			entity.State = s
		}
	case domain.AlarmDisarm:
		if s, ok := entity.State.(domain.Alarm); ok {
			s.AlarmState = "disarmed"
			entity.State = s
		}
	case domain.SirenTurnOn:
		if s, ok := entity.State.(domain.Siren); ok {
			s.IsOn = true
			entity.State = s
		}
	case domain.SirenTurnOff:
		if s, ok := entity.State.(domain.Siren); ok {
			s.IsOn = false
			entity.State = s
		}
	case domain.HumidifierTurnOn:
		if s, ok := entity.State.(domain.Humidifier); ok {
			s.IsOn = true
			entity.State = s
		}
	case domain.HumidifierTurnOff:
		if s, ok := entity.State.(domain.Humidifier); ok {
			s.IsOn = false
			entity.State = s
		}
	case domain.HumidifierSetHumidity:
		if s, ok := entity.State.(domain.Humidifier); ok {
			s.TargetHumidity = c.Humidity
			entity.State = s
		}
	case domain.HumidifierSetMode:
		if s, ok := entity.State.(domain.Humidifier); ok {
			s.Mode = c.Mode
			entity.State = s
		}
	case domain.ValveOpen:
		if s, ok := entity.State.(domain.Valve); ok {
			s.Position = 100
			entity.State = s
		}
	case domain.ValveClose:
		if s, ok := entity.State.(domain.Valve); ok {
			s.Position = 0
			entity.State = s
		}
	case domain.ValveSetPosition:
		if s, ok := entity.State.(domain.Valve); ok {
			s.Position = c.Position
			entity.State = s
		}
	case domain.MediaPlay:
		if s, ok := entity.State.(domain.MediaPlayer); ok {
			s.State = "playing"
			entity.State = s
		}
	case domain.MediaPause:
		if s, ok := entity.State.(domain.MediaPlayer); ok {
			s.State = "paused"
			entity.State = s
		}
	case domain.MediaStop:
		if s, ok := entity.State.(domain.MediaPlayer); ok {
			s.State = "idle"
			entity.State = s
		}
	case domain.MediaSetVolume:
		if s, ok := entity.State.(domain.MediaPlayer); ok {
			s.VolumeLevel = c.VolumeLevel
			entity.State = s
		}
	case domain.MediaMute:
		if s, ok := entity.State.(domain.MediaPlayer); ok {
			s.IsVolumeMuted = c.Mute
			entity.State = s
		}
	case domain.MediaNextTrack:
		// Stateless — no state mutation needed.
	case domain.MediaPreviousTrack:
		// Stateless — no state mutation needed.
	case domain.MediaSelectSource:
		if s, ok := entity.State.(domain.MediaPlayer); ok {
			s.Source = c.Source
			entity.State = s
		}
	case domain.RemoteTurnOn:
		if s, ok := entity.State.(domain.Remote); ok {
			s.IsOn = true
			entity.State = s
		}
	case domain.RemoteTurnOff:
		if s, ok := entity.State.(domain.Remote); ok {
			s.IsOn = false
			entity.State = s
		}
	case domain.RemoteSendCommand:
		// Stateless — no state mutation needed.
	case domain.CameraRecordStart:
		if s, ok := entity.State.(domain.Camera); ok {
			s.IsRecording = true
			entity.State = s
		}
	case domain.CameraRecordStop:
		if s, ok := entity.State.(domain.Camera); ok {
			s.IsRecording = false
			entity.State = s
		}
	case domain.CameraEnableMotion:
		if s, ok := entity.State.(domain.Camera); ok {
			s.MotionDetection = true
			entity.State = s
		}
	case domain.CameraDisableMotion:
		if s, ok := entity.State.(domain.Camera); ok {
			s.MotionDetection = false
			entity.State = s
		}
	default:
		log.Printf("plugin-homeassistant: unknown command %T for %s", cmd, entity.Key())
	}
	return entity
}

// applyHACommand applies a raw HA wire command (string + params map) to an entity.
// Used when commands arrive from HA via WebSocket rather than the SlideBolt messenger.
func applyHACommand(entity domain.Entity, command string, params map[string]any) domain.Entity {
	switch entity.Type {
	case "light":
		s, _ := entity.State.(domain.Light)
		switch command {
		case "turn_on":
			s.Power = true
			if b, ok := params["brightness"].(float64); ok {
				s.Brightness = int(b)
			}
			if rgb, ok := params["rgb_color"].([]any); ok && len(rgb) == 3 {
				r, rok := rgb[0].(float64)
				g, gok := rgb[1].(float64)
				b, bok := rgb[2].(float64)
				if rok && gok && bok {
					s.RGB = []int{int(r), int(g), int(b)}
				}
			}
			if ct, ok := params["color_temp_kelvin"].(float64); ok {
				s.Temperature = int(ct) // Light.Temperature is still int (mireds)
			}
		case "turn_off":
			s.Power = false
		}
		entity.State = s

	case "switch":
		s, _ := entity.State.(domain.Switch)
		switch command {
		case "turn_on":
			s.Power = true
		case "turn_off":
			s.Power = false
		}
		entity.State = s

	case "cover":
		s, _ := entity.State.(domain.Cover)
		switch command {
		case "open_cover":
			s.Position = 100
		case "close_cover":
			s.Position = 0
		case "set_cover_position":
			if p, ok := params["position"].(float64); ok {
				s.Position = int(p)
			}
		}
		entity.State = s

	case "lock":
		s, _ := entity.State.(domain.Lock)
		switch command {
		case "lock":
			s.Locked = true
		case "unlock":
			s.Locked = false
		}
		entity.State = s

	case "fan":
		s, _ := entity.State.(domain.Fan)
		switch command {
		case "turn_on":
			s.Power = true
			if s.Percentage == 0 {
				s.Percentage = 50
			}
		case "turn_off":
			s.Power = false
		case "set_percentage":
			if p, ok := params["percentage"].(float64); ok {
				s.Percentage = int(p)
				s.Power = int(p) > 0
			}
		}
		entity.State = s

	case "climate":
		s, _ := entity.State.(domain.Climate)
		switch command {
		case "set_hvac_mode":
			if m, ok := params["hvac_mode"].(string); ok {
				s.HVACMode = m
			}
		case "set_temperature":
			if t, ok := params["temperature"].(float64); ok {
				s.Temperature = t
			}
		}
		entity.State = s

	case "number":
		s, _ := entity.State.(domain.Number)
		if command == "set_value" {
			if v, ok := params["value"].(float64); ok {
				s.Value = v
			}
		}
		entity.State = s

	case "select":
		s, _ := entity.State.(domain.Select)
		if command == "select_option" {
			if opt, ok := params["option"].(string); ok {
				s.Option = opt
			}
		}
		entity.State = s

	case "text":
		s, _ := entity.State.(domain.Text)
		if command == "set_value" {
			if v, ok := params["value"].(string); ok {
				s.Value = v
			}
		}
		entity.State = s
	}
	return entity
}

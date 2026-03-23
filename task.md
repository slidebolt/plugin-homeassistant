# Integration Test Parity: All Entity Types

## Problem

Only 2/20 entity types have integration tests (light, switch). The mock server in slidebolt-hacs
proves all 20 entity types work end-to-end, but plugin-homeassistant has no equivalent coverage.
Additionally, there are **command name mismatches** between what slidebolt-hacs sends and what
`FromHA` expects for 4 entity types, meaning those entities are fundamentally broken.

## Approach

1. Fix FromHA command name mismatches (4 entity types broken)
2. Fix applyCommand gaps (commands that can be translated but aren't applied)
3. Write integration tests for all 20 entity types proving snapshot + command round-trip

---

## Phase 1: Fix FromHA Command Name Mismatches

These are bugs — the plugin receives commands from slidebolt-hacs but can't recognize them
because the action names don't match.

### 1.1 Number: `set_native_value` → `set_value`
- **HA sends**: `"set_native_value"` with `{"value": float}`
- **FromHA expects**: `"set_value"`
- **Fix**: Change FromHA to match `"set_native_value"`

### 1.2 Alarm: `alarm_arm_home` → `arm_home` (and friends)
- **HA sends**: `"alarm_arm_home"`, `"alarm_arm_away"`, `"alarm_arm_night"`, `"alarm_disarm"`, `"alarm_trigger"`
- **FromHA expects**: `"arm_home"`, `"arm_away"`, `"arm_night"`, `"disarm"` (no trigger)
- **Fix**: Change FromHA to match `"alarm_*"` prefixed names, add `"alarm_trigger"`

### 1.3 Camera: completely wrong names
- **HA sends**: `"turn_on"`, `"turn_off"`, `"enable_motion_detection"`, `"disable_motion_detection"`, `"ptz"`
- **FromHA expects**: `"record_start"`, `"record_stop"`, `"enable_motion"`, `"disable_motion"`
- **Fix**: Change FromHA to match HA names. Map turn_on→CameraRecordStart, turn_off→CameraRecordStop,
  enable_motion_detection→CameraEnableMotion, disable_motion_detection→CameraDisableMotion
- **Discussion**: PTZ has no domain command (see below)

### 1.4 MediaPlayer: most names wrong
- **HA sends**: `"media_play"`, `"media_pause"`, `"media_stop"`, `"media_next_track"`,
  `"media_previous_track"`, `"set_volume_level"`, `"mute_volume"`, `"select_source"`,
  `"turn_on"`, `"turn_off"`
- **FromHA expects**: `"play"`, `"pause"`, `"stop"`, `"next_track"`, `"previous_track"`,
  `"set_volume"`, `"mute"`, `"select_source"` (no turn_on/turn_off)
- **Fix**: Change FromHA to match `"media_*"` prefixed names, `"set_volume_level"`, `"mute_volume"`,
  add `"turn_on"`/`"turn_off"` (→ MediaPlay/MediaStop or new domain commands)

---

## Phase 2: Fix applyCommand Gaps

These are commands where FromHA can translate them but applyCommand silently ignores them.

### 2.1 Camera: 0/4 commands handled
- Add cases for CameraRecordStart, CameraRecordStop, CameraEnableMotion, CameraDisableMotion

### 2.2 Alarm: arm commands don't mutate state
- Currently: `case domain.AlarmArmHome, domain.AlarmArmAway, domain.AlarmArmNight:` → no-op comment
- Mock sets: `alarm_state = "armed_home"` etc
- **Fix**: Mutate AlarmState for all arm commands + add AlarmTrigger

### 2.3 MediaPlayer: 5/8 missing
- Existing: Play, Pause, Stop, SetVolume, Mute
- Missing: NextTrack, PreviousTrack, SelectSource (+ TurnOn/TurnOff if mapped)

### 2.4 Humidifier: SetMode missing
- FromHA produces HumidifierSetMode but applyCommand has no case for it

### 2.5 Remote: SendCommand missing
- FromHA produces RemoteSendCommand but applyCommand has no case
- Note: send_command is stateless (like button press) — may not need state mutation

---

## Phase 3: Integration Tests

One test per entity type. Each test:
1. Seeds an entity with initial state via `testEntity()`
2. Starts server, waits for HA connection
3. Verifies entity appears in HA REST API with correct state (snapshot test)
4. Sends a command via HA service API
5. Waits for HA state to reflect the update (command round-trip test)

Read-only entities (sensor, binary_sensor, event) only need snapshot verification.

### Entity Test Plan

| # | Entity Type     | Snapshot Fields to Verify                  | Command to Test                          |
|---|----------------|--------------------------------------------|------------------------------------------|
| 1 | light          | DONE (brightness, color_mode)              | DONE (turn_on + brightness)              |
| 2 | switch         | DONE (state=off)                           | turn_on → state=on                       |
| 3 | cover          | current_position, state=open               | set_cover_position(50) → position=50     |
| 4 | lock           | state=locked                               | unlock → state=unlocked                  |
| 5 | fan            | percentage, state=off                      | turn_on → state=on                       |
| 6 | sensor         | native_value, device_class, unit           | (read-only, snapshot only)               |
| 7 | binary_sensor  | state=on/off, device_class                 | (read-only, snapshot only)               |
| 8 | climate        | hvac_mode, temperature, current_temp       | set_temperature(25) → target=25          |
| 9 | button         | device_class                               | press → service returns 200              |
| 10| number         | native_value, min, max, step               | set_native_value(75) → value=75          |
| 11| select         | current_option, options list               | select_option("away") → option=away      |
| 12| text           | native_value, mode                         | set_value("Hello") → value=Hello         |
| 13| alarm          | alarm_state=disarmed                       | alarm_arm_home → state=armed_home        |
| 14| camera         | is_recording, motion_detection_enabled     | enable_motion_detection → motion=true    |
| 15| valve          | current_position, state=open               | set_valve_position(50) → position=50     |
| 16| siren          | state=off                                  | turn_on → state=on                       |
| 17| humidifier     | target_humidity, mode, state=off           | set_humidity(60) → target=60             |
| 18| media_player   | state, volume_level, source                | media_play → state=playing               |
| 19| remote         | state=on, current_activity                 | turn_off → state=off                     |
| 20| event          | event_types, device_class                  | (read-only, snapshot only)               |

---

## Results

**All tests passing** as of completion:

### Unit Tests
- `internal/translate` — all FromHA command name tests pass
- `app` — all applyCommand tests pass

### Integration Tests (38 subtests total)
- `TestIntegration_HAConnectsOnStartup` — PASS
- `TestIntegration_SnapshotEntitiesReachHA` — PASS (light, switch)
- `TestIntegration_CommandRoundTrip` — PASS (light brightness)
- `TestIntegration_AllEntitySnapshots` — PASS (19 entity types)
- `TestIntegration_AllEntityCommands` — PASS (16 commandable entities)

### Additional Fixes Made
- **Fan**: Added TURN_ON/TURN_OFF feature flags (SET_SPEED|TURN_OFF|TURN_ON = 49)
- **Climate**: Always provide min_temp/max_temp defaults (7°/35°) to avoid HA nil comparison
- **Server**: Explicit WebSocket close on Stop() for clean reconnection between tests
- **Server**: SO_REUSEADDR on listener for fast port reuse in sequential tests

### Known Issue (slidebolt-hacs bug)
- `fan.turn_on` service fails because `SlideboltFan.async_turn_on()` signature is missing
  positional args (`percentage`, `preset_mode`) that HA passes. Test uses `set_percentage` instead.

---

## Discussion Items (No Clear Mapping)

### Camera PTZ
- HA sends `"ptz"` with `{"movement": "UP", ...}`
- No domain command exists for PTZ
- Options: (a) add domain.CameraPTZ command, (b) skip for now
- **Recommendation**: skip — PTZ is niche and needs domain model work

### Camera turn_on/turn_off vs record start/stop
- HA sends turn_on/turn_off for streaming
- Domain has CameraRecordStart/CameraRecordStop for recording
- The mock maps turn_on→is_streaming, turn_off→!is_streaming
- **Current plan**: map turn_on→CameraRecordStart to get basic functionality working,
  but note this conflates streaming with recording. May need domain refinement later.

### Light RGBW/RGBWW/HS/XY/White from HA
- HA sends these via turn_on kwargs (e.g. `{"rgbw_color": [255,0,0,128]}`)
- Domain supports all of them (LightSetRGBW, etc.)
- FromHA only handles brightness, rgb_color, color_temp_kelvin today
- **Recommendation**: Add support — these are standard HA light features with clear domain mapping

### Cover stop_cover
- HA sends "stop_cover" but no domain command or FromHA handler exists
- **Recommendation**: skip — requires domain model change (domain.CoverStop)

### MediaPlayer turn_on/turn_off
- HA sends these but no direct domain command
- Mock doesn't handle them either
- **Recommendation**: skip — not a core media player action

### Remote send_command
- Stateless action (press a button on remote). No state mutation.
- applyCommand should accept it but not mutate state (like ButtonPress)

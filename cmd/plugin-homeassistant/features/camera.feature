Feature: Camera Entity

  Scenario: Create with default state
    Given a camera entity "test.dev1.cam001" named "Front Door Camera" not streaming
    When I retrieve "test.dev1.cam001"
    Then the entity type is "camera"
    And the camera is not streaming

  Scenario: Delete removes entity
    Given a camera entity "test.dev1.cam002" named "Camera" not streaming
    When I delete "test.dev1.cam002"
    Then retrieving "test.dev1.cam002" should fail

  Scenario: camera_record_start command is dispatched
    Given a command listener on "test.>"
    When I send "camera_record_start" to "test.dev1.cam001"
    Then the received command action is "camera_record_start"

  Scenario: ToHA wire state has correct fields
    Given a camera entity "test.dev1.toha" named "Camera" not streaming
    When I translate "test.dev1.toha" to HA wire format
    Then the HA wire state field "is_streaming" is false

  Scenario: FromHA record_start maps to domain command
    When HA sends "record_start" to entity type "camera" with no params
    Then the domain command type is "CameraRecordStart"

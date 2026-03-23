Feature: Binary Sensor Entity
  # Source ref: contracts/binary_sensor.md

  Scenario: Create with default state
    Given a binary_sensor entity "test.dev1.bs001" named "Motion Sensor" with on false
    When I retrieve "test.dev1.bs001"
    Then the entity type is "binary_sensor"
    And the binary sensor is off

  Scenario: Update is reflected on retrieval
    Given a binary_sensor entity "test.dev1.bs002" named "Sensor" with on false
    When I retrieve "test.dev1.bs002"
    Then the binary sensor is off

  Scenario: Delete removes entity
    Given a binary_sensor entity "test.dev1.bs003" named "Sensor" with on false
    When I delete "test.dev1.bs003"
    Then retrieving "test.dev1.bs003" should fail

  Scenario: Binary sensor with device class
    Given a binary_sensor entity "test.dev1.bs004" named "Door" with on false and device_class "door"
    When I retrieve "test.dev1.bs004"
    Then the binary sensor device_class is "door"

  Scenario: Raw discovery data is stored internally and hidden from queries
    Given a binary_sensor entity "test.dev1.bs001" named "Motion Sensor" with on false
    And I write internal data for "test.dev1.bs001" with payload '{"stateTopic":"zigbee2mqtt/motion/state"}'
    When I read internal data for "test.dev1.bs001"
    Then the internal data matches '{"stateTopic":"zigbee2mqtt/motion/state"}'
    And querying type "binary_sensor" returns only state entities

  Scenario: ToHA wire state has correct fields
    Given a binary_sensor entity "test.dev1.toha" named "Test Sensor" with on true
    When I translate "test.dev1.toha" to HA wire format
    Then the HA wire state field "is_on" is true

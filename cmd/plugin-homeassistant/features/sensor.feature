Feature: Sensor Entity
  # Source ref: contracts/sensor.md

  Scenario: Create with default state
    Given a sensor entity "test.dev1.sensor001" named "Temperature Sensor" with value "22.5" and unit "°C"
    When I retrieve "test.dev1.sensor001"
    Then the entity type is "sensor"
    And the sensor value is "22.5"
    And the sensor unit is "°C"

  Scenario: Query by type
    Given a sensor entity "test.dev1.sensor002" named "Sensor" with value "10" and unit ""
    And a binary_sensor entity "test.dev1.bs001" named "Binary" with on false
    When I query where "type" equals "sensor"
    Then the results include "test.dev1.sensor002"
    And the results do not include "test.dev1.bs001"

  Scenario: Update is reflected on retrieval
    Given a sensor entity "test.dev1.sensor003" named "Sensor" with value "10" and unit "°C"
    And I update sensor "test.dev1.sensor003" to value "25" and unit "°C"
    When I retrieve "test.dev1.sensor003"
    Then the sensor value is "25"

  Scenario: Delete removes entity
    Given a sensor entity "test.dev1.sensor004" named "Sensor" with value "10" and unit ""
    When I delete "test.dev1.sensor004"
    Then retrieving "test.dev1.sensor004" should fail

  Scenario: Sensor with device class
    Given a sensor entity "test.dev1.sensor005" named "Motion" with value "0" unit "" and device_class "motion"
    When I retrieve "test.dev1.sensor005"
    Then the sensor device_class is "motion"

  Scenario: Raw discovery data is stored internally and hidden from queries
    Given a sensor entity "test.dev1.sensor001" named "Temperature Sensor" with value "22.5" and unit "°C"
    And I write internal data for "test.dev1.sensor001" with payload '{"stateTopic":"zigbee2mqtt/sensor/state"}'
    When I read internal data for "test.dev1.sensor001"
    Then the internal data matches '{"stateTopic":"zigbee2mqtt/sensor/state"}'
    And querying type "sensor" returns only state entities

  Scenario: ToHA wire state has correct fields
    Given a sensor entity "test.dev1.toha" named "Test Sensor" with value "22.5" and unit "°C"
    When I translate "test.dev1.toha" to HA wire format
    Then the HA wire state field "state_class" equals "measurement"

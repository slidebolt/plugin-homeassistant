Feature: Fan Entity
  # Source ref: contracts/fan.md

  Scenario: Create with default state
    Given a fan entity "test.dev1.fan001" named "Ceiling Fan" with power off and percentage 0
    When I retrieve "test.dev1.fan001"
    Then the entity type is "fan"
    And the fan power is off

  Scenario: Update is reflected on retrieval
    Given a fan entity "test.dev1.fan002" named "Fan" with power off and percentage 0
    And I update fan "test.dev1.fan002" to power on and percentage 75
    When I retrieve "test.dev1.fan002"
    Then the fan power is on
    And the fan percentage is 75

  Scenario: Delete removes entity
    Given a fan entity "test.dev1.fan003" named "Fan" with power off and percentage 0
    When I delete "test.dev1.fan003"
    Then retrieving "test.dev1.fan003" should fail

  Scenario: fan_turn_on command is dispatched
    Given a command listener on "test.>"
    When I send "fan_turn_on" to "test.dev1.fan001"
    Then the received command action is "fan_turn_on"

  Scenario: fan_set_speed command is dispatched
    Given a command listener on "test.>"
    When I send "fan_set_speed" with percentage 75 to "test.dev1.fan001"
    Then the received command action is "fan_set_speed"

  Scenario: Raw discovery data is stored internally and hidden from queries
    Given a fan entity "test.dev1.fan001" named "Ceiling Fan" with power off and percentage 0
    And I write internal data for "test.dev1.fan001" with payload '{"commandTopic":"zigbee2mqtt/fan/set"}'
    When I read internal data for "test.dev1.fan001"
    Then the internal data matches '{"commandTopic":"zigbee2mqtt/fan/set"}'
    And querying type "fan" returns only state entities

  Scenario: ToHA wire state has correct fields
    Given a fan entity "test.dev1.toha" named "Test Fan" with power on and percentage 50
    When I translate "test.dev1.toha" to HA wire format
    Then the HA wire state field "is_on" is true
    And the HA wire state field "percentage" equals 50

  Scenario: FromHA turn_on maps to FanTurnOn
    When HA sends "turn_on" to entity type "fan" with no params
    Then the domain command type is "FanTurnOn"

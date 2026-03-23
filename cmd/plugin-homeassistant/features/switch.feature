Feature: Switch Entity
  # Source ref: contracts/switch.md

  Scenario: Create with default state
    Given a switch entity "test.dev1.sw001" named "Living Room Switch" with power off
    When I retrieve "test.dev1.sw001"
    Then the entity type is "switch"
    And the switch power is off

  Scenario: Update is reflected on retrieval
    Given a switch entity "test.dev1.sw002" named "Switch" with power off
    And I update switch "test.dev1.sw002" to power on
    When I retrieve "test.dev1.sw002"
    Then the switch power is on

  Scenario: Delete removes entity
    Given a switch entity "test.dev1.sw003" named "Switch" with power off
    When I delete "test.dev1.sw003"
    Then retrieving "test.dev1.sw003" should fail

  Scenario: switch_turn_on command is dispatched
    Given a command listener on "test.>"
    When I send "switch_turn_on" to "test.dev1.sw001"
    Then the received command action is "switch_turn_on"

  Scenario: switch_turn_off command is dispatched
    Given a command listener on "test.>"
    When I send "switch_turn_off" to "test.dev1.sw001"
    Then the received command action is "switch_turn_off"

  Scenario: Raw discovery data is stored internally and hidden from queries
    Given a switch entity "test.dev1.sw001" named "Living Room Switch" with power off
    And I write internal data for "test.dev1.sw001" with payload '{"commandTopic":"zigbee2mqtt/switch/set"}'
    When I read internal data for "test.dev1.sw001"
    Then the internal data matches '{"commandTopic":"zigbee2mqtt/switch/set"}'
    And querying type "switch" returns only state entities

  Scenario: ToHA wire state has correct fields
    Given a switch entity "test.dev1.toha" named "Test Switch" with power off
    When I translate "test.dev1.toha" to HA wire format
    Then the HA wire state field "is_on" is false

  Scenario: FromHA turn_on maps to SwitchTurnOn
    When HA sends "turn_on" to entity type "switch" with no params
    Then the domain command type is "SwitchTurnOn"

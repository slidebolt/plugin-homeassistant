Feature: Number Entity
  # Source ref: contracts/number.md

  Scenario: Create with default state
    Given a number entity "test.dev1.num001" named "Brightness" with value 5.0 min 1.0 max 10.0 step 0.5 unit ""
    When I retrieve "test.dev1.num001"
    Then the entity type is "number"
    And the number value is 5

  Scenario: Update is reflected on retrieval
    Given a number entity "test.dev1.num002" named "Number" with value 1.0 min 1.0 max 10.0 step 1.0 unit ""
    And I update number "test.dev1.num002" to value 7.5
    When I retrieve "test.dev1.num002"
    Then the number value is 7.5

  Scenario: Delete removes entity
    Given a number entity "test.dev1.num003" named "Number" with value 1.0 min 0.0 max 100.0 step 1.0 unit ""
    When I delete "test.dev1.num003"
    Then retrieving "test.dev1.num003" should fail

  Scenario: number_set_value command is dispatched
    Given a command listener on "test.>"
    When I send "number_set_value" with value 5.0 to "test.dev1.num001"
    Then the received command action is "number_set_value"

  Scenario: Raw discovery data is stored internally and hidden from queries
    Given a number entity "test.dev1.num001" named "Brightness" with value 5.0 min 1.0 max 10.0 step 0.5 unit ""
    And I write internal data for "test.dev1.num001" with payload '{"commandTopic":"zigbee2mqtt/number/set"}'
    When I read internal data for "test.dev1.num001"
    Then the internal data matches '{"commandTopic":"zigbee2mqtt/number/set"}'
    And querying type "number" returns only state entities

  Scenario: ToHA wire state has correct fields
    Given a number entity "test.dev1.toha" named "Test Number" with value 5.0 min 1.0 max 10.0 step 0.5 unit ""
    When I translate "test.dev1.toha" to HA wire format
    Then the HA wire state field "mode" equals "slider"

  Scenario: FromHA set_value maps to NumberSetValue
    When HA sends "set_value" to entity type "number" with params '{"value": 5.0}'
    Then the domain command type is "NumberSetValue"

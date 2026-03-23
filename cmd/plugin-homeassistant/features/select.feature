Feature: Select Entity
  # Source ref: contracts/select.md

  Scenario: Create with default state
    Given a select entity "test.dev1.sel001" named "Mode Selector" with option "eco" and options "eco,comfort,boost"
    When I retrieve "test.dev1.sel001"
    Then the entity type is "select"
    And the select option is "eco"

  Scenario: Update is reflected on retrieval
    Given a select entity "test.dev1.sel002" named "Selector" with option "eco" and options "eco,comfort,boost"
    And I update select "test.dev1.sel002" to option "boost"
    When I retrieve "test.dev1.sel002"
    Then the select option is "boost"

  Scenario: Delete removes entity
    Given a select entity "test.dev1.sel003" named "Selector" with option "eco" and options "eco,comfort"
    When I delete "test.dev1.sel003"
    Then retrieving "test.dev1.sel003" should fail

  Scenario: select_option command is dispatched
    Given a command listener on "test.>"
    When I send "select_option" with option "comfort" to "test.dev1.sel001"
    Then the received command action is "select_option"

  Scenario: Raw discovery data is stored internally and hidden from queries
    Given a select entity "test.dev1.sel001" named "Mode Selector" with option "eco" and options "eco,comfort,boost"
    And I write internal data for "test.dev1.sel001" with payload '{"commandTopic":"zigbee2mqtt/select/set"}'
    When I read internal data for "test.dev1.sel001"
    Then the internal data matches '{"commandTopic":"zigbee2mqtt/select/set"}'
    And querying type "select" returns only state entities

  Scenario: ToHA wire state has correct fields
    Given a select entity "test.dev1.toha" named "Test Select" with option "eco" and options "eco,comfort,boost"
    When I translate "test.dev1.toha" to HA wire format
    Then the HA wire state field "current_option" equals "eco"

  Scenario: FromHA select_option maps to SelectOption
    When HA sends "select_option" to entity type "select" with params '{"option": "eco"}'
    Then the domain command type is "SelectOption"

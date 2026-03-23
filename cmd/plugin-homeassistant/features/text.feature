Feature: Text Entity
  # Source ref: contracts/text.md

  Scenario: Create with default state
    Given a text entity "test.dev1.txt001" named "Label" with value "hello"
    When I retrieve "test.dev1.txt001"
    Then the entity type is "text"
    And the text value is "hello"

  Scenario: Update is reflected on retrieval
    Given a text entity "test.dev1.txt002" named "Text" with value "initial"
    And I update text "test.dev1.txt002" to value "updated"
    When I retrieve "test.dev1.txt002"
    Then the text value is "updated"

  Scenario: Delete removes entity
    Given a text entity "test.dev1.txt003" named "Text" with value "hello"
    When I delete "test.dev1.txt003"
    Then retrieving "test.dev1.txt003" should fail

  Scenario: text_set_value command is dispatched
    Given a command listener on "test.>"
    When I send "text_set_value" with value "world" to "test.dev1.txt001"
    Then the received command action is "text_set_value"

  Scenario: Raw discovery data is stored internally and hidden from queries
    Given a text entity "test.dev1.txt001" named "Label" with value "hello"
    And I write internal data for "test.dev1.txt001" with payload '{"commandTopic":"zigbee2mqtt/text/set"}'
    When I read internal data for "test.dev1.txt001"
    Then the internal data matches '{"commandTopic":"zigbee2mqtt/text/set"}'
    And querying type "text" returns only state entities

  Scenario: ToHA wire state has correct fields
    Given a text entity "test.dev1.toha" named "Test Text" with value "hello"
    When I translate "test.dev1.toha" to HA wire format
    Then the HA wire state field "native_value" equals "hello"

  Scenario: FromHA set_value maps to TextSetValue
    When HA sends "set_value" to entity type "text" with params '{"value": "hello"}'
    Then the domain command type is "TextSetValue"

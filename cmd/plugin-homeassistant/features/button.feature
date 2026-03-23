Feature: Button Entity
  # Source ref: contracts/button.md

  Scenario: Create with default state
    Given a button entity "test.dev1.btn001" named "Doorbell Button" with presses 0
    When I retrieve "test.dev1.btn001"
    Then the entity type is "button"
    And the button presses is 0

  Scenario: Delete removes entity
    Given a button entity "test.dev1.btn002" named "Button" with presses 0
    When I delete "test.dev1.btn002"
    Then retrieving "test.dev1.btn002" should fail

  Scenario: button_press command is dispatched
    Given a command listener on "test.>"
    When I send "button_press" to "test.dev1.btn001"
    Then the received command action is "button_press"

  Scenario: Raw discovery data is stored internally and hidden from queries
    Given a button entity "test.dev1.btn001" named "Doorbell Button" with presses 0
    And I write internal data for "test.dev1.btn001" with payload '{"commandTopic":"zigbee2mqtt/button/action"}'
    When I read internal data for "test.dev1.btn001"
    Then the internal data matches '{"commandTopic":"zigbee2mqtt/button/action"}'
    And querying type "button" returns only state entities

  Scenario: FromHA press maps to ButtonPress
    When HA sends "press" to entity type "button" with no params
    Then the domain command type is "ButtonPress"

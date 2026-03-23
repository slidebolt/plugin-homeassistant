Feature: Cover Entity
  # Source ref: contracts/cover.md

  Scenario: Create with default state
    Given a cover entity "test.dev1.cover001" named "Living Room Blind" with position 100
    When I retrieve "test.dev1.cover001"
    Then the entity type is "cover"
    And the cover position is 100

  Scenario: Update is reflected on retrieval
    Given a cover entity "test.dev1.cover002" named "Cover" with position 100
    And I update cover "test.dev1.cover002" to position 50
    When I retrieve "test.dev1.cover002"
    Then the cover position is 50

  Scenario: Delete removes entity
    Given a cover entity "test.dev1.cover003" named "Cover" with position 0
    When I delete "test.dev1.cover003"
    Then retrieving "test.dev1.cover003" should fail

  Scenario: cover_open command is dispatched
    Given a command listener on "test.>"
    When I send "cover_open" to "test.dev1.cover001"
    Then the received command action is "cover_open"

  Scenario: cover_set_position command is dispatched
    Given a command listener on "test.>"
    When I send "cover_set_position" with position 50 to "test.dev1.cover001"
    Then the received command action is "cover_set_position"

  Scenario: Raw discovery data is stored internally and hidden from queries
    Given a cover entity "test.dev1.cover001" named "Living Room Blind" with position 100
    And I write internal data for "test.dev1.cover001" with payload '{"commandTopic":"zigbee2mqtt/blind/set"}'
    When I read internal data for "test.dev1.cover001"
    Then the internal data matches '{"commandTopic":"zigbee2mqtt/blind/set"}'
    And querying type "cover" returns only state entities

  Scenario: ToHA wire state has correct fields
    Given a cover entity "test.dev1.toha" named "Test Cover" with position 100
    When I translate "test.dev1.toha" to HA wire format
    Then the HA wire state field "current_position" equals 100
    And the HA wire state field "state" equals "open"

  Scenario: FromHA open maps to CoverOpen
    When HA sends "open" to entity type "cover" with no params
    Then the domain command type is "CoverOpen"

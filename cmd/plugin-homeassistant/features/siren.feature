Feature: Siren Entity

  Scenario: Create with default state
    Given a siren entity "test.dev1.siren001" named "Alarm Siren" with power off
    When I retrieve "test.dev1.siren001"
    Then the entity type is "siren"
    And the siren is off

  Scenario: Delete removes entity
    Given a siren entity "test.dev1.siren002" named "Siren" with power off
    When I delete "test.dev1.siren002"
    Then retrieving "test.dev1.siren002" should fail

  Scenario: siren_turn_on command is dispatched
    Given a command listener on "test.>"
    When I send "siren_turn_on" to "test.dev1.siren001"
    Then the received command action is "siren_turn_on"

  Scenario: ToHA wire state has correct fields
    Given a siren entity "test.dev1.toha" named "Siren" with power off
    When I translate "test.dev1.toha" to HA wire format
    Then the HA wire state field "is_on" is false

  Scenario: FromHA turn_on maps to SirenTurnOn
    When HA sends "turn_on" to entity type "siren" with no params
    Then the domain command type is "SirenTurnOn"

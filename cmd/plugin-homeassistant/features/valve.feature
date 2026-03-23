Feature: Valve Entity

  Scenario: Create with default state
    Given a valve entity "test.dev1.valve001" named "Water Main" at position 100
    When I retrieve "test.dev1.valve001"
    Then the entity type is "valve"
    And the valve position is 100

  Scenario: Delete removes entity
    Given a valve entity "test.dev1.valve002" named "Valve" at position 0
    When I delete "test.dev1.valve002"
    Then retrieving "test.dev1.valve002" should fail

  Scenario: valve_open command is dispatched
    Given a command listener on "test.>"
    When I send "valve_open" to "test.dev1.valve001"
    Then the received command action is "valve_open"

  Scenario: ToHA wire state has correct fields
    Given a valve entity "test.dev1.toha" named "Valve" at position 100
    When I translate "test.dev1.toha" to HA wire format
    Then the HA wire state field "current_valve_position" equals 100
    And the HA wire state field "state" equals "open"

  Scenario: FromHA open maps to ValveOpen
    When HA sends "open" to entity type "valve" with no params
    Then the domain command type is "ValveOpen"

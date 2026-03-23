Feature: Alarm Entity

  Scenario: Create with default state
    Given an alarm entity "test.dev1.alarm001" named "Home Alarm" with state "disarmed"
    When I retrieve "test.dev1.alarm001"
    Then the entity type is "alarm"
    And the alarm state is "disarmed"

  Scenario: Update alarm state
    Given an alarm entity "test.dev1.alarm002" named "Alarm" with state "disarmed"
    And I update "test.dev1.alarm002" alarm state to "armed_home"
    When I retrieve "test.dev1.alarm002"
    Then the alarm state is "armed_home"

  Scenario: Delete removes entity
    Given an alarm entity "test.dev1.alarm003" named "Alarm" with state "disarmed"
    When I delete "test.dev1.alarm003"
    Then retrieving "test.dev1.alarm003" should fail

  Scenario: alarm_disarm command is dispatched
    Given a command listener on "test.>"
    When I send "alarm_disarm" to "test.dev1.alarm001"
    Then the received command action is "alarm_disarm"

  Scenario: ToHA wire state has correct fields
    Given an alarm entity "test.dev1.toha" named "Alarm" with state "disarmed"
    When I translate "test.dev1.toha" to HA wire format
    Then the HA wire state field "alarm_state" equals "disarmed"

  Scenario: FromHA disarm maps to domain command
    When HA sends "disarm" to entity type "alarm" with no params
    Then the domain command type is "AlarmDisarm"

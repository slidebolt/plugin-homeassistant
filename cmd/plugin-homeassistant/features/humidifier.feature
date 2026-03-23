Feature: Humidifier Entity

  Scenario: Create with default state
    Given a humidifier entity "test.dev1.hum001" named "Bedroom Humidifier" with target humidity 45
    When I retrieve "test.dev1.hum001"
    Then the entity type is "humidifier"
    And the humidifier target humidity is 45

  Scenario: Delete removes entity
    Given a humidifier entity "test.dev1.hum002" named "Humidifier" with target humidity 50
    When I delete "test.dev1.hum002"
    Then retrieving "test.dev1.hum002" should fail

  Scenario: humidifier_turn_on command is dispatched
    Given a command listener on "test.>"
    When I send "humidifier_turn_on" to "test.dev1.hum001"
    Then the received command action is "humidifier_turn_on"

  Scenario: ToHA wire state has correct fields
    Given a humidifier entity "test.dev1.toha" named "Humidifier" with target humidity 45
    When I translate "test.dev1.toha" to HA wire format
    Then the HA wire state field "target_humidity" equals 45

  Scenario: FromHA turn_on maps to HumidifierTurnOn
    When HA sends "turn_on" to entity type "humidifier" with no params
    Then the domain command type is "HumidifierTurnOn"

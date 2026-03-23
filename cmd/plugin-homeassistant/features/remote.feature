Feature: Remote Entity

  Scenario: Create with default state
    Given a remote entity "test.dev1.remote001" named "TV Remote" with power on
    When I retrieve "test.dev1.remote001"
    Then the entity type is "remote"
    And the remote is on

  Scenario: Delete removes entity
    Given a remote entity "test.dev1.remote002" named "Remote" with power off
    When I delete "test.dev1.remote002"
    Then retrieving "test.dev1.remote002" should fail

  Scenario: remote_turn_off command is dispatched
    Given a command listener on "test.>"
    When I send "remote_turn_off" to "test.dev1.remote001"
    Then the received command action is "remote_turn_off"

  Scenario: ToHA wire state has correct fields
    Given a remote entity "test.dev1.toha" named "Remote" with power on
    When I translate "test.dev1.toha" to HA wire format
    Then the HA wire state field "is_on" is true

  Scenario: FromHA turn_off maps to RemoteTurnOff
    When HA sends "turn_off" to entity type "remote" with no params
    Then the domain command type is "RemoteTurnOff"

Feature: Media Player Entity

  Scenario: Create with default state
    Given a media_player entity "test.dev1.mp001" named "Living Room Speaker" with state "paused"
    When I retrieve "test.dev1.mp001"
    Then the entity type is "media_player"
    And the media player state is "paused"

  Scenario: Delete removes entity
    Given a media_player entity "test.dev1.mp002" named "Speaker" with state "idle"
    When I delete "test.dev1.mp002"
    Then retrieving "test.dev1.mp002" should fail

  Scenario: media_play command is dispatched
    Given a command listener on "test.>"
    When I send "media_play" to "test.dev1.mp001"
    Then the received command action is "media_play"

  Scenario: ToHA wire state has correct fields
    Given a media_player entity "test.dev1.toha" named "Speaker" with state "paused"
    When I translate "test.dev1.toha" to HA wire format
    Then the HA wire state field "state" equals "paused"

  Scenario: FromHA play maps to MediaPlay
    When HA sends "play" to entity type "media_player" with no params
    Then the domain command type is "MediaPlay"

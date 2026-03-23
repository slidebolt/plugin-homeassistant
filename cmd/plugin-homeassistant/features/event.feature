Feature: Event Entity

  Scenario: Create with default state
    Given an event entity "test.dev1.evt001" named "Doorbell" with device class "doorbell"
    When I retrieve "test.dev1.evt001"
    Then the entity type is "event"
    And the event device class is "doorbell"

  Scenario: Delete removes entity
    Given an event entity "test.dev1.evt002" named "Button" with device class "button"
    When I delete "test.dev1.evt002"
    Then retrieving "test.dev1.evt002" should fail

  Scenario: ToHA wire state has correct fields
    Given an event entity "test.dev1.toha" named "Doorbell" with device class "doorbell"
    When I translate "test.dev1.toha" to HA wire format
    Then the HA wire state field "device_class" equals "doorbell"

Feature: Lock Entity
  # Source ref: contracts/lock.md

  Scenario: Create with default state
    Given a lock entity "test.dev1.lock001" named "Front Door Lock" with locked true
    When I retrieve "test.dev1.lock001"
    Then the entity type is "lock"
    And the lock is locked

  Scenario: Update is reflected on retrieval
    Given a lock entity "test.dev1.lock002" named "Lock" with locked true
    And I update lock "test.dev1.lock002" to locked false
    When I retrieve "test.dev1.lock002"
    Then the lock is unlocked

  Scenario: Delete removes entity
    Given a lock entity "test.dev1.lock003" named "Lock" with locked true
    When I delete "test.dev1.lock003"
    Then retrieving "test.dev1.lock003" should fail

  Scenario: lock_lock command is dispatched
    Given a command listener on "test.>"
    When I send "lock_lock" to "test.dev1.lock001"
    Then the received command action is "lock_lock"

  Scenario: lock_unlock command is dispatched
    Given a command listener on "test.>"
    When I send "lock_unlock" to "test.dev1.lock001"
    Then the received command action is "lock_unlock"

  Scenario: Raw discovery data is stored internally and hidden from queries
    Given a lock entity "test.dev1.lock001" named "Front Door Lock" with locked true
    And I write internal data for "test.dev1.lock001" with payload '{"commandTopic":"zigbee2mqtt/lock/set"}'
    When I read internal data for "test.dev1.lock001"
    Then the internal data matches '{"commandTopic":"zigbee2mqtt/lock/set"}'
    And querying type "lock" returns only state entities

  Scenario: ToHA wire state has correct fields
    Given a lock entity "test.dev1.toha" named "Test Lock" with locked true
    When I translate "test.dev1.toha" to HA wire format
    Then the HA wire state field "is_locked" is true

  Scenario: FromHA lock maps to LockLock
    When HA sends "lock" to entity type "lock" with no params
    Then the domain command type is "LockLock"

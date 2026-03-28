# Home Assistant Pairing And Naming Spec

## Purpose

This spec defines the canonical names and pairing flow for the SlideBolt
Home Assistant integration.

Goals:

- Use one name for the Home Assistant long-lived access token everywhere.
- Use one name for the Home Assistant client identity everywhere.
- Remove ambiguous `token` naming from the WebSocket pairing flow.
- Replace the current shared-secret WebSocket auth model with a simple TOFU
  pairing model.

## Problem In Current Code

The current code overloads `HA_TOKEN` with two different meanings:

- In [`app/config.go`](./app/config.go), `HA_TOKEN` is used as a WebSocket
  shared secret.
- In [`cmd/plugin-homeassistant/test.integration.entities_test.go`](./cmd/plugin-homeassistant/test.integration.entities_test.go),
  `HA_TOKEN` means the Home Assistant long-lived access token used for REST API
  calls.

This is confusing and makes the architecture harder to reason about.

## Canonical Terms

### 1. Home Assistant long-lived access token

Meaning:
The token issued by Home Assistant for authenticated REST or WebSocket API
calls into Home Assistant.

Canonical name:

- Env var: `HA_LONG_LIVED_ACCESS_TOKEN`
- Code identifier: `haLongLivedAccessToken`
- Docs text: "Home Assistant long-lived access token"

Rules:

- Do not call this `HA_TOKEN`.
- Do not call this `access_token` in SlideBolt-specific pairing code unless the
  value is literally being passed to a Home Assistant API that requires that
  field name.
- This token is not part of the SlideBolt pairing handshake.

### 2. Home Assistant client identity

Meaning:
The stable UUID sent by the Home Assistant SlideBolt integration to identify
that HA instance when connecting to SlideBolt.

Canonical name:

- Wire field: `client_id`
- Code identifier: `clientID`
- Docs text: "Home Assistant client ID"

Rules:

- `client_id` is a UUID generated and persisted by the Home Assistant side.
- `client_id` is used only for SlideBolt pairing and reconnect validation.
- `client_id` is not a bearer token and must not be described as a token.

### 3. Trusted paired Home Assistant client identity

Meaning:
The `client_id` that SlideBolt has accepted and stored as the paired HA
instance.

Canonical name:

- Storage field: `trusted_client_id`
- Code identifier: `trustedClientID`
- Docs text: "trusted Home Assistant client ID"

Rules:

- This is the only client identity trusted by SlideBolt after pairing.
- If a reconnect presents a different `client_id`, access is denied.

## Names To Remove

The following names should be removed from the SlideBolt HA pairing path:

- `HA_TOKEN`
- `Token` for WebSocket auth
- `access_token` in the SlideBolt hello handshake
- generic log messages like "invalid token" for pairing failures

These names are ambiguous because they imply bearer authentication rather than
client identity pairing.

## TOFU Pairing Model

TOFU means "trust on first use".

Behavior:

1. If SlideBolt has no stored `trusted_client_id`, the first valid HA
   connection that presents a non-empty `client_id` becomes the trusted client.
2. SlideBolt stores that `client_id` durably.
3. All later WebSocket connections must present the same `client_id`.
4. If the `client_id` does not match, SlideBolt denies access and does not send
   the entity snapshot.

Security note:

- TOFU protects the open port after the first pairing.
- TOFU does not protect the first claim. The first HA client to connect wins.

## Wire Protocol

### Hello request from Home Assistant to SlideBolt

```json
{
  "type": "hello",
  "client_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

Rules:

- `type` must be `hello`
- `client_id` is required
- `client_id` must be a non-empty UUID string

### Hello response from SlideBolt to Home Assistant

```json
{
  "type": "hello",
  "auth": true,
  "server_id": "slidebolt-system-uuid"
}
```

Rules:

- `auth: true` means the presented `client_id` is trusted
- `auth: false` means the client is not trusted and no snapshot will follow
- `server_id` remains the SlideBolt instance identity already in use today

### Snapshot behavior

- Snapshot is sent only after `auth: true`
- No entity data is sent when `auth: false`

## Persistence

Pairing state must be stored in SlideBolt durable private storage, not in
runtime config.

Storage target:

- `storage.Internal`

Suggested storage key:

- `plugin-homeassistant.pairing`

Suggested stored document:

```json
{
  "trusted_client_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

Rules:

- Keep the record minimal
- Do not store the Home Assistant long-lived access token in the pairing record
- Pairing state is owned by SlideBolt, not by entity profiles

## Configuration Names

### Runtime config for plugin-homeassistant

Allowed names:

- `HA_PORT`
- `SYSTEM_UUID`
- `SYSTEM_MAC`

Remove:

- `HA_TOKEN`

### Test and tooling config for calling Home Assistant APIs

Canonical env var:

- `HA_LONG_LIVED_ACCESS_TOKEN`

Uses:

- integration tests that call HA REST APIs
- any tooling that talks to Home Assistant directly

## Logging Names

Use these phrases in logs:

- `pairing established for client_id=<uuid>`
- `pairing rejected for client_id=<uuid>`
- `client_id mismatch from <remote_addr>`
- `missing client_id from <remote_addr>`

Avoid:

- `invalid token`
- `bad token`
- `auth token`

## Migration Plan

### Phase 1: naming cleanup

- Rename integration-test env var usage from `HA_TOKEN` to
  `HA_LONG_LIVED_ACCESS_TOKEN`
- Rename WebSocket wire field from `access_token` to `client_id`
- Rename config field `Token` to nothing; remove it

### Phase 2: pairing persistence

- Add a small pairing store in `plugin-homeassistant/app`
- On first successful hello with a non-empty `client_id`, persist
  `trusted_client_id`
- On later hellos, compare against stored `trusted_client_id`

### Phase 3: test updates

Replace current token-based vulnerability tests with pairing tests:

- unpaired first client with `client_id` succeeds and stores trust
- second client with same `client_id` succeeds
- second client with different `client_id` fails
- missing `client_id` fails
- unauthorized clients never receive a snapshot

## Non-Goals

- Multi-controller trust
- Pairing approval UI
- Pairing reset flow
- Cryptographic proof of Home Assistant identity

Those can be added later if needed. This spec only standardizes naming and the
simple TOFU model.

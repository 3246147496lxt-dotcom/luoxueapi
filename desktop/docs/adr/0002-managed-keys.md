# ADR 0002: Device-Managed API Keys

- Status: Accepted
- Date: 2026-07-28
- Scope: macOS Codex MVP

## Context

An existing API key is bound to one group. The MVP requires group changes to affect new Codex sessions while existing sessions continue with their original group. A single mutable key cannot preserve that behavior. Desktop-created keys must also remain hidden from ordinary user key management and be disabled as a unit when a device is revoked.

## Decision

- Extend `api_keys` with nullable `managed_device_id` and add `purpose='desktop'`. Desktop keys require a non-null OpenAI `group_id`.
- Enforce one active key per device and group with a partial unique index on `(managed_device_id, group_id)` for non-deleted desktop keys.
- Add a desktop managed-key repository. Ordinary key CRUD continues to query only `purpose='user'`; gateway authentication explicitly accepts `purpose IN ('user', 'desktop')`; Web-chat principals remain separate.
- `PUT /api/v1/desktop/managed-keys/{group_id}` is an idempotent ensure operation. It revalidates device ownership, device status, user access to the active OpenAI group, and subscription state before returning the device's key.
- The managed key is delivered only to the authenticated device over TLS and stored in macOS Keychain. The Vue layer, Web device page, list responses, logs, analytics, and diagnostics receive only key IDs and masked metadata.
- The local gateway selects a managed key through its session-to-group binding. The cloud gateway remains responsible for billing, account scheduling, upstream credentials, and failover.
- Device usage endpoints derive key IDs from `managed_device_id`; clients cannot submit arbitrary key IDs for aggregation.

## Security invariants

- A device can ensure, rotate, or use only keys whose `managed_device_id` equals its authenticated `device_id`.
- Provisioning is unavailable for revoked devices, inactive users, inaccessible groups, non-OpenAI groups, or expired subscriptions.
- Managed keys cannot be renamed, rebound to another group, assigned a custom value, or exposed through ordinary `/keys` endpoints.
- Revocation disables every managed key in the same database transaction and publishes authentication-cache invalidations.
- Raw key values are redacted from request bodies, idempotency records, audit logs, error messages, and diagnostic bundles.

## Failure recovery

- Concurrent ensure requests converge through the partial unique index; the losing request reads the existing device/group key.
- If Keychain persistence fails after provisioning, the app does not take over Codex. It retries the same ensure operation or explicitly rotates the key after user confirmation.
- Rotation creates the replacement and switches local state before disabling the old key; failure before the switch leaves the old key usable, while failure after the switch is reconciled from the server's active-key record.
- A failed cache invalidation is retried and surfaced as an operational error; the database status remains authoritative.
- Usage aggregation continues from immutable `usage_logs` after a key is disabled or a device is revoked.

## Consequences

The design reuses the existing API-key gateway and billing path, but requires schema migrations, purpose-aware repository boundaries, managed-key cache invalidation, and desktop-specific usage queries. The MVP also inherits the current server-side raw API-key representation; migrating authentication lookup to keyed hashes remains separate security debt.

## Rejected alternatives

- Mutating one device key's group: breaks existing-session pinning and creates routing races.
- Reusing a user-created key: prevents reliable device-wide revocation and exposes unrelated user configuration.
- Pre-creating keys for every group: creates unused secrets and unnecessary database rows.
- Authenticating gateway traffic directly with the device token: requires a second billing principal and a much larger gateway authorization change.

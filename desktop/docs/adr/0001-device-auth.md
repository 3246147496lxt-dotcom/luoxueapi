# ADR 0001: Desktop Device Authentication

- Status: Accepted
- Date: 2026-07-28
- Scope: macOS Codex MVP

## Context

The desktop app needs a long-lived relationship with a LuoxueAPI account without receiving the user's password or reusing the Web console's full-privilege JWT. Web session binding is also unsuitable for a Mac that moves between networks. Pairing must support browser approval, a five-device limit, immediate remote revocation, and replay-resistant token rotation.

## Decision

- Add a dedicated desktop authentication boundary and `/api/v1/desktop` middleware. Desktop tokens are never accepted by Web user or admin routes.
- The app creates a PKCE S256 pairing with a 256-bit device code and a short human user code. Redis stores only hashes and pairing metadata under separate device-code and user-code keys for 10 minutes.
- Browser approval requires an authenticated Web session and recent step-up verification. Approval reserves a `desktop_devices` row while holding a per-user database lock; `pending` plus `active` devices may not exceed five.
- Token exchange verifies PKCE and atomically changes the Redis state from `approved` to `consumed`. A consumed, expired, denied, or mismatched pairing cannot be exchanged again.
- Access tokens are short-lived JWTs with a distinct `aud=luoxue-desktop`, `device_id`, narrow scopes, `token_version`, `jti`, and expiry. Refresh tokens are rotating opaque secrets; only their hashes and family state are stored in a durable `desktop_device_sessions` table.
- Device access tokens are stored by Rust in macOS Keychain and are not exposed to the Vue WebView.
- Revocation atomically marks the device revoked, increments `token_version`, revokes its session families, disables its managed keys, and invalidates device and API-key caches.

## Security invariants

- The app never receives a password, Web refresh token, admin permission, payment permission, or ordinary API-key management permission.
- Raw device codes and refresh tokens are not persisted server-side or written to logs, URLs, diagnostics, or audit bodies.
- Pairing and token exchange fail closed when Redis, PKCE validation, device-limit enforcement, or durable session storage is unavailable.
- Every desktop API request validates token audience, scope, device status, and `token_version`. Any cache of device status must be explicitly invalidated on revocation.
- Pairing approval displays the device name, platform, requested permissions, and spending capability before confirmation.

## Failure recovery

- Redis loss or pairing expiry requires starting a new pairing; an unexchanged pending device reservation expires and is removed by cleanup.
- If approval succeeds but exchange fails, the app may retry with the same device code until the pairing expires; the atomic consume transition ensures only one exchange succeeds.
- Refresh rotation permits only a short, server-recorded retry grace for a lost response. Reuse outside that grace revokes the whole token family.
- A Keychain write failure leaves Codex untouched. The app revokes or discards the new desktop session and asks the user to pair again.
- Remote revocation takes effect for management APIs immediately and for gateway traffic when managed-key cache invalidation reaches the cloud gateway.

## Consequences

This adds a device model, a durable session model, Redis pairing state, a separate token issuer/validator, step-up approval, and cleanup work. It avoids coupling desktop lifetime to Web login lifetime and gives remote device management a precise revocation boundary.

## Rejected alternatives

- Reusing the Web JWT: grants excessive authority and inherits network/UA session binding.
- Sending email/password to the app: expands credential exposure and complicates 2FA.
- Storing long-lived bearer tokens only in Redis: Redis loss would log out every installed app and weaken durable replay tracking.
- A non-PKCE deep link containing an API key: leaks credentials through browser, protocol-handler, and shell history.

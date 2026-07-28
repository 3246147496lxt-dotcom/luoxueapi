# ADR 0004: Localhost Gateway and Session Pinning

- Status: Accepted
- Date: 2026-07-28
- Scope: macOS Codex MVP

## Context

The desktop app needs a thin local HTTP gateway so Codex can keep a localhost endpoint while the app selects a device-managed group key. The gateway handles sensitive authorization material and streaming responses, and any local process or browser can attempt to reach loopback. Group changes must apply only to new sessions without inspecting or persisting prompt contents.

## Decision

- Run the gateway in the Tauri Rust process. Bind IPv4 `127.0.0.1` only, using the persisted preferred port when available. The listener is acquired and held before Codex configuration is changed; a conflict selects and persists a new port.
- Authenticate every request with a random 256-bit local Bearer token using constant-time comparison. Keychain is canonical, while the required copy in Codex `auth.json` is treated as a local-only capability.
- Validate `Host` against the bound loopback address and reject requests carrying `Origin`. CORS does not provide the security boundary.
- After percent-decoding and path normalization, allow only `GET /models`, `POST /responses`, and `POST /responses/compact`, plus their exact `/v1` aliases. Reject all other methods, paths, upgrades, encoded separators, and traversal forms.
- Build a new outbound request to a release-configured HTTPS LuoxueAPI origin. Disable redirects, replace authorization with the selected managed key, strip cookies and hop-by-hop/proxy headers, and forward only an explicit header allowlist.
- Stream SSE without full buffering, with cancellation propagation and bounded backpressure. Apply request-body, header, concurrency, connect, first-byte, idle, and total non-streaming limits.
- Resolve session identity from `session_id`, then `conversation_id`, then `prompt_cache_key`. Persist only `HMAC(device_secret, session_id) -> group_id, managed_key_id, expiry` in local SQLite. Do not derive identity from prompt content.
- The first request for a new session atomically binds the current group. Existing bindings retain their key across group changes and app restarts. Requests without an explicit session signal use the current group without creating a durable binding.
- Explicit request models must exist in the pinned group's real `/models` result. An unavailable pinned group or key fails with an actionable local error rather than silently changing groups.

## Security invariants

- The gateway never listens on `0.0.0.0`, a LAN address, Unix wildcard, or user-configurable upstream origin.
- Incoming authorization is never forwarded; managed keys never enter Codex files, Vue state, local request metadata, logs, notifications, or diagnostics.
- Prompt and response bodies are never persisted. Seven-day request history contains only time, model, status, token counts, duration, first-token latency, and request ID.
- Redirects cannot carry authorization to another host. Host, path, method, local token, and managed-key availability all fail closed.
- Session identifiers are stored only as keyed hashes and cannot be used to reconstruct prompt content.

## Failure recovery

- Port conflicts are resolved before configuration mutation. If no port can be bound, takeover remains disabled.
- Keychain denial, missing managed key, invalid local token, or revoked device prevents forwarding and produces a redacted diagnostic code.
- Cloud loss preserves the session binding and returns a retryable error; it does not switch groups or keys.
- Client disconnect cancels the cloud request and releases stream resources. Gateway restart reloads unexpired session bindings and reconciles their keys with Keychain.
- Repeated cloud authentication failure marks the managed key unavailable, notifies the app, and initiates safe Codex configuration restoration when revocation is confirmed.

## Consequences

The gateway stays small and reuses cloud billing, scheduling, and failover. It requires careful HTTP parsing, secret handling, SQLite lifecycle, SSE tests, and real-client contract fixtures. Persisted session pinning makes group changes predictable but retains old managed keys until their bindings expire.

## Rejected alternatives

- Direct Codex-to-cloud configuration: cannot preserve session-specific group selection or hide managed keys from Codex files.
- A separate Go sidecar: adds process supervision, packaging, and update failure modes without MVP value.
- Binding to all interfaces and relying on a token: unnecessarily exposes the service to the LAN.
- Content-derived session hashes: touches prompt data and can change across turns.
- Silent fallback from an unavailable pinned group to the current group: can break response continuity and billing expectations.

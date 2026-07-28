# ADR 0003: Codex Configuration Transaction and Recovery

- Status: Accepted
- Date: 2026-07-28
- Scope: macOS Codex MVP

## Context

Codex CLI and Codex Desktop share `~/.codex/config.toml` and `~/.codex/auth.json`. The app must modify both files, although the filesystem cannot atomically commit two independent files. Users and other tools may edit unrelated settings during takeover, and a crash can occur after either write. Restoring a complete backup would destroy legitimate user changes.

## Decision

- Use format-preserving TOML editing and structured JSON editing. The managed provider is uniquely named `luoxue_desktop`; the app does not replace an existing `OpenAI` provider table.
- The owned field set is explicit and versioned: the `luoxue_desktop` provider subtree, selected top-level provider/default model fields, and `OPENAI_API_KEY`. All other fields remain user-owned.
- Acquire an application-level configuration lock and reject unparseable files, symlinks, unsafe ownership/permissions, and detected CCSwitch takeover before writing.
- Before mutation, create encrypted backups and a durable journal under Application Support. The journal records operation ID, file identity, permissions, original hashes, original and applied values for every owned field, backup hashes, and phase.
- Bind and hold the local gateway listener before changing files. Apply in this order: local token in `auth.json`, journal `auth_written`, provider and endpoint in `config.toml`, journal `applied`, then reread and verify both files.
- Every write uses a same-directory exclusive temporary file, preserved ownership and safe modes, file `fsync`, atomic rename, and parent-directory `fsync`.
- Restore uses a three-way merge of original, applied, and current values. A field is restored only when current still equals the app-applied value. Non-owned fields are always preserved; conflicting owned fields require user resolution.
- Restore order is `config.toml` first, then `auth.json`, followed by gateway shutdown and Keychain cleanup.

## Security invariants

- The cloud managed key never enters Codex configuration; Codex receives only a random local gateway token.
- Backups containing authentication material are encrypted with an AEAD key rooted in macOS Keychain and are never included in logs or diagnostics.
- No write follows a symlink, changes unrelated fields, weakens file permissions, or proceeds without a held gateway listener.
- Parsing, backup, journal persistence, write verification, and conflict detection fail closed.
- The app never claims successful takeover until both files reread successfully and a local authenticated health check passes.

## Failure recovery

- Journal phases are `prepared`, `auth_written`, `applied`, `restoring`, and `restored`.
- A crash in `prepared` removes unused temporary files. A crash in `auth_written` restores the authentication field. A crash in `applied` restarts the gateway and verifies takeover. A crash in `restoring` continues the three-way restore.
- Failure during apply triggers reverse-order rollback using the same merge rules; the journal remains until verification succeeds.
- External changes to owned fields place takeover in `conflict` state and stop automatic rewrites. External changes to non-owned fields are retained.
- Explicit quit, logout, remote revocation, and "prepare to uninstall" restore configuration while the gateway is still running. Normal system shutdown does not attempt a last-second restore; takeover therefore requires working login auto-start.

## Consequences

Configuration work becomes a recoverable state machine rather than a pair of file writes. It adds encrypted backup management, journal migrations, fixtures, filesystem fault injection, and conflict UI, but prevents silent loss of user configuration.

## Rejected alternatives

- Replacing whole files from templates or backups: destroys concurrent and unrelated user changes.
- Using ordinary TOML serialization: may discard comments, ordering, and formatting.
- Writing `config.toml` before the gateway is bound: creates a dead endpoint window.
- Restoring only on application exit: cannot cover crashes, force quit, or power loss.
- Silently overwriting CCSwitch-managed fields: creates two competing configuration owners.

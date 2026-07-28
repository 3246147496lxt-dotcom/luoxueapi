# LuoxueAPI Desktop

`落雪API Desktop` is the macOS client for LuoxueAPI customers who use Codex.
It pairs a Mac with the web account, runs a localhost-only Responses gateway,
and manages the user's Codex configuration without exposing managed cloud keys.

## MVP boundary

- macOS 13+, Apple Silicon and Intel Universal DMG.
- Codex CLI and Codex Desktop through their shared `~/.codex` configuration.
- Tauri 2, Vue 3 and TypeScript, with the gateway in the Rust process.
- Browser PKCE pairing and device-scoped sessions stored in Keychain.
- Local request metadata only. Prompts, response bodies and sensitive headers
  are never persisted.
- The web product remains authoritative for accounts, billing, balance and
  device management. The desktop app owns local configuration and diagnostics.

Windows, Claude Code, Cursor, third-party Sub2API instances and local protocol
translation are deliberately outside this MVP.

## Development

```bash
corepack pnpm install
corepack pnpm dev
corepack pnpm tauri dev
```

The browser-only development mode uses an explicit mock adapter. Production
builds talk to the Rust commands only; Vue never receives a device refresh token
or managed API key.

## Structure

```text
desktop/
|-- docs/adr/             Architecture decisions
|-- src/                  Vue application
|-- src-tauri/src/
|   |-- cloud.rs          Fixed-origin cloud API client
|   |-- config.rs         Journaled merge and restore
|   |-- gateway.rs        Localhost Responses proxy
|   |-- storage.rs        Session pins and request metadata
|   |-- keychain.rs       Keychain access
|   |-- state.rs          Pairing and runtime lifecycle
|   `-- lib.rs            Tauri lifecycle and commands
|-- package.json
`-- vite.config.ts
```

See `docs/adr` for the security and recovery contracts.

## Verification

```bash
corepack pnpm typecheck
corepack pnpm test
corepack pnpm build
cd src-tauri
cargo check
cargo test
```

Release builds inject `LUOXUE_API_BASE_URL` and `LUOXUE_UPDATE_PUBKEY` at
compile time. The UI does not provide a custom upstream field. Never commit the
matching update private key.

## macOS package

Install both Rust macOS targets once on the build Mac, then create the Universal
App and DMG from the same source tree:

```bash
rustup target add aarch64-apple-darwin x86_64-apple-darwin
corepack pnpm build:mac
```

The normal command builds the Universal App and DMG without requiring release
signing material. A release that also emits signed Tauri updater artifacts uses:

```bash
export LUOXUE_UPDATE_PUBKEY='the committed release public key value'
export TAURI_SIGNING_PRIVATE_KEY='the private key or its secure file path'
export TAURI_SIGNING_PRIVATE_KEY_PASSWORD='the signing key password'
corepack pnpm build:mac:update
```

The current MVP is intentionally unsigned. Gatekeeper will therefore require
the documented "Open Anyway" flow even though the updater package itself will
be signed separately by Tauri when release signing is configured.

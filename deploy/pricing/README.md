# Official reference prices

`model_pricing_overrides.json` contains exact-model USD reference quotes checked on
2026-09-15. Each entry records its primary vendor source and scope. Copy this file
to the configured pricing data directory as `model_pricing_overrides.json` (the
production bind mount is `deploy/data/`). It is loaded on startup and whenever the
reference feed is refreshed, after remote/default data. The remote cache and hash
remain untouched. Force a price refresh or restart after installing a new file.

The file replaces each model's price fields as a whole; omitted non-price capabilities and context metadata are retained from the feed. Omit unknown price fields; use numeric `0` only
when the vendor explicitly publishes a free price. Invalid override files are
rejected as a whole. Keep a previous version for rollback.

## Scope

- Use international USD pay-as-you-go prices; do not convert domestic CNY prices.
- Qwen uses Singapore / International and **implicit caching**. Its separately
  priced explicit cache API is documented in source notes, not mixed into the
  implicit cache fields.
- DeepSeek reference entries contain **peak list prices**. This channel's fixed
  base prices use those peak references, before its unchanged 0.45 multiplier.
  Explicit channel prices do not change with the clock. The separate global
  billing path retains DeepSeek's peak/off-peak schedule and has updated Flash
  off-peak rates.
- MiniMax M3 uses the current effective published price and its <=512K / >512K
  tiers. The integer 524288 follows the same model's official API definition of
  512K; both cache read and input/output double above it.
- MiMo's explicitly free cache creation is a vendor promotion without a published
  end date. Review this entry when the vendor changes the offer.
- The `deepseek-flash` canonical reference is included, but is not added to this
  channel's allowlist unless an account supports that request ID.

Seven upstream request IDs remain unresolved: `deepseek-v4-flash-0731`,
`deepseek-v4-pro-0813`, `deepseek-v4.1-flash`,
`deepseek-v4.1-flash-expires-on-0910`, `deepseek-v4.1-flash/Global`, `k3`,
`hy3-preview`. Do not infer reference prices merely from prefixes or suffixes.
The last has a verified domestic Tencent retirement/redirect notice, but no
verified international alias rule for this upstream route.

No scheduled auto-research is configured. The downloaded third-party feed can
continue updating unoverridden models; these reviewed overrides remain pinned
until explicitly revised.

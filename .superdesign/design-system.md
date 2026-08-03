# Luoxue API Snow Clay Design System

## Status and Authority

Status: **canonical and active**.

This file governs Superdesign work for the entire Luoxue API product. The old public `/home` page is the visual ground truth from which the shared language is extracted; authentication and product surfaces are applications of the same system at different information densities.

There is one active visual language: **Luoxue Snow Clay**.

- Runtime token authority: `frontend/src/styles/luoxue-clay-tokens.css`.
- Visual ground truth: `frontend/src/views/HomeView.clay.css` and the `public-site-page--snow` branch in `frontend/src/components/public/PublicSiteLayout.vue`.
- Application guidance: this file and `DESIGN.admin.md`.
- `Current UI reproduction` means faithful reproduction of existing Snow Clay UI, not activation of a second theme.
- Layout variations may change composition and density, but must not introduce another palette, type system, material model, or component vocabulary.

The `Ink Snow Vermilion Mode` section retained later in this file is archival design exploration only. It is inactive, must not be selected by default, and must not be used for implementation unless the user explicitly revives it in a future request.

## Product Context

- Product: Luoxue API, a Chinese AI API gateway and relay service.
- Primary audience: Chinese AI coding beginners, individual developers, and API users.
- Primary job: create an account, create an API key, choose a valid group, connect an OpenAI-compatible client, and verify the first request.
- Secondary jobs: inspect usage, token cost, key limits, model availability, and troubleshooting guidance.
- Brand personality: friendly, clear, trustworthy, tactile, optimistic.
- Copy language: Chinese first. Keep API, GPT, OpenAI, Python, cURL, CC Switch, Codex, Token, and model names as technical terms.
- Locale behavior: translate only product-owned interface strings through the i18n catalog.
- Custom-content invariant: administrator- or user-provided copy is content, not an interface string. Render it verbatim across locales and themes; never translate, rewrite, normalize, or replace it during a visual migration.

## Existing Page Contract

Preserve these sections, anchors, and content relationships:

1. Public navigation with Luoxue API brand, product anchors, tutorial, locale, theme, and account action.
2. Hero with support status, headline, short description, primary CTA, tutorial CTA, and runnable API example.
3. Three concise trust facts.
4. Product capability section with the real dashboard screenshot and three management benefits.
5. Three-step onboarding sequence.
6. Provider status section. GPT is supported. Claude, Gemini, and Antigravity are visibly unsupported.
7. FAQ accordion.
8. Final account/tutorial CTA.
9. Public footer.

Do not add pricing tables, testimonials, customer logos, fake metrics, fake uptime, fake stock counters, or unsupported product claims.

## Whole-product Surface Architecture

The product is one brand with deliberately different information densities, not one identical card treatment copied everywhere. Every route belongs to one of these reusable page archetypes:

1. **Public discovery** — home and public catalog. Expressive Snow Clay composition, strongest brand moment, generous spacing.
2. **Focused flow / state** — setup, authentication, callbacks, payment transitions, success and failure. One task, one dominant panel, no application sidebar.
3. **Overview / analytics** — user and administrator dashboards. A decision summary followed by trends and ranked detail.
4. **Monitoring / operations** — resource health, traffic, incidents, logs, channel health and risk control. Compact local navigation, dense scan paths and semantic status before decoration.
5. **Data workspace / CRUD** — keys, users, groups, accounts, proxies, orders and other management lists. One coherent filter/table/detail workspace rather than stacked decorative cards.
6. **Form / settings / editor** — profile, system settings and documentation management. Stable section navigation, grouped fields and one explicit save state.
7. **Commerce / entitlement** — purchase, subscriptions, redeem and orders. Clear price, entitlement and next-action hierarchy without fake urgency.
8. **Reference / detail** — key usage, legal and custom content. Readable lightweight shell with metadata and task-specific tools.

The authenticated desktop shell uses `AppSidebar` as the full-height primary frame and does not render a global top header. Page context belongs to a semantic content-level `h1`; identity, wallet, subscription status, preferences and help live in the account dock at the bottom of the sidebar. Mobile retains one compact app bar only for navigation access and immediate context. Public discovery keeps `PublicSiteLayout`; focused flows use the compact auth/state shell. Shared brand tokens, controls, status semantics, typography and motion remain identical across all archetypes.

## Data and Operations Readability

Dense product surfaces follow **state → impact → next action**. The first viewport answers what changed, what is affected, and what the operator should do; it does not try to show every available chart.

### Decision hierarchy

- Use one compact page header and one task-level navigation row. Avoid separate oversized title, filter, navigation and summary cards that repeat the same context.
- A decision strip may contain three to five metrics. Each metric has one short label, one dominant tabular number, and one comparison or operational hint. Do not place paragraphs inside KPI cells.
- Group related metrics inside one shared surface with dividers. Do not render four unrelated floating cards when a single comparison band communicates the relationship better.
- Put the critical queue or primary chart directly after the decision strip. Secondary healthy detail stays collapsed or moves to a drawer.
- One row or card gets one primary action. Secondary actions belong in overflow, detail drawers or the destination management page.

### Monitoring and resource health

- The top task navigation is fixed to `账号/IP / 流量性能 / 异常告警 / 日志排障`. Resource health then uses `概览 / 账号池 / IP资源` as local navigation.
- `账号/IP` opens by default. Its first view emphasizes schedulable capacity, zero-capacity or low-redundancy groups, proxy health and unresolved severe incidents.
- Exception queues sort by operational impact and show at most five items before a focused management link. Healthy items are disclosed progressively.
- Status treatment always includes a text label and, when useful, a small icon or dot. Violet never represents health or failure.
- Drawers explain cause, last check, countdown and associated objects; monitoring may retest proxies but must not perform account mutations.

### Charts

- Use a chart only for a temporal, proportional or distribution relationship that is harder to read as text.
- Every chart begins with a plain-language takeaway and its primary value. Axes and legends are supporting evidence, not the headline.
- Use violet for the selected/primary series and gray-lavender for comparison or inactive series. Ice blue is optional brand information only and must not become a default secondary series; do not use cyan or teal on account/IP monitoring. Semantic colors remain reserved for actual status.
- A chart should normally show no more than three emphasized series. Provide direct labels or a compact legend; do not rely on color alone.
- Gridlines use hairline color, tooltips use the shared overlay surface, and numbers use tabular numerals. Never add gradient-filled chart backgrounds, neon glow or decorative 3D charts.
- Prefer a line/area chart for trend, horizontal bars for ranked comparison, a compact donut only for a small closed composition, and tables for exact operational values.
- Match the graphic to the data contract: use line/area charts only when a real historical series exists; use a 100% stacked bar for a current state composition; use labeled progress bars for capacity or redundancy; use ranked bars for group/platform comparison; keep exact identities, timestamps and actions in rows or a table.
- A KPI may include a micro-bar or sparkline only when its comparison points are returned by the backend. Never manufacture an account-health history, forecast, baseline, or recovery curve from a latest-snapshot response.
- On the first-version account/IP workspace, visualize schedulable share, account-state composition, group redundancy and proxy-health composition. Keep the exception queue textual because cause, affected object, countdown and next action are more important than shape.
- For traffic, latency, throughput and error-rate workspaces, use the reference-inspired pattern of one dominant violet line or area, a muted comparison series, direct hover values and a compact range selector. The chart must occupy one clear surface rather than sit inside nested cards.

### Data workspaces

- Desktop rows target 48–56px with sticky headers, visible selection, predictable action placement and a persistent filter summary.
- The filter bar exposes common filters and search; advanced filters disclose on demand. Applied filters remain visible and removable.
- Details open in a drawer when the operator should keep list context. Destructive actions require explicit confirmation and never execute from a URL parameter.
- Mobile replaces wide tables with one-column record cards; preserve the primary value, status and one action before secondary metadata.

### Density and shape

- Overview surfaces may use 22–26px radii and restrained Snow Clay depth.
- Table workspaces use about 14px radii and near-flat elevation.
- Forms use 14–17px radii with grouped section boundaries.
- Operations use about 12px radii, hairlines and minimal decorative shadow. Density comes from shorter spacing and fewer container layers, never unreadably small type.
- Authenticated content may reach 1600px, while the sidebar remains about 200px expanded or 60–68px collapsed. Desktop content starts at the top of the viewport with no global-header reservation. Mobile shell offsets come from the shared compact-bar token (56px plus safe-area inset), never from page-local magic numbers.

## Current UI Reproduction Tokens

The current `/home` source imports `HomeView.clay.css` and consumes the shared `public-site-page--snow` branch in `PublicSiteLayout.vue`. Those source files are the absolute ground truth for the reproduction draft; do not reproduce the older teal public-shell defaults that are overridden at runtime.

Key implemented values, provided only as a cross-check against the source:

- Canvas: `#F4F1FA`
- Solid surface: `#FFFFFF`
- Recessed surface: `#EFEBF5`
- Primary text: `#332F3A`
- Secondary text: `#635F69`
- Hairline: `rgba(91, 80, 112, 0.14)`
- Primary violet: `#7C3AED`
- Ice blue: `#0B8BED`
- Code well: `#120F18`
- Typefaces: `Nunito` for display and `DM Sans` for body/UI, with the existing Chinese fallbacks
- Content width: 1240px for home content and 1192px for the public shell
- Existing radii, shadows, spacing, typography, and section geometry must come from the source code.

## Target Design Read

High-fidelity digital clay for a technical product, interpreted as premium matte silicone and injection-molded forms. It should feel playful without becoming childish, and substantial without hiding information.

Reference object: a premium translucent ice-blue vinyl snowflake displayed among soft violet and pink silicone controls under a diffused top-left studio light.

## Target Color System

### Core

- Canvas: `#F4F1FA`
- Elevated surface: `rgba(255, 255, 255, 0.76)`
- Solid surface: `#FFFFFF`
- Recessed surface: `#EFEBF5`
- Primary text: `#332F3A`
- Secondary text: `#635F69`
- Hairline: `rgba(91, 80, 112, 0.14)`

### Candy Accents

- Primary violet: `#7C3AED`
- Violet highlight: `#A78BFA`
- Pink: `#DB2777`
- Sky: `#0EA5E9`
- Emerald: `#10B981`
- Amber: `#F59E0B`

### Luoxue Brand Bridge

- Ice blue: `#0B8BED`
- Deep ice: `#075985`
- Pale ice: `#EAF5FF`
- Cloud blue: `#8DCCFF`
- In every primary brand lockup, keep the site-name stem in the active ink color and render a real trailing `API` token in ice blue (`#0B8BED` on light surfaces, `#8DCCFF` on dark surfaces). Preserve administrator-defined names without a trailing `API` as one unmodified ink-colored string.

Use the candy accents as named roles, not random decoration. Violet owns primary conversion actions. Sky and ice blue own brand identity and information. Emerald is reserved for the real supported state. Amber is reserved for real caution. Pink may appear as a secondary material highlight but never as a fake status.

### Legacy Palette Ban

The original teal/cyan primary theme is retired.

- Do not use `#14B8A6`, `#0D9488`, or `#0F766E` for primary actions, selected navigation, focus rings, links, charts, or large decorative surfaces.
- Do not create teal-to-cyan gradients or treat blue/teal as the product's primary identity.
- Ice blue `#0B8BED` remains allowed only for the Luoxue snowflake, brand bridge, information, and selected data accents.
- Emerald remains semantic success/healthy status and must not be confused with the retired teal primary.
- When legacy Tailwind `primary-*` utilities appear in existing code, interpret them as migration debt; new work must use the Snow Clay semantic roles.

### Shared Component Contract

`frontend/src/styles/luoxue-clay-components.css` is the shared component application layer. It consumes, but never redefines, the canonical values in `luoxue-clay-tokens.css`.

- Actions: `.btn` plus `primary`, `secondary`, `ghost`, `danger`, `success`, and `warning` variants.
- Form controls: `.input`, `Input.vue`, `TextArea.vue`, `Select.vue`, `SearchInput.vue`, and `Toggle.vue` share one focus, error, disabled, and dark-theme vocabulary.
- Surfaces: `.card`, `.glass-card`, and `.card-glass` are aliases for the same semantic surface. Page density may tune radius and elevation, but not introduce another material system.
- Data: `DataTable.vue` and `Pagination.vue` share surface, border, hover, focus, selected, loading, empty, and error semantics.
- Status: `StatusBadge.vue` and `.badge-*` use violet only for product selection/primary emphasis; green, amber, red, and neutral remain semantic.
- New shared primitives must be exported from `frontend/src/components/common/index.ts` and must include default, hover, focus-visible, disabled, loading where applicable, error where applicable, and reduced-motion behavior.

### Dark Theme

- Canvas: `#17131F`
- Elevated surface: `rgba(39, 31, 50, 0.84)`
- Solid surface: `#251E2F`
- Recessed surface: `#120F18`
- Primary text: `#F8F5FC`
- Secondary text: `#C5BCCF`
- Hairline: `rgba(255, 255, 255, 0.12)`
- Primary violet: `#A78BFA`
- Sky: `#38BDF8`
- Emerald: `#34D399`

Drafts should use one page theme at a time. Do not alternate light and dark sections within one page.

## Typography

- Display and headings: Nunito, weights 700, 800, 900.
- Body and UI: DM Sans, weights 400, 500, 700.
- Code: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace.
- Chinese fallback: PingFang SC, Microsoft YaHei, sans-serif.

### Scale

- Hero: 52px mobile, 64px tablet, 72px desktop. Weight 900, line-height 1.08.
- Section title: 34px mobile, 44px desktop. Weight 800, line-height 1.15.
- Card title: 20px to 26px. Weight 800.
- Body: 16px to 18px, line-height 1.65.
- Supporting UI: 13px to 15px, weight 600 or 700.
- Code: 13px to 14px, line-height 1.7.

Hero headline must stay within two lines on desktop. Body copy should stay within 65 to 75 characters per line. Keep all letter spacing at zero except the existing code font behavior.

## Shape System

- Large hero or section material: 48px to 60px.
- Primary clay card: 32px.
- Supporting material: 24px.
- Button and recessed control: 20px.
- Small badge: 12px or full circle.
- Image nested inside a 32px frame: 24px.

Rule: outer radius minus 8px for one directly nested visual. Do not nest decorative cards inside cards. Use spacing, bands, or real content hierarchy instead.

## Clay Lighting and Shadows

Light comes from the top-left. Every major material uses coordinated outer depth, top-left highlight, inner color bounce, and inner rim light.

### Deep Surface

```css
box-shadow:
  30px 30px 60px #cdc6d9,
  -30px -30px 60px #ffffff,
  inset 10px 10px 20px rgba(139, 92, 246, 0.05),
  inset -10px -10px 20px rgba(255, 255, 255, 0.8);
```

### Floating Card

```css
box-shadow:
  16px 16px 32px rgba(160, 150, 180, 0.2),
  -10px -10px 24px rgba(255, 255, 255, 0.9),
  inset 6px 6px 12px rgba(139, 92, 246, 0.03),
  inset -6px -6px 12px rgba(255, 255, 255, 1);
```

### Convex Button

```css
box-shadow:
  12px 12px 24px rgba(139, 92, 246, 0.3),
  -8px -8px 16px rgba(255, 255, 255, 0.4),
  inset 4px 4px 8px rgba(255, 255, 255, 0.4),
  inset -4px -4px 8px rgba(0, 0, 0, 0.1);
```

### Pressed Surface

```css
box-shadow:
  inset 10px 10px 20px #d9d4e3,
  inset -10px -10px 20px #ffffff;
```

Dark-theme shadows must replace white highlights with low-opacity white and use near-black ambient occlusion. Do not copy the light shadow values directly into dark mode.

## Components

### Navigation

- One desktop line, 72px to 80px tall.
- Rounded floating clay shell with the real Luoxue snowflake logo and site name.
- Compact icon controls stay at least 44px square.
- Account action is a convex violet control.
- Mobile navigation becomes one compact clay shell and a full-width recessed menu.

### Primary Button

- 56px tall, 20px radius, one-line label.
- Violet highlight to violet base gradient is allowed on the button only.
- Hover: translateY(-4px) and strengthen the coordinated shadow.
- Active: scale(0.92) and switch to the pressed shadow.
- Focus: visible 4px violet ring with offset.

### Secondary Button

- Solid or translucent white material with dark text.
- Same dimensions and physics as the primary action.
- Never transparent without a visible material boundary.

### Code Example

- Keep cURL and Python tabs, copy action, tab semantics, and code content.
- Use a dark recessed clay well rather than a fake browser frame.
- The active tab is visibly pressed into the toolbar.
- Code remains horizontally scrollable only within its own well.

### Dashboard Visual

- Use the real dashboard screenshot in implementation.
- In Superdesign, represent it as a clear screenshot placeholder with the same 1600:757 ratio.
- Frame it as one raised clay display. Do not draw a fake dashboard from generic rectangles.

### Capability and Step Items

- Preserve the three real capabilities and three real onboarding steps.
- Avoid three identical floating cards. Use an asymmetric bento or one connected clay composition with varied cell sizes.
- Step numbers are semantic and may remain circular.

### Provider Status

- Provider icons remain recognizable.
- GPT uses emerald supported treatment.
- Unsupported providers remain readable, muted, and explicitly labeled.
- Do not hide unsupported providers through low opacity.

### FAQ

- Closed rows look lightly raised or flush.
- Open rows become recessed.
- Keep native details/summary mental model, plus/minus state, keyboard access, and comfortable touch targets.

### Final CTA

- One dominant conversion panel, not a duplicate hero.
- Preserve dynamic account intent and tutorial action.

## Layout and Composition

- Desktop maximum width: 1180px to 1240px.
- Mobile gutter: 16px. Desktop gutter: 24px to 32px.
- Hero is an asymmetric split and fits the first viewport with both actions visible.
- The product dashboard or code example is a first-viewport visual signal.
- Vary section rhythm. Use at least four layout families across the page.
- Keep the current anchor order and page information architecture.
- Do not add decorative scroll cues, numbered section eyebrows, weather, location strips, version labels, or fake metrics.
- Ambient color should come from broad integrated lighting fields and clay forms. Avoid small floating bokeh dots or unrelated decorative orbs.

## Imagery and Icons

- Brand logo: `frontend/public/logo.png`.
- Hero visual: `frontend/public/brand/home-dashboard.webp`.
- Product visual: `frontend/public/brand/home-dashboard.webp`.
- Provider icons: existing `PlatformIcon.vue` paths.
- UI icons: existing `Icon.vue` paths.
- Superdesign should use the canonical logo PNG and recognizable icon shapes. Raster images may be represented as labeled placeholders in drafts.

## Motion

- Initial hero choreography communicates hierarchy: brand visual, headline, then actions.
- Cards lift only when interactive.
- Buttons squish on active input.
- FAQ motion communicates open and closed state.
- Ambient clay fields may drift 12 to 16px over 10 to 12 seconds.
- Use exponential ease-out for transitions. No bounce or elastic easing.
- Respect `prefers-reduced-motion`; reduce all perpetual motion to static.

## Responsive Behavior

- Below 1024px, desktop nav collapses to a menu and split sections stack.
- Below 768px, all asymmetric grids become one column.
- Primary CTAs become full-width when needed, but labels never wrap.
- Hero headline reduces to 42px to 52px and stays within three lines.
- Code tabs and copy control remain usable at 320px; only the code content scrolls horizontally.
- Keep four-layer shadows at reduced spread on small screens to avoid muddy edges.
- All touch targets are at least 44px.

## Accessibility

- Body copy and placeholders meet WCAG AA contrast.
- Focus-visible states are obvious on every link, button, tab, and summary.
- Preserve heading order, tab roles, tabpanel relationships, live copy feedback, image alt text, and provider aria labels.
- Muted text may not be lighter than `#635F69` on the light canvas.
- Color is never the only carrier of provider status.

## Archived Exploration — Ink Snow Vermilion Mode（墨雪朱印）

> Inactive archive. Do not use this mode for new drafts or implementation unless the user explicitly requests that this exploration be revived.

### Activation and Precedence

Use this mode only when the prompt explicitly requests `Ink Snow Vermilion` or `墨雪朱印`. It is mutually exclusive with `Current UI reproduction` and `Clay redesign variants`.

When active, this section overrides `Target Design Read`, `Target Color System`, `Typography`, `Shape System`, `Clay Lighting and Shadows`, `Motion`, and all clay-specific component treatments. Continue to obey `Product Context`, `Existing Page Contract`, content truth, routes, anchors, responsive behavior, accessibility semantics, and real asset requirements.

### Design Read

Create a contemporary Chinese technical editorial interface: snow is expressed through calm negative space and clean paper-white surfaces; ink is expressed through decisive black typography, rules, and controls; cinnabar appears only as a rare brand signature.

The result should feel precise, quiet, trustworthy, and modern—not antique, ceremonial, decorative, or nostalgic. Do not imitate rice paper, brush calligraphy, ink-wash painting, traditional borders, fabricated seal characters, or generic “Chinese style” ornament.

The snowflake remains the primary brand association without relying on blue. Preserve the exact six-arm geometry of the real Luoxue snowflake asset and render it in ink, paper white, or cinnabar. Do not invent a generic snow icon or a replacement logo.

### Color System

#### Core

- Snow canvas: `#F7F6F1`
- Clean paper surface: `#FFFEFB`
- Ink: `#171717`
- Graphite copy: `#5C5A56`
- Snow wash: `#ECEAE3`
- Frost rule: `#D8D5CC`
- Code well: `#111212`
- Code copy: `#F3F0E8`
- Cinnabar brand seal: `#C4422E`

#### Dark Mode

- Night ink canvas: `#111212`
- Dark paper surface: `#1A1A19`
- Dark primary text: `#F3F0E8`
- Dark graphite copy: `#B8B3A8`
- Dark frost rule: `#34322D`
- Cinnabar brand seal remains `#C4422E`

Dark mode stays achromatic and editorial. It must not introduce blue, teal, violet, colored fog, or glow.

#### Semantic Status

- Supported / success: `#217A4B`
- Warning: `#9A6700`
- Destructive / error: `#7A1F1F`

Cinnabar is a brand color, never a status color. Keep it to approximately 3–5% of the visible page: the seal signature, one selected-state detail, or a short editorial accent. Never fill every CTA, heading, icon, and border with cinnabar.

Destructive states must use the deeper error color together with an explicit destructive icon and label such as “错误”, “失败”, or “删除”. Never place destructive states inside the square seal geometry. Unsupported providers use graphite styling and the explicit label “暂不支持”; they do not use cinnabar or danger red.

Do not introduce ice blue, cyan, teal, electric violet, pink, or other candy accents. Do not use gradients, neon, colored glow, translucent color fog, or alternating light and dark page sections. Truthful colors already present inside a real product screenshot are not reusable UI palette colors.

### Typography

- Display and major Chinese headings: `"Noto Serif SC", "Source Han Serif SC", "Songti SC", serif`, weights 700 and 900.
- Body and UI: `Inter, "PingFang SC", "Microsoft YaHei", sans-serif`, weights 400, 500, 600, and 700.
- Code: `ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace`.
- Hero: 44px mobile, 60px desktop; weight 900; line-height 1.08.
- Section title: 30px mobile, 40px desktop; weight 700; line-height 1.2.
- Body: 16px to 18px; line-height 1.7.
- Supporting UI: 13px to 15px; weight 600.
- Code: 13px to 14px; line-height 1.7.

Use the serif face selectively for hierarchy, not for every label or paragraph. Keep Chinese letter spacing natural. Do not use brush-script, handwriting, faux calligraphy, all-caps decoration, or gradient text.

### Shape, Depth, and Rules

- Editorial panels: 0px to 8px radius.
- Product frames and major interactive surfaces: 12px to 16px radius.
- Buttons and compact controls: 10px to 12px radius.
- Status pills may use a full radius.
- Cinnabar seal: compact square, 0px to 3px radius.
- Dividers and frames: 1px frost or ink rules.

The interface is predominantly flat. Use spacing, rules, contrast, and alignment for hierarchy. Shadows are allowed only for genuinely floating navigation, menus, dialogs, and hover states, using a restrained neutral shadow such as `0 8px 24px rgba(23, 23, 23, 0.06)`. Do not use clay extrusion, neumorphic highlights, large blur fields, glossy depth, or decorative card stacking.

### Brand Signature

- Use the real Luoxue snowflake geometry as the principal symbol.
- Render the main snowflake in ink or paper white.
- A single compact cinnabar square may sit behind one corner of the snowflake or contain the exact snowflake silhouette as a seal signature.
- Do not fabricate Chinese seal characters or redesign the logo.
- A low-opacity ink snowflake may appear once as a large cropped hero watermark.
- Do not scatter snowflake particles, repeat the logo as a pattern, or turn the page into a winter illustration.

### Component Treatments

#### Navigation

Use a clean paper or snow surface with one crisp bottom rule. Keep the real Luoxue snowflake and name prominent. Avoid a floating pill-shaped clay shell. The account action is an ink-black control with paper-white text; locale and theme controls remain compact, legible, and at least 44px square.

#### Primary and Secondary Actions

Primary actions use ink black with paper-white text. Secondary actions use paper white with ink text and a visible frost or ink border. Hover states may lift by 1px to 2px or invert contrast. Active states compress subtly. Focus-visible uses a clear 2px ink outline with offset. Button labels never wrap.

Cinnabar is not the default CTA fill. It may appear as a small directional mark, selected underline, or seal beside the action.

#### Hero

Preserve the existing headline, support statement, description, actions, and runnable API example. Use an asymmetric editorial split with a strong typographic block and a dark code well, separated by negative space or a single rule. The snowflake may anchor the composition as an exact monochrome mark or cropped watermark. Do not use the ice-blue 3D snowflake image as the dominant visual in this mode.

#### Code Example

Keep the real cURL and Python tabs, copy behavior, semantics, and code. Use the deep ink code well with paper-white code. The active tab may use a short cinnabar underline or seal-sized marker. Do not use fake browser chrome, colored glow, or decorative terminal dots.

#### Product Visual

Keep the real dashboard screenshot in implementation. In Superdesign, use a clearly labeled placeholder with the same `1600:757` ratio. Frame it once with an ink rule and restrained radius; do not redraw a fake dashboard or place it inside multiple decorative cards.

#### Capabilities and Steps

Preserve the three real capabilities and three real onboarding steps. Avoid three identical cards. Use editorial columns, an asymmetric grid, or one connected sequence with rules and deliberate variation. Step numbers may use ink circles; reserve the cinnabar mark for one active or concluding moment.

#### Provider Status

Keep provider icons recognizable. GPT uses the real supported green state. Claude, Gemini, and Antigravity remain readable in graphite and explicitly say “暂不支持”. Do not use cinnabar or danger red for unsupported providers, and do not hide them with excessive opacity.

#### FAQ and Final CTA

FAQ rows are primarily separated by rules. The open state may use a snow-wash background and an ink indicator. Preserve keyboard and summary behavior.

The final CTA should feel like the editorial conclusion rather than a duplicated hero: generous snow-white space, decisive ink typography, one primary ink action, one secondary outlined action, and at most one cinnabar seal signature.

### Motion and Responsive Behavior

Use restrained 160ms to 240ms transitions for hierarchy and feedback. No perpetual ambient drift, bounce, elastic easing, parallax snow, or decorative particle motion. Respect `prefers-reduced-motion`.

Keep the existing responsive section order. Below 1024px, collapse navigation and stack split sections. Below 768px, editorial columns become one column. Preserve 16px mobile gutters, 44px touch targets, usable code controls at 320px, and zero horizontal page overflow.

### Ink Snow Vermilion Quality Gate

- No ice blue, cyan, teal, electric violet, candy pink, or clay material.
- No gradient, neon, glow, neumorphism, glossy depth, or colored ambient fog.
- No fabricated seal characters, faux calligraphy, rice-paper texture, or antique Chinese ornament.
- No generic snowflake replacing the real Luoxue geometry.
- No cinnabar used for errors, unsupported providers, or every CTA.
- No generic three-equal-card feature row.
- No fake product screenshot, fake metrics, or unsupported claims.
- No decorative cards nested inside other decorative cards.
- No unreadable graphite copy or ambiguous provider status.
- No wrapped desktop CTA, mobile overflow, or reduced-motion violation.

## Clay Variant Boundaries

Allowed variation topics:

- Hero composition and which real visual dominates.
- Section rhythm and bento geometry.
- Relative emphasis of violet candy material versus ice-blue Luoxue material.
- Amount of translucency versus solid matte clay.

Locked across all Clay variants:

- Fonts, color roles, radii scale, clay physics, and shadow architecture.
- Chinese copy and section order.
- Dynamic CTA meanings.
- GPT-only support truth.
- Existing routes, anchor IDs, accessibility semantics, and real brand assets.

## Quality Gate

- No generic three-equal-card feature row.
- No fake product screenshot.
- No card nested inside another decorative card.
- No gradient body text.
- No uncontrolled purple glow.
- No unreadable muted text.
- No desktop CTA wrapping.
- No hero taller than the initial viewport.
- No horizontal page overflow at 320px.
- No motion without reduced-motion behavior.
- No visible claim unsupported by current product content.

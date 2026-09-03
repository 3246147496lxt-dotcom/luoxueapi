# Luoxue API Snow Clay Design System

## Dashboard GPT Usage Lower Region — Current Design Override

This section is the active authority for the current Superdesign Dashboard iteration. Preserve the existing Work Sidebar, mobile header, account-balance card, Ultra quota card, quota percentage/progress geometry, and all geometry above them exactly. The usage region below those two top cards keeps the measured DeepSeek layout but is customized for the `gpt-5.6-sol` model.

- The entire Dashboard uses one active theme at a time; never alternate a light top region with a dark usage region. Every page, Sidebar, top card, usage control, usage card, chart card, heading, label, axis, grid, and border must consume the existing semantic `--workspace-*` theme aliases.
- Purple is not part of the authenticated Workspace or Dashboard in either theme. Neutral surfaces stay dominant; one semantic blue ramp owns actions, focus, selection and detail data, orange owns consumption data, and green/amber/red remain truthful status colors.
- Light mode: page and usage canvas `var(--workspace-canvas)` = `#FCFCFC`; every summary/chart/top card uses `var(--workspace-card-surface)` = `#FFFFFF`; controls use `var(--workspace-surface-subtle)` = `#F7F7F8`; text uses `--workspace-text`, `--workspace-text-secondary`, and `--workspace-text-muted`; borders use `--workspace-border`. The lower region must look like a natural continuation of the original light Dashboard background, not a dark insert.
- Dark mode under `html.dark`: the same elements automatically resolve to the original Workspace dark tokens: canvas `#000000`, Sidebar canvas `#000000`, cards/surfaces `#171717`, primary text `#ECECEC`, secondary text `#B4B4B4`, muted text `#8A8A8A`, and border/divider `rgb(255 255 255 / 0.1)`. Do not introduce a second custom dark palette.
- Keep `Inter, system-ui, -apple-system, "Segoe UI", Roboto, sans-serif` for the usage typography while consuming the same active theme colors as the surrounding Workspace.
- The reproduced inner content width is exactly 920px at desktop. Use 32px vertical rhythm, 12px gaps between sibling cards, no borders, no shadows, and no gradients or glow outside the chart data fills.
- The usage region starts directly with the exact 36px filter toolbar; do not render a timezone notice or leave reserved space for one. Keep only the time-dimension pill, 18px radius, 14px type, and `0 14px` horizontal padding, plus `清除筛选条件`, the 36px `导出` pill, and the vertical-more action. Remove the API Key filter completely. All control surfaces and text follow the active Workspace theme.
- Next row contains exactly three equal 102px summary cards with 12px gaps. Card surface is `var(--workspace-card-surface)`, with a 1px `var(--workspace-border)` border in light and dark modes, radius 16px, padding `16px 20px`; labels 14/400/22px, values 29/500/36px. Content: `消费金额 ¥0.45 CNY`, `API 请求次数 79`, and `Tokens 8,445,868`.
- The main chart is exactly 920px × 341px, surface `var(--workspace-card-surface)`, 1px theme border, radius 16px, padding `20px 20px 12px`. Header text `消费金额（CNY）` is 14/500/22px with adjacent muted `¥0.45`. Do not render the `模型 / API Key` segmented control or any API Key option. Chart axes consume `--workspace-text-muted`; grid lines use the active theme divider; Y labels are 0, 0.3, 0.6, and X labels are 00:00, 08:00, 15:00, 23:00. Use the exact orange series colors `#FF810C`, `#FFA10A`, and `#FFC104`; the visible sample peaks at 22:00–23:00. Its compact inverse tooltip names the model `gpt-5.6-sol`.
- After a 24px gap, render the exact 16/500/24px model heading `gpt-5.6-sol`. After 16px, render two 454px × 340px chart cards with a 12px gap; each uses `var(--workspace-card-surface)`, a 1px theme border, radius 16px, and padding `23px 20px 12px`.
- Left detail chart header is `API 请求次数 79`, axes 0/50/100 and 00:00/08:00/15:00/23:00, with the reference blue line/area using `#0C70F3` and `#70B2FE` and a narrow late-evening peak.
- Right detail chart header is `Tokens 8,445,868`, axes 0/5M/10M and 00:00/08:00/15:00/23:00, with the reference stacked blue bar using `#0C70F3`, `#60B3FE`, and `#A0DCFD` near 23:00.
- Desktop and mobile may stack responsively, but desktop dimensions, colors, radii, padding, typography, ordering, and chart proportions must match the measured reference exactly. Do not retain the previous light chart component, orange range buttons, model select, empty-state card, legend, or any additional content below the preserved top cards.

## Authenticated Workspace Runtime Token Contract — Implemented

- Chat, Work and Account share one runtime authority: `frontend/src/styles/luoxue-clay-tokens.css`.
- The only authenticated-workspace namespace is `--workspace-*`. The former `--shell-*`, `--work-*` and `--account-*` namespaces are retired.
- Runtime consumers reference semantic roles without literal fallbacks or local palette redeclarations. This document describes visual intent; it is not a second source of executable Token values.
- Shared foundations cover color primitives, one UI typography stack, spacing, radii and Sidebar dimensions. Chat, Work and the teleported Account popover may keep separate semantic color roles where their approved values genuinely differ, but typography does not fork by surface.
- Token consolidation preserves approved theme values. The Workspace Shell is the explicit exception for geometry: Chat and Work now share one 260px expanded width and one 68px collapsed width.

## Workspace Typography Contract — Active

- Chat, Work and the personal Account surface inherit one runtime entry: `--workspace-font-ui: Inter, "PingFang SC", "Microsoft YaHei", sans-serif`.
- The role scale is fixed: Brand 18/700, Page Title 28/600, Navigation 14/500, Body 14/400, Secondary 12/400, and Numeric 32/600.
- Components must inherit the Workspace UI family instead of declaring local UI, navigation, dashboard, account, or composer font stacks. `--workspace-font-mono` remains reserved for API keys, endpoints, identifiers, and code.
- The personal Workspace body class owns the inherited entry so teleported menus and dialogs remain typographically consistent. Administrator, public, and authentication surfaces keep their existing typography contracts.

## Workspace Shell Component Contract — Implemented

- `WorkspaceSidebarFrame` is the shared structural base for the Work `AppSidebar` and the Chat history rail. It owns expanded/collapsed width, column layout, overflow, border, surface, content padding, and the `header → mode-switch → content → footer` slot order.
- `WorkspaceSidebarHeader` is the single header structure. Expanded mode keeps the brand left and search/collapse actions right on one 52px row; collapsed mode exposes the same 44px expansion target.
- `AppModeSwitch` is controlled only by `activeMode` and its `change` event. It owns height, radius, font, active/hover/focus states, and light/dark Token consumption; shell parents must not style it with `:deep()` or local overrides.
- Desktop collapsed state is controlled by the owning shell adapter, while the shared Frame/Header define identical visual and interaction semantics. Existing Work and Chat mobile drawer breakpoints, modal semantics, and focus traps remain mode-owned.

## Sidebar User Account Menu — Active Isolated Design Override

This is the active authority for the current Superdesign exploration. It overrides conflicting shell/account-dock guidance below **only for the fixed user account area at the bottom of the left Sidebar and its opened account menu**. This is a design-only task; it does not authorize Vue, TypeScript, Tailwind, routing, store, or business-logic changes.

### Frozen scope

- Do not redesign or move the Chat / Work switch, Sidebar brand, main navigation, Dashboard, Chat page, composer, or any other page content.
- Preserve the real 260px desktop Sidebar, existing route/data contracts, account bindings, responsive behavior, keyboard interaction, focus return, outside-click/Escape close, mobile sheet behavior, and light/dark theme support.
- Show the target as a standalone User Menu component in its real Sidebar-bottom context. Do not invent a new application shell or marketing composition.

### Closed account entry

- Fixed to the Sidebar bottom; use a compact 52–56px neutral row, not a card.
- First line: the existing bound circular 32px avatar plus the bound display name. Keep the name concise, 14px, medium/semibold, and ellipsized when needed.
- Second line: only the bound primary plan label such as `Pro` in 12px muted text.
- Keep the existing small neutral chevrons-up-down arrow affordance at the far edge. Hover/open state may use only a quiet neutral fill.
- Do not expose email, numeric balance, frozen balance, quota charts, subscription counts, upgrade text, or any upgrade button in the closed entry.

### Open account menu

- Desktop: one 248px product Account Popover anchored 8px above the entry. Use a 1px neutral hairline, 14px radius, clipped integrated sections, and a restrained floating shadow. This is a popover, not a settings panel and not a stack of cards.
- First section — identity header: one integrated full-width top region that inherits the popover's top corners; it is not an inset card and, in dark mode, it uses the exact same background as the rest of the popover. Show the bound avatar, bound display name, and bound plan label only. Use a 36px avatar here for a clear but compact identity hierarchy. Do not show balance, email, upgrade, badges, billing copy, a trailing action, or a contrasting header band.
- Place a full-width hairline divider directly after the identity header.
- Second section — `常用设置`: show a quiet 11px group label, then exactly these 40px monochrome icon-and-label rows in order: `个性化` with `slidersHorizontal`, `个人资料` with `user`, and `设置` with `cog`.
- Place a full-width hairline divider between the two action groups.
- Third section — `辅助功能`: show a quiet 11px group label, then exactly these 40px monochrome icon-and-label rows in order: `帮助` with `questionCircle`, and `退出登录` with `logout`.
- Rows use an 18px existing line icon, a 14px label, a quiet neutral hover/focus fill, and no trailing chevrons. Keep logout monochrome by default; do not add a red block or promotional treatment.
- Do not include `我的账户`, `套餐管理`, `余额`, any upgrade action, a plan-management row, numeric balance, or other actions in this design branch.

### Visual tokens and tone

- Product reference: ChatGPT account entry, Claude user menu, and Linear account menu. The tone is professional, quiet, compact, and unmistakably application UI rather than an admin console or store.
- Light theme roles: review canvas `#F5F5F5`; Sidebar `#FCFCFC`; popover `#FFFFFF`; integrated identity header `#F7F7F8`; row hover/focus `#F3F4F6`; border and dividers `#E5E7EB`; primary text `#111827`; secondary text and icons `#6B7280`; inverse/avatar text `#FFFFFF`; neutral fallback avatar `#111827`.
- Dark popup roles: page/Sidebar `#0F0F10`; the complete popover container, identity region, both labeled action groups, and every non-hover interior region use one continuous `#353535` surface; popup row hover/focus uses `rgba(255, 255, 255, 0.08)`; popup border uses `rgba(255, 255, 255, 0.10)`; dividers use `rgba(255, 255, 255, 0.12)`; primary text uses `#F5F5F5`; secondary text, plan text, icons, and group labels use `#C7C7CC`; neutral fallback avatar may remain `#262626` with `#F5F5F5` initials because it is identity content rather than a section surface.
- Dark dock roles remain unchanged and separate from the popup correction: Sidebar `#0F0F10`; dock hover/open surface `#262626`; dock primary text `#F5F5F5`; dock plan text and chevrons `#A1A1AA`.
- Light popover shadow: `0 12px 30px rgba(0, 0, 0, 0.12), 0 2px 8px rgba(0, 0, 0, 0.06)`.
- Dark popover shadow: `0 14px 32px rgba(0, 0, 0, 0.38), 0 2px 8px rgba(0, 0, 0, 0.24)`.
- The popup must read above the Sidebar as one uniform gray card through its single surface, hairline dividers, spacing, typography weight, and shadow—not alternating section fills or a bright edge. Never use `#262626`, `#0F0F10`, black, or another darker surface for any header, body, group, or footer region inside the dark popup. Never add a white rim, glow, pure-black surface, high-contrast outline, or inset highlight.
- UI font inherits the shared `var(--workspace-font-ui)` stack. Use compact 4px/8px spacing logic.
- No gradients, broad color fills, purple buttons, purple upgrade treatment, glow, glass, clay/neumorphism, large cards, wallet/store styling, promotional copy, or invented user data contracts.
- A component-spec presentation may show equivalent light and dark examples side by side for review; these are two theme states of the same component, not alternating sections of a product page.

### Approval gate

- Treat the current Account Popover draft as ground truth and apply this dark-surface correction as a single replace iteration on that same draft; do not open a new design direction.
- Do not implement the approved result in production code until the user explicitly asks to proceed.

## End-User Work Accent Refinement V4 — Approval Draft

This is the latest authority for the current SuperDesign exploration. It is a **Work-mode-only visual refinement** of the already implemented end-user workspace. It overrides the V3 monochrome rules below only where they conflict with this section. It does not authorize implementation until the user confirms the generated design.

### Frozen scope

- Chat is completely frozen. Do not change the ChatGPT-like layout, 260px history rail, conversation grouping, message presentation, composer, model selector, attachment/voice controls, input styling, spacing, or interaction behavior.
- Do not pass Chat page/components into this design exploration and do not infer a new Chat palette from the Work proposal.
- Preserve the current Work information architecture, routes, business data, loading/error/empty states, API contracts, responsive breakpoints, and component hierarchy. This is a local visual refinement, not a product redesign.
- The Dashboard remains an end-user surface. Never introduce administrator KPIs, gateway health, system monitoring, operations data, server capacity, global incidents, or infrastructure status.

### Work visual ratio and tokens

The Work workspace uses approximately 80% neutral structure and 20% restrained brand accents. White cards, neutral typography, gray borders, and generous whitespace remain dominant. Purple identifies selection, focus, primary action, and important user-owned data; it is not decorative atmosphere.

Work consumes the canonical `--workspace-work-*` roles plus shared `--workspace-radius-*`, `--workspace-space-*`, `--workspace-font-*` and `--workspace-sidebar-*` foundations. Exact runtime values live only in `frontend/src/styles/luoxue-clay-tokens.css`.

- Never remap the shared `--workspace-action` roles to the Work violet roles, because Chat consumes the neutral action roles and must remain visually untouched.
- No purple gradients, purple page fields, large violet cards, rainbow provider palettes, glow, glass, clay/neumorphism, ornamental AI shapes, or colored marketing surfaces.
- `#7C3AED` and `#6D28D9` may be used for normal text/icons and focus states. `#8B5CF6` and `#A78BFA` are graph/fill colors only, not small text on white.
- Focus-visible uses a 2px `#7C3AED` outline with a 2px offset. Selected states also use background, weight, marker, or `aria-current`; never rely on hue alone.

### Work Application Shell

- Keep the desktop sidebar exactly 260px and preserve the real logo, `[ Chat ][ Work ]` switch, navigation order, resources, account dock, mobile drawer, and current density.
- Navigation remains exactly: 工作台 — 仪表盘, 模型中心, API 密钥, 使用记录; 账户 — 余额, 套餐, 订单; 资源 — 文档, 公告, 设置.
- Work active navigation uses `#F5F3FF` background, `#6D28D9` text/icon, 600 weight, and an optional 2px `#7C3AED` marker. Hover remains neutral `#F3F4F6` with dark text; brand color is reserved for active/focus.
- On Work pages, the Work segment may use deep-purple text and a subtle `#DDD6FE` boundary while the segment surface remains white. The Chat-mode appearance is unchanged.
- The compact account avatar may use deep purple. Upgrade remains a small outlined action with violet text/border and soft hover; do not turn the footer into a purple panel.

### Dashboard visual refinement

- Preserve the current header, four KPI cards, dominant model-usage panel, recent conversations, 2×2 quick actions, API information, and account information in the same order and geometry.
- Cards remain white with a 1px `#E5E7EB` border, 14px radius, and only the very light card shadow above. Do not add colored card bodies, top stripes, or hover lift.
- KPI cards use 32–36px soft icon wells: balance violet, today usage soft violet, Token indigo, and current plan neutral with the existing truthful green status dot. Balance may be the only purple primary value; other values remain `#111827`.
- Remove grayscale treatment from the real balance mark. Use tabular numerals and keep units visually subordinate.
- Model usage keeps exact labels, values, ranking rows, and bar lengths. Tracks are `#F3F4F6`; the primary bar is `#7C3AED`, with later rows using the same hue at restrained lighter strengths. Data is still conveyed through names, numbers, and length—not color alone.
- If the existing trend data is visualized, keep it inside the existing model panel: violet primary line, indigo secondary line, neutral grid, no gradient, glow, invented data, or extra card.
- Recent conversations keep the current structure. Only use a very soft violet hover and violet chevron/icon hover.
- Quick actions keep the current 2×2 layout and destinations. Use small violet/indigo icon wells; New Chat may receive the only subtly prioritized boundary. Do not create four colored tiles.
- API endpoint cards stay neutral. The primary endpoint badge may use violet soft/border/text; code areas remain `#F9FAFB`; copy/open buttons become violet only on hover/focus.
- Account identity may use a deep-purple avatar and a violet API-key icon. Semantic active status remains green.
- Increase low-contrast 10–11px `#9CA3AF` labels to at least 12px `#6B7280` (or `#4B5563` when compact). Maintain 44px touch targets, 16px mobile gutters, stacked model metrics, and no horizontal overflow at 320px.

### Approval gate

- Generate one current-UI reproduction first, then exactly one branch implementing this restrained Work accent direction.
- Present the SuperDesign canvas and preview for confirmation. Do not migrate this V4 proposal into production Vue/CSS before explicit user approval.

## End-User AI Workspace V3 — Approved and Implemented

This section is the active authority for the user-confirmed authenticated **end-user** Application Shell, Work Dashboard, and Chat Shell direction, now migrated into the project's Vue/CSS implementation. It overrides every conflicting authenticated-shell, Snow Clay, administrator, operations, monitoring, and violet-primary rule below.

### Product boundary

- This is the paid end-user product workspace, comparable in interaction density to ChatGPT Plus, Claude Pro, OpenAI Console, Linear, and Vercel Dashboard.
- It is not an administrator dashboard, operations console, global monitoring surface, API gateway health center, or system data center.
- Preserve the existing user-facing business contracts: AI chat, model usage, API calls, credit balance, subscriptions, orders, account identity, announcements, settings, loading/error/empty states, and current feature visibility.
- Do not expose global service health, gateway status, system capacity, operational incidents, or administrator KPIs on the user Dashboard.

### Monochrome visual language

- Canvas: `#FFFFFF` or `#FAFAFA`; sidebar may use `#FAFAFA` while primary work surfaces remain white.
- Card/surface: `#FFFFFF`.
- Border: `#E5E7EB`; strong neutral border: `#D1D5DB`.
- Primary text: `#111111`; secondary text: `#374151`; muted text: `#6B7280`.
- Hover/selected neutral fill: `#F5F5F5` or `#F3F4F6`; primary action: near-black `#111111` with white text.
- Brand color is permitted only in the real logo, a tiny truthful status dot, or a subtle hover detail. It must not fill page backgrounds, large buttons, active navigation blocks, KPI cards, charts, or decorative regions.
- No purple gradients, broad colored panels, neon/glow, glass, clay/neumorphism, ornamental AI effects, oversized radii, or decorative floating shapes.
- Surfaces rely on hairline borders. Optional elevation is limited to `0 1px 2px rgba(0, 0, 0, 0.04)`; overlays alone may use a slightly deeper neutral shadow.

### Geometry and typography

The geometry contract is implemented through the canonical `--workspace-font-*`, `--workspace-space-*`, `--workspace-radius-*`, `--workspace-sidebar-*` and semantic color roles. Exact values live only in `frontend/src/styles/luoxue-clay-tokens.css`.

- Typography: `Inter`, `PingFang SC`, `Microsoft YaHei`, sans-serif; monospace only for API keys, endpoints, identifiers, and code.
- Use an 8px spacing system. Desktop content gutters are 24–32px; mobile gutters are 16px.
- Cards use 12–16px radii, buttons 8–10px, inputs 12px. Do not use 20–28px application-card radii.
- Workspace roles are fixed at Brand 18/700, Page Title 28/600, Navigation 14/500, Body 14/400, Secondary 12/400, and Numeric 32/600. Numeric metrics use tabular numerals.
- Touch targets are at least 44px on touch layouts; focus-visible uses a 2px neutral/ink outline with sufficient contrast.

### Application Shell

- Desktop sidebar is fixed at `260px`, full height, white or `#FAFAFA`, with one right hairline and no decorative shadow.
- Order: real brand logo, neutral `[ Chat ][ Work ]` segmented switch, mode-owned navigation, flexible spacer, compact account dock.
- The mode switch uses a white active segment with black text and a subtle border/shadow; it never uses a purple active fill.
- Work navigation contains exactly three primary groups in this approval concept:
  1. 工作台: 仪表盘, 模型中心, API 密钥, 使用记录.
  2. 账户: 余额, 套餐, 订单.
  3. 资源: 文档, 公告, 设置.
- Feature-gated secondary capabilities remain business contracts but are not promoted into the requested primary Work navigation. They may later live behind contextual quick links or a subordinate overflow without changing routes or data.
- Chat navigation contains only New chat and conversation history. It must not show API, balance, plans, orders, Dashboard, global status, or administrator destinations.
- Below 1024px each mode has exactly one header/menu trigger and one focus-managed drawer. Never stack a Work sidebar and Chat history rail.

### Work Dashboard hierarchy

- Header: concise personalized greeting, optional supporting line, and low-emphasis date-range/refresh controls.
- First region: four equal user-owned summary cards — current balance, today's usage, token consumption, current plan. Each card has one dominant value and one concise supporting fact; no admin/system KPI.
- Second region: model usage (dominant evidence surface), recent conversations, and compact quick entries for Chat, API key, usage, and balance/plan tasks.
- Third region: API information and account information. Endpoints and IDs use monospace, with copy/open actions. Account information summarizes identity, plan, billing/renewal, or membership state from existing contracts.
- Announcements may appear as a user resource/compact feed. Global service availability, channel monitor results, gateway uptime, and operations status do not appear on the Dashboard.
- Charts are monochrome: near-black primary series and neutral grays, with semantic colors only when the data truly represents success/warning/error. Never use a rainbow model palette by default.

### Chat Shell hierarchy

- One 260px history sidebar: logo, neutral mode switch, black/white New chat control, optional search, grouped history, account dock.
- Main area is edge-to-edge white with a minimal header, centered message stream, generous whitespace, and a fixed/anchored bottom composer.
- Assistant messages are flat; user messages may use a soft neutral bubble. Composer uses one neutral border, 12–16px radius, model label, attachment, voice, and an ink send control.
- Preserve existing streaming, model selection, file upload, voice input, retry/copy, history management, drawer focus, Escape close, focus return, and `inert` behavior.

### Implementation status

- The Application Shell, end-user Dashboard, and Chat visual direction are implemented in Vue/CSS.
- `ChatComposer` and the chat input form remain unchanged, as explicitly requested by the user.

## Authenticated Application Shell V2 — Active Scoped Override

This section is the active authority for the authenticated personal product shell introduced for the Chat / Work redesign. It overrides conflicting Snow Clay guidance below **only inside the personal Application Shell**. Public discovery, focused purchase/auth flows, and administrator surfaces retain their existing systems unless they are explicitly migrated later.

### Product modes

- The personal product has two explicit modes: `Chat` and `Work`. Mode is semantic application state, not a content-density variant.
- Both modes share brand identity, account access, keyboard/focus behavior, and one compact mode switch. Their navigation and content frames remain distinct.
- `Chat` is an AI-assistant environment. Its left rail contains only shell identity/mode controls, New chat, conversation search/history, and the account dock. It must never expose API keys, balance, subscriptions, orders, admin, or other Work navigation.
- `Work` is the developer workspace. Its sidebar groups are exactly Workbench, Account, and Resources, with feature-gated secondary tools preserved in a compact More section when available.
- Never infer product mode from a layout variant such as `variant="chat"`; mode must be declared explicitly as `shellMode="chat" | "work"` or equivalent route metadata.

### Visual direction

The shell is a quiet, professional enterprise SaaS workspace influenced by ChatGPT, OpenAI Console, Linear, and the restraint of Snow Clay. It is not a marketing page, an AI-tech showcase, or a neumorphic/clay composition. Long-session comfort, scan speed, and focus take priority.

- No large violet fields, purple fog, neon, decorative gradients, glassmorphism, or floating ornamental shapes.
- Use flat surfaces, hairline boundaries, strong typography, and restrained neutral elevation.
- Violet is reserved for selected mode/navigation, primary actions, focus, and rare emphasis. Neutral or ink controls are preferred for routine actions.

### Workspace shell tokens

The V2 `--shell-*` namespace is retired. Chat and shared shell chrome consume the canonical `--workspace-*` roles from `frontend/src/styles/luoxue-clay-tokens.css`; Work-specific accents consume `--workspace-work-*` from the same file.

- Use the shared `var(--workspace-font-ui)` stack. Do not use display typography inside the shell.
- Use the fixed Workspace role scale: Brand 18/700, Page Title 28/600, Navigation 14/500, Body 14/400, Secondary 12/400, and Numeric 32/600. Never introduce intermediate weights such as 450, 550, 650, or 680.
- Use 4px/8px spacing logic. Desktop shell gutters are 24px–32px. Touch targets are at least 44px on touch layouts.

### Shared application frame

- Desktop viewport reference: 1440×900. Mobile references: 390×844 and 320px minimum width.
- The shell fills `100dvh`, owns overflow, and has no marketing header or footer.
- Desktop Sidebar is 260px expanded and 68px collapsed in both Chat and Work. Mobile drawers retain their existing host breakpoints and width formulas through canonical Workspace tokens.
- One compact mode switch appears near the brand at the shell level. It is not repeated inside page content.
- Account identity may live in the sidebar footer. Balance and subscription summaries may appear in the account overlay, but never as Chat navigation items.
- Mobile has exactly one application header and one menu trigger. Avoid nested hamburger controls.

### Chat shell

- One left history rail only; remove the current global Work sidebar plus nested history-sidebar composition.
- Rail order: brand/mode switch, prominent New chat action, search, grouped conversation history, account footer.
- Main area is a calm white conversation canvas with a 56px minimal toolbar, large scrollable message region, and bottom composer. Do not wrap the entire chat surface in a card.
- Keep messages and composer centered at approximately 820px and 780px. Assistant content is flat; user content may use a soft neutral bubble.
- Empty state is small and centered, with one concise greeting and supporting line; no hero illustration or promotional card grid.
- Composer uses a 12px–16px restrained radius, one neutral border, subtle floating shadow, attachment entry, visible model label, voice input, and ink send control. Preserve all file, voice, streaming, balance, retry, history, and accessibility behavior.
- On mobile, history opens as the existing focus-managed drawer. The single Chat toolbar owns the menu trigger and page context.

### Work shell

- Left navigation groups and order:
  1. Workbench: Dashboard, Model Center, API Keys, Usage, Service Status.
  2. Account: Balance, Subscriptions, Orders.
  3. Resources: Documentation, Announcements, Settings.
- Preserve existing feature flags. Model Center may use the existing catalog destination until an authenticated wrapper exists. Documentation remains a sanitized external destination; Announcements and Settings remain actions rather than fake routes.
- Preserve valuable feature-gated tools and custom menu items in a visually subordinate `More` group; do not silently remove business capabilities.
- Main workspace canvas is `#F8FAFC`. Page-owned content uses white 16px cards with hairlines; tables and operational surfaces may use 12px–14px radii and near-flat elevation.
- Existing pages continue to own their title and task actions. Do not add a redundant global desktop header.

### Interaction, responsive, and accessibility

- Hover uses neutral background or a slight border change; avoid lifting every control. Motion is 140ms–200ms ease-out and disabled under reduced-motion.
- Focus-visible uses a clear 2px violet outline/ring with offset. Selected states pair color with background, marker, weight, or `aria-current`.
- Sidebar labels, mode controls, drawer, menus, and dialogs remain keyboard reachable. Preserve focus trap, Escape close, focus return, and `inert` semantics already implemented in Chat.
- At widths below 1024px, Work sidebar and Chat history become drawers. At 768px and below, use 16px gutters and 44px controls. No horizontal overflow at 320px.
- Dark theme may remain supported through existing semantic aliases, but this design pass is judged in the light theme and must not introduce a second palette.

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

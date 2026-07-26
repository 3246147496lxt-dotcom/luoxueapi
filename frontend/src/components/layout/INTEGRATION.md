# Layout Integration Guide

This guide describes the current authenticated shell. Use Snow Clay runtime tokens
from `frontend/src/styles/luoxue-clay-tokens.css` and design guidance from
`.superdesign/design-system.md`.

## Add an authenticated page

Define localized route metadata:

```ts
{
  path: '/usage',
  name: 'Usage',
  component: () => import('@/views/user/UsageView.vue'),
  meta: {
    requiresAuth: true,
    title: 'Usage',
    titleKey: 'usage.title',
    descriptionKey: 'usage.description',
  },
}
```

Wrap the view in `AppLayout` and render a semantic page heading:

```vue
<template>
  <AppLayout>
    <AdminPageHeader
      :title="t('usage.title')"
      :description="t('usage.description')"
    />

    <!-- Page content -->
  </AppLayout>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { AdminPageHeader, AppLayout } from '@/components/layout'

const { t } = useI18n()
</script>
```

Use a native `h1` directly when the standard header does not fit the page. Compact
workspaces may use an `sr-only` heading, but must not omit it. The mobile route title
is resolved separately and does not replace this heading.

## Shell composition

`AppLayout` mounts both shell branches:

- Desktop: full-height `AppSidebar`, no global top bar, and a margin-adjusted content
  area.
- Mobile: fixed `AppMobileHeader`, drawer-style `AppSidebar`, and a shared backdrop.

Do not mount these components again inside a page. Do not add local padding to
compensate for the mobile bar; `AppLayout` already applies
`--app-shell-top-offset`.

For viewport-bound layouts, use the shared tokens:

```css
.page-workspace {
  min-height: calc(100dvh - var(--app-shell-top-offset));
}

.page-sticky-control {
  top: calc(var(--app-shell-top-offset) + 1rem);
}
```

Avoid fixed shell-height constants in pages.

## Navigation

Main navigation is rendered by `AppSidebar.vue`. Keep route labels localized and
preserve role, capability, simple-mode, and feature-flag visibility checks.

Account, support, and standalone navigation destinations are declared in
`frontend/src/navigation/shellDestinations.ts`. Use that registry when adding or
removing links for subscriptions, wallet, orders, profile, home, model catalog,
contact, or documentation so the sidebar and account panel share one visibility
policy.

On mobile, the menu button in `AppMobileHeader` controls
`useAppStore().mobileOpen`. Selecting a sidebar route closes the drawer.

## Account entry point

`SidebarAccountDock` is mounted at the bottom of `AppSidebar`. Do not add a second
user dropdown or page-level account launcher.

`SidebarAccountOverlay` presents account summary and actions:

- desktop: an anchored dialog beside the dock;
- mobile: a modal bottom sheet with safe-area padding;
- both: route-aware dismissal, outside-click and Escape handling, theme and locale
  controls, announcements, support, onboarding replay, and logout.

When changing account actions, keep destination selection in
`shellDestinations.ts`, display data in `useAccountSummary`, and the view behavior in
the dock/overlay pair.

## Authentication pages

Authentication views use `AuthLayout` directly:

```vue
<template>
  <AuthLayout>
    <h1>{{ t('auth.login') }}</h1>
    <!-- Form -->
  </AuthLayout>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { AuthLayout } from '@/components/layout'

const { t } = useI18n()
</script>
```

## State responsibilities

- `useAuthStore`: session, role, identity, and logout.
- `useAppStore`: desktop collapse state, mobile drawer state, and public settings.
- `useSubscriptionStore`: account subscription data.
- `useAnnouncementStore`: unread announcements.
- `useOnboardingStore`: guide replay.

Pages should consume these stores for page behavior, not duplicate shell state.

## Testing checklist

When changing the shell, verify:

- `AppLayout` renders the correct desktop and mobile branches;
- `AppMobileHeader` toggles the sidebar and resolves localized route titles;
- `AppSidebar` preserves visibility rules and closes after mobile navigation;
- `SidebarAccountDock` works in expanded and collapsed sidebar states;
- `SidebarAccountOverlay` positions beside its anchor on desktop and behaves as an
  accessible bottom sheet on mobile;
- focus returns to the correct trigger and background scroll is restored;
- authenticated pages retain a semantic `h1`;
- light and dark Snow Clay tokens remain consistent.

Run the related layout unit tests plus `vue-tsc --noEmit` after implementation
changes. Documentation-only edits need formatting and reference checks.

## Troubleshooting

### Mobile menu does not open

- Confirm the route is inside `AppLayout`.
- Check `useAppStore().mobileOpen` and the `lg` breakpoint.
- Check whether another modal currently owns the page scroll lock.

### Account panel has stale data

- Confirm auth and subscription stores have initialized.
- Check `useAccountSummary` rather than duplicating balance calculations.
- Verify capability state before expecting optional destinations.

### Page is offset or clipped

- Remove local shell-height constants.
- Use `--app-shell-top-offset` for viewport height, sticky top, and scroll margin.
- Keep page padding inside the page content rather than recreating shell chrome.

# Layout Components

Vue 3 layout components for the Sub2API frontend. The authenticated product shell uses
Luoxue Snow Clay tokens from `frontend/src/styles/luoxue-clay-tokens.css`; visual
decisions belong in `.superdesign/design-system.md`.

## Authenticated shell

```text
AppLayout
├── AppMobileHeader        mobile navigation context only
├── AppSidebar
│   ├── AppBrand
│   ├── navigation
│   └── SidebarAccountDock
│       └── SidebarAccountOverlay
└── main page slot
```

Desktop uses a full-height sidebar and has no global top bar. Each page owns its
visible heading and must render one semantic `h1`. On mobile, `AppMobileHeader`
provides the navigation trigger, brand, and compact route title.

Do not reserve a page-specific top height. Shell-aware heights and sticky offsets
must use `--app-shell-top-offset`; the token is zero on desktop and includes the
compact mobile bar and safe-area inset on small screens.

## Components

### `AppLayout.vue`

`AppLayout` composes the authenticated shell and adjusts the main content margin
when the sidebar is collapsed.

```vue
<template>
  <AppLayout>
    <section>
      <h1>{{ t('dashboard.title') }}</h1>
      <!-- Page content -->
    </section>
  </AppLayout>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { AppLayout } from '@/components/layout'

const { t } = useI18n()
</script>
```

Use `variant="home-clay"` for the admin content-density adapter and
`variant="chat"` for the viewport-bound chat workspace.

### `AppSidebar.vue`

The sidebar owns authenticated navigation on desktop and mobile:

- persistent and collapsible on desktop;
- opened as a drawer by `AppMobileHeader` on mobile;
- role-, capability-, and simple-mode-aware navigation;
- active-route highlighting;
- `SidebarAccountDock` fixed at the bottom of the sidebar.

`AppLayout` mounts the sidebar automatically.

### `AppMobileHeader.vue`

The compact header is rendered only below the desktop breakpoint. It contains:

- the sidebar menu trigger;
- `AppBrand`;
- the localized page title resolved by `usePageContext()`.

Provide `route.meta.titleKey` whenever possible, with `route.meta.title` as a
fallback. The compact title supplements rather than replaces the page's `h1`.

### `SidebarAccountDock.vue` and `SidebarAccountOverlay.vue`

The account dock is the single entry point for identity and personal account
actions. Its expanded state keeps the capability-gated Upgrade action beside the
account summary; its collapsed state keeps the avatar trigger visible. Opening it
shows identity and balance details, personal profile and preferences, theme,
language, onboarding, and logout.

Task destinations such as subscriptions, wallet, orders, and quota tools belong to
the workspace navigation. Announcements and help resources belong to the scrolling
sidebar's `Support` group; do not add them back to the account overlay.

The overlay is anchored to the dock on desktop and becomes a modal bottom sheet on
mobile. It handles focus return, Escape dismissal, outside-click dismissal, and
mobile background scroll locking. These components are internal to `AppSidebar`;
pages should link to account destinations instead of mounting another account menu.

### `AuthLayout.vue`

`AuthLayout` provides the responsive authentication shell for login, registration,
recovery, verification, and callback pages. It exposes the main content slot and an
optional footer slot.

### Page helpers

- `AdminPageHeader` provides the standard page `h1`, description, and actions.
- `TablePageLayout` provides a bounded table workspace with header and filter slots.

## Route metadata

Route metadata drives the mobile title and browser document title:

```ts
{
  path: '/dashboard',
  name: 'Dashboard',
  component: () => import('@/views/user/DashboardView.vue'),
  meta: {
    requiresAuth: true,
    title: 'Dashboard',
    titleKey: 'dashboard.title',
  },
}
```

Custom menu pages are resolved from their configured labels.

## Store dependencies

- `useAuthStore`: authentication, role, user identity, and logout.
- `useAppStore`: sidebar state, public settings, and shell capabilities.
- `useSubscriptionStore`: account subscription summary.
- `useAnnouncementStore`: unread announcement state.
- `useOnboardingStore`: onboarding replay.

## Responsive and accessibility contract

- The desktop breakpoint is `lg` (`1024px`).
- The mobile header and sidebar drawer share `useAppStore().mobileOpen`.
- Account and navigation triggers expose expanded state and dialog relationships.
- The mobile account sheet traps focus, restores focus on close, and locks background
  scrolling.
- Every authenticated view must keep one semantic `h1`; workspace-style views may
  make it visually hidden.

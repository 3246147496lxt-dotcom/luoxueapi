# Frontend routing architecture

The web UI has two same-origin application entries with separate route graphs:

- `index.ts` owns public, authentication, and user routes.
- `admin.ts` owns `/admin` routes only.
- `routes/public.ts`, `routes/user.ts`, and `routes/admin.ts` are the route manifests.
- `createAppRouter.ts` builds either router without importing the other entry's routes.
- `guards.ts` contains shared authentication, feature-access, canonicalization, and
  full-page entry-bridge behavior.

Both applications keep the existing URL contract. Navigating from a user route to
`/admin/**`, or from an admin route to a non-admin URL, performs a full-page navigation
so the Go server can return the correct HTML entry. Because both entries remain on the
same origin, the existing auth session persists across that navigation.

## Adding a route

1. Put public/authentication routes in `routes/public.ts`.
2. Put authenticated personal-workspace routes in `routes/user.ts`.
3. Put administrator routes in `routes/admin.ts` and set both
   `requiresAuth: true` and `requiresAdmin: true`.
4. Keep views lazy-loaded with `() => import(...)`.
5. Add or update the route contract tests under `__tests__`.

Do not import the admin route manifest from the user entry or vice versa. Shared guard
code may load administrator-only stores dynamically after the route and role require
them, but it must not statically pull those stores into the user entry graph.

## Server contract

Production routing is paired with the embedded Go frontend server:

- `/admin` and `/admin/**` fall back to `admin/index.html`.
- Other SPA routes fall back to `index.html`.
- `/api`, gateway endpoints, documentation assets, and real static assets bypass the
  SPA fallback as before.

Client-side guards improve navigation and UX only. API authorization remains enforced
by the backend JWT/admin middleware.

## Verification

Run the router tests, typecheck, and a production build. Direct refreshes of both a
user deep link and an admin deep link must boot the matching entry, and cross-entry
navigation must preserve query strings and hashes.

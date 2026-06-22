# Admin Frontend — Agent Context

## Stack

Vue 3 + Vite + TypeScript. `oidc-client-ts` for auth (Zitadel OIDC), wrapped in the
`useAuth` composable; `@tanstack/vue-query` for data fetching, Vue Router for routing.

The UI is the **"Helm" design system** — a faithful, hand-authored port of a
Claude Design export (not shadcn-vue, not a CLI-generated component set). Tokens
and component styles live in `src/assets/{theme,components,shell}.css`; the Vue
primitives that consume them live in `src/components/ui/`. Tailwind is present
for incidental utilities, but the design system owns colors and component styling
— prefer the design classes (`.btn`, `.card`, `.table`, `.page`, `.grid-kpi`, …)
over Tailwind color/spacing utilities.

## Icons

Use the in-house `<Icon>` component from `src/components/ui/Icon.vue`:

```vue
<Icon name="mail" :size="15" />
<Icon name="arrow-right" :size="14" />
```

Icon paths are a lucide-derived map in `src/components/ui/icons.ts`. To add an
icon, add its inner SVG markup to that map keyed by name — do **not** reach for a
separate icon library. (`unplugin-icons` is still wired in `vite.config.ts` from
the earlier scaffold, but the design system standardizes on `<Icon>` for visual
consistency; don't mix the two.) Size via the `:size` prop, not a CSS class.

## Theming and customization

All colors are CSS custom properties in `src/assets/theme.css`. To change the
theme, edit those variables — never hardcode color values in components; use
`var(--token)` or the design CSS classes.

- **Light/dark** is the `data-theme` attribute on `<html>` (`light` | `dark`) —
  **not** a `.dark` class, and not `useColorMode`.
- **Density** is the `data-density` attribute (`comfortable` | `compact`), which
  drives the `--d-*` scale.
- Both (plus brand accent) are owned by the **`useTweaks` composable**
  (`src/composables/useTweaks.ts`): it applies them to `<html>` and persists to
  `localStorage` under `helm.tweaks`. The topbar toggle calls it. `index.html`
  re-applies the persisted theme/density before first paint to avoid a flash.

## UI components

The primitives in `src/components/ui/` are hand-authored SFCs (Button, Card,
Badge, Input, Select, Switch, Avatar, Tabs, Segmented, Dropdown + Menu parts,
Dialog, Sheet, Toast, Progress, IconChip, EmptyState, Tag, …). **Edit them
directly** — there is no generator to overwrite them. Build screens by composing
these primitives with the design CSS layout classes.

Conventions worth matching when extending:
- `icon` / `iconRight` props take an **icon name string** (e.g. `icon="download"`),
  with `#icon` / `#iconRight` slots for overrides.
- Components that re-bind `$attrs` set `defineOptions({ inheritAttrs: false })`.
- Toasts: `const toast = useToast()` (`src/composables/useToast.ts`), then
  `toast({ title, desc?, variant? })`. `<ToastRegion>` is mounted once in `AppLayout`.
- Charts live in `src/components/charts/` (lightweight inline SVG).

## Shell & navigation

The app shell (`src/components/shell/`) is Sidebar, TopBar (⌘K command palette
trigger, theme toggle, notifications), CommandPalette, MobileNav, and UserMenu
(wired to real OIDC identity via `useAuth` + signOut). The nav model and screen→path map are
in `shell/nav.ts`; **route names equal screen ids** so the active link derives
from the current route.

Navigate with the `useNav` composable: `const nav = useNav(); nav('organizations',
{ open: orgId })`. Params become route query (deep-linkable, e.g. `?create=1`).

## Form validation

Validate on blur to mark a field dirty, then re-validate on every keystroke while
it is dirty — immediate feedback as the user corrects an invalid value. Never
validate on keystroke alone; only after the field has been blurred once.

```ts
const dirty = reactive({ email: false })
const errors = reactive({ email: '' })
watch(() => form.email, () => { if (dirty.email) errors.email = validateEmail() })
function markDirty(field: 'email') { dirty[field] = true; errors[field] = validate(field) }
```

## Data fetching

Use `@tanstack/vue-query` for all real API calls (`useQuery` for reads,
`useMutation` for writes, invalidate after mutations). API base URL comes from
`import.meta.env.VITE_API_BASE_URL`.

The current screens render **mock data** from `src/lib/data.ts`. When wiring live
data, replace those arrays with service-core queries (via `useApi`) — the screens
depend only on the shapes exported there.

## Auth

Use `useAuth()` (`src/composables/useAuth.ts`, backed by oidc-client-ts) for auth
state and operations. `AppLayout` handles the auth guard — it redirects to the
Zitadel sign-in when `isAuthenticated` is false. Do not add auth checks elsewhere.

```ts
const { getAccessToken } = useAuth()
const token = await getAccessToken() // audience is fixed via the OIDC scope
```

Runtime OIDC config (issuer, clientId, projectId) is loaded by the Docker entrypoint
from `/run/infra/admin-app.json` (written by zitadel-init) into `VITE_OIDC_*` vars;
see `src/config.ts`.

## File structure

```
src/
  assets/        # index.css (Tailwind) + theme/components/shell.css (design system)
  components/
    ui/          # design-system primitives (hand-authored SFCs) + Icon + icons.ts
    charts/      # inline-SVG charts
    shell/       # Sidebar, TopBar, CommandPalette, MobileNav, UserMenu, nav.ts
    layout/      # AppLayout (auth guard + shell)
  composables/   # useTweaks, useNav, useToast
  lib/
    data.ts      # mock data (replace with API calls)
    types.ts
    utils.ts     # cn() and other helpers
  pages/         # one file per route (Overview, Organizations, Users, …)
  router/
    index.ts
  main.ts
  App.vue
```

## Environment variables

All env vars are prefixed `VITE_`. See `.env.example` for the full list. Never
hardcode URLs or IDs.

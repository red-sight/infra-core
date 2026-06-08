# Admin Frontend — Agent Context

## Stack

Vue 3 + Vite + TypeScript. shadcn-vue + Tailwind CSS for UI. `@logto/vue` for auth. `@tanstack/vue-query` for data fetching. Vue Router for routing.

## Icons

Always use **unplugin-icons** auto-imported components — never import from `lucide-vue-next` or any other icon library directly.

`vite.config.ts` configures `Icons({ compiler: 'vue3' })` and `Components({ resolvers: [IconsResolver()] })`, so icon components are resolved automatically at build time with no explicit imports required.

**Naming convention:** `I` + PascalCase collection + PascalCase icon name.

```vue
<!-- correct -->
<ILucideMail class="size-4" />
<ILucideArrowRight class="size-5" />

<!-- wrong — do not do this -->
import { Mail } from 'lucide-vue-next'
```

Size via Tailwind class (`size-4`, `size-5`, etc.), not `:size` prop.

Available collections: everything in `@iconify/json` (lucide, mdi, heroicons, …).

## Form validation

Validate on blur to mark a field dirty, then re-validate on every keystroke while it is dirty — so the user gets immediate feedback as they correct an invalid value.

```ts
const dirty = reactive({ email: false })
const errors = reactive({ email: '' })

watch(() => form.email, () => { if (dirty.email) errors.email = validateEmail() })

function markDirty(field: 'email') {
  dirty[field] = true
  errors[field] = validate(field)
}
```

```vue
<input @blur="markDirty('email')" />
```

Never validate on keystroke alone — only after the field has been blurred at least once.

## Theming and customization

All colors are CSS custom properties defined in `src/assets/index.css`. To change the theme, edit only the CSS variables in `:root` and `.dark` — never hardcode color values in components.

Dark mode is controlled by the `dark` class on `<html>`. Use the `useColorMode()` composable from `@vueuse/core` — it supports `'auto'` (follows system), `'dark'`, and `'light'`.

## shadcn-vue components

Add components via CLI:
```sh
npx shadcn-vue@latest add <component>
```

Components are copied into `src/components/ui/`. Do not edit them unless customizing for the project — re-running the CLI will overwrite changes.

## Data fetching

Use `@tanstack/vue-query` for all API calls:
- `useQuery` for reads
- `useMutation` for writes
- Invalidate relevant queries after mutations

API base URL comes from `import.meta.env.VITE_API_BASE_URL`.

## Auth

Use `useLogto()` composable for auth state and operations. The `AppLayout` component handles the auth guard — it redirects to Logto sign-in when `isAuthenticated` is false. Do not add auth checks elsewhere.

Access tokens for API calls:
```ts
const { getAccessToken } = useLogto()
const token = await getAccessToken(import.meta.env.VITE_LOGTO_API_RESOURCE)
```

## File structure

```
src/
  assets/        # global CSS only
  components/
    ui/          # shadcn-vue components (generated)
    layout/      # AppLayout, AppHeader, etc.
  composables/   # shared Vue composables
  lib/
    utils.ts     # cn() and other helpers
  pages/         # one file per route
  router/
    index.ts
  main.ts
  App.vue
```

## Environment variables

All env vars are prefixed `VITE_`. See `.env.example` for the full list. Never hardcode URLs or IDs.

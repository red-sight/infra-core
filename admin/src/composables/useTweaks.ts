// Tweaks — single source of truth for theme / density / accent.
// Applies them to <html> (data-theme, data-density, and accent CSS vars) and
// persists to localStorage. A module-level reactive object makes this a shared
// singleton across every component that calls useTweaks().
import { reactive, readonly, watch } from 'vue'

export type Theme = 'light' | 'dark'
export type Density = 'comfortable' | 'compact'

export interface Tweaks {
  theme: Theme
  density: Density
  accent: string
}

const STORAGE_KEY = 'helm.tweaks'

// Accent → { hover } map. The default near-black/white is theme-aware, so it
// clears the overrides and lets theme.css drive --primary.
const ACCENTS: Record<string, { hover: string } | null> = {
  '#18181b': null,
  '#2563eb': { hover: '#1d4ed8' },
  '#7c3aed': { hover: '#6d28d9' },
  '#16a34a': { hover: '#15803d' },
  '#d97706': { hover: '#b45309' },
}

export const ACCENT_OPTIONS = Object.keys(ACCENTS)

const DEFAULTS: Tweaks = { theme: 'light', density: 'comfortable', accent: '#18181b' }

function load(): Tweaks {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) return { ...DEFAULTS, ...(JSON.parse(raw) as Partial<Tweaks>) }
  } catch {
    /* ignore malformed storage */
  }
  return { ...DEFAULTS }
}

const state = reactive<Tweaks>(load())

function applyAccent(hex: string) {
  const s = document.documentElement.style
  const a = ACCENTS[hex]
  if (!a) {
    for (const v of ['--primary', '--primary-hover', '--primary-foreground', '--ring']) {
      s.removeProperty(v)
    }
  } else {
    s.setProperty('--primary', hex)
    s.setProperty('--primary-hover', a.hover)
    s.setProperty('--primary-foreground', '#ffffff')
    s.setProperty('--ring', hex)
  }
}

function apply(t: Tweaks) {
  const el = document.documentElement
  el.setAttribute('data-theme', t.theme)
  el.setAttribute('data-density', t.density)
  applyAccent(t.accent)
}

let started = false
function start() {
  if (started) return
  started = true
  watch(
    state,
    (t) => {
      apply(t)
      try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(t))
      } catch {
        /* storage may be unavailable */
      }
    },
    { immediate: true, deep: true },
  )
}

export function useTweaks() {
  start()
  return {
    tweaks: readonly(state),
    setTheme: (v: Theme) => (state.theme = v),
    toggleTheme: () => (state.theme = state.theme === 'dark' ? 'light' : 'dark'),
    setDensity: (v: Density) => (state.density = v),
    setAccent: (v: string) => (state.accent = v),
  }
}

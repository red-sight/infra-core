<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import Icon from '@/components/ui/Icon.vue'
import Kbd from '@/components/ui/Kbd.vue'
import { NAV } from './nav'
import { useNav } from '@/composables/useNav'
import { ORGS, USERS } from '@/lib/data'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()
const nav = useNav()

interface Item {
  type: string
  label: string
  icon: string
  go: string
  meta?: string
}

const q = ref('')
const sel = ref(0)
const inputRef = ref<HTMLInputElement | null>(null)

const pages: Item[] = NAV.map((n) => ({ type: 'Page', label: n.label, icon: n.icon, go: n.id }))
const orgs: Item[] = ORGS.map((o) => ({ type: 'Organization', label: o.name, icon: 'building-2', go: 'organizations', meta: o.slug }))
const users: Item[] = USERS.map((u) => ({ type: 'User', label: u.name, icon: 'user', go: 'users', meta: u.email }))

const items = computed<Item[]>(() => {
  if (!q.value.trim()) return pages
  const t = q.value.toLowerCase()
  return [...pages, ...orgs, ...users]
    .filter((i) => i.label.toLowerCase().includes(t) || (i.meta || '').toLowerCase().includes(t))
    .slice(0, 9)
})

// Group headers: show the type label the first time it appears.
const rows = computed(() => {
  let last: string | null = null
  return items.value.map((it, i) => {
    const header = it.type !== last ? it.type : null
    last = it.type
    return { it, i, header }
  })
})

watch(q, () => (sel.value = 0))
watch(
  () => props.open,
  (v) => {
    if (v) {
      q.value = ''
      nextTick(() => setTimeout(() => inputRef.value?.focus(), 30))
      document.addEventListener('keydown', onKey)
    } else {
      document.removeEventListener('keydown', onKey)
    }
  },
)
onBeforeUnmount(() => document.removeEventListener('keydown', onKey))

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
  else if (e.key === 'ArrowDown') {
    e.preventDefault()
    sel.value = Math.min(sel.value + 1, items.value.length - 1)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    sel.value = Math.max(sel.value - 1, 0)
  } else if (e.key === 'Enter' && items.value[sel.value]) {
    select(items.value[sel.value])
  }
}

function select(it: Item) {
  nav(it.go)
  emit('close')
}
</script>

<template>
  <Teleport to="body">
    <template v-if="open">
      <div class="overlay" style="z-index: 110" @click="emit('close')" />
      <div
        style="
          position: fixed;
          top: 14vh;
          left: 50%;
          transform: translateX(-50%);
          z-index: 111;
          width: min(560px, calc(100vw - 32px));
          background: var(--popover);
          border: 1px solid var(--border);
          border-radius: var(--radius-lg);
          box-shadow: var(--shadow-lg);
          overflow: hidden;
          animation: fadeUp 0.16s ease both;
        "
      >
        <div style="display: flex; align-items: center; gap: 10px; padding: 14px 16px; border-bottom: 1px solid var(--border)">
          <Icon name="search" :size="17" style="color: var(--muted-foreground)" />
          <input
            ref="inputRef"
            v-model="q"
            placeholder="Search pages, organizations, users…"
            style="flex: 1; border: none; outline: none; background: transparent; font-size: 14.5px; color: var(--foreground)"
          />
          <Kbd>Esc</Kbd>
        </div>
        <div style="max-height: 360px; overflow: auto; padding: 6px">
          <div
            v-if="items.length === 0"
            style="padding: 28px; text-align: center; color: var(--muted-foreground); font-size: 13px"
          >
            No results for “{{ q }}”.
          </div>
          <template v-for="row in rows" :key="row.i">
            <div v-if="row.header" class="menu__label" :style="{ paddingTop: row.i === 0 ? '2px' : '8px' }">
              {{ row.header }}s
            </div>
            <button
              class="cmd-item"
              :data-active="sel === row.i"
              @mouseenter="sel = row.i"
              @click="select(row.it)"
            >
              <Icon :name="row.it.icon" :size="16" style="color: var(--muted-foreground)" />
              <span style="flex: 1; text-align: left">{{ row.it.label }}</span>
              <span v-if="row.it.meta" style="font-size: 12px; color: var(--muted-foreground)">{{ row.it.meta }}</span>
              <Icon v-if="sel === row.i" name="corner-down-left" :size="14" style="color: var(--muted-foreground)" />
            </button>
          </template>
        </div>
      </div>
    </template>
  </Teleport>
</template>

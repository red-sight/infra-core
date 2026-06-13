<script setup lang="ts">
import { computed } from 'vue'
import Icon from '@/components/ui/Icon.vue'
import Kbd from '@/components/ui/Kbd.vue'
import Dropdown from '@/components/ui/Dropdown.vue'
import MenuLabel from '@/components/ui/MenuLabel.vue'
import { NAV } from './nav'
import { useTweaks } from '@/composables/useTweaks'

const props = defineProps<{ current: string }>()
const emit = defineEmits<{ openCmd: []; openMobile: [] }>()

const { tweaks, toggleTheme } = useTweaks()
const title = computed(() => NAV.find((n) => n.id === props.current)?.label ?? '')
</script>

<template>
  <header class="topbar">
    <button
      class="btn btn--ghost btn--icon btn--sm mobile-only"
      aria-label="Open menu"
      style="display: none"
      @click="emit('openMobile')"
    >
      <Icon name="menu" :size="18" />
    </button>

    <div class="topbar__crumb">
      <Icon name="layers" :size="14" style="color: var(--muted-foreground)" />
      <Icon name="chevron-right" :size="13" style="color: var(--muted-foreground); opacity: 0.6" />
      <span style="font-weight: 600">{{ title }}</span>
    </div>

    <div style="flex: 1" />

    <button class="cmd-trigger" @click="emit('openCmd')">
      <Icon name="search" :size="14" />
      <span>Search…</span>
      <Kbd>⌘K</Kbd>
    </button>

    <button
      class="btn btn--ghost btn--icon btn--sm"
      aria-label="Toggle theme"
      title="Toggle theme"
      @click="toggleTheme"
    >
      <Icon :name="tweaks.theme === 'dark' ? 'sun' : 'moon'" :size="17" />
    </button>

    <Dropdown align="end" :width="230">
      <template #trigger>
        <button class="btn btn--ghost btn--icon btn--sm" aria-label="Notifications" style="position: relative">
          <Icon name="bell" :size="17" />
          <span
            style="position: absolute; top: 5px; right: 5px; width: 7px; height: 7px; border-radius: 999px; background: var(--destructive); border: 2px solid var(--background)"
          />
        </button>
      </template>
      <MenuLabel>Notifications</MenuLabel>
      <div style="padding: 4px 9px 8px; font-size: 12.5px; color: var(--muted-foreground); line-height: 1.5">
        <div style="display: flex; gap: 8px; padding: 6px 0">
          <Icon name="alert-triangle" :size="14" style="color: var(--warn-fg); margin-top: 2px" />
          <span><b style="color: var(--foreground)">Ridgeline Capital</b> payment is past due.</span>
        </div>
        <div style="display: flex; gap: 8px; padding: 6px 0">
          <Icon name="user-plus" :size="14" style="color: var(--info-fg); margin-top: 2px" />
          <span>3 new seat invitations accepted.</span>
        </div>
      </div>
    </Dropdown>
  </header>
</template>

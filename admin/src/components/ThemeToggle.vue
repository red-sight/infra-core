<script setup lang="ts">
import { useColorMode } from '@vueuse/core'

type Mode = 'auto' | 'dark' | 'light'

const mode = useColorMode()
const cycle: Record<Mode, Mode> = { auto: 'light', light: 'dark', dark: 'auto' }
const labels: Record<Mode, string> = { auto: 'Auto', light: 'Light', dark: 'Dark' }

function toggle() {
  mode.value = cycle[mode.value as Mode] ?? 'auto'
}
</script>

<template>
  <button
    type="button"
    :title="`Theme: ${labels[mode as Mode] ?? 'Auto'} — click to change`"
    class="inline-flex items-center gap-1.5 rounded-md px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
    @click="toggle"
  >
    <ILucideMonitor v-if="mode === 'auto'" class="size-4" />
    <ILucideSun v-else-if="mode === 'light'" class="size-4" />
    <ILucideMoon v-else class="size-4" />
    <span class="sr-only">{{ labels[mode as Mode] ?? 'Auto' }}</span>
  </button>
</template>

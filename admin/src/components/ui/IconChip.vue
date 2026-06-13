<script setup lang="ts">
import { computed } from 'vue'
import Icon from './Icon.vue'

const props = withDefaults(
  defineProps<{ name: string; tone?: string; size?: number; iconSize?: number }>(),
  { tone: 'muted', size: 36, iconSize: 17 },
)

const TONES: Record<string, [string, string]> = {
  muted: ['var(--muted)', 'var(--muted-foreground)'],
  primary: ['var(--accent)', 'var(--foreground)'],
  success: ['var(--success-bg)', 'var(--success-fg)'],
  danger: ['var(--danger-bg)', 'var(--danger-fg)'],
  warning: ['var(--warn-bg)', 'var(--warn-fg)'],
  info: ['var(--info-bg)', 'var(--info-fg)'],
  violet: ['var(--violet-bg)', 'var(--violet-fg)'],
}

const style = computed(() => {
  const [bg, fg] = TONES[props.tone] ?? TONES.muted
  return { width: `${props.size}px`, height: `${props.size}px`, background: bg, color: fg }
})
</script>

<template>
  <span class="icon-chip" :style="style"><Icon :name="name" :size="iconSize" /></span>
</template>

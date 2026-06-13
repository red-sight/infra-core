<script setup lang="ts">
import { computed } from 'vue'
import { ICON_PATHS } from './icons'

const props = withDefaults(
  defineProps<{
    name: string
    size?: number
    strokeWidth?: number
  }>(),
  { size: 16, strokeWidth: 2 },
)

const inner = computed(() => {
  const d = ICON_PATHS[props.name]
  if (!d && props.name) console.warn('Missing icon:', props.name)
  return d ?? ''
})
</script>

<template>
  <svg
    :width="size"
    :height="size"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    :stroke-width="strokeWidth"
    stroke-linecap="round"
    stroke-linejoin="round"
    v-html="inner"
  />
</template>

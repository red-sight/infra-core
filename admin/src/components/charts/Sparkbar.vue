<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{ values: number[]; color?: string; height?: number }>(),
  { color: 'var(--primary)', height: 36 },
)

const max = computed(() => Math.max(...props.values))
</script>

<template>
  <div :style="{ display: 'flex', alignItems: 'flex-end', gap: '3px', height: `${height}px` }">
    <div
      v-for="(v, i) in values"
      :key="i"
      :style="{
        flex: 1,
        height: `${(v / max) * 100}%`,
        minHeight: '2px',
        background: color,
        borderRadius: '2px',
        opacity: 0.35 + 0.65 * (i / values.length),
      }"
    />
  </div>
</template>

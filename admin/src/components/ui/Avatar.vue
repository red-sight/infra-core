<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    name?: string
    size?: 'xs' | 'sm' | 'md' | 'lg'
    square?: boolean
    color?: string
  }>(),
  { name: '', size: 'md' },
)

const AV_COLORS = ['#18181b', '#2563eb', '#16a34a', '#d97706', '#7c3aed', '#dc2626', '#0891b2', '#db2777']

function avColor(s: string) {
  let h = 0
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) >>> 0
  return AV_COLORS[h % AV_COLORS.length]
}

const initials = computed(() =>
  props.name
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((w) => w[0])
    .join('')
    .toUpperCase(),
)
const classes = computed(() => ['avatar', `avatar--${props.size}`, props.square && 'avatar--square'])
const bg = computed(() => props.color || avColor(props.name))
</script>

<template>
  <span :class="classes" :style="{ background: bg }">{{ initials }}</span>
</template>

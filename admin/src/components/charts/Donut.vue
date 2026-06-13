<script setup lang="ts">
import { computed } from 'vue'

interface Slice {
  count: number
  color: string
}

const props = withDefaults(defineProps<{ data: Slice[]; size?: number; thickness?: number }>(), {
  size: 150,
  thickness: 22,
})

const total = computed(() => props.data.reduce((s, d) => s + d.count, 0))
const r = computed(() => (props.size - props.thickness) / 2)
const c = computed(() => props.size / 2)
const circ = computed(() => 2 * Math.PI * r.value)

const segments = computed(() => {
  let off = 0
  return props.data.map((d) => {
    const len = (d.count / total.value) * circ.value
    const seg = { color: d.color, len, gap: circ.value - len, offset: -off }
    off += len
    return seg
  })
})
</script>

<template>
  <svg :viewBox="`0 0 ${size} ${size}`" :width="size" :height="size">
    <circle :cx="c" :cy="c" :r="r" fill="none" stroke="var(--muted)" :stroke-width="thickness" />
    <circle
      v-for="(seg, i) in segments"
      :key="i"
      :cx="c"
      :cy="c"
      :r="r"
      fill="none"
      :stroke="seg.color"
      :stroke-width="thickness"
      :stroke-dasharray="`${seg.len} ${seg.gap}`"
      :stroke-dashoffset="seg.offset"
      :transform="`rotate(-90 ${c} ${c})`"
      stroke-linecap="butt"
    />
    <text :x="c" :y="c - 4" text-anchor="middle" font-size="22" font-weight="700" fill="var(--foreground)">
      {{ total }}
    </text>
    <text :x="c" :y="c + 14" text-anchor="middle" font-size="10.5" fill="var(--muted-foreground)">
      total orgs
    </text>
  </svg>
</template>

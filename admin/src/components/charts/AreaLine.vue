<script setup lang="ts">
import { computed, ref } from 'vue'

interface SeriesKey {
  key: string
  label: string
  color: string
}

const props = withDefaults(
  defineProps<{ data: Record<string, any>[]; height?: number; keys: SeriesKey[] }>(),
  { height: 220 },
)

const hover = ref<number | null>(null)

const W = 720
const padL = 36
const padR = 12
const padT = 14
const padB = 26
const ticks = 4

const h = computed(() => props.height)
const iw = W - padL - padR
const ih = computed(() => h.value - padT - padB)

const max = computed(
  () => Math.max(...props.data.flatMap((d) => props.keys.map((k) => d[k.key]))) * 1.15,
)

const xLabelKey = computed(() => Object.keys(props.data[0] ?? {})[0])

function x(i: number) {
  return padL + (iw * i) / (props.data.length - 1)
}
function y(v: number) {
  return padT + ih.value - (v / max.value) * ih.value
}
function line(k: string) {
  return props.data.map((d, i) => `${i ? 'L' : 'M'}${x(i)},${y(d[k])}`).join(' ')
}
function area(k: string) {
  return `${line(k)} L${x(props.data.length - 1)},${padT + ih.value} L${padL},${padT + ih.value} Z`
}

const tooltipX = computed(() =>
  hover.value == null ? 0 : Math.min(Math.max(x(hover.value) - 52, 2), W - 106),
)
</script>

<template>
  <svg :viewBox="`0 0 ${W} ${h}`" width="100%" style="display: block" @mouseleave="hover = null">
    <defs>
      <linearGradient v-for="k in keys" :id="`ag-${k.key}`" :key="k.key" x1="0" y1="0" x2="0" y2="1">
        <stop offset="0%" :stop-color="k.color" stop-opacity="0.18" />
        <stop offset="100%" :stop-color="k.color" stop-opacity="0" />
      </linearGradient>
    </defs>

    <g v-for="i in ticks + 1" :key="`grid-${i}`">
      <line
        :x1="padL"
        :y1="padT + (ih * (i - 1)) / ticks"
        :x2="W - padR"
        :y2="padT + (ih * (i - 1)) / ticks"
        stroke="var(--border)"
        stroke-width="1"
      />
      <text
        :x="padL - 8"
        :y="padT + (ih * (i - 1)) / ticks + 3"
        text-anchor="end"
        font-size="10"
        fill="var(--muted-foreground)"
      >
        {{ Math.round(max - (max * (i - 1)) / ticks) }}
      </text>
    </g>

    <path v-for="k in keys" :key="`a-${k.key}`" :d="area(k.key)" :fill="`url(#ag-${k.key})`" />
    <path
      v-for="k in keys"
      :key="`l-${k.key}`"
      :d="line(k.key)"
      fill="none"
      :stroke="k.color"
      stroke-width="2"
      stroke-linejoin="round"
      stroke-linecap="round"
    />

    <g v-for="(d, i) in data" :key="`pt-${i}`">
      <text :x="x(i)" :y="h - 6" text-anchor="middle" font-size="10" fill="var(--muted-foreground)">
        {{ d[xLabelKey] }}
      </text>
      <rect
        :x="x(i) - iw / data.length / 2"
        :y="padT"
        :width="iw / data.length"
        :height="ih"
        fill="transparent"
        @mouseenter="hover = i"
      />
      <line
        v-if="hover === i"
        :x1="x(i)"
        :y1="padT"
        :x2="x(i)"
        :y2="padT + ih"
        stroke="var(--border-strong)"
        stroke-width="1"
        stroke-dasharray="3 3"
      />
      <circle
        v-for="k in keys"
        v-show="hover === i"
        :key="`c-${k.key}`"
        :cx="x(i)"
        :cy="y(d[k.key])"
        r="3.5"
        fill="var(--card)"
        :stroke="k.color"
        stroke-width="2"
      />
    </g>

    <g v-if="hover != null">
      <rect
        :x="tooltipX"
        :y="padT + 2"
        width="104"
        :height="18 + keys.length * 15"
        rx="6"
        fill="var(--popover)"
        stroke="var(--border)"
      />
      <text
        v-for="(k, j) in keys"
        :key="`tt-${k.key}`"
        :x="tooltipX + 9"
        :y="padT + 19 + j * 15"
        font-size="11"
        fill="var(--foreground)"
      >
        <tspan :fill="k.color">●</tspan> {{ k.label }}: {{ data[hover][k.key] }}
      </text>
    </g>
  </svg>
</template>

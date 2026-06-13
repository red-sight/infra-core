<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'

const props = withDefaults(
  defineProps<{
    align?: 'start' | 'end'
    width?: number
    up?: boolean
    block?: boolean
  }>(),
  { align: 'end', up: false, block: false },
)

const open = ref(false)
const root = ref<HTMLElement | null>(null)

function onDocMouseDown(e: MouseEvent) {
  if (root.value && !root.value.contains(e.target as Node)) open.value = false
}
function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') open.value = false
}

watch(open, (v) => {
  if (v) {
    document.addEventListener('mousedown', onDocMouseDown)
    document.addEventListener('keydown', onKey)
  } else {
    document.removeEventListener('mousedown', onDocMouseDown)
    document.removeEventListener('keydown', onKey)
  }
})
onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocMouseDown)
  document.removeEventListener('keydown', onKey)
})

const menuStyle = computed(() => {
  const s: Record<string, string | number> = { position: 'absolute', zIndex: 50 }
  if (props.up) s.bottom = 'calc(100% + 6px)'
  else s.top = 'calc(100% + 6px)'
  s[props.align === 'end' ? 'right' : 'left'] = 0
  if (props.width) s.width = `${props.width}px`
  return s
})
const rootStyle = computed(() => ({
  position: 'relative' as const,
  display: props.block ? 'block' : 'inline-flex',
  width: props.block ? '100%' : undefined,
}))
</script>

<template>
  <div ref="root" :style="rootStyle">
    <span :style="block ? { display: 'block' } : undefined" @click="open = !open">
      <slot name="trigger" />
    </span>
    <div v-if="open" class="menu" :style="menuStyle" @click="open = false">
      <slot />
    </div>
  </div>
</template>

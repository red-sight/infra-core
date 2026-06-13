<script setup lang="ts">
import { onBeforeUnmount, useSlots, watch } from 'vue'
import Icon from './Icon.vue'

const props = defineProps<{
  open: boolean
  title?: string
  desc?: string
  width?: number
}>()

const emit = defineEmits<{ close: [] }>()
const slots = useSlots()

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}

watch(
  () => props.open,
  (v) => {
    if (v) {
      document.addEventListener('keydown', onKey)
      document.body.style.overflow = 'hidden'
    } else {
      document.removeEventListener('keydown', onKey)
      document.body.style.overflow = ''
    }
  },
)
onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKey)
  document.body.style.overflow = ''
})
</script>

<template>
  <Teleport to="body">
    <template v-if="open">
      <div class="overlay" @click="emit('close')" />
      <div
        class="sheet"
        role="dialog"
        aria-modal="true"
        :style="width ? { width: `min(${width}px, 100vw)` } : undefined"
      >
        <div class="sheet__head">
          <div>
            <div class="sheet__title"><slot name="title">{{ title }}</slot></div>
            <div v-if="desc || slots.desc" class="sheet__desc"><slot name="desc">{{ desc }}</slot></div>
          </div>
          <button class="icon-x" aria-label="Close" @click="emit('close')"><Icon name="x" :size="18" /></button>
        </div>
        <div class="sheet__body"><slot /></div>
        <div v-if="slots.footer" class="sheet__foot"><slot name="footer" /></div>
      </div>
    </template>
  </Teleport>
</template>

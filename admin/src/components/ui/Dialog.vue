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
        class="dialog"
        role="dialog"
        aria-modal="true"
        :style="width ? { width: `min(${width}px, calc(100vw - 32px))` } : undefined"
      >
        <div v-if="title || desc || slots.title" class="dialog__head">
          <div>
            <div class="dialog__title"><slot name="title">{{ title }}</slot></div>
            <div v-if="desc" class="dialog__desc">{{ desc }}</div>
          </div>
          <button class="icon-x" aria-label="Close" @click="emit('close')"><Icon name="x" :size="18" /></button>
        </div>
        <div class="dialog__body"><slot /></div>
        <div v-if="slots.footer" class="dialog__foot"><slot name="footer" /></div>
      </div>
    </template>
  </Teleport>
</template>

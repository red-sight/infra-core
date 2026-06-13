<script setup lang="ts">
import Icon from './Icon.vue'
import { toasts } from '@/composables/useToast'

function iconFor(variant?: string) {
  return variant === 'error' ? 'x-circle' : variant === 'info' ? 'info' : 'check-circle'
}
</script>

<template>
  <Teleport to="body">
    <div class="toast-region">
      <div v-for="t in toasts" :key="t.id" :class="['toast', `toast--${t.variant || 'success'}`]">
        <span class="toast__icon"><Icon :name="iconFor(t.variant)" :size="18" /></span>
        <div style="min-width: 0">
          <div class="toast__title">{{ t.title }}</div>
          <div v-if="t.desc" class="toast__desc">{{ t.desc }}</div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

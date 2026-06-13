<script setup lang="ts">
import Icon from '@/components/ui/Icon.vue'
defineEmits<{ click: [] }>()
</script>

<template>
  <button
    style="display: flex; align-items: center; gap: 10px; flex-shrink: 0"
    @click="$emit('click')"
  >
    <span
      style="
        width: 32px;
        height: 32px;
        border-radius: 8px;
        background: var(--primary);
        color: var(--primary-foreground);
        display: flex;
        align-items: center;
        justify-content: center;
      "
    >
      <Icon name="layers" :size="18" />
    </span>
    <span style="display: flex; flex-direction: column; line-height: 1.1; text-align: left">
      <span style="font-weight: 700; font-size: 14.5px; letter-spacing: -0.02em">Helm</span>
      <span style="font-size: 11px; color: var(--muted-foreground); font-weight: 500">Admin Console</span>
    </span>
  </button>
</template>

<script setup lang="ts">
// Removable filter chip. The design references <Tag onRemove> but ships no .tag
// CSS, so it is styled inline from tokens here.
import Icon from './Icon.vue'

defineEmits<{ remove: [] }>()
</script>

<template>
  <span
    style="
      display: inline-flex;
      align-items: center;
      gap: 5px;
      padding: 2px 4px 2px 9px;
      font-size: 12px;
      font-weight: 500;
      border-radius: var(--radius-pill);
      background: var(--muted);
      color: var(--foreground);
      border: 1px solid var(--border);
    "
  >
    <slot />
    <button
      type="button"
      aria-label="Remove filter"
      style="
        display: inline-flex;
        padding: 2px;
        border-radius: 999px;
        color: var(--muted-foreground);
      "
      @click="$emit('remove')"
    >
      <Icon name="x" :size="12" />
    </button>
  </span>
</template>

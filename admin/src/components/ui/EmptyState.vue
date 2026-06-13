<script setup lang="ts">
import Icon from './Icon.vue'

withDefaults(defineProps<{ icon?: string; title?: string; desc?: string }>(), { icon: 'search' })
</script>

<template>
  <div class="empty-state">
    <div class="empty-state__icon"><Icon :name="icon" :size="20" /></div>
    <div style="font-weight: 600; color: var(--foreground); font-size: 14px">{{ title }}</div>
    <div v-if="desc" style="margin-top: 4px; font-size: 13px">{{ desc }}</div>
    <div v-if="$slots.action" style="margin-top: 16px"><slot name="action" /></div>
  </div>
</template>

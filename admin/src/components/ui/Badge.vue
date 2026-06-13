<script setup lang="ts">
import { computed } from 'vue'
import Icon from './Icon.vue'

const props = withDefaults(
  defineProps<{
    variant?: 'solid' | 'outline' | 'muted' | 'success' | 'danger' | 'warning' | 'info' | 'violet'
    dot?: boolean
    icon?: string
  }>(),
  { variant: 'muted' },
)

const classes = computed(() => ['badge', `badge--${props.variant}`])
</script>

<template>
  <span :class="classes">
    <span v-if="dot" class="badge__dot" />
    <Icon v-if="icon" :name="icon" :size="11" />
    <slot />
  </span>
</template>

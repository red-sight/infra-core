<script setup lang="ts">
import { computed, useSlots } from 'vue'
import Icon from './Icon.vue'

const props = withDefaults(
  defineProps<{
    variant?: 'primary' | 'secondary' | 'outline' | 'ghost' | 'destructive' | 'danger-outline'
    size?: 'sm' | 'md' | 'lg'
    block?: boolean
    icon?: string
    iconRight?: string
    iconSize?: number
    tag?: string
  }>(),
  { variant: 'primary', size: 'md', iconSize: 15, tag: 'button' },
)

const slots = useSlots()
const hasLabel = computed(() => !!slots.default)

const classes = computed(() => [
  'btn',
  `btn--${props.variant}`,
  props.size !== 'md' && `btn--${props.size}`,
  !hasLabel.value && 'btn--icon',
  props.block && 'btn--block',
])
</script>

<template>
  <component :is="tag" :class="classes" :type="tag === 'button' ? 'button' : undefined">
    <slot name="icon"><Icon v-if="icon" :name="icon" :size="iconSize" /></slot>
    <span v-if="hasLabel"><slot /></span>
    <slot name="iconRight"><Icon v-if="iconRight" :name="iconRight" :size="iconSize" /></slot>
  </component>
</template>

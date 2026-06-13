<script setup lang="ts">
import { computed, useSlots } from 'vue'

const props = defineProps<{
  title?: string
  desc?: string
  bordered?: boolean
}>()

const slots = useSlots()
const classes = computed(() => ['card__head', props.bordered && 'card__head--bordered'])
</script>

<template>
  <div :class="classes" style="display: flex; align-items: flex-start; justify-content: space-between; gap: 16px">
    <div style="min-width: 0">
      <div v-if="title || slots.title" class="card__title"><slot name="title">{{ title }}</slot></div>
      <div v-if="desc || slots.desc" class="card__desc"><slot name="desc">{{ desc }}</slot></div>
    </div>
    <div v-if="slots.actions" style="flex-shrink: 0; display: flex; gap: 8px; align-items: center">
      <slot name="actions" />
    </div>
  </div>
</template>

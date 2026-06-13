<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{ value?: number; color?: string }>(), { value: 0 })
const width = computed(() => `${Math.max(0, Math.min(100, props.value))}%`)
</script>

<template>
  <div class="progress">
    <div class="progress__bar" :style="{ width, background: color }" />
  </div>
</template>

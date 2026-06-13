<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  pad?: boolean
  hover?: boolean
}>()

const classes = computed(() => ['card', props.pad && 'card--pad', props.hover && 'card--hover'])
</script>

<template>
  <div :class="classes"><slot /></div>
</template>

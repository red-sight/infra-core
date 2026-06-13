<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  checked?: boolean
  disabled?: boolean
  size?: 'sm'
}>()

const emit = defineEmits<{ change: [checked: boolean] }>()
const classes = computed(() => ['switch', props.size === 'sm' && 'switch--sm', props.disabled && 'switch--disabled'])
</script>

<template>
  <label :class="classes">
    <input
      type="checkbox"
      :checked="checked"
      :disabled="disabled"
      @change="emit('change', ($event.target as HTMLInputElement).checked)"
    />
    <span class="switch__track"><span class="switch__thumb" /></span>
  </label>
</template>

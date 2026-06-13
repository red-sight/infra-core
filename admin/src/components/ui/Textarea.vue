<script setup lang="ts">
import { computed } from 'vue'

defineOptions({ inheritAttrs: false })

const props = defineProps<{
  modelValue?: string
  error?: string | null
}>()

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const classes = computed(() => ['input', props.error && 'input--err'])
</script>

<template>
  <textarea
    :class="classes"
    :value="modelValue"
    v-bind="$attrs"
    @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
  />
</template>

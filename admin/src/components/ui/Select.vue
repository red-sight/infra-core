<script setup lang="ts">
import { computed } from 'vue'

defineOptions({ inheritAttrs: false })

type Option = string | { value: string; label: string }

const props = withDefaults(
  defineProps<{
    modelValue?: string
    options?: Option[]
    placeholder?: string
    filter?: boolean
  }>(),
  { options: () => [] },
)

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const normalized = computed(() =>
  props.options.map((o) => (typeof o === 'string' ? { value: o, label: o } : o)),
)
const classes = computed(() => ['select', props.filter && 'filter-sel'])
</script>

<template>
  <select
    :class="classes"
    :value="modelValue"
    v-bind="$attrs"
    @change="emit('update:modelValue', ($event.target as HTMLSelectElement).value)"
  >
    <option v-if="placeholder" value="">{{ placeholder }}</option>
    <option v-for="o in normalized" :key="o.value" :value="o.value">{{ o.label }}</option>
    <slot />
  </select>
</template>

<script setup lang="ts">
import { computed } from 'vue'

type Option = string | { value: string; label: string }

const props = defineProps<{ options: Option[]; modelValue: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const normalized = computed(() =>
  props.options.map((o) => (typeof o === 'string' ? { value: o, label: o } : o)),
)
</script>

<template>
  <div class="segmented">
    <button
      v-for="o in normalized"
      :key="o.value"
      :class="['segmented__item', modelValue === o.value && 'segmented__item--active']"
      @click="emit('update:modelValue', o.value)"
    >
      {{ o.label }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

type Tab = string | { value: string; label: string; count?: number }

const props = defineProps<{ tabs: Tab[]; modelValue: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const normalized = computed(() =>
  props.tabs.map((t) => (typeof t === 'string' ? { value: t, label: t, count: undefined } : t)),
)
</script>

<template>
  <div class="tabs">
    <button
      v-for="t in normalized"
      :key="t.value"
      :class="['tab', modelValue === t.value && 'tab--active']"
      @click="emit('update:modelValue', t.value)"
    >
      {{ t.label }}<span v-if="t.count != null" class="tab__count">{{ t.count }}</span>
    </button>
  </div>
</template>

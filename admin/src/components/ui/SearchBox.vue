<script setup lang="ts">
import Icon from './Icon.vue'

defineOptions({ inheritAttrs: false })

withDefaults(defineProps<{ modelValue?: string; placeholder?: string }>(), {
  placeholder: 'Search…',
})

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
</script>

<template>
  <div class="searchbox">
    <Icon name="search" :size="15" />
    <input
      :value="modelValue"
      :placeholder="placeholder"
      v-bind="$attrs"
      @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    />
  </div>
</template>

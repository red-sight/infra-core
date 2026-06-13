<script setup lang="ts">
import { computed } from 'vue'
import Icon from './Icon.vue'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    modelValue?: string | number
    icon?: string
    error?: string | null
    type?: string
  }>(),
  { type: 'text' },
)

const emit = defineEmits<{ 'update:modelValue': [value: string | number] }>()

const classes = computed(() => ['input', props.icon && 'input--icon', props.error && 'input--err'])

function onInput(e: Event) {
  const target = e.target as HTMLInputElement
  emit('update:modelValue', props.type === 'number' ? target.valueAsNumber || 0 : target.value)
}
</script>

<template>
  <div v-if="icon" class="input-wrap">
    <span class="input-wrap__icon"><Icon :name="icon" :size="15" /></span>
    <input :class="classes" :type="type" :value="modelValue" v-bind="$attrs" @input="onInput" />
  </div>
  <input v-else :class="classes" :type="type" :value="modelValue" v-bind="$attrs" @input="onInput" />
</template>

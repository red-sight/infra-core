<script setup lang="ts">
import Icon from './Icon.vue'

defineProps<{
  label?: string
  required?: boolean
  hint?: string
  error?: string | null
  htmlFor?: string
}>()
</script>

<template>
  <div class="field">
    <label v-if="label" class="field__label" :for="htmlFor">
      {{ label }}<span v-if="required" class="field__req">*</span>
    </label>
    <slot />
    <span v-if="error" class="field__err"><Icon name="alert-circle" :size="12" />{{ error }}</span>
    <span v-else-if="hint" class="field__hint">{{ hint }}</span>
  </div>
</template>

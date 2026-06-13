<script setup lang="ts">
import { computed } from 'vue'
import Icon from './Icon.vue'

defineOptions({ inheritAttrs: false })

const props = defineProps<{ icon?: string; danger?: boolean }>()
const classes = computed(() => ['menu__item', props.danger && 'menu__item--danger'])
</script>

<template>
  <button type="button" :class="classes" v-bind="$attrs">
    <Icon v-if="icon" :name="icon" :size="15" />
    <slot />
  </button>
</template>

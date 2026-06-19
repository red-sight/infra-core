<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router'
import { useRouter } from 'vue-router'
import Icon from './Icon.vue'

export interface Crumb {
  label: string
  // Clickable when set and not the last crumb; the final crumb is always the
  // current page and renders as static text.
  to?: RouteLocationRaw
}

const props = defineProps<{ items: Crumb[] }>()
const router = useRouter()

function isLink(c: Crumb, i: number) {
  return c.to != null && i < props.items.length - 1
}
</script>

<template>
  <nav class="breadcrumb" aria-label="Breadcrumb">
    <template v-for="(c, i) in items" :key="i">
      <Icon v-if="i > 0" name="chevron-right" :size="13" />
      <button v-if="isLink(c, i)" type="button" @click="router.push(c.to!)">{{ c.label }}</button>
      <span v-else class="breadcrumb__current" :aria-current="i === items.length - 1 ? 'page' : undefined">
        {{ c.label }}
      </span>
    </template>
  </nav>
</template>

<style scoped>
/* Base color and hover come from the global .breadcrumb rules; reset the native
   button chrome so crumbs read as inline text links. */
.breadcrumb button {
  background: none;
  border: 0;
  padding: 0;
  margin: 0;
  font: inherit;
  color: inherit;
  cursor: pointer;
}
.breadcrumb__current {
  color: var(--foreground);
  font-weight: 500;
  max-width: 42ch;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>

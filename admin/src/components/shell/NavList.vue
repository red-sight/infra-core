<script setup lang="ts">
import Icon from '@/components/ui/Icon.vue'
import { NAV, NAV_GROUPS } from './nav'

defineProps<{ current: string }>()
defineEmits<{ navigate: [id: string] }>()
</script>

<template>
  <nav class="sidebar__nav">
    <div v-for="group in NAV_GROUPS" :key="group" class="side-group">
      <div class="side-group__label">{{ group }}</div>
      <button
        v-for="n in NAV.filter((x) => x.group === group)"
        :key="n.id"
        :class="['side-link', current === n.id && 'side-link--active']"
        @click="$emit('navigate', n.id)"
      >
        <Icon :name="n.icon" :size="17" />
        <span>{{ n.label }}</span>
      </button>
    </div>
  </nav>
</template>

<script setup lang="ts">
import Logo from './Logo.vue'
import NavList from './NavList.vue'
import UserMenu from './UserMenu.vue'
import { useNav } from '@/composables/useNav'

defineProps<{ current: string }>()
const nav = useNav()
</script>

<template>
  <aside class="sidebar">
    <div class="sidebar__head"><Logo @click="nav('overview')" /></div>
    <NavList :current="current" @navigate="nav($event)" />
    <div class="sidebar__foot"><UserMenu :width="216" align="start" up /></div>
  </aside>
</template>

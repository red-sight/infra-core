<script setup lang="ts">
import Icon from '@/components/ui/Icon.vue'
import Logo from './Logo.vue'
import NavList from './NavList.vue'
import UserMenu from './UserMenu.vue'
import { useNav } from '@/composables/useNav'

defineProps<{ open: boolean; current: string }>()
const emit = defineEmits<{ close: [] }>()
const nav = useNav()

function go(id: string) {
  nav(id)
  emit('close')
}
</script>

<template>
  <Teleport to="body">
    <template v-if="open">
      <div class="overlay" style="z-index: 60" @click="emit('close')" />
      <div class="mobile-drawer">
        <div class="sidebar__head" style="justify-content: space-between">
          <Logo @click="go('overview')" />
          <button class="icon-x" @click="emit('close')"><Icon name="x" :size="18" /></button>
        </div>
        <NavList :current="current" @navigate="go" />
        <div class="sidebar__foot"><UserMenu up @click="emit('close')" /></div>
      </div>
    </template>
  </Teleport>
</template>

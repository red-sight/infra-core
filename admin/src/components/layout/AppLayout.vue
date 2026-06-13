<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useLogto } from '@logto/vue'
import { useRoute } from 'vue-router'
import Sidebar from '@/components/shell/Sidebar.vue'
import TopBar from '@/components/shell/TopBar.vue'
import MobileNav from '@/components/shell/MobileNav.vue'
import CommandPalette from '@/components/shell/CommandPalette.vue'
import ToastRegion from '@/components/ui/ToastRegion.vue'

const { isAuthenticated, isLoading, signIn } = useLogto()
const route = useRoute()

const current = computed(() => (route.name as string) ?? 'overview')

const cmdOpen = ref(false)
const mobileOpen = ref(false)

watch(
  [isLoading, isAuthenticated],
  ([loading, auth]) => {
    if (!loading && !auth) signIn(import.meta.env.VITE_REDIRECT_URI)
  },
  { immediate: true },
)

function onKey(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    cmdOpen.value = !cmdOpen.value
  }
}
onMounted(() => document.addEventListener('keydown', onKey))
onBeforeUnmount(() => document.removeEventListener('keydown', onKey))
</script>

<template>
  <template v-if="isAuthenticated">
    <div class="app-shell">
      <Sidebar :current="current" />
      <div class="app-main">
        <TopBar :current="current" @open-cmd="cmdOpen = true" @open-mobile="mobileOpen = true" />
        <main :key="current">
          <RouterView />
        </main>
      </div>
    </div>
    <MobileNav :open="mobileOpen" :current="current" @close="mobileOpen = false" />
    <CommandPalette :open="cmdOpen" @close="cmdOpen = false" />
    <ToastRegion />
  </template>

  <div v-else-if="isLoading" style="display: flex; height: 100vh; align-items: center; justify-content: center">
    <div
      style="width: 24px; height: 24px; border-radius: 999px; border: 2px solid var(--border); border-top-color: var(--foreground); animation: spin 0.8s linear infinite"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useAuth } from '@/composables/useAuth'
import { useRoute } from 'vue-router'
import Sidebar from '@/components/shell/Sidebar.vue'
import TopBar from '@/components/shell/TopBar.vue'
import MobileNav from '@/components/shell/MobileNav.vue'
import CommandPalette from '@/components/shell/CommandPalette.vue'
import ToastRegion from '@/components/ui/ToastRegion.vue'

const { isAuthenticated, isLoading, signIn } = useAuth()
const route = useRoute()

// Sidebar highlight, topbar title, and the <main> remount key all key off the nav
// section (a screen id). Detail routes carry meta.section to map back to their
// top-level item; for nav screens the route name already equals the screen id.
// Keying <main> by section (not the leaf route name) keeps the detail page mounted
// across its tab routes instead of remounting on every tab switch.
const section = computed(() => (route.meta.section as string) ?? (route.name as string) ?? 'overview')

const cmdOpen = ref(false)
const mobileOpen = ref(false)

watch(
  [isLoading, isAuthenticated],
  ([loading, auth]) => {
    if (!loading && !auth) signIn()
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
      <Sidebar :current="section" />
      <div class="app-main">
        <TopBar :current="section" @open-cmd="cmdOpen = true" @open-mobile="mobileOpen = true" />
        <main :key="section">
          <RouterView />
        </main>
      </div>
    </div>
    <MobileNav :open="mobileOpen" :current="section" @close="mobileOpen = false" />
    <CommandPalette :open="cmdOpen" @close="cmdOpen = false" />
    <ToastRegion />
  </template>

  <div v-else-if="isLoading" style="display: flex; height: 100vh; align-items: center; justify-content: center">
    <div
      style="width: 24px; height: 24px; border-radius: 999px; border: 2px solid var(--border); border-top-color: var(--foreground); animation: spin 0.8s linear infinite"
    />
  </div>
</template>

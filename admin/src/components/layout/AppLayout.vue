<script setup lang="ts">
import { useLogto } from '@logto/vue'
import { watch } from 'vue'
import AppHeader from './AppHeader.vue'

const { isAuthenticated, isLoading, signIn } = useLogto()

watch(
  [isLoading, isAuthenticated],
  ([loading, auth]) => {
    if (!loading && !auth) {
      signIn(import.meta.env.VITE_REDIRECT_URI)
    }
  },
  { immediate: true },
)
</script>

<template>
  <template v-if="isAuthenticated">
    <AppHeader />
    <main class="container py-6">
      <RouterView />
    </main>
  </template>

  <div v-else-if="isLoading" class="flex h-screen items-center justify-center">
    <div class="size-6 animate-spin rounded-full border-2 border-muted border-t-foreground" />
  </div>
</template>

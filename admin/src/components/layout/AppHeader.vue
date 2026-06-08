<script setup lang="ts">
import { useLogto } from '@logto/vue'
import { useQuery } from '@tanstack/vue-query'
import ThemeToggle from '../ThemeToggle.vue'

const { signOut, fetchUserInfo } = useLogto()

const { data: user } = useQuery({
  queryKey: ['user-info'],
  queryFn: () => fetchUserInfo(),
  staleTime: 5 * 60 * 1000,
})

function handleSignOut() {
  signOut(window.location.origin)
}
</script>

<template>
  <header class="sticky top-0 z-50 border-b bg-background/80 backdrop-blur">
    <div class="container flex h-14 items-center justify-between">
      <span class="text-sm font-semibold tracking-tight">Admin</span>

      <div class="flex items-center gap-2">
        <ThemeToggle />

        <div class="flex items-center gap-2 pl-2 border-l">
          <span v-if="user" class="text-sm text-muted-foreground">
            {{ user.name ?? user.email }}
          </span>
          <button
            type="button"
            class="inline-flex items-center gap-1.5 rounded-md px-2 py-1.5 text-sm text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
            @click="handleSignOut"
          >
            <ILucideLogOut class="size-4" />
            <span class="sr-only">Sign out</span>
          </button>
        </div>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { useTweaks } from '@/composables/useTweaks'

// Start the tweaks controller — applies persisted theme / density / accent to
// <html> and keeps it in sync.
useTweaks()
</script>

<template>
  <RouterView />
</template>

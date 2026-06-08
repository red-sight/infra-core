<script setup lang="ts">
import { useColorMode } from '@vueuse/core'

// Initialize color mode — applies .dark class to <html> based on stored preference or system setting.
useColorMode()
</script>

<template>
  <RouterView />
</template>

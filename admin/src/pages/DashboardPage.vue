<script setup lang="ts">
import { useLogto } from '@logto/vue'
import { ref, onMounted } from 'vue'

const { getIdTokenClaims } = useLogto()

const name = ref<string | null>(null)
const email = ref<string | null>(null)

onMounted(async () => {
  const claims = await getIdTokenClaims()
  name.value = claims?.name ?? claims?.username ?? null
  email.value = claims?.email ?? null
})
</script>

<template>
  <div class="flex flex-col gap-1">
    <h1 class="text-2xl font-semibold tracking-tight">
      Welcome{{ name ? `, ${name}` : '' }}
    </h1>
    <p v-if="email" class="text-sm text-muted-foreground">{{ email }}</p>
  </div>
</template>

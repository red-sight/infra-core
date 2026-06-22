<script setup lang="ts">
import { useAuth } from '@/composables/useAuth'
import { useRouter } from 'vue-router'
import { onMounted, ref } from 'vue'

const { handleSignInCallback } = useAuth()
const router = useRouter()

const step = ref('mounted')
const error = ref<string | null>(null)

onMounted(async () => {
  step.value = 'calling handleSignInCallback'
  try {
    await handleSignInCallback()
    step.value = 'callback done, navigating…'
    await router.replace('/')
  } catch (e) {
    error.value = e instanceof Error ? `${e.name}: ${e.message}` : String(e)
    step.value = 'error'
  }
})
</script>

<template>
  <div style="display:flex;flex-direction:column;align-items:center;justify-content:center;height:100vh;gap:16px;font-family:monospace;font-size:13px">
    <div v-if="error" style="color:red;max-width:600px;word-break:break-all">
      <strong>Error:</strong> {{ error }}
      <br><a href="/" style="color:blue">Go home</a>
    </div>
    <template v-else>
      <div style="width:24px;height:24px;border:2px solid #ccc;border-top-color:#555;border-radius:50%;animation:spin 0.8s linear infinite" />
      <div style="color:#888">{{ step }}</div>
    </template>
  </div>
</template>

<style>
@keyframes spin { to { transform: rotate(360deg) } }
</style>

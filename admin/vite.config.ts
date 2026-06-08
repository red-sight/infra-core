import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import Icons from 'unplugin-icons/vite'
import Components from 'unplugin-vue-components/vite'
import IconsResolver from 'unplugin-icons/resolver'
import { fileURLToPath } from 'url'

export default defineConfig({
  plugins: [
    vue(),
    Icons({ compiler: 'vue3' }),
    Components({
      resolvers: [IconsResolver()],
      dts: false,
    }),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    host: true,
    port: 5173,
    hmr: {
      overlay: false,
    },
  },
  optimizeDeps: {
    include: [
      '@logto/vue',
      '@tanstack/vue-query',
      '@vueuse/core',
      'radix-vue',
    ],
  },
})

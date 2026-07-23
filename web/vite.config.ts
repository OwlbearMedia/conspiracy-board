import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    host: true, // reachable from the nginx container
    port: 5173,
    strictPort: true,
  },
})

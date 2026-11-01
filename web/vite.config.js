import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// Inside the dev container the backend is on localhost:8097 (JFSHARE_PORT).
// When running Vite on the host against the dev stack, set
// VITE_API_TARGET=http://localhost:8098 instead.
const backendTarget =
  process.env.VITE_API_TARGET || `http://localhost:${process.env.JFSHARE_PORT || 8097}`

export default defineConfig({
  plugins: [svelte()],
  build: {
    outDir: 'dist',
    emptyOutDir: true
  },
  server: {
    host: true,
    proxy: {
      '/api': backendTarget
    }
  }
})

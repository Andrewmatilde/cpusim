import path from "path"
import tailwindcss from "@tailwindcss/vite"
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    proxy: {
      '/config': {
        target: 'http://localhost:9090',
        changeOrigin: true,
      },
      '/status': {
        target: 'http://localhost:9090',
        changeOrigin: true,
      },
      '/health': {
        target: 'http://localhost:9090',
        changeOrigin: true,
      },
      '/experiments': {
        target: 'http://localhost:9090',
        changeOrigin: true,
      },
    },
  },
})

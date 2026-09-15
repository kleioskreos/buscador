import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 5173,
    // En Docker el backend se alcanza por nombre de servicio.
    // En desarrollo local, usar el puerto mapeado.
    proxy: {
      '/api': {
        target: process.env.VITE_API_URL || 'http://backend:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
    chunkSizeWarningLimit: 1000,
  },
})
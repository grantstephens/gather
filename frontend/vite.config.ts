import { defineConfig } from 'vite'
import preact from '@preact/preset-vite'

export default defineConfig({
  plugins: [preact()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules/leaflet')) return 'vendor-leaflet'
          if (id.includes('node_modules/marked') || id.includes('node_modules/dompurify')) return 'vendor-markdown'
          if (id.includes('node_modules/date-fns')) return 'vendor-date-fns'
        },
      },
    },
  },
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:8090',
      '/_': 'http://127.0.0.1:8090',
    },
  },
})

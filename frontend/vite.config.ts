import path from 'path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { '@': path.resolve(__dirname, './src') },
  },
  server: {
    port: 3000,
    proxy: {
      '/auth': 'http://localhost:8085',
      '/wallets': 'http://localhost:8085',
      '/deposits': 'http://localhost:8085',
      '/conversions': 'http://localhost:8085',
      '/institutions': 'http://localhost:8085',
      '/payouts': 'http://localhost:8085',
    },
  },
})

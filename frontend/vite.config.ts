import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// In development the API (and the mock payment provider page) run on another port;
// the dev server proxies them so the browser sees a single origin, like nginx in Docker.
const target = process.env.VITE_API_TARGET || 'http://localhost:18080';

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    strictPort: true,
    proxy: {
      '/api': { target, changeOrigin: true },
      '/mock-provider': { target, changeOrigin: true },
    },
  },
});

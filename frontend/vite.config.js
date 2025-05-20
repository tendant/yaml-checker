import { defineConfig } from 'vite';
import solidPlugin from 'vite-plugin-solid';

export default defineConfig({
  plugins: [solidPlugin()],
  server: {
    port: 3000, // Development server port
    proxy: {
      '/api': {
        target: 'http://localhost:4000', // Proxy backend API requests to the Go server
        changeOrigin: true,
        secure: false,
      },
    },
  },
  build: {
    target: 'esnext',
    outDir: 'dist',
  },
});

import vue from '@vitejs/plugin-vue';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [vue()],
  base: '/',
  build: {
    assetsDir: 'bitapi-assets'
  },
  server: {
    port: 5181,
    allowedHosts: true,
    proxy: {
      '/api': 'http://127.0.0.1:8091',
      '/v1': 'http://127.0.0.1:8091',
      '/responses': 'http://127.0.0.1:8091',
      '/uploads': 'http://127.0.0.1:8091',
      '/health': 'http://127.0.0.1:8091'
    }
  },
  preview: {
    port: 5181,
    host: '0.0.0.0',
    allowedHosts: true,
    proxy: {
      '/api': 'http://127.0.0.1:8091',
      '/v1': 'http://127.0.0.1:8091',
      '/responses': 'http://127.0.0.1:8091',
      '/uploads': 'http://127.0.0.1:8091',
      '/health': 'http://127.0.0.1:8091'
    }
  }
});

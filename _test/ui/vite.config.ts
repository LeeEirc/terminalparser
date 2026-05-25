import { fileURLToPath, URL } from 'node:url'
import type { ConfigEnv, UserConfig } from 'vite';
import process from 'node:process';
import { resolve } from 'node:path';
import vue from '@vitejs/plugin-vue'
import vueJsx from '@vitejs/plugin-vue-jsx'
import vueDevTools from 'vite-plugin-vue-devtools'
import tailwindcss from '@tailwindcss/vite';
import { defineConfig, loadEnv } from 'vite';

// https://vite.dev/config/
export default defineConfig(({ mode }: ConfigEnv): UserConfig => {
  const root = process.cwd();
  const env = loadEnv(mode, root);
return {
  plugins: [
    vue(),
    vueJsx(),
    vueDevTools(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
  },
  base: env.VITE_PUBLIC_PATH,
  server: {
    port: 9540,
    proxy: {
      '^/ws/ssh/': {
        target: env.VITE_KOKO_WS_URL,
        ws: true,
        changeOrigin: true,
      },
      '^/api/ssh/': {
        target: env.VITE_KOKO_API_URL,
        changeOrigin: true,
      },
    },
  },
}})

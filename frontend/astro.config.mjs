import { defineConfig } from 'astro/config';
import react from '@astrojs/react';
import tailwind from '@astrojs/tailwind';

export default defineConfig({
  output: 'static',
  outDir: '../internal/web/dist',
  integrations: [
    react(),
    tailwind({ applyBaseStyles: false })
  ],
  vite: {
    server: {
      proxy: {
        '/api': 'http://localhost:8000',
        '/events': 'http://localhost:8000',
        '/healthz': 'http://localhost:8000',
        '/sessions': 'http://localhost:8000',
        '/projects': 'http://localhost:8000',
        '/stats': 'http://localhost:8000',
        '/analytics': 'http://localhost:8000',
        '/pricing': 'http://localhost:8000',
        '/budgets': 'http://localhost:8000',
        '/notifications': 'http://localhost:8000',
        '/config': 'http://localhost:8000',
        '/artifacts': 'http://localhost:8000'
      }
    }
  }
});

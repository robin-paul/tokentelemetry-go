/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{astro,html,js,jsx,md,mdx,svelte,ts,tsx,vue}'],
  darkMode: ['class', '[data-theme="dark"]'],
  theme: {
    extend: {
      colors: {
        canvas: 'var(--tt-canvas)',
        sunken: 'var(--tt-sunken)',
        panel: 'var(--tt-panel)',
        raised: 'var(--tt-raised)',
        overlay: 'var(--tt-overlay)',
        border: 'var(--tt-border)',
        borderStrong: 'var(--tt-border-strong)',
        agent: {
          claude: 'var(--agent-claude)',
          codex: 'var(--agent-codex)',
          gemini: 'var(--agent-gemini)',
          antigravity: 'var(--agent-antigravity)',
          qwen: 'var(--agent-qwen)',
          cursor: 'var(--agent-cursor)',
          copilot: 'var(--agent-copilot)',
          opencode: 'var(--agent-opencode)',
          grok: 'var(--agent-grok)',
          pi: 'var(--agent-pi)',
          cline: 'var(--agent-cline)',
          muse: 'var(--agent-muse)',
          prime: 'var(--agent-prime)',
          dsh: 'var(--agent-dsh)',
        }
      }
    },
  },
  plugins: [],
}

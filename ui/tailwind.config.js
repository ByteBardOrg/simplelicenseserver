/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './index.html',
    './src/**/*.{vue,js,ts,jsx,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        surface: {
          DEFAULT: '#0f131d',
          lowest: '#0a0e18',
          low: '#171b26',
          container: '#1c1f2a',
          high: '#262a35',
          highest: '#31353f',
        },
        ink: {
          DEFAULT: '#dfe2f1',
          strong: '#ffffff',
          soft: '#c6ccdc',
          muted: '#8f98ad',
          faint: '#6f7a91',
        },
        primary: {
          DEFAULT: '#a4c9ff',
          container: '#60a5fa',
          on: '#06142c',
        },
        success: '#4ae176',
        warning: '#f2b36d',
        critical: '#ff7a8a',
        outline: '#414751',
      },
      fontFamily: {
        sans: ['"Inter Variable"', 'Inter', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        label: ['"Space Grotesk Variable"', '"Space Grotesk"', '"Inter Variable"', 'ui-sans-serif', 'sans-serif'],
        mono: ['"JetBrains Mono Variable"', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'monospace'],
      },
      boxShadow: {
        ghost: 'inset 0 0 0 1px rgba(65, 71, 81, 0.28)',
        float: '0 24px 64px -12px rgba(2, 6, 16, 0.65), inset 0 0 0 1px rgba(65, 71, 81, 0.28)',
        glow: '0 0 0 1px rgba(164, 201, 255, 0.35), 0 8px 28px -6px rgba(96, 165, 250, 0.45)',
      },
    },
  },
  plugins: [],
}

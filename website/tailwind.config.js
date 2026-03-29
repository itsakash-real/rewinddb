/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
      },
      colors: {
        bg: '#0a0a0a',
        surface: '#111111',
        'surface-hover': '#161616',
        border: '#1a1a1a',
        'border-hover': '#2a2a2a',
        accent: {
          DEFAULT: '#7dd3fc',
          hover: '#38bdf8',
          dim: 'rgba(125, 211, 252, 0.08)',
        },
        text: {
          DEFAULT: '#f0f0f0',
          secondary: '#b0b0b0',
          muted: '#94a3b8',
        },
        success: '#4ade80',
        warning: '#fbbf24',
        error: '#f87171',
      },
      boxShadow: {
        'glow-sm': '0 0 16px -2px rgba(56, 189, 248, 0.25)',
        'glow-md': '0 0 24px -4px rgba(56, 189, 248, 0.35)',
        'glow-lg': '0 0 40px -8px rgba(56, 189, 248, 0.5)',
        'glow-button': '0 0 20px -5px rgba(56, 189, 248, 0.4), inset 0 1px 0 0 rgba(255, 255, 255, 0.1)',
      },
      maxWidth: {
        content: '1000px',
      },
      animation: {
        blink: 'blink 1s step-end infinite',
        'marquee-left': 'marqueeLeft 60s linear infinite',
        'marquee-right': 'marqueeRight 60s linear infinite',
      },
      keyframes: {
        blink: {
          '0%, 100%': { opacity: 1 },
          '50%': { opacity: 0 },
        },
        marqueeLeft: {
          '0%': { transform: 'translateX(0)' },
          '100%': { transform: 'translateX(-50%)' },
        },
        marqueeRight: {
          '0%': { transform: 'translateX(-50%)' },
          '100%': { transform: 'translateX(0)' },
        },
      },
    },
  },
  plugins: [],
}

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
        bg: '#050508',
        surface: '#0e0e12',
        border: '#1c1c24',
        'border-light': '#2a2a36',
        purple: {
          DEFAULT: '#a78bfa',
          dim: '#7c3aed',
          bright: '#c4b5fd',
          glow: '#8b5cf6',
        },
        cyan: { DEFAULT: '#67e8f9', dim: '#0891b2' },
        green: { DEFAULT: '#4ade80', dim: '#16a34a' },
        red: { DEFAULT: '#f87171' },
        text: {
          DEFAULT: '#e8e8f0',
          muted: '#6b6b80',
          dim: '#3a3a50',
        },
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'hero-glow':
          'radial-gradient(ellipse 80% 50% at 50% -20%, rgba(139,92,246,0.15), transparent)',
        'purple-glow':
          'radial-gradient(ellipse 60% 40% at 50% 50%, rgba(139,92,246,0.08), transparent)',
      },
      animation: {
        'fade-up': 'fadeUp 0.6s ease forwards',
        blink: 'blink 1s step-end infinite',
        'gradient-x': 'gradientX 4s ease infinite',
        float: 'float 6s ease-in-out infinite',
      },
      keyframes: {
        fadeUp: {
          from: { opacity: 0, transform: 'translateY(24px)' },
          to: { opacity: 1, transform: 'translateY(0)' },
        },
        blink: {
          '0%, 100%': { opacity: 1 },
          '50%': { opacity: 0 },
        },
        gradientX: {
          '0%, 100%': { backgroundPosition: '0% 50%' },
          '50%': { backgroundPosition: '100% 50%' },
        },
        float: {
          '0%, 100%': { transform: 'translateY(0)' },
          '50%': { transform: 'translateY(-8px)' },
        },
      },
    },
  },
  plugins: [],
}

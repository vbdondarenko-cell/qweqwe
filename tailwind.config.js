/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        background: '#050506',
        surface: '#0D0E10',
        elevated: '#141518',
        zone: '#1A1C20',
        red: {
          DEFAULT: '#FF2D35',
          signal: '#FF3B42',
          deep: '#9F171E',
        },
        critical: '#EF4444',
        success: '#22C55E',
        warning: '#F59E0B',
        info: '#3B82F6',
        text: {
          primary: '#F7F8FA',
          dimmed: '#A5A9B0',
          muted: '#747982',
        },
        border: {
          DEFAULT: '#24262B',
        },
      },
      fontFamily: {
        display: ['Outfit', 'sans-serif'],
        body: ['Inter', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
      animation: {
        'pulse-ring': 'pulseRing 2s ease-out infinite',
        'fade-slide-up': 'fadeSlideUp 0.4s ease-out forwards',
        'fade-in': 'fadeIn 0.3s ease-out forwards',
        'scale-in': 'scaleIn 0.25s cubic-bezier(0.34, 1.56, 0.64, 1) forwards',
        'slide-up-sheet': 'slideUpSheet 0.35s cubic-bezier(0.16, 1, 0.3, 1) forwards',
        'bpm-beat': 'bpmBeat 1.2s ease-in-out infinite',
        'count-up': 'countUp 0.3s ease-out',
        'shimmer': 'shimmer 1.8s linear infinite',
        'glow-pulse': 'glowPulse 2.5s ease-in-out infinite',
      },
      keyframes: {
        pulseRing: {
          '0%': { transform: 'scale(0.85)', opacity: '0.8' },
          '100%': { transform: 'scale(1.6)', opacity: '0' },
        },
        fadeSlideUp: {
          '0%': { opacity: '0', transform: 'translateY(12px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.9)' },
          '100%': { opacity: '1', transform: 'scale(1)' },
        },
        slideUpSheet: {
          '0%': { transform: 'translateY(100%)' },
          '100%': { transform: 'translateY(0)' },
        },
        bpmBeat: {
          '0%, 100%': { transform: 'scale(1)', opacity: '1' },
          '50%': { transform: 'scale(1.3)', opacity: '0.7' },
        },
        countUp: {
          '0%': { transform: 'translateY(8px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' },
        },
        glowPulse: {
          '0%, 100%': { boxShadow: '0 0 12px rgba(255, 45, 53, 0.4)' },
          '50%': { boxShadow: '0 0 24px rgba(255, 45, 53, 0.7)' },
        },
      },
    },
  },
  plugins: [],
};

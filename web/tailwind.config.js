/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{html,js,svelte,ts}'],
  theme: {
    extend: {
      colors: {
        bg:      '#0f1117',
        surface: '#1a1d27',
        border:  '#2a2d3e',
        muted:   '#4a5068',
        text:    '#e2e8f0',
        dim:     '#8892a4',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      }
    }
  },
  plugins: []
};

/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{astro,html,js,ts,md,mdx}'],
  darkMode: ['class', '[data-theme="dark"]'],
  theme: {
    extend: {
      colors: {
        // Bold, bright tomato brand palette.
        tomato: {
          50: '#fff1ef',
          100: '#ffe0db',
          200: '#ffc5bd',
          300: '#ff9c8d',
          400: '#ff6a54',
          500: '#f5402a', // primary accent
          600: '#e02412',
          700: '#bd1a0c',
          800: '#9c1a10',
          900: '#7f1c15',
        },
        leaf: {
          400: '#4caf50',
          500: '#3d9142',
          600: '#2f7434',
        },
      },
      fontFamily: {
        // JetBrains Mono — house display face for big bold headings.
        display: ['"JetBrains Mono"', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'monospace'],
        // Body copy: system stack (self-hosting only the display face).
        sans: [
          'system-ui', '-apple-system', 'BlinkMacSystemFont', '"Segoe UI"',
          'Roboto', 'Helvetica', 'Arial', 'sans-serif',
        ],
        mono: ['"JetBrains Mono"', 'ui-monospace', 'SFMono-Regular', 'Menlo', 'monospace'],
      },
      maxWidth: {
        content: '1400px', // §3 desktop content max-width
      },
      // The two blocks below keep Tailwind 3 defaults that Tailwind 4 changed.
      // Breakpoints in px: Tailwind 4 defines them in rem, which moves every
      // breakpoint for a visitor whose browser font size is not 16px.
      screens: { sm: '640px', md: '768px', lg: '1024px', xl: '1280px', '2xl': '1536px' },
      // Line heights in rem: Tailwind 4 pairs each text size with a unitless
      // ratio, which nested text of another size (the <code> in .doc pre)
      // inherits as a ratio, and which rounds differently off a 16px root.
      fontSize: {
        xs: ['0.75rem', { lineHeight: '1rem' }],
        sm: ['0.875rem', { lineHeight: '1.25rem' }],
        base: ['1rem', { lineHeight: '1.5rem' }],
        lg: ['1.125rem', { lineHeight: '1.75rem' }],
        xl: ['1.25rem', { lineHeight: '1.75rem' }],
        '2xl': ['1.5rem', { lineHeight: '2rem' }],
        '3xl': ['1.875rem', { lineHeight: '2.25rem' }],
        '4xl': ['2.25rem', { lineHeight: '2.5rem' }],
      },
    },
  },
  plugins: [],
};

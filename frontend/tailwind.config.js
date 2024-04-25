const defaultTheme = require('tailwindcss/defaultTheme');

module.exports = {
  content: ['./src/**/*.{js,jsx,ts,tsx}'],
  theme: {
    extend: {
      colors: {
        alabaster: '#f8f8f8',
        amaranth: '#ef2e55',
        amaranthTransparent: '#ef2e5520',
        athens: '#e5e7eb',
        athensTransparent: '#e5e7eb20',
        boulder: '#7a7a7a',
        curiousBlue: '#149deb',
        curiousBlueTransparent: '#149deb20',
        flushOrange: '#FC8603',
        flushOrangeTransparent: '#FC860320',
        mountainMeadow: '#23d160',
        mountainMeadowTransparent: '#23d16020',
        paleSky: '#6b7280',
        primary: '#244365',
        tundora: '#4a4a4a',
        tundoraTransparent: '#4a4a4a20'
      },
      flex: {
        2: '2 2 0%'
      },
      fontFamily: {
        sans: ['Inter', ...defaultTheme.fontFamily.sans],
        mono: ['Roboto Mono', 'mono']
      }
    }
  },
  plugins: [require('@tailwindcss/forms'), require('@tailwindcss/line-clamp')],
  safelist: [
    {
      pattern: /(bg|border)-(amaranth|athens|curiousBlue|flushOrange|mountainMeadow|tundora)(|Transparent)/
    }
  ]
};

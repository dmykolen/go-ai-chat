const plugin = require('tailwindcss/plugin');
const colors = require('tailwindcss/colors');

/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./static/**/*.{html,js,go}", "./views/**/*.{html,js,go}"],
  presets: [process.env.NODE_ENV === 'development' ? require('./tailwind.config.DEV.js') : [], 'node_modules/tailwindcss/stubs/defaultConfig.stub.js'],
  // presets: [process.env.NODE_ENV === 'development' ? ['./tailwind.config.DEV.js'] : './tailwind.config.DEV.js'],
  // presets: [require('./tailwind.config.DEV.js')],
  theme: {
    extend: {
      Audio: ["hover", "focus"],
      backgroundImage: {
        "robot-bg": "url('./static/img/robot_image_3.png')",
      },
      colors: {
        blue: {
          100: "#CCF2FF",
          200: "#99E5FF",
          300: "#66D9FF",
          400: "#33CCFF",
          500: "#00BFFF",
          600: "#009FCC",
          700: "#007F99",
          800: "#005F66",
          900: "#003F33",
        },
        yellow: {
          100: "#FFFFCC",
          200: "#FFFF99",
          300: "#FFFF66",
          400: "#FFFF33",
          500: "#FFFF00",
          600: "#CCCC00",
          700: "#999900",
          800: "#666600",
          900: "#333300",
        },
      },
      gradientColorStops: {
        "ukraine-blue": {
          start: "#00BFFF",
          end: "#003F33",
        },
        "ukraine-yellow": {
          start: "#FFFF00",
          end: "#333300",
        },
      },
      backgroundImage: (theme) => ({
        "blue-gradient": `linear-gradient(to right, ${theme("gradientColorStops.ukraine-blue.start")}, ${theme("gradientColorStops.ukraine-blue.end")})`,
        "yellow-gradient": `linear-gradient(to right, ${theme("gradientColorStops.ukraine-yellow.start")}, ${theme("gradientColorStops.ukraine-yellow.end")})`,
      }),
      animation: {
        'gradient-x': 'gradient-x 3s ease infinite',
        'gradient-y': 'gradient-y 3s ease infinite',
        'gradient-xy': 'gradient-xy 3s ease infinite',
      },
      keyframes: {
        'gradient-x': {
          '0%, 100%': { backgroundPosition: 'left center' },
          '50%': { backgroundPosition: 'right center' },
        },
        'gradient-y': {
          '0%, 100%': { backgroundPosition: 'top center' },
          '50%': { backgroundPosition: 'bottom center' },
        },
        'gradient-xy': {
          '0%, 100%': { backgroundPosition: 'left top' },
          '50%': { backgroundPosition: 'right bottom' },
        },
      },
      boxShadow: {
        'neon-blue': '0 0 15px rgba(0, 191, 255, 0.5)',
        'neon-yellow': '0 0 15px rgba(255, 255, 0, 0.5)',
        '3d': '0 4px #999',
      },
    },
    container: {
      center: true,
      padding: "1rem",
    },
  },
  daisyui: {
    themes: [
      {
        ua: {
          primary: '#00BFFF',
          secondary: '#FFFF00',
          accent: '#FFDD00',
          neutral: '#2A2E37',
          'base-100': '#121212', // Dark background
          'base-200': '#1E1E1E', // Slightly lighter dark background
          'base-300': '#292929', // Even lighter dark background
          'base-content': '#E0E0E0', // Light content for dark backgrounds
          info: '#2094F3',
          success: '#009485',
          warning: '#FF9900',
          error: '#FF5724',
          'primary-gradient': 'linear-gradient(to right, #00BFFF, #003F33)',
          'secondary-gradient': 'linear-gradient(to right, #FFFF00, #333300)',
          'primary-content': '#FFFFFF',
          'secondary-content': '#003F33',
          'accent-content': '#151505',
          // 'accent-content': '#2A2E37',
        },
      },
      "dark",
      // "bumblebee",
      "dim",
      "sunset",
      // "cupcake",
      // "cyberpunk",
      // "valentine",
      // "halloween",
      "forest",
      // "luxury",
    ],
  },
  plugins: [
    require("@tailwindcss/forms")({
      strategy: "class",
    }),
    require("@tailwindcss/typography"),
    require("daisyui"),
    plugin(function ({ addComponents, theme }) {
      const newComponents = {
        '.bg-primary-to-secondary': {
          backgroundImage: `linear-gradient(to bottom right, theme(colors.primary), theme(colors.secondary))`,
        },
      };

      addComponents(newComponents);
    }),
  ],
};

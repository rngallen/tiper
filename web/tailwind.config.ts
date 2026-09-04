import type { Config } from "tailwindcss";

const config: Config = {
  darkMode: "class",
  content: ["./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        // TIPER DFMS — navy #033860 / accent #dc295c
        brand: {
          50: "#eef4f8",
          100: "#d5e4ee",
          200: "#a9c9dd",
          300: "#74a6c4",
          400: "#4580a5",
          500: "#2c6489",
          600: "#1e4f70",
          700: "#16405c",
          800: "#033860",
          900: "#022a48",
        },
        accent: {
          DEFAULT: "#dc295c",
          dark: "#b81f4a",
        },
      },
      fontFamily: {
        sans: [
          "var(--font-sans)",
          "Source Sans 3",
          "ui-sans-serif",
          "system-ui",
          "sans-serif",
        ],
        display: [
          "var(--font-sans)",
          "Source Sans 3",
          "ui-sans-serif",
          "system-ui",
          "sans-serif",
        ],
      },
      keyframes: {
        "fade-up": {
          "0%": { opacity: "0", transform: "translateY(18px)" },
          "100%": { opacity: "1", transform: "translateY(0)" },
        },
        "fade-in": {
          "0%": { opacity: "0" },
          "100%": { opacity: "1" },
        },
        "ken-burns": {
          "0%": { transform: "scale(1)" },
          "100%": { transform: "scale(1.06)" },
        },
      },
      animation: {
        "fade-up": "fade-up 0.8s ease-out both",
        "fade-in": "fade-in 1s ease-out both",
        "ken-burns": "ken-burns 18s ease-out forwards",
      },
    },
  },
  plugins: [],
};

export default config;

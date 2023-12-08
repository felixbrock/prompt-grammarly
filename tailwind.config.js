/** @type {import('tailwindcss').Config} */
const defaultTheme = require("tailwindcss/defaultTheme");

module.exports = {
  content: ["./internal/components/**/*.templ"],
  theme: {
    extend: {
      fontFamily: {
        sans: ["Inter var", ...defaultTheme.fontFamily.sans],
      },
      maxWidth: {
        "1/2": "50%",
      },
      minWidth: {
        "1/4": "25%",
      },
      colors: {
        lemonaiMain: "#d65cf7",
      },
    },
  },
  plugins: [require("@tailwindcss/forms")],
};

import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Backend API URL for development proxy (configurable via environment variable)
const API_TARGET = process.env.VITE_API_URL || "http://localhost:8080";

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    open: true,
    proxy: {
      "/api": {
        target: API_TARGET,
        changeOrigin: true,
      },
    },
  },
  test: {
    globals: true,
    environment: "jsdom",
    setupFiles: "./src/test/setup.js",
    css: true,
  },
});

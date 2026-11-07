import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/matches": { target: "http://localhost:8080", ws: true },
      "/healthz": "http://localhost:8080",
    },
  },
  test: {
    include: ["src/**/*.{test,spec}.{ts,tsx}"],
    environment: "jsdom",
    setupFiles: ["./src/test-setup.ts"],
    css: true,
  },
});

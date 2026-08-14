import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  build: {
    target: "es2022",
    sourcemap: false,
    cssCodeSplit: false,
    rollupOptions: {
      output: {
        entryFileNames: "assets/studio.js",
        chunkFileNames: "assets/[name].js",
        assetFileNames: "assets/studio.[ext]",
      },
    },
  },
});

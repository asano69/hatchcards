import { defineconfig } from "vite";
import solid from "vite-plugin-solid";

export default defineconfig({
  plugins: [solid()],
  server: {
    host: "0.0.0.0",
    port: 3001,
    allowedhosts: true,
    changeorigin: true,
    proxy: {
      // use 127.0.0.1 explicitly to avoid localhost resolving to ::1 (ipv6)
      // while pocketbase only listens on 127.0.0.1 (ipv4).
      "/api": "http://127.0.0.1:3000",
      "/_": "http://127.0.0.1:3000",
    },
  },
  build: {
    outDir: "../internal/assets/static/dist",
    emptyOutDir: true,
    rollupOptions: {
      input: "src/main.jsx",
      output: {
        entryFileNames: "index.js",
        assetFileNames: "index.[ext]",
      },
    },
  },
});

import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const devProxyTarget = process.env.VITE_DEV_PROXY_TARGET || "http://localhost:8080";

export default defineConfig({
  base: "/tutorial-docs/",
  optimizeDeps: {
    include: ["react", "react-dom/client"],
  },
  server: {
    host: "0.0.0.0",
    allowedHosts: ["terminal.local"],
    proxy: {
      "/api": {
        target: devProxyTarget,
        changeOrigin: true,
      },
    },
    warmup: {
      clientFiles: ["./src/main.jsx"],
    },
  },
  plugins: [react()],
});

import { readdirSync, rmSync } from "node:fs";
import { fileURLToPath } from "node:url";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig(({ command, mode }) => ({
	plugins: [
		react(),
		tailwindcss(),
		{
			name: "limpar-build-preservando-checkout",
			buildStart() {
				if (command !== "build") return;
				for (const name of readdirSync(new URL("./dist/", import.meta.url))) {
					if (name !== ".gitkeep") rmSync(new URL(`./dist/${name}`, import.meta.url), { recursive: true, force: true });
				}
			},
		},
	],
	resolve: { alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) } },
	define: { __HARNESS__: JSON.stringify(mode === "teste" && command === "serve") },
	build: { emptyOutDir: false },
	server: {
		host: "127.0.0.1",
		port: 5173,
		strictPort: true,
		proxy: Object.fromEntries(
			["/health", "/ready", "/receipts", "/api"].map((path) => [path, { target: "http://127.0.0.1:8080", changeOrigin: false }]),
		),
	},
}));

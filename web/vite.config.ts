import { readdirSync, rmSync } from "node:fs";
import { fileURLToPath } from "node:url";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig, loadEnv } from "vite";

/** Configuração exclusiva do servidor Vite; não é importada pelo navegador. */
export function destinoDaAPI(valor?: string) {
	if (!valor?.trim()) return "http://127.0.0.1:8080";
	const erro = new Error("APROVA_API_PROXY deve ser uma origem HTTP(S), sem credenciais, caminho, consulta ou fragmento.");
	let url: URL;
	try {
		url = new URL(valor);
	} catch {
		throw erro;
	}
	if (!["http:", "https:"].includes(url.protocol) || url.username || url.password || url.pathname !== "/" || url.search || url.hash)
		throw erro;
	return url.origin;
}

export default defineConfig(({ command, mode }) => {
	const destino = destinoDaAPI(loadEnv(mode, process.cwd(), "APROVA_").APROVA_API_PROXY);
	return {
		// Desenvolvimento, regressão e o teste de proxy usam servidores simultâneos.
		cacheDir: `node_modules/.vite/${mode}`,
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
				["/health", "/ready", "/receipts", "/api", "/auth"].map((path) => [path, { target: destino, changeOrigin: false }]),
			),
		},
	};
});

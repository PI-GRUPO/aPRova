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
	const env = loadEnv(mode, process.cwd(), "APROVA_");
	const destino = destinoDaAPI(env.APROVA_API_PROXY);
	return {
		// Desenvolvimento, regressão e o teste de proxy usam servidores simultâneos.
		cacheDir: `node_modules/.vite/${mode}`,
		plugins: [
			react(),
			tailwindcss(),
			{
				name: "validar-build-dashboard",
				generateBundle(_, bundle) {
					for (const output of Object.values(bundle)) {
						if (output.type === "chunk" && Object.keys(output.modules).some((id) => id.includes("/src/mocks/")))
							this.error("O build inclui fixtures do dashboard.");
					}
					// O Go incorpora dist no binário mesmo em checkouts sem o frontend compilado.
					this.emitFile({ type: "asset", fileName: ".gitkeep", source: "" });
				},
			},
		],
		resolve: { alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) } },
		define: {
			__MOCKS__: JSON.stringify(
				command === "serve" &&
					process.env.NODE_ENV !== "production" &&
					["development", "teste", "test"].includes(mode) &&
					env.APROVA_DASHBOARD_MOCKS === "1",
			),
			__HARNESS__: JSON.stringify(mode === "teste" && command === "serve"),
		},
		build: { emptyOutDir: true },
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

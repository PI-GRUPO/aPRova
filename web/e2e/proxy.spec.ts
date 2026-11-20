import { createServer as criarServidorHTTP } from "node:http";
import type { AddressInfo } from "node:net";
import { fileURLToPath } from "node:url";
import { expect, test } from "@playwright/test";
import { createServer as criarVite } from "vite";
import { destinoDaAPI } from "../vite.config";

test("destino do proxy aceita somente origens e não revela configurações inválidas", () => {
	expect(destinoDaAPI()).toBe("http://127.0.0.1:8080");
	expect(destinoDaAPI("  ")).toBe("http://127.0.0.1:8080");
	expect(destinoDaAPI("https://api.example.test/")).toBe("https://api.example.test");
	expect(destinoDaAPI("http://127.0.0.1:18080")).toBe("http://127.0.0.1:18080");
	for (const valor of [
		"invalido",
		"//example.test",
		"file:///tmp/api",
		"https://usuario:segredo@example.test",
		"https://example.test/api",
		"https://example.test/?token=segredo",
		"https://example.test/#segredo",
	]) {
		expect(() => destinoDaAPI(valor)).toThrow(
			/^APROVA_API_PROXY deve ser uma origem HTTP\(S\), sem credenciais, caminho, consulta ou fragmento\.$/,
		);
	}
});

test("Vite encaminha API e OAuth ao destino configurado preservando erros, cookies e CSRF", async ({ request }) => {
	const recebidas: { caminho: string; metodo?: string; cookie?: string; csrf?: string; origem?: string }[] = [];
	const api = criarServidorHTTP((req, res) => {
		recebidas.push({
			caminho: req.url ?? "",
			metodo: req.method,
			cookie: req.headers.cookie,
			csrf: req.headers["x-csrf-token"] as string | undefined,
			origem: req.headers.origin,
		});
		if (req.url?.startsWith("/auth/github")) {
			res.writeHead(303, {
				Location: "https://github.com/login/oauth/authorize?state=fixture",
				"Set-Cookie": "_oauth=fixture; HttpOnly; Path=/; SameSite=Lax",
			});
			res.end();
		} else {
			res.writeHead(req.url === "/api/ausente" ? 404 : 200, { "Content-Type": "application/json" });
			res.end(JSON.stringify({ origem: "api-isolada" }));
		}
	});
	await new Promise<void>((resolve) => api.listen(0, "127.0.0.1", resolve));
	const anterior = process.env.APROVA_API_PROXY;
	let vite: Awaited<ReturnType<typeof criarVite>> | undefined;
	try {
		process.env.APROVA_API_PROXY = `http://127.0.0.1:${(api.address() as AddressInfo).port}`;
		vite = await criarVite({
			root: fileURLToPath(new URL("../", import.meta.url)),
			mode: "teste-proxy",
			server: { port: 0, strictPort: false },
		});
		await vite.listen();
		const endereco = vite.httpServer?.address();
		if (!endereco || typeof endereco === "string") throw new Error("Servidor Vite sem porta HTTP.");
		const origem = `http://127.0.0.1:${endereco.port}`;
		for (const caminho of ["/health", "/ready", "/receipts/fixture", "/api/sessao", "/api/analises?installation_id=42"]) {
			const res = await request.get(origem + caminho);
			expect(res.status()).toBe(200);
			expect(await res.json()).toEqual({ origem: "api-isolada" });
		}
		const ausente = await request.get(`${origem}/api/ausente`, { headers: { Accept: "text/html" } });
		expect(ausente.status()).toBe(404);
		expect(ausente.headers()["content-type"]).toBe("application/json");
		const login = await request.get(`${origem}/auth/github`, { maxRedirects: 0 });
		expect(login.status()).toBe(303);
		expect(login.headers().location).toBe("https://github.com/login/oauth/authorize?state=fixture");
		expect(login.headers()["set-cookie"]).toContain("HttpOnly");
		await request.get(`${origem}/auth/github/callback?code=fixture&state=fixture`, { maxRedirects: 0 });
		await request.post(`${origem}/api/sessao/sair`, {
			headers: { Cookie: "sessao=fixture", "X-CSRF-Token": "csrf-fixture", Origin: origem },
		});
		expect(recebidas.map((r) => r.caminho)).toContain("/api/analises?installation_id=42");
		expect(recebidas.map((r) => r.caminho)).toContain("/auth/github/callback?code=fixture&state=fixture");
		expect(recebidas.at(-1)).toEqual({
			caminho: "/api/sessao/sair",
			metodo: "POST",
			cookie: "sessao=fixture",
			csrf: "csrf-fixture",
			origem,
		});
	} finally {
		if (anterior === undefined) delete process.env.APROVA_API_PROXY;
		else process.env.APROVA_API_PROXY = anterior;
		await vite?.close();
		await new Promise<void>((resolve, reject) => api.close((erro) => (erro ? reject(erro) : resolve())));
	}
});

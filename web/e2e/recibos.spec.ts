import { text } from "node:stream/consumers";
import { expect, type Page, test } from "@playwright/test";

const id = "6d290152-af24-4462-a525-72adf3b22d6c";
const anterior = "31c84bc0-0000-4000-8000-000000000001";
const chave = { key_id: "chave-teste", algoritmo: "Ed25519", chave_publica_base64: "Y2hhdmU=" };
const documento = {
	id,
	task_run_id: "9eaa0538-add6-4361-8bfc-e74acec86c07",
	key_id: chave.key_id,
	hash_sha256: "a".repeat(64),
	payload_base64: "ZXhhdG8=",
	assinatura_base64: "YXNzaW5hdHVyYQ==",
	recibo_anterior_id: anterior,
	payload: {
		repositorio: "equipe/projeto",
		numero_pr: 1,
		head_sha: "6be930a",
		emitido_em: "2026-11-09T11:00:47Z",
		nivel: "alto",
		nivel_deterministico: "alto",
		sandbox: { testes_passaram: 10, testes_falharam: 1, tipo_falha: "teste" },
		motivos: [{ id: "falha_sandbox", evidencias: ["1 de 11 testes falharam"] }],
		anterior: { id: anterior, hash: "b".repeat(64) },
	},
};
const verificacao = {
	id,
	valido: true,
	hash_sha256: documento.hash_sha256,
	key_id: chave.key_id,
	verificacoes: ["hash", "chave", "assinatura", "conteudo", "elo"].map((nome) => ({ nome, resultado: "ok" })),
};
async function simular(page: Page, doc: unknown = documento, resultado: unknown = verificacao, chaves: unknown[] = [chave]) {
	await page.route("**/receipts/**", async (route) => {
		const path = new URL(route.request().url()).pathname;
		await route.fulfill({ json: path.endsWith("/keys") ? { chaves } : path.endsWith("/verify") ? resultado : doc });
	});
}
for (const largura of [1920, 1440, 390, 360]) {
	test(`consulta, verificação, exportação e temas em ${largura}px`, async ({ page }) => {
		await simular(page);
		await page.setViewportSize({ width: largura, height: largura >= 768 ? 900 : 844 });
		await page.goto("/recibos");
		await page.getByLabel("UUID do recibo").fill(id);
		await page.getByRole("button", { name: "Localizar recibo" }).click();
		await expect(page.getByRole("heading", { name: `Recibo / ${id.slice(0, 8)}…` })).toBeVisible();
		await expect(page.getByText("Ainda não executada nesta sessão.")).toBeVisible();
		await expect(page.getByRole("button", { name: "Verificar cadeia da instalação" })).toBeDisabled();
		await page.evaluate(() => document.fonts.ready);
		for (const tema of ["escuro", "claro"]) {
			if (tema === "claro") await page.getByRole("button", { name: "Usar tema claro" }).click();
			if (largura === 1920) {
				const borda = await page.locator(".recibo-assinado").evaluate((painel) => {
					const estilo = getComputedStyle(painel, "::after");
					return { repeticao: estilo.backgroundRepeat, imagem: estilo.backgroundImage, largura: parseFloat(estilo.width) };
				});
				expect(borda.repeticao).toBe("repeat-x");
				expect(borda.imagem).not.toBe("none");
				expect(borda.largura).toBeGreaterThan(558);
			}
			await page.screenshot({ path: `../.capturas-mcp/f2/local-${largura}-${tema}.png`, fullPage: true });
			expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
			expect(
				await page
					.locator("img:visible")
					.evaluateAll((imgs) =>
						imgs.every(
							(img) =>
								img instanceof HTMLImageElement &&
								img.complete &&
								img.naturalWidth > 0 &&
								img.width === img.naturalWidth &&
								img.height === img.naturalHeight,
						),
					),
			).toBeTruthy();
		}
		await page.getByRole("button", { name: "Verificar recibo", exact: true }).click();
		await expect(page.getByText("Recibo válido.", { exact: true })).toBeVisible();
		await expect(page.getByRole("cell", { name: "Confere", exact: true })).toHaveCount(5);
		if (largura < 768) {
			await expect(page.getByRole("heading", { name: "Verificação do recibo", exact: true })).toBeFocused();
			await page.screenshot({ path: `../.capturas-mcp/validacao-final/verificacao-${largura}.png`, fullPage: true });
			await page.getByRole("button", { name: "Voltar ao recibo", exact: true }).click();
			await expect(page.getByRole("button", { name: "Verificar recibo", exact: true })).toBeFocused();
		}
		const downloadPromise = page.waitForEvent("download");
		await page.getByRole("button", { name: "Exportar recibo", exact: true }).click();
		const download = await downloadPromise;
		expect(JSON.parse(await text(await download.createReadStream()))).toEqual(documento);
		const keyPromise = page.waitForEvent("download");
		await page.getByRole("button", { name: /Exportar chave pública de/ }).click();
		const downloadChave = await keyPromise;
		expect(await text(await downloadChave.createReadStream())).toBe(chave.chave_publica_base64);
		await page.reload();
		await expect(page.getByText("Ainda não executada nesta sessão.")).toBeVisible();
	});
}
test("mostra falha e conferências não verificadas, sem transformar 200 em sucesso", async ({ page }) => {
	await simular(page, documento, {
		...verificacao,
		valido: false,
		falha: "hash",
		detalhe: "O hash não confere.",
		verificacoes: verificacao.verificacoes.map((c, i) => ({ ...c, resultado: i === 0 ? "falhou" : "nao_verificada" })),
	});
	await page.goto(`/recibos/${id}`);
	await page.getByRole("button", { name: "Verificar recibo", exact: true }).click();
	await expect(page.getByText("Não foi possível validar o recibo.", { exact: true })).toBeVisible();
	await expect(page.getByRole("cell", { name: "Não verificada", exact: true })).toHaveCount(4);
	await expect(page.getByText("Recibo válido.", { exact: true })).toHaveCount(0);
});
test("payload ausente, chave desconhecida e primeiro recibo são explícitos", async ({ page }) => {
	await simular(page, { ...documento, payload: null, recibo_anterior_id: null }, verificacao, []);
	await page.goto(`/recibos/${id}`);
	await expect(page.getByText(/Conteúdo legível ausente/)).toBeVisible();
	await expect(page.getByText(/Nenhuma chave correspondente/)).toBeVisible();
	await expect(page.getByRole("link", { name: "Consultar recibo anterior" })).toHaveCount(0);
	await expect(page.getByRole("button", { name: /Exportar chave pública de/ })).toHaveCount(0);
});
for (const status of [404, 500, 503])
	test(`erro ${status} permite recuperação`, async ({ page }) => {
		await page.route("**/receipts/**", (route) => route.fulfill({ status, json: { status: "erro" } }));
		await page.goto(`/recibos/${id}`);
		await expect(page.getByRole("heading", { name: "Não foi possível abrir o recibo" })).toBeVisible();
		await page.unrouteAll();
		await simular(page);
		await page.getByRole("button", { name: "Tentar novamente" }).click();
		await expect(page.getByText("Ainda não executada nesta sessão.")).toBeVisible();
	});
test("troca de UUID cancela consulta e limpa a verificação anterior", async ({ page }) => {
	await simular(page);
	await page.goto(`/recibos/${id}?installation_id=43`);
	await page.getByRole("button", { name: "Verificar recibo", exact: true }).click();
	await expect(page.getByText("Recibo válido.", { exact: true })).toBeVisible();
	await page.route(`**/receipts/${anterior}`, async (route) => {
		await new Promise((resolve) => setTimeout(resolve, 300));
		await route.fulfill({ json: { ...documento, id: anterior, recibo_anterior_id: null } });
	});
	await page.getByRole("link", { name: "Consultar recibo anterior" }).click();
	await expect(page).toHaveURL(`/recibos/${anterior}?installation_id=43`);
	await expect(page.getByText("Recibo válido.", { exact: true })).toHaveCount(0);
	await page.getByRole("link", { name: "Voltar para recibos" }).click();
	await expect(page).toHaveURL("/recibos?installation_id=43");
	await expect(page.getByRole("heading", { name: "Recibos e auditoria", exact: true })).toBeVisible();
	await page.unrouteAll({ behavior: "wait" });
});
test("verificação com identidade divergente é recusada", async ({ page }) => {
	await simular(page, documento, { ...verificacao, id: anterior });
	await page.goto(`/recibos/${id}`);
	await page.getByRole("button", { name: "Verificar recibo", exact: true }).click();
	await expect(page.getByRole("alert")).toContainText("formato esperado");
	await expect(page.getByText("Recibo válido.", { exact: true })).toHaveCount(0);
});

test("handlers Go verificam recibo assinado e exportação confere independentemente", async ({ page, request }) => {
	const idLocal = "00000000-0000-4000-8000-000000000002";
	await page.route("**/receipts/**", async (route) => {
		const path = new URL(route.request().url()).pathname;
		const response = await request.get(`http://127.0.0.1:18080${path}`);
		await route.fulfill({ response });
	});
	await page.goto(`/recibos/${idLocal}`);
	await expect(page.getByText("equipe-e2e/projeto", { exact: true })).toBeVisible();
	await page.getByRole("button", { name: "Verificar recibo", exact: true }).click();
	await expect(page.getByText("Recibo válido.", { exact: true })).toBeVisible();
	const promessa = page.waitForEvent("download");
	await page.getByRole("button", { name: "Exportar recibo", exact: true }).click();
	const download = await promessa;
	const exportado = JSON.parse(await text(await download.createReadStream()));
	const chaves = await (await request.get("http://127.0.0.1:18080/receipts/keys")).json();
	const { createHash, createPublicKey, verify } = await import("node:crypto");
	const payload = Buffer.from(exportado.payload_base64, "base64");
	expect(createHash("sha256").update(payload).digest("hex")).toBe(exportado.hash_sha256);
	const publica = createPublicKey({
		key: Buffer.concat([Buffer.from("302a300506032b6570032100", "hex"), Buffer.from(chaves.chaves[0].chave_publica_base64, "base64")]),
		format: "der",
		type: "spki",
	});
	expect(verify(null, payload, publica, Buffer.from(exportado.assinatura_base64, "base64"))).toBeTruthy();
});

test("erro de rede na verificação não mantém resultado válido e permite repetir", async ({ page }) => {
	await simular(page);
	await page.goto(`/recibos/${id}`);
	const verificar = page.getByRole("button", { name: "Verificar recibo", exact: true });
	await verificar.click();
	await expect(page.getByText("Recibo válido.", { exact: true })).toBeVisible();
	await page.route("**/verify", (route) => route.abort("failed"));
	await verificar.click();
	await expect(page.getByRole("alert")).toContainText("Não foi possível conectar");
	await expect(page.getByText("Recibo válido.", { exact: true })).toHaveCount(0);
	await page.unroute("**/verify");
	await verificar.click();
	await expect(page.getByText("Recibo válido.", { exact: true })).toBeVisible();
});
test("ajuda aparece ao passar o mouse ou focar o ícone", async ({ page }) => {
	await simular(page);
	await page.goto(`/recibos/${id}`);
	const ajuda = page.getByRole("button", { name: "Ajuda: Conteúdo assinado", exact: true });
	const balao = page.getByRole("tooltip");
	await expect(balao).toBeHidden();
	await ajuda.hover();
	await expect(balao).toContainText("O conteúdo registrado só tem sua integridade confirmada após a verificação.");
	await expect(balao).toBeVisible();
	await expect(page.getByRole("dialog")).toHaveCount(0);
	await balao.hover();
	await expect(balao).toBeVisible();
	await page.getByRole("heading", { name: /^Recibo \/ / }).hover();
	await expect(balao).toBeHidden();
	await ajuda.focus();
	await expect(balao).toBeVisible();
	await expect(ajuda).toHaveAccessibleDescription(/O conteúdo registrado/);
	await page.keyboard.press("Escape");
	await expect(balao).toBeHidden();
	await expect(ajuda).toBeFocused();
});

test.describe("ajuda em tela de toque", () => {
	test.use({ viewport: { width: 320, height: 844 }, hasTouch: true });
	test("abre o balão ao tocar e fecha ao tocar fora", async ({ page }) => {
		await simular(page);
		await page.addInitScript(() => {
			localStorage.setItem("aprova-tema", "claro");
		});
		await page.goto(`/recibos/${id}`);
		const ajuda = page.getByRole("button", { name: "Ajuda: Conteúdo assinado", exact: true });
		const balao = page.getByRole("tooltip");
		await ajuda.tap();
		await expect(balao).toBeVisible();
		await expect(page.getByRole("dialog")).toHaveCount(0);
		const caixa = await balao.boundingBox();
		expect(caixa).not.toBeNull();
		if (caixa) {
			expect(caixa.x).toBeGreaterThanOrEqual(0);
			expect(caixa.x + caixa.width).toBeLessThanOrEqual(320);
		}
		await page.getByRole("heading", { name: /^Recibo \/ / }).tap();
		await expect(balao).toBeHidden();
	});
});

test("consulta rejeita UUID inválido", async ({ page }) => {
	await simular(page);
	await page.goto("/recibos");
	await page.getByLabel("UUID do recibo").fill("inválido");
	await page.getByRole("button", { name: "Localizar recibo" }).click();
	await expect(page.getByRole("alert").filter({ hasText: "Informe um UUID válido." })).toBeVisible();
});

for (const largura of [390, 320])
	for (const tema of ["escuro", "claro"]) {
		test(`resultado de verificação mobile em ${largura}px e ${tema}`, async ({ page }) => {
			await simular(page);
			await page.setViewportSize({ width: largura, height: 844 });
			await page.addInitScript((t) => {
				localStorage.setItem("aprova-tema", t);
			}, tema);
			await page.goto(`/recibos/${id}`);
			await page.getByRole("button", { name: "Verificar recibo", exact: true }).click();
			await expect(page.getByRole("heading", { name: "Verificação do recibo", exact: true })).toBeFocused();
			await expect(page.getByRole("cell", { name: "Confere", exact: true })).toHaveCount(5);
			await page.evaluate(() => document.fonts.ready);
			const tabela = await page.getByRole("table", { name: "Conferências do núcleo" }).boundingBox();
			expect(tabela).toMatchObject({ x: 16, y: 276, width: largura - 32 });
			await page.screenshot({ path: `../.capturas-mcp/validacao-final/verificacao-${largura}-${tema}.png`, fullPage: true });
			expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBeTruthy();
			await page.getByRole("button", { name: "Voltar ao recibo", exact: true }).click();
			await expect(page.getByRole("button", { name: "Verificar recibo", exact: true })).toBeFocused();
			await expect(page.getByRole("button", { name: "Exportar recibo", exact: true })).toBeVisible();
			await page.reload();
			await expect(page.getByText("Ainda não executada nesta sessão.")).toBeVisible();
		});
	}

test("resultado mobile conserva falha e etapas não verificadas", async ({ page }) => {
	await simular(page, documento, {
		...verificacao,
		valido: false,
		falha: "hash",
		verificacoes: verificacao.verificacoes.map((c, i) => ({ ...c, resultado: i === 0 ? "falhou" : "nao_verificada" })),
	});
	await page.setViewportSize({ width: 320, height: 568 });
	await page.goto(`/recibos/${id}`);
	await page.getByRole("button", { name: "Verificar recibo", exact: true }).click();
	await expect(page.getByRole("heading", { name: "Verificação do recibo", exact: true })).toBeVisible();
	await expect(page.getByRole("cell", { name: "Falhou", exact: true })).toHaveCount(1);
	await expect(page.getByRole("cell", { name: "Não verificada", exact: true })).toHaveCount(4);
	await expect(page.getByText("Recibo válido.", { exact: true })).toHaveCount(0);
});

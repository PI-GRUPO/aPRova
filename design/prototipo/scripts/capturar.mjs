import { mkdirSync } from "node:fs";
import { chromium } from "@playwright/test";

const [, , rota = "/", nome = "tela", tema = "escuro", largura = "1440", altura = "900"] = process.argv;
const pasta = ".impeccable/review";
mkdirSync(pasta, { recursive: true });

const navegador = await chromium.launch();
const contexto = await navegador.newContext({
	viewport: { width: Number(largura), height: Number(altura) },
	deviceScaleFactor: 1,
	locale: "pt-BR",
	timezoneId: "America/Sao_Paulo",
});
await contexto.addInitScript((escolhido) => localStorage.setItem("aprova-tema", escolhido), tema);
const pagina = await contexto.newPage();
await pagina.goto(`http://localhost:5173${rota}`);
await pagina.waitForLoadState("networkidle");
await pagina.evaluate(() => document.fonts.ready);
await pagina.screenshot({ path: `${pasta}/${nome}.png`, fullPage: true });
await navegador.close();
console.log(`${pasta}/${nome}.png`);

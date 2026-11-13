import { spawnSync } from "node:child_process";
import { chromium } from "@playwright/test";

const rotas = process.argv.slice(2);
const alvos = (rotas.length > 0 ? rotas : ["/"]).map((rota) => `http://localhost:5173${rota}`);
const resultado = spawnSync("npx", ["impeccable", "detect", ...alvos], {
	stdio: "inherit",
	env: { ...process.env, IMPECCABLE_BROWSER: process.env.IMPECCABLE_BROWSER ?? chromium.executablePath() },
});
process.exit(resultado.status ?? 1);

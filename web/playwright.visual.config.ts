import { defineConfig } from "@playwright/test";

export default defineConfig({
	testDir: "./e2e",
	testMatch: "visao-geral.spec.ts",
	forbidOnly: true,
	workers: 1,
	reporter: [["list"]],
	use: { baseURL: "http://127.0.0.1:5173", locale: "pt-BR", timezoneId: "America/Sao_Paulo", trace: "retain-on-failure" },
	webServer: { command: "npm run dev -- --mode teste", url: "http://127.0.0.1:5173", reuseExistingServer: false, timeout: 60000 },
});

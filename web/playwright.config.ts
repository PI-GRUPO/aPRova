import { defineConfig } from "@playwright/test";
export default defineConfig({
	testDir: "./e2e",
	fullyParallel: true,
	forbidOnly: true,
	reporter: [["list"]],
	use: {
		baseURL: "http://127.0.0.1:5183",
		locale: "pt-BR",
		timezoneId: "America/Sao_Paulo",
		viewport: { width: 1440, height: 900 },
		trace: "retain-on-failure",
	},
	projects: [{ name: "chromium", use: { browserName: "chromium" } }],
	webServer: [
		{
			command: "npm run dev -- --mode teste --port 5183",
			url: "http://127.0.0.1:5183",
			reuseExistingServer: false,
			timeout: 60000,
			gracefulShutdown: { signal: "SIGTERM", timeout: 3000 },
		},
		{
			command: "go run ./e2e/servidor",
			url: "http://127.0.0.1:18080/health",
			reuseExistingServer: false,
			timeout: 120000,
			gracefulShutdown: { signal: "SIGTERM", timeout: 3000 },
		},
	],
});

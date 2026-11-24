import { listarRepositorios } from "./analises";
import { type Consulta, type Escopo, indisponivel } from "./fonte";
import { consultarJSON } from "./http";
import { data, filtrosPeriodo, inteiro, invalido, lerPainel, lista, registro, texto } from "./painel";
import { uuidValido } from "./recibos";
export type RegistroRecibo = { id: string; repo: string; pr: number; sha: string; emissao: string };
type Colecao = { repositorios: string[]; itens: RegistroRecibo[]; pagina?: number; limite?: number; total?: number; snapshot?: string };
export const consultarColecao: Consulta<Colecao> = __MOCKS__
	? async ({ repositorio }, filtros, signal) => {
			const [{ recibosDemonstracao }, { referenciaMock }] = await Promise.all([import("@/mocks/recibos"), import("@/mocks/gestao")]);
			signal.throwIfAborted();
			const periodo = filtros.get("periodo") ?? "30";
			const inicio = referenciaMock.getTime() - (["7", "30", "90"].includes(periodo) ? Number(periodo) : 30) * 86400000;
			return {
				repositorios: [...new Set(recibosDemonstracao.map((r) => r.repo))],
				itens: recibosDemonstracao.filter((r) => (!repositorio || r.repo === repositorio) && Date.parse(r.emissao) > inicio),
			};
		}
	: async (escopo, f, signal) => {
			const [colecao, repositorios] = await Promise.all([
				lerRecibos(escopo, filtrosPeriodo(f, true), signal),
				listarRepositorios(escopo.instalacao, signal),
			]);
			if (colecao.itens.some((i) => !repositorios.includes(i.repo))) return invalido();
			return { ...colecao, repositorios };
		};
const falhasDaCadeia = {
	hash: "O hash do recibo não confere.",
	chave_desconhecida: "A chave pública usada para assinar o recibo não está disponível.",
	assinatura: "A assinatura do recibo não confere.",
	conteudo: "O conteúdo assinado não confere com o registro.",
	elo: "O vínculo com o recibo anterior está quebrado.",
};

export const verificarCadeia = {
	disponivel: !__MOCKS__,
	async executar(instalacao: string, signal: AbortSignal) {
		if (__MOCKS__) return indisponivel();
		const q = new URLSearchParams({ installation_id: instalacao });
		const r = registro(await consultarJSON(`/api/web/recibos/cadeia?${q}`, { privada: true, signal, timeout: 30000 }));
		const total = inteiro(r.total);
		const verificados = inteiro(r.verificados);
		if (r.installation_id !== Number(instalacao) || typeof r.valida !== "boolean") return invalido();
		if (r.valida) {
			if (verificados !== total || [r.primeiro_recibo_invalido, r.recibo_anterior_ausente, r.falha].some((v) => v !== undefined))
				return invalido();
			return { valida: true as const, total };
		}
		const id = texto(r.primeiro_recibo_invalido);
		const falha = texto(r.falha);
		const ausente = r.recibo_anterior_ausente;
		if (
			verificados >= total ||
			!uuidValido(id) ||
			!Object.hasOwn(falhasDaCadeia, falha) ||
			(ausente !== undefined && (falha !== "elo" || typeof ausente !== "string" || !uuidValido(ausente)))
		)
			return invalido();
		return {
			valida: false as const,
			id,
			detalhe: typeof ausente === "string" ? `Recibo anterior ausente: ${ausente}.` : falhasDaCadeia[falha as keyof typeof falhasDaCadeia],
		};
	},
};
async function lerRecibos(escopo: Escopo, q: URLSearchParams, signal: AbortSignal) {
	const { dados, pagina, limite, snapshot } = await lerPainel("/api/web/recibos", escopo, q, signal);
	const itens = lista(dados.itens, (r) => {
		const id = texto(r.id),
			repo = texto(r.repo);
		if (!uuidValido(id) || (escopo.repositorio && repo !== escopo.repositorio)) return invalido();
		const hash = texto(r.hash_sha256);
		if (!/^[a-f0-9]{64}$/.test(hash)) return invalido();
		return { id, repo, pr: inteiro(r.pr), sha: texto(r.sha), emissao: data(r.emissao), key_id: texto(r.key_id), hash_sha256: hash };
	});
	const total = inteiro(dados.total);
	if (itens.length !== Math.max(0, Math.min(limite, total - (pagina - 1) * limite))) return invalido();
	return { itens, total, snapshot, pagina, limite };
}
function campoCSV(valor: string | number) {
	const s = String(valor);
	return `"${/^\s*[=+@-]|^[\t\r\n]/.test(s) ? "'" : ""}${s.replaceAll('"', '""')}"`;
}
export const exportarColecao = {
	disponivel: !__MOCKS__,
	async executar(escopo: Escopo, dias: number, formato: "json" | "csv", signal: AbortSignal, snapshot: string) {
		const { itens, total } = await lerRecibos(escopo, new URLSearchParams({ periodo: String(dias), limite: "500", snapshot }), signal);
		if (total > itens.length) throw new Error("Reduza o período ou filtre por repositório para exportar até 500 recibos.");
		if (formato === "json") return new Blob([JSON.stringify(itens, null, 2)], { type: "application/json" });
		return new Blob(
			[
				[
					"id,repo,pr,sha,emissao,key_id,hash_sha256",
					...itens.map((r) => [r.id, r.repo, r.pr, r.sha, r.emissao, r.key_id, r.hash_sha256].map(campoCSV).join(",")),
				].join("\r\n"),
			],
			{ type: "text/csv;charset=utf-8" },
		);
	},
};

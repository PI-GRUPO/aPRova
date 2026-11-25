import { listarRepositorios } from "./analises";
import { type Consulta, type Escopo, indisponivel, type Operacao, operacaoPendente } from "./fonte";
import { data, filtrosPeriodo, inteiro, invalido, lerPainel, lista, texto } from "./painel";
export type Configuracao = {
	limite: number;
	caminhos: string[];
	versao: number;
	historico: { versao: number; pessoa: string; data: string }[];
};
export type Sugestao = {
	id: string;
	repo: string;
	titulo: string;
	origem: string;
	estado: "pendente" | "em_revisao" | "recusada" | "aplicada";
	resumo: string;
	evidencia: string;
	limiteProposto: number | null;
};
export type Gestao = {
	paginacao?: { pagina: number; limite: number; snapshot: string; total: number };
	repositorios: string[];
	configuracoes: Record<string, Configuracao>;
	sla: Record<string, { primeiro: number; segundo: number }>;
	sugestoes: Sugestao[];
	regras: { id: string; descricao: string; nivel: string; acionamentos: number; avaliadasAltas?: number; aprovadas: number }[];
	calibracoes: { id: string; nome: string; risco: string; resultado: string }[];
	auditoria: { id: string; data: string; pessoa: string; acao: string; pr: number; origem: string; decisao: string; motivo: string }[];
};
const leituraGestao: Consulta<Gestao> = __MOCKS__
	? async (escopo, filtros, signal) => {
			const { lerGestaoMock } = await import("@/mocks/gestao");
			signal.throwIfAborted();
			return lerGestaoMock(escopo.repositorio, filtros);
		}
	: indisponivel;
function consultarRegistros(recurso: "regras" | "auditoria"): Consulta<Gestao> {
	return async (escopo, f, signal) => {
		const q = filtrosPeriodo(f, recurso === "auditoria");
		if (recurso === "auditoria") {
			const resultado = f.get("recusadas") === "1" ? "recusada" : f.get("resultado")?.toLowerCase();
			if (resultado) q.set("decisao", resultado);
			if (f.get("origem")) q.set("origem", f.get("origem") ?? "");
		}
		const [v, repositorios] = await Promise.all([lerPainel(recurso, escopo, q, signal), listarRepositorios(escopo.instalacao, signal)]);
		const base: Gestao = { repositorios, configuracoes: {}, sla: {}, sugestoes: [], regras: [], calibracoes: [], auditoria: [] };
		function repo(r: Record<string, unknown>) {
			const nome = texto(r.repo);
			if (!repositorios.includes(nome) || (escopo.repositorio && nome !== escopo.repositorio)) return invalido();
			return nome;
		}
		if (recurso === "regras") {
			base.regras = lista(v.dados.regras, (r) => ({
				id: texto(r.id),
				nivel: texto(r.nivel),
				descricao: texto(r.descricao),
				acionamentos: inteiro(r.acionamentos),
				avaliadasAltas: inteiro(r.avaliadas_altas),
				aprovadas: inteiro(r.aprovadas),
			}));
			if (
				base.regras.some(
					(r) =>
						!["alto", "medio", "baixo"].includes(r.nivel) ||
						r.aprovadas > (r.avaliadasAltas ?? 0) ||
						(r.avaliadasAltas ?? 0) > r.acionamentos,
				)
			)
				return invalido();
			base.calibracoes = lista(v.dados.calibracoes, (r) => ({
				id: texto(r.id),
				nome: `${repo(r)} #${inteiro(r.pr)}`,
				risco: `${texto(r.inicial)} → ${texto(r.final)}`,
				resultado: texto(r.resultado),
			}));
		} else {
			const total = inteiro(v.dados.total);
			base.auditoria = lista(v.dados.itens, (r) => ({
				id: texto(r.id),
				data: data(r.data),
				pessoa: r.pessoa === null ? "Login não registrado" : texto(r.pessoa),
				acao: `${texto(r.acao)} · ${repo(r)}`,
				pr: inteiro(r.pr),
				// A API anterior só listava tentativas do Slack e omitia a origem.
				origem: r.origem === undefined ? "Slack" : texto(r.origem),
				decisao: texto(r.decisao),
				motivo: texto(r.motivo),
			}));
			if (
				base.auditoria.some((a) => !["Slack", "Painel"].includes(a.origem) || (f.get("origem") && a.origem !== f.get("origem"))) ||
				base.auditoria.length !== Math.max(0, Math.min(v.limite, total - (v.pagina - 1) * v.limite))
			)
				return invalido();
			base.paginacao = { total, pagina: v.pagina, limite: v.limite, snapshot: v.snapshot };
		}
		return base;
	};
}
export const consultasGestao = {
	"/regras": __MOCKS__ ? leituraGestao : consultarRegistros("regras"),
	"/auditoria": __MOCKS__ ? leituraGestao : consultarRegistros("auditoria"),
	"/configuracao": leituraGestao,
	"/sugestoes": leituraGestao,
	"/sla": leituraGestao,
} satisfies Record<string, Consulta<Gestao>>;
export const salvarConfiguracao: Operacao<
	(escopo: Escopo, limite: number, caminhos: string[], versao: number, proposta?: string) => Promise<Configuracao>
> = operacaoPendente;
export const salvarSla: Operacao<(escopo: Escopo, primeiro: number, segundo: number) => Promise<void>> = operacaoPendente;
export const revisarSugestao: Operacao<(escopo: Escopo, id: string, acao: "recusada" | "em_revisao") => Promise<void>> = operacaoPendente;

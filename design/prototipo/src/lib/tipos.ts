export type Nivel = "baixo" | "medio" | "alto";

export type Estado =
	| "pendente"
	| "mapeando_contexto"
	| "gerando"
	| "testando_sandbox"
	| "avaliando_risco"
	| "aguardando_aprovacao"
	| "aplicando"
	| "concluido"
	| "rejeitado"
	| "falhou";

export type TipoDeFalha = "nenhuma" | "teste" | "compilacao" | "infraestrutura";

export interface Motivo {
	id: string;
	nivel: Nivel;
	descricao: string;
	evidencias: string[];
}

export interface Etapa {
	from_state: Estado;
	to_state: Estado;
	detail: string;
	created_at: string;
}

export interface ResultadoDoSandbox {
	attempt: number;
	compiled: boolean;
	tests_passed: number;
	tests_failed: number;
	duration_ms: number;
	failure_type: TipoDeFalha;
	created_at: string;
}

export interface GateDeRisco {
	deterministic_level: Nivel;
	level: Nivel;
	reason: string;
	rules_hit: Motivo[];
	new_dependencies: { ecossistema: string; nome: string; versao: string }[];
}

export interface Aprovacao {
	decision: "aprovada" | "alteracoes_solicitadas";
	reviewer: string;
	origin: "github" | "slack";
	feedback_reason: string | null;
	decided_at: string;
}

export interface Analise {
	id: string;
	tenant_id: number;
	repository: string;
	pr_number: number;
	head_branch: string;
	head_sha: string;
	author: string;
	event_action: string;
	state: Estado;
	received_at: string;
	created_at: string;
	updated_at: string;
	steps: Etapa[];
	sandbox_results: ResultadoDoSandbox[];
	risk_gate: GateDeRisco | null;
	receipt_id: string | null;
	approval: Aprovacao | null;
}

export interface ConteudoDoRecibo {
	versao: number;
	id: string;
	task_run_id: string;
	tenant_id: number;
	repositorio: string;
	numero_pr: number;
	head_sha: string;
	diff_sha256: string;
	diff_truncado: boolean;
	sandbox: {
		compilou: boolean;
		testes_passaram: number;
		testes_falharam: number;
		tipo_falha: TipoDeFalha;
		tentativa: number;
	};
	nivel_deterministico: Nivel;
	nivel: Nivel;
	motivos: { id: string; evidencias: string[] }[];
	emitido_em: string;
	anterior: { id: string; hash: string } | null;
	key_id: string;
}

export interface Recibo {
	id: string;
	tenant_id: number;
	task_run_id: string;
	key_id: string;
	hash_sha256: string;
	payload_base64: string;
	assinatura_base64: string;
	recibo_anterior_id: string | null;
	signed_at: string;
	payload: ConteudoDoRecibo;
}

export interface Regra {
	id: string;
	nivel: Nivel;
	descricao: string;
	quando: string;
}

export interface Instalacao {
	tenant_id: number;
	conta: string;
	repositorios: string[];
	periodo_dias: number;
	gerado_em: string;
	limite_de_arquivos: number;
}

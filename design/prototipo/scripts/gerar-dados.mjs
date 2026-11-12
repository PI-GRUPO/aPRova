import { createHash, generateKeyPairSync, sign } from "node:crypto";
import { mkdirSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const pastaDeDados = join(dirname(fileURLToPath(import.meta.url)), "..", "src", "dados");
const agora = Date.parse("2026-11-11T17:00:00Z");
const tenant = 58291734;
const minuto = 60_000;
const hora = 60 * minuto;
const dia = 24 * hora;

function sorteador(semente) {
	let estado = semente;
	return () => {
		estado = (estado + 0x6d2b79f5) | 0;
		let t = Math.imul(estado ^ (estado >>> 15), 1 | estado);
		t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
		return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
	};
}

const sortear = sorteador(20261111);
const inteiro = (minimo, maximo) => minimo + Math.floor(sortear() * (maximo - minimo + 1));
const hex = (tamanho) => Array.from({ length: tamanho }, () => inteiro(0, 15).toString(16)).join("");
const uuid = () => `${hex(8)}-${hex(4)}-4${hex(3)}-${"89ab"[inteiro(0, 3)]}${hex(3)}-${hex(12)}`;
const sha256 = (bytes) => createHash("sha256").update(bytes).digest("hex");
const iso = (instante) => new Date(instante).toISOString();

const regras = [
	{
		id: "falha_sandbox",
		nivel: "alto",
		descricao: "os testes ou a compilação falharam no sandbox",
		quando: "o resultado do sandbox é falha de teste ou de compilação",
	},
	{
		id: "muitos_arquivos",
		nivel: "alto",
		descricao: "o PR altera mais arquivos que o limite",
		quando: "o PR altera mais de 20 arquivos",
	},
	{
		id: "configuracao_ci",
		nivel: "alto",
		descricao: "o PR altera configuração de build ou de CI",
		quando: "altera workflow, Dockerfile, Makefile, Compose ou arquivo de build equivalente",
	},
	{
		id: "dependencia_nova",
		nivel: "alto",
		descricao: "o PR acrescenta dependência nova, que precisa de revisão humana",
		quando: "acrescenta dependência em go.mod, requirements, pyproject.toml ou pom.xml, ou o diff do manifesto não pôde ser lido",
	},
	{
		id: "segredo_versionado",
		nivel: "alto",
		descricao: "o PR versiona um arquivo que costuma guardar segredo",
		quando: "versiona .env, chave privada ou credencial",
	},
	{
		id: "area_sensivel",
		nivel: "alto",
		descricao: "o PR altera um caminho sensível",
		quando: "altera auth/, security/, payments/, migrations/, db/, database/ ou um arquivo .sql",
	},
	{
		id: "codigo_sem_teste",
		nivel: "medio",
		descricao: "o PR altera código sem alterar nenhum teste",
		quando: "altera código Go, Python ou Java sem alterar nenhum teste",
	},
];
const regraPorId = Object.fromEntries(regras.map((regra) => [regra.id, regra]));

const repositorios = {
	testes: {
		nome: "Azulpasta/aPRova-TESTES",
		testes: 10,
		autor: "agente-frete[bot]",
	},
	cobranca: {
		nome: "Azulpasta/cobranca-api",
		testes: 148,
		autor: "agente-cobranca[bot]",
	},
	painel: {
		nome: "Azulpasta/painel-entregas",
		testes: 63,
		autor: "agente-painel[bot]",
	},
};

const muitosArquivos = Array.from({ length: 27 }, (_, indice) => `painel/rotas/rota_${String(indice + 1).padStart(2, "0")}.py`);

const cenarios = {
	limpo: { motivos: [] },
	semTeste: { motivos: [["codigo_sem_teste", ["internal/cobranca/juros.go"]]] },
	semTestePainel: {
		motivos: [["codigo_sem_teste", ["painel/entregas/janela.py", "painel/entregas/rota.py"]]],
	},
	testeFalhou: {
		falha: "teste",
		falharam: 1,
		motivos: [["falha_sandbox", null]],
	},
	testesFalharam: {
		falha: "teste",
		falharam: 4,
		motivos: [
			["falha_sandbox", null],
			["codigo_sem_teste", ["internal/cobranca/boleto.go"]],
		],
	},
	naoCompila: {
		falha: "compilacao",
		motivos: [["falha_sandbox", ["o projeto não compilou"]]],
	},
	dependencia: {
		motivos: [["dependencia_nova", ["go.mod"]]],
		dependencias: [
			{
				ecossistema: "Go",
				nome: "github.com/shopspring/decimal",
				versao: "v1.4.0",
			},
		],
	},
	dependenciaPython: {
		motivos: [
			["dependencia_nova", ["requirements.txt"]],
			["codigo_sem_teste", ["painel/exportacao/planilha.py"]],
		],
		dependencias: [{ ecossistema: "PyPI", nome: "openpyxl", versao: "3.1.5" }],
	},
	diffIlegivel: {
		motivos: [["dependencia_nova", ["pom.xml: não foi possível ler o diff"]]],
	},
	ci: {
		motivos: [["configuracao_ci", [".github/workflows/ci.yml", "Dockerfile"]]],
	},
	sensivel: {
		motivos: [["area_sensivel", ["internal/auth/sessao.go", "internal/auth/sessao_test.go"]]],
	},
	migracao: {
		motivos: [
			["area_sensivel", ["migrations/000031_parcelas.up.sql", "migrations/000031_parcelas.down.sql"]],
			["codigo_sem_teste", ["internal/cobranca/parcela.go"]],
		],
	},
	segredo: {
		motivos: [
			["segredo_versionado", [".env.producao"]],
			["configuracao_ci", ["docker-compose.yml"]],
		],
	},
	muitos: {
		motivos: [
			["muitos_arquivos", muitosArquivos],
			["codigo_sem_teste", muitosArquivos.slice(0, 6)],
		],
	},
};

const roteiro = [
	["testes", 1, "teste-peso-zero", 2 * dia + 3 * hora, "aguardando_aprovacao", "testeFalhou"],
	["cobranca", 214, "agente/juros-compostos", 26 * hora, "aguardando_aprovacao", "dependencia"],
	["painel", 97, "agente/exportar-planilha", 7 * hora + 20 * minuto, "aguardando_aprovacao", "dependenciaPython"],
	["cobranca", 217, "agente/parcelamento", 3 * hora + 5 * minuto, "aguardando_aprovacao", "migracao"],
	["painel", 99, "agente/reorganiza-rotas", 48 * minuto, "aguardando_aprovacao", "muitos"],
	["cobranca", 219, "agente/arredondamento", 9 * minuto, "avaliando_risco", "limpo"],
	["painel", 100, "agente/filtro-por-bairro", 6 * minuto, "testando_sandbox", "limpo"],
	["testes", 9, "agente/frete-gratis", 4 * minuto, "testando_sandbox", "limpo"],
	["cobranca", 220, "agente/remove-campo-legado", 2 * minuto, "mapeando_contexto", "limpo"],
	["painel", 101, "agente/corrige-fuso", 1 * minuto, "pendente", "limpo"],
	["cobranca", 218, "agente/log-estruturado", 22 * minuto, "aplicando", "semTeste"],
	["cobranca", 216, "agente/retry-webhook", 1 * dia + 2 * hora, "concluido", "limpo"],
	["painel", 98, "agente/janela-de-entrega", 1 * dia + 6 * hora, "concluido", "semTestePainel"],
	["testes", 8, "agente/frete-por-faixa", 2 * dia + 5 * hora, "concluido", "limpo"],
	["cobranca", 215, "agente/sessao-expira", 3 * dia + 1 * hora, "concluido", "sensivel", "aprovada"],
	["painel", 96, "agente/pipeline-cache", 3 * dia + 4 * hora, "rejeitado", "ci"],
	["cobranca", 213, "agente/boleto-vencido", 4 * dia + 2 * hora, "concluido", "testesFalharam", "aprovada"],
	["testes", 7, "agente/peso-negativo", 5 * dia + 3 * hora, "concluido", "limpo"],
	["painel", 95, "agente/mapa-de-calor", 5 * dia + 7 * hora, "falhou", "infraestrutura"],
	["cobranca", 212, "agente/multa-diaria", 6 * dia + 1 * hora, "concluido", "semTeste"],
	["cobranca", 211, "agente/credenciais-homologacao", 7 * dia + 5 * hora, "rejeitado", "segredo"],
	["painel", 94, "agente/ordenacao-rotas", 8 * dia + 2 * hora, "concluido", "limpo"],
	["testes", 6, "agente/taxa-distancia", 8 * dia + 6 * hora, "concluido", "naoCompila", "aprovada"],
	["cobranca", 210, "agente/cliente-http", 9 * dia + 3 * hora, "concluido", "dependencia", "aprovada"],
	["painel", 93, "agente/tela-motorista", 10 * dia + 4 * hora, "concluido", "semTestePainel"],
	["cobranca", 209, "agente/estorno-parcial", 11 * dia + 2 * hora, "falhou", "substituido"],
	["cobranca", 209, "agente/estorno-parcial", 11 * dia + 1 * hora, "concluido", "limpo"],
	["painel", 92, "agente/pom-relatorios", 12 * dia + 5 * hora, "rejeitado", "diffIlegivel"],
	["testes", 5, "agente/frete-internacional", 13 * dia + 3 * hora, "concluido", "limpo"],
	["cobranca", 208, "agente/conciliacao", 14 * dia + 6 * hora, "concluido", "limpo"],
	["painel", 91, "agente/cache-enderecos", 15 * dia + 2 * hora, "falhou", "infraestrutura"],
	["cobranca", 207, "agente/indice-vencimento", 16 * dia + 4 * hora, "concluido", "migracao", "aprovada"],
	["painel", 90, "agente/notificacao-atraso", 17 * dia + 1 * hora, "concluido", "limpo"],
	["testes", 4, "agente/desconto-volume", 18 * dia + 5 * hora, "concluido", "semTeste"],
	["cobranca", 206, "agente/dockerfile-distroless", 20 * dia + 3 * hora, "concluido", "ci", "aprovada"],
	["painel", 89, "agente/exporta-csv", 21 * dia + 2 * hora, "concluido", "limpo"],
	["cobranca", 205, "agente/pix-copia-cola", 23 * dia + 6 * hora, "concluido", "limpo"],
	["painel", 88, "agente/agrupa-paradas", 24 * dia + 4 * hora, "concluido", "testeFalhou", "aprovada"],
	["testes", 3, "agente/arredonda-centavos", 26 * dia + 2 * hora, "concluido", "limpo"],
	["cobranca", 204, "agente/juros-pro-rata", 27 * dia + 5 * hora, "falhou", "infraestrutura"],
	["painel", 87, "agente/tema-impressao", 29 * dia + 1 * hora, "concluido", "semTestePainel"],
];

const ordemDosNiveis = { baixo: 0, medio: 1, alto: 2 };
const revisores = ["httpslua", "marina-tl", "rafael-rev"];

function montarGate(cenario, resultado) {
	const motivos = cenario.motivos
		.map(([id, evidencias]) => {
			const regra = regraPorId[id];
			const total = resultado.tests_passed + resultado.tests_failed;
			const lista = evidencias ?? [`${resultado.tests_failed} de ${total} testes falharam`];
			const descricao = id === "muitos_arquivos" ? `o PR altera ${lista.length} arquivos, acima do limite de 20` : regra.descricao;
			return {
				id,
				nivel: regra.nivel,
				descricao,
				evidencias: [...lista].sort(),
			};
		})
		.sort((a, b) => regras.findIndex((r) => r.id === a.id) - regras.findIndex((r) => r.id === b.id));
	const nivel = motivos.reduce((maior, motivo) => (ordemDosNiveis[motivo.nivel] > ordemDosNiveis[maior] ? motivo.nivel : maior), "baixo");
	return {
		deterministic_level: nivel,
		level: nivel,
		reason: motivos.length === 0 ? "nenhuma regra acionada" : motivos.map((m) => `${m.id}: ${m.descricao}`).join("\n"),
		rules_hit: motivos,
		new_dependencies: cenario.dependencias ?? [],
	};
}

function resultadoDoSandbox(repositorio, cenario) {
	if (cenario.falha === "compilacao") {
		return {
			compiled: false,
			tests_passed: 0,
			tests_failed: 0,
			failure_type: "compilacao",
		};
	}
	const falharam = cenario.falharam ?? 0;
	const total = repositorio.testes + (falharam > 0 ? 1 : 0);
	return {
		compiled: true,
		tests_passed: total - falharam,
		tests_failed: falharam,
		failure_type: falharam > 0 ? "teste" : "nenhuma",
	};
}

function montarAnalise([chave, numeroPR, ramo, haQuanto, estado, nomeDoCenario, decisao]) {
	const repositorio = repositorios[chave];
	const id = uuid();
	const recebido = agora - haQuanto;
	const etapas = [];
	let relogio = recebido + inteiro(400, 1900);
	const transicao = (de, para, detalhe = "") => {
		etapas.push({
			from_state: de,
			to_state: para,
			detail: detalhe,
			created_at: iso(relogio),
		});
	};
	const analise = {
		id,
		tenant_id: tenant,
		repository: repositorio.nome,
		pr_number: numeroPR,
		head_branch: ramo,
		head_sha: hex(40),
		author: repositorio.autor,
		event_action: numeroPR % 3 === 0 ? "synchronize" : "opened",
		state: estado,
		received_at: iso(recebido),
		created_at: iso(recebido + inteiro(20, 180)),
		updated_at: iso(recebido),
		steps: etapas,
		sandbox_results: [],
		risk_gate: null,
		receipt_id: null,
		approval: null,
	};
	const fechar = () => {
		analise.updated_at = etapas.length > 0 ? etapas[etapas.length - 1].created_at : analise.created_at;
		return analise;
	};

	if (estado === "pendente") return fechar();
	transicao("pendente", "mapeando_contexto");
	if (estado === "mapeando_contexto") return fechar();

	if (nomeDoCenario === "substituido") {
		relogio += inteiro(900, 2400);
		transicao("mapeando_contexto", "testando_sandbox");
		relogio += inteiro(8000, 15000);
		transicao("testando_sandbox", "falhou", "substituído por commit mais novo");
		return fechar();
	}

	relogio += inteiro(900, 2400);
	transicao("mapeando_contexto", "testando_sandbox");
	if (estado === "testando_sandbox") return fechar();

	if (nomeDoCenario === "infraestrutura") {
		for (let tentativa = 1; tentativa <= 3; tentativa++) {
			const duracao = inteiro(41000, 78000);
			relogio += duracao + (tentativa > 1 ? 30000 * 2 ** (tentativa - 2) : 0);
			analise.sandbox_results.push({
				attempt: tentativa,
				compiled: false,
				tests_passed: 0,
				tests_failed: 0,
				duration_ms: duracao,
				failure_type: "infraestrutura",
				created_at: iso(relogio),
			});
		}
		transicao("testando_sandbox", "falhou", "tentativas esgotadas em testando_sandbox: aprova: o job do sandbox não publicou resultado");
		return fechar();
	}

	const cenario = cenarios[nomeDoCenario];
	const duracao = chave === "cobranca" ? inteiro(52000, 96000) : inteiro(31000, 47000);
	relogio += duracao;
	const resultado = {
		attempt: 1,
		...resultadoDoSandbox(repositorio, cenario),
		duration_ms: duracao,
		created_at: iso(relogio),
	};
	analise.sandbox_results.push(resultado);
	relogio += inteiro(60, 240);
	transicao("testando_sandbox", "avaliando_risco");
	if (estado === "avaliando_risco") return fechar();

	analise.risk_gate = montarGate(cenario, resultado);
	relogio += inteiro(300, 1400);
	analise.decidido_em = relogio;
	const alto = analise.risk_gate.level === "alto";

	if (!alto) {
		transicao("avaliando_risco", "aplicando", `risco ${analise.risk_gate.level}`);
		if (estado === "aplicando") return fechar();
		relogio += inteiro(900, 2600);
		transicao("aplicando", "concluido", "resultado publicado");
		return fechar();
	}

	transicao("avaliando_risco", "aguardando_aprovacao", "risco alto");
	if (estado === "aguardando_aprovacao") return fechar();

	const revisor = revisores[inteiro(0, revisores.length - 1)];
	const origem = numeroPR % 2 === 0 ? "slack" : "github";
	relogio += inteiro(25, 600) * minuto;
	const aprovada = decisao === "aprovada";
	analise.approval = {
		decision: aprovada ? "aprovada" : "alteracoes_solicitadas",
		reviewer: revisor,
		origin: origem,
		feedback_reason: aprovada ? ["risco_aceito", "regra_conservadora", "falso_positivo"][numeroPR % 3] : null,
		decided_at: iso(relogio - inteiro(500, 3000)),
	};
	if (!aprovada) {
		transicao("aguardando_aprovacao", "rejeitado", `alterações solicitadas por ${revisor} via ${origem}`);
		return fechar();
	}
	transicao("aguardando_aprovacao", "aplicando", `aprovado por ${revisor} via ${origem}`);
	relogio += inteiro(900, 2600);
	transicao("aplicando", "concluido", "resultado publicado");
	return fechar();
}

const analises = roteiro.map(montarAnalise);

const { publicKey: chavePublica, privateKey: chavePrivada } = generateKeyPairSync("ed25519");
const chavePublicaBruta = chavePublica.export({ type: "spki", format: "der" }).subarray(-32);
const keyId = sha256(chavePublicaBruta).slice(0, 16);

const recibos = [];
const decididas = analises.filter((analise) => analise.risk_gate !== null).sort((a, b) => a.decidido_em - b.decidido_em);
for (const analise of decididas) {
	const anterior = recibos[recibos.length - 1] ?? null;
	const sandbox = analise.sandbox_results[analise.sandbox_results.length - 1];
	const emitidoEm = new Date(analise.decidido_em).toISOString().replace("Z", "000Z");
	const id = uuid();
	const payload = {
		versao: 1,
		id,
		task_run_id: analise.id,
		tenant_id: tenant,
		repositorio: analise.repository,
		numero_pr: analise.pr_number,
		head_sha: analise.head_sha,
		diff_sha256: hex(64),
		diff_truncado: false,
		sandbox: {
			compilou: sandbox.compiled,
			testes_passaram: sandbox.tests_passed,
			testes_falharam: sandbox.tests_failed,
			tipo_falha: sandbox.failure_type,
			tentativa: sandbox.attempt,
		},
		nivel_deterministico: analise.risk_gate.deterministic_level,
		nivel: analise.risk_gate.level,
		motivos: analise.risk_gate.rules_hit.map((motivo) => ({
			id: motivo.id,
			evidencias: motivo.evidencias,
		})),
		emitido_em: emitidoEm,
		anterior: anterior ? { id: anterior.id, hash: anterior.hash_sha256 } : null,
		key_id: keyId,
	};
	const bytes = Buffer.from(JSON.stringify(payload), "utf8");
	recibos.push({
		id,
		tenant_id: tenant,
		task_run_id: analise.id,
		key_id: keyId,
		hash_sha256: sha256(bytes),
		payload_base64: bytes.toString("base64"),
		assinatura_base64: sign(null, bytes, chavePrivada).toString("base64"),
		recibo_anterior_id: anterior ? anterior.id : null,
		signed_at: emitidoEm,
		payload,
	});
	analise.receipt_id = id;
}
for (const analise of analises) delete analise.decidido_em;

const instalacao = {
	tenant_id: tenant,
	conta: "Azulpasta",
	repositorios: Object.values(repositorios).map((repositorio) => repositorio.nome),
	periodo_dias: 30,
	gerado_em: iso(agora),
	limite_de_arquivos: 20,
};

const chaves = {
	chaves: [
		{
			key_id: keyId,
			algoritmo: "Ed25519",
			chave_publica_base64: chavePublicaBruta.toString("base64"),
		},
	],
};

mkdirSync(pastaDeDados, { recursive: true });
const gravar = (nome, conteudo) => writeFileSync(join(pastaDeDados, nome), `${JSON.stringify(conteudo, null, "\t")}\n`);
gravar("instalacao.json", instalacao);
gravar("analises.json", analises);
gravar("recibos.json", recibos);
gravar("chaves.json", chaves);
gravar("regras.json", regras);

const porEstado = Object.groupBy(analises, (analise) => analise.state);
console.log(`${analises.length} análises, ${recibos.length} recibos, key_id ${keyId}`);
console.log(
	Object.entries(porEstado)
		.map(([estado, lista]) => `${estado}: ${lista.length}`)
		.join(", "),
);

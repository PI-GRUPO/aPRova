import { formatDistanceStrict } from "date-fns";
import { ptBR } from "date-fns/locale";
import { Link } from "react-router";
import type { saudeRepositorios } from "@/servicos/analises";
import { SecaoGestao, TabelaGestao } from "./Gestao";
import { Botao } from "./Primitivos";

type SaudeDaInstalacao = Awaited<ReturnType<typeof saudeRepositorios>>;
type SaudeDoRepositorio = SaudeDaInstalacao["itens"][number];

const motivosDaProtecao = new Map([
	[
		"acesso_insuficiente",
		"O GitHub não expôs a configuração. Confira o acesso da sua conta e da instalação. A proteção clássica exige Administração (leitura).",
	],
	["check_sem_origem", "O check aPRova aparece na regra sem um App de origem confirmado."],
	["branch_ou_app_nao_informado", "O GitHub não informou a branch padrão ou o App da instalação."],
	["consulta_indisponivel", "O GitHub não respondeu à consulta de proteção. Tente atualizar."],
	["dados_incompletos", "O GitHub retornou dados incompletos sobre a proteção."],
]);
const rotulosDeAcesso = { none: "Não concedida", read: "Leitura", write: "Escrita", admin: "Administração" };
const horasSemAnalise = 24;

function linkDaProtecao(nome: string) {
	if (!/^[A-Za-z0-9][A-Za-z0-9-]{0,38}\/(?!\.{1,2}$)[A-Za-z0-9._-]{1,100}$/.test(nome)) return undefined;
	return `https://github.com/${nome}/settings/branches`;
}

function ProtecaoDaBranch({ nome, branch, check, motivo }: SaudeDoRepositorio) {
	if (check === true) return <span className="texto-quebrado">Obrigatório em {branch}</span>;
	const destino = linkDaProtecao(nome);
	return (
		<div className="saude-repo-celula">
			<strong className={check === false ? "texto-sem-protecao" : undefined}>
				{check === false ? `Não obrigatório em ${branch}` : "Não confirmado"}
			</strong>
			{check === null && <p className="apoio">{motivosDaProtecao.get(motivo ?? "") ?? "Confira a proteção no GitHub."}</p>}
			{destino && (
				<a href={destino} aria-label={`Ver proteção de ${nome} no GitHub`}>
					Ver proteção no GitHub
				</a>
			)}
		</div>
	);
}

function UltimaAnalise({ ultima, agora }: { ultima: string | null; agora: Date }) {
	const avisoWebhook = "Se houve PR recente, confira as entregas do webhook no GitHub.";
	if (!ultima) {
		return (
			<div className="saude-repo-celula">
				<span>Sem análises recebidas</span>
				<p className="apoio">{avisoWebhook}</p>
			</div>
		);
	}
	const data = new Date(ultima);
	const inativo = agora.getTime() - data.getTime() >= horasSemAnalise * 60 * 60_000;
	return (
		<div className="saude-repo-celula">
			<time dateTime={ultima}>{formatDistanceStrict(data, agora, { addSuffix: true, locale: ptBR })}</time>
			<p className="apoio">{data.toLocaleString("pt-BR", { timeZone: "UTC" })} UTC</p>
			{inativo && (
				<p className="apoio">
					Sem análises nas últimas {horasSemAnalise} h. {avisoWebhook}
				</p>
			)}
		</div>
	);
}

function PermissoesDaInstalacao({ permissoes }: Pick<SaudeDaInstalacao, "permissoes">) {
	return (
		<SecaoGestao titulo="Permissões do GitHub App">
			<dl className="permissoes-instalacao">
				{permissoes.map(({ chave, nome, minimo, concedida, suficiente, uso }) => (
					<div key={chave}>
						<dt>{nome}</dt>
						<dd>
							<strong className={suficiente === false ? "texto-sem-protecao" : undefined}>
								{suficiente === false
									? `Falta ${rotulosDeAcesso[minimo].toLowerCase()}`
									: (concedida && rotulosDeAcesso[concedida]) || "Não informada"}
							</strong>
							{suficiente === false && <span> · Atual: {rotulosDeAcesso[concedida ?? "none"]}</span>}
							<p className="apoio">{uso}</p>
						</dd>
					</div>
				))}
			</dl>
			{permissoes.some((p) => p.suficiente === false) && (
				<p className="apoio">Peça a quem administra o GitHub App para revisar as permissões e confirme o novo acesso na instalação.</p>
			)}
		</SecaoGestao>
	);
}

export function SaudeDosRepositorios({
	instalacao,
	dados,
	atualizar,
}: {
	instalacao: string;
	dados: SaudeDaInstalacao;
	atualizar: () => void;
}) {
	const agora = new Date();
	const semProtecao = dados.itens.filter((repo) => repo.check === false).length;
	return (
		<>
			<SecaoGestao
				titulo="Saúde por repositório"
				ajuda="A consulta verifica se a branch padrão exige o check publicado por esta instalação do aPRova."
			>
				{semProtecao > 0 && (
					<div className="alerta-saude-repositorios" role="alert">
						<strong>
							{semProtecao === 1
								? "O check aPRova não é obrigatório em um repositório desta página."
								: `O check aPRova não é obrigatório em ${semProtecao} repositórios desta página.`}
						</strong>
						<p>Sem essa exigência, um resultado de risco alto do aPRova pode não impedir o merge.</p>
					</div>
				)}
				{dados.itens.some((repo) => repo.check !== true) && (
					<p className="apoio">
						Para exigir o check, abra Settings no GitHub e edite a proteção ou o ruleset da branch padrão. Em Require status checks to pass,
						adicione o check aPRova e selecione o App aPRova como origem.
					</p>
				)}
				{dados.itens.length ? (
					<TabelaGestao
						compacta
						legenda="Saúde por repositório"
						colunas={["Repositório", "Check obrigatório", "Última análise"]}
						linhas={dados.itens.map((repo) => ({
							id: repo.nome,
							celulas: [
								<Link
									key="repo"
									className="texto-quebrado"
									to={`/analises?${new URLSearchParams({ installation_id: instalacao, repositorio: repo.nome })}`}
								>
									{repo.nome}
								</Link>,
								<ProtecaoDaBranch key="protecao" {...repo} />,
								<UltimaAnalise key="ultima" ultima={repo.ultima} agora={agora} />,
							],
						}))}
					/>
				) : (
					<p className="apoio">
						{dados.total === 0 ? "Nenhum repositório ativo disponível nesta instalação." : "Nenhum repositório nesta página."}
					</p>
				)}
				<div className="acoes-gestao">
					<Botao variante="primario" onClick={atualizar}>
						Atualizar repositórios
					</Botao>
				</div>
			</SecaoGestao>
			<PermissoesDaInstalacao permissoes={dados.permissoes} />
		</>
	);
}

// Package slackhome apresenta a App Home a partir de interações autenticadas.
package slackhome

import (
	"context"
	"html"
	"net/url"
	"strings"
)

type Secao string

const (
	Geral        Secao = "geral"
	Pendencias   Secao = "pendencias"
	Detalhes     Secao = "detalhes"
	Historico    Secao = "historico"
	Configuracao Secao = "configuracao"
)

var secoes = []struct {
	id                         Secao
	titulo, descricao, caminho string
}{
	{Geral, "Visão geral", "O resumo de repositórios e execuções depende da confirmação do seu acesso pelo núcleo.", "/repositorios"},
	{Pendencias, "Pendências", "A fila, o motivo do risco e o tempo de espera exigem acesso confirmado ao repositório.", "/analises"},
	{Detalhes, "Detalhes", "No dashboard, selecione uma análise para consultar commit, testes, gate, decisões e recibo disponíveis.", "/analises"},
	{Historico, "Histórico e relatórios", "O relatório semanal cobre sete dias UTC. Consulte o histórico e as métricas com sua sessão GitHub.", "/relatorio-semanal"},
	{Configuracao, "Integrações e configuração", "A configuração pelo Slack aguarda vínculo individual, permissões e gravação no núcleo. O Trello é opcional.", "/conta"},
}

func secaoValida(s Secao) bool {
	for _, item := range secoes {
		if item.id == s {
			return true
		}
	}
	return false
}

type bloco = map[string]any

func texto(s string) bloco     { return bloco{"type": "plain_text", "text": s} }
func paragrafo(s string) bloco { return bloco{"type": "section", "text": texto(s)} }
func botao(id, nome, valor string) bloco {
	return bloco{"type": "button", "action_id": "aprova_home_" + id, "text": texto(nome), "value": valor}
}
func opcoes() []bloco {
	var lista []bloco
	for _, item := range secoes {
		lista = append(lista, bloco{"text": texto(item.titulo), "value": item.id})
	}
	return lista
}

// Filtro é apenas apresentação. O dashboard resolve a instalação com a sessão do navegador.
type Filtro struct {
	Secao   Secao
	Periodo string
}

func periodoValido(p string) bool { return p == "7" || p == "30" || p == "90" || p == "todos" }
func filtroDaView(meta string) (Filtro, bool) {
	secao, periodo, temPeriodo := strings.Cut(meta, ":")
	if !temPeriodo {
		periodo = "30"
	}
	f := Filtro{Secao(secao), periodo}
	return f, secaoValida(f.Secao) && periodoValido(f.Periodo)
}
func (f Filtro) metadata() string { return string(f.Secao) + ":" + f.Periodo }
func (f Filtro) caminhoAnalises() string {
	q := url.Values{"periodo": {f.Periodo}}
	if f.Secao == Pendencias {
		q.Set("estado", "aguardando_aprovacao")
	}
	return "/analises?" + q.Encode()
}

type Pessoa struct{ Workspace, Usuario string }

// Cada capacidade deve ser consultada com a identidade do evento. A porta não
// recebe instalação, repositório nem credencial fornecida pelo cliente Slack.
type Recurso string

const (
	Vinculo      Recurso = "vinculo"
	Catalogo     Recurso = "catalogo"
	Fila         Recurso = "pendencias"
	Evidencias   Recurso = "analise_e_recibo"
	Metricas     Recurso = "historico_e_relatorios"
	Decisoes     Recurso = "acoes"
	Preferencias Recurso = "preferencias_sla_onboarding"
	Integracoes  Recurso = "integracoes_e_alertas"
)

type Situacao int

const (
	SemContrato Situacao = iota
	SemVinculo
	SemPermissao
	FalhaConsulta
	ListaVazia
)

// Resultado descreve a capacidade da interface. ListaVazia exige leitura
// autorizada, concluída e correlacionada pelo núcleo para o titular.
type Resultado struct {
	Titular  Pessoa
	Recurso  Recurso
	Situacao Situacao
}

type AcessoIndividual interface {
	Consultar(context.Context, Pessoa, Recurso) Resultado
}
type acessoPendente struct{}

func (acessoPendente) Consultar(_ context.Context, p Pessoa, recurso Recurso) Resultado {
	return Resultado{p, recurso, SemContrato}
}

func aviso(s Situacao) string {
	switch s {
	case SemVinculo:
		return "Sua conta Slack ainda não está vinculada ao GitHub."
	case SemPermissao:
		return "Você não tem permissão para consultar estes dados."
	case FalhaConsulta:
		return "Falha ao consultar seu acesso. Atualize a Home para tentar novamente."
	case ListaVazia:
		return "Nenhum item no recorte autorizado."
	default:
		return "A API ainda não confirma o acesso individual pelo Slack."
	}
}

var recursos = map[Secao][]Recurso{
	Geral: {Vinculo, Catalogo}, Pendencias: {Fila}, Detalhes: {Evidencias, Decisoes},
	Historico: {Metricas}, Configuracao: {Preferencias, Integracoes},
}

type View struct {
	Type     string  `json:"type"`
	Callback string  `json:"callback_id,omitempty"`
	Metadata string  `json:"private_metadata,omitempty"`
	Blocks   []bloco `json:"blocks"`
}

func construir(ctx context.Context, c Config, pessoa Pessoa, filtro Filtro, acesso AcessoIndividual, estado EstadoPublico) View {
	secao := filtro.Secao
	v := View{Type: "home", Callback: "aprova_home", Metadata: filtro.metadata()}
	destinoDM := "https://slack.com/app_redirect?" + url.Values{"app": {c.AppID}, "team": {pessoa.Workspace}}.Encode()
	v.Blocks = append(v.Blocks,
		bloco{"type": "header", "text": texto("aPRova")},
		bloco{"type": "section", "text": bloco{"type": "mrkdwn", "text": "<" + html.EscapeString(destinoDM) + "|Conversar com o aPRova> · Consulte a saúde da API na aba Mensagens."}},
	)

	lista := opcoes()
	for i, item := range secoes {
		if item.id != secao {
			continue
		}
		v.Blocks = append(v.Blocks, bloco{"type": "actions", "elements": []bloco{{"type": "static_select", "action_id": "aprova_home_secao", "placeholder": texto("Escolher seção"), "options": lista, "initial_option": lista[i]}}}, bloco{"type": "header", "text": texto(item.titulo)})
		if !estado.Carregando {
			v.Blocks = append(v.Blocks, paragrafo(item.descricao))
			situacao := SemContrato
			for _, recurso := range recursos[secao] {
				resultado := acesso.Consultar(ctx, pessoa, recurso)
				situacao = resultado.Situacao
				if resultado.Titular != pessoa || resultado.Recurso != recurso || (situacao == ListaVazia && recurso != Fila && recurso != Catalogo) {
					situacao = FalhaConsulta
				}
				if situacao != SemContrato {
					break
				}
			}
			v.Blocks = append(v.Blocks, paragrafo(aviso(situacao)))
			if estado.Login == Disponivel && c.WebURL != "" {
				caminho := item.caminho
				if secao == Pendencias || secao == Detalhes {
					caminho = filtro.caminhoAnalises()
				}
				links := "<" + html.EscapeString(c.WebURL+caminho) + "|Abrir no dashboard com minha conta GitHub>"
				if secao == Historico {
					links += " · <" + html.EscapeString(c.WebURL+filtro.caminhoAnalises()) + "|Ver análises do período>"
				}
				v.Blocks = append(v.Blocks, bloco{"type": "section", "text": bloco{"type": "mrkdwn", "text": links}})
			}
		}
	}
	if estado.Carregando {
		v.Blocks = append(v.Blocks, paragrafo("Consultando a API…"))
		return v
	}
	v.Blocks = append(v.Blocks, paragrafo(estado.descricao()), bloco{"type": "actions", "elements": []bloco{botao("atualizar", "Atualizar", string(secao)), botao("navegar", "Seção e período…", string(secao))}})
	return v
}
func modal(hash string, atual Filtro) bloco {
	secoesDisponiveis := opcoes()
	var inicial bloco
	for i, item := range secoes {
		if item.id == atual.Secao {
			inicial = secoesDisponiveis[i]
		}
	}
	var periodos []bloco
	var periodoAtual bloco
	for _, p := range []string{"7", "30", "90", "todos"} {
		nome := "Últimos " + p + " dias"
		if p == "todos" {
			nome = "Todo o histórico"
		}
		opcao := bloco{"text": texto(nome), "value": p}
		periodos = append(periodos, opcao)
		if atual.Periodo == p {
			periodoAtual = opcao
		}
	}
	periodo := bloco{"type": "input", "block_id": "intervalo", "label": texto("Período para abrir a lista de análises no dashboard"), "element": bloco{"type": "static_select", "action_id": "periodo", "options": periodos, "initial_option": periodoAtual}}
	return bloco{"type": "modal", "callback_id": "aprova_home_navegacao", "private_metadata": hash, "title": texto("Navegar no aPRova"), "submit": texto("Abrir"), "close": texto("Cancelar"), "blocks": []bloco{{"type": "input", "block_id": "destino", "label": texto("Qual seção deseja abrir?"), "element": bloco{"type": "static_select", "action_id": "secao", "options": secoesDisponiveis, "initial_option": inicial, "placeholder": texto("Escolher seção")}}, periodo}}
}

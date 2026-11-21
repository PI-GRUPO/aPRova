package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/acaoremota"
	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/saude"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const tempoLimiteRequisicao = 30 * time.Second

// Dependencia identifica um serviço externo consultado pela rota de prontidão.
type Dependencia = saude.Dependencia

// RegistroDeExecucoes descreve a persistência exigida pelo handler de webhook.
type RegistroDeExecucoes interface {
	RegistrarPullRequest(ctx context.Context, evento domain.EventoPullRequest) (domain.Execucao, error)
	RegistrarRevisao(ctx context.Context, decisao domain.DecisaoDeReview) error
	RegistrarFechamento(ctx context.Context, fechamento domain.FechamentoDePR) error
}

// ProcessadorDeAcoesRemotas decide e executa uma ação pedida por canal
// externo, a partir do corpo já autenticado.
type ProcessadorDeAcoesRemotas interface {
	Processar(ctx context.Context, corpo []byte) acaoremota.Decisao
}

// AvisoDeFila mostra no pull request que a análise do task_run novo espera o
// teto da instalação. Falha no aviso não impede o task_run de seguir na fila.
type AvisoDeFila interface {
	AvisarSeAcimaDoTeto(ctx context.Context, taskRun domain.TaskRun) error
}

type FonteDeRelatorios interface {
	RelatorioSemanal(ctx context.Context, tenantID int64, inicio, fim time.Time) (domain.RelatorioSemanal, error)
}

// Opcoes reúne as dependências com que o servidor HTTP é construído.
type Opcoes struct {
	Dependencias        []Dependencia
	SegredoWebhook      []byte
	Execucoes           RegistroDeExecucoes
	AvisoDeFila         AvisoDeFila
	SegredoAcoesRemotas []byte
	AcoesRemotas        ProcessadorDeAcoesRemotas
	Relatorios          FonteDeRelatorios
	ConsultasAnalises   FonteDeAnalises
	Recibos             VerificadorDeRecibos
	Desfechos           FonteDeDesfechos
	Commits             LeitorDeCommits
	URLPublica          string
	SlugDoApp           string
	Agora               func() time.Time

	TetoDeLeiturasPorMinuto int
	ProxyConfiavel          bool
}

// Servidor agrupa as dependências externas usadas pelos handlers HTTP.
type Servidor struct {
	dependencias        []Dependencia
	segredoWebhook      []byte
	execucoes           RegistroDeExecucoes
	avisoDeFila         AvisoDeFila
	segredoAcoesRemotas []byte
	acoesRemotas        ProcessadorDeAcoesRemotas
	relatorios          FonteDeRelatorios
	consultasAnalises   FonteDeAnalises
	recibos             VerificadorDeRecibos
	desfechos           FonteDeDesfechos
	commits             LeitorDeCommits
	urlPublica          string
	slugDoApp           string
	agora               func() time.Time
	limiteDeLeituras    *limitePorIP
}

// NovoServidor constrói o servidor HTTP a partir das opções informadas.
func NovoServidor(opcoes Opcoes) *Servidor {
	agora := opcoes.Agora
	if agora == nil {
		agora = time.Now
	}

	return &Servidor{
		dependencias:        opcoes.Dependencias,
		segredoWebhook:      opcoes.SegredoWebhook,
		execucoes:           opcoes.Execucoes,
		avisoDeFila:         opcoes.AvisoDeFila,
		segredoAcoesRemotas: opcoes.SegredoAcoesRemotas,
		acoesRemotas:        opcoes.AcoesRemotas,
		relatorios:          opcoes.Relatorios,
		consultasAnalises:   opcoes.ConsultasAnalises,
		recibos:             opcoes.Recibos,
		desfechos:           opcoes.Desfechos,
		commits:             opcoes.Commits,
		urlPublica:          strings.TrimRight(opcoes.URLPublica, "/"),
		slugDoApp:           opcoes.SlugDoApp,
		agora:               agora,
		limiteDeLeituras:    novoLimitePorIP(opcoes.TetoDeLeiturasPorMinuto, opcoes.ProxyConfiavel, agora),
	}
}

// Rotas devolve o roteador com os middlewares e as rotas registradas.
func (s *Servidor) Rotas() http.Handler {
	roteador := chi.NewRouter()

	roteador.Use(identificarRequisicao)
	roteador.Use(registrarAcesso)
	roteador.Use(middleware.Recoverer)
	roteador.Use(middleware.Timeout(tempoLimiteRequisicao))

	roteador.Get("/health", saude.Viva)
	roteador.Get("/health/live", saude.Viva)
	roteador.Get("/ready", saude.Pronta(s.dependencias))
	roteador.Get("/health/ready", saude.Pronta(s.dependencias))
	roteador.Post("/webhooks/github", s.manipularWebhookGitHub)
	roteador.Post("/acoes-remotas", s.manipularAcaoRemota)
	roteador.Get("/reports/weekly", s.manipularRelatorioSemanal)
	roteador.Get("/api/analises", s.manipularListaAnalises)
	roteador.Get("/api/analises/{id}", s.manipularConsultaAnalise)
	roteador.Route("/receipts", func(recibos chi.Router) {
		recibos.Use(s.limiteDeLeituras.segurar)
		recibos.Get("/keys", s.manipularChavesDeRecibo)
		recibos.Get("/{id}", s.manipularRecibo)
		recibos.Get("/{id}/verify", s.manipularVerificacaoDeRecibo)
		recibos.Get("/{id}/revert", s.manipularReversaoDeRecibo)
	})

	return roteador
}

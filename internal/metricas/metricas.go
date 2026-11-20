// Package metricas define as métricas do aPRova no formato do Prometheus.
// Os rótulos são sempre de baixa cardinalidade: estado, nível, motivo,
// resultado, etapa, linguagem, tipo de falha e tipo de erro. Nunca task_run_id,
// número de PR ou SHA.
package metricas

import (
	"errors"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/platform"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

// EtapaPublicacao é o rótulo da duração das chamadas que publicam o resultado
// no pull request, que não são um estado do task_run.
const EtapaPublicacao = "publicacao"

const (
	nivelAusente    = "sem_nivel"
	erroTransitorio = "transitorio"
	erroDePermissao = "permissao"
	erroDefinitivo  = "definitivo"
	prefixoDosNomes = "aprova_"
)

var limitesDaDuracaoDaEtapa = []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120, 300}

// Motor reúne as métricas que o worker atualiza ao conduzir task_runs.
type Motor struct {
	analises        *prometheus.CounterVec
	motivos         *prometheus.CounterVec
	calibracoes     *prometheus.CounterVec
	duracaoDaEtapa  *prometheus.HistogramVec
	execucoes       *prometheus.CounterVec
	novasTentativas *prometheus.CounterVec
	recibos         prometheus.Counter
	leasesRetomados prometheus.Counter
	errosDoGitHub   *prometheus.CounterVec
}

// NovoMotor cria as métricas do motor e as registra em registro.
func NovoMotor(registro prometheus.Registerer) *Motor {
	m := &Motor{
		analises: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: prefixoDosNomes + "analises_total",
			Help: "Análises encerradas, por estado final e nível de risco.",
		}, []string{"estado", "nivel"}),
		motivos: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: prefixoDosNomes + "motivos_do_gate_total",
			Help: "Motivos acionados pelo gate de risco, por id do motivo.",
		}, []string{"motivo"}),
		calibracoes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: prefixoDosNomes + "calibracoes_total",
			Help: "Calibrações por histórico, pelo resultado gravado em calibration_status.",
		}, []string{"resultado"}),
		duracaoDaEtapa: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    prefixoDosNomes + "duracao_da_etapa_segundos",
			Help:    "Duração de cada passagem do motor por uma etapa, em segundos.",
			Buckets: limitesDaDuracaoDaEtapa,
		}, []string{"etapa"}),
		execucoes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: prefixoDosNomes + "execucoes_do_sandbox_total",
			Help: "Execuções do sandbox, por linguagem e tipo de falha.",
		}, []string{"linguagem", "tipo_falha"}),
		novasTentativas: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: prefixoDosNomes + "novas_tentativas_total",
			Help: "Novas tentativas agendadas depois de uma falha transitória, por etapa.",
		}, []string{"etapa"}),
		recibos: prometheus.NewCounter(prometheus.CounterOpts{
			Name: prefixoDosNomes + "recibos_emitidos_total",
			Help: "Recibos assinados emitidos.",
		}),
		leasesRetomados: prometheus.NewCounter(prometheus.CounterOpts{
			Name: prefixoDosNomes + "leases_vencidos_retomados_total",
			Help: "Task_runs reivindicados de um trabalhador cujo prazo venceu.",
		}),
		errosDoGitHub: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: prefixoDosNomes + "erros_github_total",
			Help: "Erros da API do GitHub vistos pelo motor, por tipo.",
		}, []string{"tipo"}),
	}
	registro.MustRegister(m.analises, m.motivos, m.calibracoes, m.duracaoDaEtapa, m.execucoes,
		m.novasTentativas, m.recibos, m.leasesRetomados, m.errosDoGitHub)
	return m
}

// AnaliseEncerrada conta um task_run que chegou ao estado final, com o nível
// do gate ou sem_nivel quando não houve decisão.
func (m *Motor) AnaliseEncerrada(estado domain.EstadoTaskRun, nivel domain.NivelRisco) {
	rotuloDoNivel := string(nivel)
	if rotuloDoNivel == "" {
		rotuloDoNivel = nivelAusente
	}
	m.analises.WithLabelValues(string(estado), rotuloDoNivel).Inc()
}

// MotivosDoGate conta cada motivo acionado numa decisão do gate.
func (m *Motor) MotivosDoGate(motivos []domain.MotivoDeRisco) {
	for _, motivo := range motivos {
		m.motivos.WithLabelValues(motivo.ID).Inc()
	}
}

// Calibracao conta uma calibração pelo estado gravado.
func (m *Motor) Calibracao(estado domain.EstadoCalibracao) {
	m.calibracoes.WithLabelValues(string(estado)).Inc()
}

// DuracaoDaEtapa observa quanto durou uma passagem pela etapa.
func (m *Motor) DuracaoDaEtapa(etapa string, duracao time.Duration) {
	m.duracaoDaEtapa.WithLabelValues(etapa).Observe(duracao.Seconds())
}

// ExecucaoDoSandbox conta uma execução do sandbox que devolveu resultado.
func (m *Motor) ExecucaoDoSandbox(linguagem sandbox.Linguagem, tipo sandbox.TipoFalha) {
	m.execucoes.WithLabelValues(string(linguagem), string(tipo)).Inc()
}

// NovaTentativa conta uma nova tentativa agendada na etapa.
func (m *Motor) NovaTentativa(etapa domain.EstadoTaskRun) {
	m.novasTentativas.WithLabelValues(string(etapa)).Inc()
}

// ReciboEmitido conta um recibo assinado.
func (m *Motor) ReciboEmitido() {
	m.recibos.Inc()
}

// LeaseVencidoRetomado conta um task_run reivindicado depois de o prazo de
// outro trabalhador vencer.
func (m *Motor) LeaseVencidoRetomado() {
	m.leasesRetomados.Inc()
}

// ErroDoGitHub conta um erro da API do GitHub como transitorio, permissao ou
// definitivo.
func (m *Motor) ErroDoGitHub(err error) {
	m.errosDoGitHub.WithLabelValues(tipoDoErroDoGitHub(err)).Inc()
}

func tipoDoErroDoGitHub(err error) string {
	switch {
	case errors.Is(err, platform.ErrGitHubIndisponivel), errors.Is(err, platform.ErrLimiteDeRequisicoes):
		return erroTransitorio
	case errors.Is(err, platform.ErrSemPermissao):
		return erroDePermissao
	default:
		return erroDefinitivo
	}
}

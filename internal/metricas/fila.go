package metricas

import (
	"context"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const tempoLimiteDaContagem = 2 * time.Second

// ContagemDaFila devolve quantos task_runs esperam na fila.
type ContagemDaFila func(context.Context) (int, error)

type coletorDaFila struct {
	descricao *prometheus.Desc
	contar    ContagemDaFila
}

// RegistrarTamanhoDaFila registra aprova_fila_pendentes, lida a cada coleta
// por contar. Se a contagem falha, a métrica fica de fora daquela coleta.
func RegistrarTamanhoDaFila(registro prometheus.Registerer, contar ContagemDaFila) {
	registro.MustRegister(coletorDaFila{
		descricao: prometheus.NewDesc(prefixoDosNomes+"fila_pendentes", "Task_runs pendentes na fila no momento da coleta.", nil, nil),
		contar:    contar,
	})
}

func (c coletorDaFila) Describe(descricoes chan<- *prometheus.Desc) {
	descricoes <- c.descricao
}

func (c coletorDaFila) Collect(metricas chan<- prometheus.Metric) {
	ctx, cancelar := context.WithTimeout(context.Background(), tempoLimiteDaContagem)
	defer cancelar()

	pendentes, err := c.contar(ctx)
	if err != nil {
		slog.Warn("tamanho da fila indisponível na coleta de métricas", "erro", err)
		return
	}
	metricas <- prometheus.MustNewConstMetric(c.descricao, prometheus.GaugeValue, float64(pendentes))
}

// Package observabilidade monta o listener interno do processo, com as
// métricas no formato do Prometheus e as rotas de saúde.
package observabilidade

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/PI-GRUPO/aPRova/internal/saude"
)

const tempoLimiteDaLeituraDoCabecalho = 5 * time.Second

// NovoRegistro devolve um registro com as métricas do runtime do Go e do
// processo, onde as métricas do aPRova também são registradas.
func NovoRegistro() *prometheus.Registry {
	registro := prometheus.NewRegistry()
	registro.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	return registro
}

// NovoServidor devolve o servidor do listener interno em endereco, com
// /metrics lido de coletor, /health/live e /health/ready sobre dependencias.
func NovoServidor(endereco string, coletor prometheus.Gatherer, dependencias []saude.Dependencia) *http.Server {
	rotas := http.NewServeMux()
	rotas.Handle("GET /metrics", promhttp.HandlerFor(coletor, promhttp.HandlerOpts{}))
	rotas.HandleFunc("GET /health/live", saude.Viva)
	rotas.Handle("GET /health/ready", saude.Pronta(dependencias))

	return &http.Server{
		Addr:              endereco,
		Handler:           rotas,
		ReadHeaderTimeout: tempoLimiteDaLeituraDoCabecalho,
	}
}

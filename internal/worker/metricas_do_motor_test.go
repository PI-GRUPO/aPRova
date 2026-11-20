package worker

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/metricas"
	"github.com/PI-GRUPO/aPRova/internal/platform"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

type rotulos map[string]string

func metricaComRotulos(t *testing.T, registro *prometheus.Registry, nome string, esperados rotulos) *dto.Metric {
	t.Helper()

	familias, err := registro.Gather()
	if err != nil {
		t.Fatalf("coletar métricas: %v", err)
	}
	for _, familia := range familias {
		if familia.GetName() != nome {
			continue
		}
		for _, metrica := range familia.GetMetric() {
			if temOsRotulos(metrica, esperados) {
				return metrica
			}
		}
	}
	return nil
}

func temOsRotulos(metrica *dto.Metric, esperados rotulos) bool {
	encontrados := 0
	for _, par := range metrica.GetLabel() {
		if valor, existe := esperados[par.GetName()]; existe {
			if valor != par.GetValue() {
				return false
			}
			encontrados++
		}
	}
	return encontrados == len(esperados)
}

func contador(t *testing.T, registro *prometheus.Registry, nome string, esperados rotulos) float64 {
	t.Helper()

	metrica := metricaComRotulos(t, registro, nome, esperados)
	if metrica == nil {
		return 0
	}
	return metrica.GetCounter().GetValue()
}

func observacoes(t *testing.T, registro *prometheus.Registry, nome string, esperados rotulos) uint64 {
	t.Helper()

	metrica := metricaComRotulos(t, registro, nome, esperados)
	if metrica == nil {
		return 0
	}
	return metrica.GetHistogram().GetSampleCount()
}

func exigirContador(t *testing.T, registro *prometheus.Registry, nome string, esperados rotulos, valor float64) {
	t.Helper()

	if obtido := contador(t, registro, nome, esperados); obtido != valor {
		t.Errorf("%s%v = %v, esperava %v", nome, esperados, obtido, valor)
	}
}

func cenarioComMetricas(nivel domain.NivelRisco, resultados ...sandbox.ResultadoSandbox) (*cenario, *prometheus.Registry) {
	c := novoCenario(nivel, resultados...)
	registro := prometheus.NewRegistry()
	c.opcoes.Metricas = metricas.NovoMotor(registro)
	return c, registro
}

func TestAnaliseConcluidaAtualizaAsMetricasDoMotor(t *testing.T) {
	c, registro := cenarioComMetricas(domain.RiscoBaixo)
	c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	exigirContador(t, registro, "aprova_analises_total", rotulos{"estado": "concluido", "nivel": "baixo"}, 1)
	exigirContador(t, registro, "aprova_recibos_emitidos_total", nil, 1)
	exigirContador(t, registro, "aprova_calibracoes_total", rotulos{"resultado": "desligada"}, 1)
	exigirContador(t, registro, "aprova_execucoes_do_sandbox_total", rotulos{"linguagem": "go", "tipo_falha": "nenhuma"}, 1)
	for _, etapa := range []string{"mapeando_contexto", "testando_sandbox", "avaliando_risco", "publicacao"} {
		if observacoes(t, registro, "aprova_duracao_da_etapa_segundos", rotulos{"etapa": etapa}) == 0 {
			t.Errorf("a duração da etapa %s não foi observada", etapa)
		}
	}
}

func TestMotivosDoGateSaoContadosPorID(t *testing.T) {
	c, registro := cenarioComMetricas(domain.RiscoMedio)
	c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	exigirContador(t, registro, "aprova_motivos_do_gate_total", rotulos{"motivo": "area_sensivel"}, 1)
	exigirContador(t, registro, "aprova_analises_total", rotulos{"estado": "concluido", "nivel": "medio"}, 1)
}

func TestAnaliseRejeitadaContaComONivelDoGate(t *testing.T) {
	c, registro := cenarioComMetricas(domain.RiscoAlto)
	taskRun := c.armazenamento.novoTaskRun(14)
	c.conduzirTudo(t)

	c.armazenamento.registrarAprovacao(taskRun.ID, domain.DecisaoAlteracoesSolicitadas, "httpslua", domain.OrigemGitHub)
	c.conduzirTudo(t)

	exigirContador(t, registro, "aprova_analises_total", rotulos{"estado": "rejeitado", "nivel": "alto"}, 1)
}

func TestFalhaDeInfraestruturaContaExecucoesTentativasEAnaliseSemNivel(t *testing.T) {
	falha := sandbox.ResultadoSandbox{SaidaBruta: "docker indisponível", TipoFalha: sandbox.FalhaInfraestrutura}
	c, registro := cenarioComMetricas(domain.RiscoBaixo, falha, falha, falha)
	c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	exigirContador(t, registro, "aprova_execucoes_do_sandbox_total", rotulos{"linguagem": "go", "tipo_falha": "infraestrutura"}, 3)
	exigirContador(t, registro, "aprova_novas_tentativas_total", rotulos{"etapa": "testando_sandbox"}, 2)
	exigirContador(t, registro, "aprova_analises_total", rotulos{"estado": "falhou", "nivel": "sem_nivel"}, 1)
}

func TestErrosDoGitHubSaoContadosPorTipo(t *testing.T) {
	c, registro := cenarioComMetricas(domain.RiscoBaixo)
	c.leitor.erros = []error{platform.ErrGitHubIndisponivel, platform.ErrLimiteDeRequisicoes, platform.ErrSemPermissao}
	c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	exigirContador(t, registro, "aprova_erros_github_total", rotulos{"tipo": "transitorio"}, 2)
	exigirContador(t, registro, "aprova_erros_github_total", rotulos{"tipo": "permissao"}, 1)
	exigirContador(t, registro, "aprova_novas_tentativas_total", rotulos{"etapa": "mapeando_contexto"}, 2)
}

func TestPoolContaLeaseVencidoRetomado(t *testing.T) {
	c, registro := cenarioComMetricas(domain.RiscoBaixo)
	taskRun := c.armazenamento.novoTaskRun(14)
	c.armazenamento.marcarLeaseVencido(taskRun.ID)
	pool, err := NovoPool(OpcoesPool{
		Motor: c.motor(t), Armazenamento: c.armazenamento, Nome: "pool-de-teste",
		IntervaloOcioso: time.Millisecond, Metricas: c.opcoes.Metricas,
	})
	if err != nil {
		t.Fatalf("montar o pool: %v", err)
	}

	go pool.Rodar(t.Context())
	for range 200 {
		if c.armazenamento.taskRun(taskRun.ID).Estado.Terminal() {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	pool.Parar(time.Second)

	exigirContador(t, registro, "aprova_leases_vencidos_retomados_total", nil, 1)
}

package integracaopostgres

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/metricas"
	"github.com/PI-GRUPO/aPRova/internal/observabilidade"
	"github.com/PI-GRUPO/aPRova/internal/worker"
)

func lerMetricas(t *testing.T, servidor *http.Server) string {
	t.Helper()

	gravador := httptest.NewRecorder()
	servidor.Handler.ServeHTTP(gravador, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/metrics", nil))
	corpo, _ := io.ReadAll(gravador.Body)
	return string(corpo)
}

func TestAnaliseCompletaFazAsMetricasSubirem(t *testing.T) {
	banco := novoBancoDaFila(t)
	registro := observabilidade.NovoRegistro()
	metricasDoMotor := metricas.NovoMotor(registro)
	metricas.RegistrarTamanhoDaFila(registro, banco.repo.ContarPendentes)
	servidor := observabilidade.NovoServidor(":0", registro, nil)
	motor, _ := motorQuePublicaNoPR(t, banco, domain.RiscoBaixo, func(opcoes *worker.Opcoes) { opcoes.Metricas = metricasDoMotor })
	banco.novoTaskRunDoPR(t, 14)

	antes := lerMetricas(t, servidor)
	conduzirAteParar(t, banco, motor)
	depois := lerMetricas(t, servidor)

	if !strings.Contains(antes, "aprova_fila_pendentes 1\n") {
		t.Errorf("antes da análise a fila tem um pendente:\n%s", antes)
	}
	for _, esperada := range []string{
		`aprova_analises_total{estado="concluido",nivel="baixo"} 1`,
		"aprova_recibos_emitidos_total 1",
		`aprova_calibracoes_total{resultado="desligada"} 1`,
		`aprova_execucoes_do_sandbox_total{linguagem="go",tipo_falha="nenhuma"} 1`,
		`aprova_duracao_da_etapa_segundos_count{etapa="testando_sandbox"}`,
		`aprova_duracao_da_etapa_segundos_count{etapa="publicacao"}`,
		"aprova_fila_pendentes 0\n",
	} {
		if !strings.Contains(depois, esperada) {
			t.Errorf("/metrics depois da análise não tem %q", esperada)
		}
	}
}

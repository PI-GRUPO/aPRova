package integracaopostgres

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/PI-GRUPO/aPRova/internal/observabilidade"
	"github.com/PI-GRUPO/aPRova/internal/saude"
)

func TestHealthReadyFalhaComOBancoFora(t *testing.T) {
	banco := novoBancoDaFila(t)
	servidor := observabilidade.NovoServidor(":0", prometheus.NewRegistry(), []saude.Dependencia{
		{Nome: "postgres", Verificar: banco.repo.ConferirVersaoDoBanco},
	})
	pronto := func() int {
		gravador := httptest.NewRecorder()
		servidor.Handler.ServeHTTP(gravador, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/health/ready", nil))
		return gravador.Code
	}

	if status := pronto(); status != http.StatusOK {
		t.Fatalf("com o banco migrado na versão esperada, /health/ready responde 200, obtive %d", status)
	}
	banco.pool.Close()
	if status := pronto(); status != http.StatusServiceUnavailable {
		t.Errorf("com o banco fora, /health/ready responde 503, obtive %d", status)
	}
}

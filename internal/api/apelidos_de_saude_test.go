package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEReadySaoApelidosDeHealthLiveEHealthReady(t *testing.T) {
	rotas := NovoServidor(Opcoes{Dependencias: []Dependencia{dependenciaQuebrada("postgres")}}).Rotas()
	status := func(caminho string) int {
		gravador := httptest.NewRecorder()
		rotas.ServeHTTP(gravador, httptest.NewRequestWithContext(t.Context(), http.MethodGet, caminho, nil))
		return gravador.Code
	}

	if status("/health/live") != http.StatusOK || status("/health") != http.StatusOK {
		t.Errorf("/health e /health/live respondem 200: %d e %d", status("/health"), status("/health/live"))
	}
	if status("/health/ready") != http.StatusServiceUnavailable || status("/ready") != http.StatusServiceUnavailable {
		t.Errorf("/ready e /health/ready respondem 503 com o banco fora: %d e %d", status("/ready"), status("/health/ready"))
	}
}

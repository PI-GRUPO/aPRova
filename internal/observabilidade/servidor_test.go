package observabilidade

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/PI-GRUPO/aPRova/internal/metricas"
	"github.com/PI-GRUPO/aPRova/internal/saude"
)

func pedirAoServidor(t *testing.T, servidor *http.Server, caminho string) (int, string, string) {
	t.Helper()

	gravador := httptest.NewRecorder()
	servidor.Handler.ServeHTTP(gravador, httptest.NewRequestWithContext(t.Context(), http.MethodGet, caminho, nil))
	corpo, _ := io.ReadAll(gravador.Body)
	return gravador.Code, gravador.Header().Get("Content-Type"), string(corpo)
}

func TestServidorInternoExpoeAsMetricasNoFormatoDoPrometheus(t *testing.T) {
	registro := NovoRegistro()
	metricas.NovoMotor(registro).ReciboEmitido()
	servidor := NovoServidor(":9090", registro, nil)

	status, tipo, corpo := pedirAoServidor(t, servidor, "/metrics")

	if servidor.Addr != ":9090" {
		t.Errorf("o servidor escuta no endereço configurado, obtive %q", servidor.Addr)
	}
	if status != http.StatusOK || !strings.HasPrefix(tipo, "text/plain") {
		t.Fatalf("esperava 200 em texto, obtive %d %q", status, tipo)
	}
	for _, nome := range []string{"aprova_recibos_emitidos_total 1", "go_goroutines", "process_"} {
		if !strings.Contains(corpo, nome) {
			t.Errorf("/metrics não tem %q:\n%s", nome, corpo)
		}
	}
}

func TestServidorInternoRespondeVivoEPronto(t *testing.T) {
	foraDoAr := []saude.Dependencia{{Nome: "postgres", Verificar: func(context.Context) error { return errors.New("banco fora") }}}
	servidor := NovoServidor(":9090", prometheus.NewRegistry(), foraDoAr)

	if status, _, _ := pedirAoServidor(t, servidor, "/health/live"); status != http.StatusOK {
		t.Errorf("/health/live responde 200 enquanto o processo está de pé, obtive %d", status)
	}
	if status, _, corpo := pedirAoServidor(t, servidor, "/health/ready"); status != http.StatusServiceUnavailable || !strings.Contains(corpo, "postgres") {
		t.Errorf("/health/ready responde 503 com o banco fora, obtive %d %s", status, corpo)
	}
	if status, _, _ := pedirAoServidor(t, servidor, "/webhooks/github"); status != http.StatusNotFound {
		t.Errorf("o servidor interno só tem métricas e saúde, obtive %d", status)
	}
}

func enderecoLivre(t *testing.T) string {
	t.Helper()

	ouvinte, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reservar porta: %v", err)
	}
	endereco := ouvinte.Addr().String()
	if err := ouvinte.Close(); err != nil {
		t.Fatalf("liberar porta: %v", err)
	}
	return endereco
}

func TestIniciarEscutaNoEnderecoEPararDesliga(t *testing.T) {
	endereco := enderecoLivre(t)
	parar := Iniciar(NovoServidor(endereco, prometheus.NewRegistry(), nil))

	var resposta *http.Response
	var err error
	for range 50 {
		resposta, err = http.Get("http://" + endereco + "/health/live")
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil || resposta.StatusCode != http.StatusOK {
		t.Fatalf("o listener interno precisa responder em %s: %v", endereco, err)
	}
	_ = resposta.Body.Close()

	if err := parar(t.Context()); err != nil {
		t.Fatalf("parar: %v", err)
	}
	if _, err := http.Get("http://" + endereco + "/health/live"); err == nil {
		t.Error("depois de parar, o listener interno não responde mais")
	}
}

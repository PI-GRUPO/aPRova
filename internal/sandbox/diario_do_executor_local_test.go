package sandbox

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/rastro"
)

type saidaDoDiario struct {
	mutex sync.Mutex
	saida bytes.Buffer
}

func (s *saidaDoDiario) Write(p []byte) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.saida.Write(p)
}

func (s *saidaDoDiario) linhas() []map[string]any {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var linhas []map[string]any
	leitor := bufio.NewScanner(bytes.NewReader(s.saida.Bytes()))
	for leitor.Scan() {
		var campos map[string]any
		_ = json.Unmarshal(leitor.Bytes(), &campos)
		linhas = append(linhas, campos)
	}
	return linhas
}

func TestLinhasDoExecutorLocalLevamAIdentificacaoDoContexto(t *testing.T) {
	saida := &saidaDoDiario{}
	anterior := slog.Default()
	slog.SetDefault(slog.New(rastro.NovoHandler(slog.NewJSONHandler(saida, nil))))
	t.Cleanup(func() { slog.SetDefault(anterior) })

	comando := comandoDasEtapas(respostasDasEtapas{
		preparo: linhaDoPreparoPronto, teste: linhaDoResultadoDoTeste,
		falhar: map[string]error{"volume rm": errors.New("volume em uso")},
	})
	ctx := rastro.Com(t.Context(), "task_run_id", "0742a9a6-3866-4ddf-8b1a-1b0501aea6d3", "tenant_id", 58291734)

	if _, err := novoExecutorLocal(opcoesLocaisDeTeste(), comando).Executar(ctx, pedidoValido()); err != nil {
		t.Fatalf("executar: %v", err)
	}

	linhas := saida.linhas()
	removendoVolume := false
	for _, campos := range linhas {
		if campos["msg"] == "falha ao remover o volume da execução" {
			removendoVolume = true
		}
		if campos["task_run_id"] != "0742a9a6-3866-4ddf-8b1a-1b0501aea6d3" || campos["tenant_id"] != float64(58291734) {
			t.Errorf("linha do executor sem a identificação do contexto: %v", campos)
		}
	}
	if !removendoVolume {
		t.Errorf("a falha ao remover o volume precisa ser registrada: %v", linhas)
	}
}

package platform

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/Azulpasta/aPRova/internal/rastro"
)

func TestLogDoSaldoDeRequisicoesLevaOsAtributosDoContexto(t *testing.T) {
	var saida bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(rastro.NovoHandler(slog.NewJSONHandler(&saida, nil))))
	t.Cleanup(func() { slog.SetDefault(anterior) })

	github := &githubDoPR{arquivos: 1, diff: diffDoFrete, cabecalhos: http.Header{"X-Ratelimit-Remaining": {"4990"}, "X-Ratelimit-Limit": {"5000"}}}
	cliente := NovoCliente(OpcoesCliente{Tokens: &fornecedorDeTokenFalso{}, URLBase: github.servidor(t).URL})
	ctx := rastro.Com(t.Context(), "task_run_id", "0742a9a6-3866-4ddf-8b1a-1b0501aea6d3")

	if _, err := cliente.ContextoDoPullRequest(ctx, 161192271, "Azulpasta", "aPRova-TESTES", 14); err != nil {
		t.Fatalf("ler contexto: %v", err)
	}

	linhas := strings.Split(strings.TrimSpace(saida.String()), "\n")
	if len(linhas) == 0 || linhas[0] == "" {
		t.Fatal("o saldo de requisições não foi registrado")
	}
	for _, linha := range linhas {
		var entrada map[string]any
		if err := json.Unmarshal([]byte(linha), &entrada); err != nil {
			t.Fatalf("linha de log não é JSON: %s", linha)
		}
		if entrada["task_run_id"] != "0742a9a6-3866-4ddf-8b1a-1b0501aea6d3" {
			t.Errorf("linha sem o task_run_id do contexto: %s", linha)
		}
	}
}

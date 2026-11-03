package rastro

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func registrarCom(t *testing.T, ctx context.Context, registrar func(*slog.Logger, context.Context)) map[string]any {
	t.Helper()

	var saida bytes.Buffer
	registrar(slog.New(NovoHandler(slog.NewJSONHandler(&saida, nil))), ctx)

	var linha map[string]any
	if err := json.Unmarshal(saida.Bytes(), &linha); err != nil {
		t.Fatalf("linha de log não é JSON: %s", saida.String())
	}
	return linha
}

func TestLinhaRegistradaComContextoLevaOsAtributosDoContexto(t *testing.T) {
	ctx := Com(t.Context(), "task_run_id", "0742a9a6-3866-4ddf-8b1a-1b0501aea6d3")

	linha := registrarCom(t, ctx, func(diario *slog.Logger, ctx context.Context) {
		diario.InfoContext(ctx, "saldo de requisições do github", "restantes", 4990)
	})

	if linha["task_run_id"] != "0742a9a6-3866-4ddf-8b1a-1b0501aea6d3" || linha["restantes"] != float64(4990) {
		t.Errorf("a linha precisa levar o task_run_id do contexto e os próprios atributos: %v", linha)
	}
}

func TestAtributosSeAcumulamSemAlterarOContextoDeOrigem(t *testing.T) {
	externo := Com(t.Context(), "task_run_id", "c41d7e9a-2f3b-4c8d-a5e6-0b1c2d3e4f5a")
	interno := Com(externo, "etapa", "testando_sandbox")

	linhaInterna := registrarCom(t, interno, func(diario *slog.Logger, ctx context.Context) { diario.InfoContext(ctx, "dentro") })
	linhaExterna := registrarCom(t, externo, func(diario *slog.Logger, ctx context.Context) { diario.InfoContext(ctx, "fora") })

	if linhaInterna["task_run_id"] == nil || linhaInterna["etapa"] != "testando_sandbox" {
		t.Errorf("o contexto interno leva os dois atributos: %v", linhaInterna)
	}
	if _, vazou := linhaExterna["etapa"]; vazou {
		t.Errorf("acrescentar no interno não pode mudar o externo: %v", linhaExterna)
	}
}

func TestLinhaSemAtributosNoContextoFicaComoEra(t *testing.T) {
	var saida bytes.Buffer
	diario := slog.New(NovoHandler(slog.NewJSONHandler(&saida, nil))).With("componente", "worker").WithGroup("detalhes")

	diario.InfoContext(context.Background(), "iniciado", "concorrencia", 4)

	if texto := saida.String(); !strings.Contains(texto, `"componente":"worker"`) || !strings.Contains(texto, `"detalhes":{"concorrencia":4}`) {
		t.Errorf("With e WithGroup precisam continuar funcionando: %s", texto)
	}
}

package worker

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/rastro"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

const (
	marcaDoDiff           = "conteudo-do-diff-7f3a"
	marcaDoPatch          = "conteudo-do-patch-2c8e"
	marcaDaSaidaDoSandbox = "saida-do-sandbox-9b1c"
)

func TestDiarioDoWorkerNuncaLevaDiffNemSaidaDoSandbox(t *testing.T) {
	global := &saidaSincronizada{}
	anterior := slog.Default()
	slog.SetDefault(slog.New(rastro.NovoHandler(slog.NewJSONHandler(global, &slog.HandlerOptions{Level: slog.LevelDebug}))))
	t.Cleanup(func() { slog.SetDefault(anterior) })

	falhaDeInfraestrutura := sandbox.ResultadoSandbox{SaidaBruta: "docker: " + marcaDaSaidaDoSandbox, TipoFalha: sandbox.FalhaInfraestrutura}
	c := novoCenario(domain.RiscoBaixo, falhaDeInfraestrutura, falhaDeInfraestrutura, falhaDeInfraestrutura)
	c.leitor.diff = "diff --git a/x b/x\n+" + marcaDoDiff
	c.leitor.patch = "+" + marcaDoPatch
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoFalhou {
		t.Fatalf("o cenário precisa esgotar as tentativas do sandbox, terminou em %s", estado)
	}
	if !strings.Contains(c.armazenamento.ultimaEtapa(taskRun.ID).detalhe, marcaDaSaidaDoSandbox) {
		t.Fatal("o diagnóstico gravado no task_run continua com a saída do sandbox; só o log não a leva")
	}
	for _, linha := range append(c.diario.linhas(), global.linhas()...) {
		for _, marca := range []string{marcaDoDiff, marcaDoPatch, marcaDaSaidaDoSandbox} {
			if strings.Contains(linha, marca) {
				t.Errorf("o log levou conteúdo do PR (%s): %s", marca, linha)
			}
		}
	}
}

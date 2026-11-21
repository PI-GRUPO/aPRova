package publicacao

import (
	"testing"
)

func TestIniciarLevaParaEmAndamentoOCheckRunQueJaEstavaNaFila(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	taskRun := taskRunDeTeste("9eaa0538-add6-4361-8bfc-e74acec86c07", shaDoPrimeiro)
	naFila := github.checkRunExistente(shaDoPrimeiro, taskRun.ID)
	naFila.Situacao = "queued"
	if err := registro.GravarCheckRun(t.Context(), taskRun.ID, naFila.ID); err != nil {
		t.Fatalf("gravar check run: %v", err)
	}

	if err := publicadorContra(t, github, registro).Iniciar(t.Context(), taskRun); err != nil {
		t.Fatalf("iniciar: %v", err)
	}

	if github.quantosPedidos("POST", "/check-runs") != 0 {
		t.Error("o Check Run que já estava na fila é reaproveitado, não duplicado")
	}
	checks := github.checkRuns()
	if len(checks) != 1 || checks[0].Situacao != "in_progress" || checks[0].Saida.Titulo != "Análise em andamento" {
		t.Errorf("o Check Run da fila passa a em andamento: %+v", checks)
	}
}

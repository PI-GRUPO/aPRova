package publicacao

import (
	"context"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
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

type contadorFalso struct {
	iniciadas int
	teto      int
}

func (c contadorFalso) AnalisesIniciadasNaUltimaHora(_ context.Context, _ int64) (int, error) {
	return c.iniciadas, nil
}

func (c contadorFalso) TetoDeAnalisesPorHora() int {
	return c.teto
}

func avisoContra(t *testing.T, github *githubFalso, registro *registroFalso, contador contadorFalso) AvisoDeFila {
	t.Helper()
	return AvisoDeFila{Publicador: publicadorContra(t, github, registro), Contador: contador}
}

func TestInstalacaoNoTetoGanhaCheckRunNaFilaComAExplicacao(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	taskRun := taskRunDeTeste("9eaa0538-add6-4361-8bfc-e74acec86c07", shaDoPrimeiro)

	if err := avisoContra(t, github, registro, contadorFalso{iniciadas: 60, teto: 60}).AvisarSeAcimaDoTeto(t.Context(), taskRun); err != nil {
		t.Fatalf("avisar: %v", err)
	}

	checks := github.checkRuns()
	if len(checks) != 1 {
		t.Fatalf("esperava um Check Run na fila, há %d", len(checks))
	}
	naFila := checks[0]
	if naFila.Situacao != "queued" || naFila.Conclusao != "" || naFila.IDExterno != taskRun.ID || naFila.SHAHead != shaDoPrimeiro {
		t.Errorf("o Check Run fica em queued, sem conclusão, ligado ao task_run: %+v", naFila)
	}
	if !strings.Contains(naFila.Saida.Resumo, "60 análises por hora") || !strings.Contains(naFila.Saida.Resumo, "não foi descartada") {
		t.Errorf("o resumo explica o teto e que a análise espera: %q", naFila.Saida.Resumo)
	}
	if id, _ := registro.CheckRunDoTaskRun(t.Context(), taskRun.ID); id != naFila.ID {
		t.Errorf("o id do Check Run da fila é gravado para o worker reaproveitar: gravado %d, esperava %d", id, naFila.ID)
	}
}

func TestInstalacaoAbaixoDoTetoNaoGanhaAviso(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	taskRun := taskRunDeTeste("9eaa0538-add6-4361-8bfc-e74acec86c07", shaDoPrimeiro)

	if err := avisoContra(t, github, registro, contadorFalso{iniciadas: 59, teto: 60}).AvisarSeAcimaDoTeto(t.Context(), taskRun); err != nil {
		t.Fatalf("avisar: %v", err)
	}

	if len(github.checkRuns()) != 0 {
		t.Error("abaixo do teto o worker cria o Check Run ao começar, o aviso não cria nada")
	}
}

func TestAvisoDaFilaDepoisIniciarEPublicarUsamUmSoCheckRun(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	taskRun := taskRunDeTeste("9eaa0538-add6-4361-8bfc-e74acec86c07", shaDoPrimeiro)
	aviso := avisoContra(t, github, registro, contadorFalso{iniciadas: 61, teto: 60})

	if err := aviso.AvisarSeAcimaDoTeto(t.Context(), taskRun); err != nil {
		t.Fatalf("avisar: %v", err)
	}
	if err := aviso.AvisarSeAcimaDoTeto(t.Context(), taskRun); err != nil {
		t.Fatalf("avisar de novo: %v", err)
	}
	if err := aviso.Publicador.Iniciar(t.Context(), taskRun); err != nil {
		t.Fatalf("iniciar: %v", err)
	}
	if err := aviso.Publicador.Publicar(t.Context(), publicacaoDeTeste(taskRun, domain.RiscoBaixo)); err != nil {
		t.Fatalf("publicar: %v", err)
	}

	checks := github.checkRuns()
	if len(checks) != 1 || checks[0].Situacao != "completed" {
		t.Errorf("fila, andamento e conclusão no mesmo Check Run: %+v", checks)
	}
}

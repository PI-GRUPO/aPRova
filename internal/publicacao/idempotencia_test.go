package publicacao

import (
	"strings"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
	"github.com/PI-GRUPO/aPRova/internal/worker"
)

const (
	urlPublicaDeTeste = "https://aprova.example.com"
	shaDoPrimeiro     = "6be930afc83bb7213d3d4e111f2c6eb7fd008a42"
	shaDoSegundo      = "95eb58e1d0c4a7b3f2e6d9c8b1a0f4e3d2c1b0a9"
)

func taskRunDeTeste(id, sha string) domain.TaskRun {
	return domain.TaskRun{
		ID: id, TenantID: 58291734, Repositorio: "Azulpasta/aPRova-TESTES", NumeroPR: 1,
		SHAHead: sha, Autor: "agente-frete[bot]", Estado: domain.EstadoAplicando,
	}
}

func publicacaoDeTeste(taskRun domain.TaskRun, nivel domain.NivelRisco) worker.Publicacao {
	motivos := []domain.MotivoDeRisco{}
	if nivel == domain.RiscoAlto {
		motivos = append(motivos, domain.MotivoDeRisco{ID: "falha_sandbox", Nivel: domain.RiscoAlto, Descricao: "os testes ou a compilação falharam no sandbox", Evidencias: []string{"1 de 11 testes falharam"}})
	}
	if nivel == domain.RiscoMedio {
		motivos = append(motivos, domain.MotivoDeRisco{ID: "codigo_sem_teste", Nivel: domain.RiscoMedio, Descricao: "o PR altera código sem alterar testes", Evidencias: []string{"go/frete/frete.go"}})
	}
	return worker.Publicacao{
		TaskRun: taskRun,
		Gate:    domain.GateDeRisco{TaskRunID: taskRun.ID, NivelDeterministico: nivel, Nivel: nivel, Motivos: motivos},
		Sandbox: sandbox.ResultadoSandbox{Compilou: true, TestesPassaram: 10, TestesFalharam: 1, DuracaoMS: 42459, TipoFalha: sandbox.FalhaTeste},
		Recibo:  receipt.Emitido{ID: "6d290152-af24-4462-a525-72adf3b22d6c", Hash: []byte(strings.Repeat("\x5a", 32)), IDChave: "ed25519:4f1c2a9b"},
	}
}

func publicadorContra(t *testing.T, github *githubFalso, registro *registroFalso) *Publicador {
	t.Helper()

	publicador, err := NovoPublicador(OpcoesPublicador{
		GitHub: github.cliente(), Registro: registro, SlugDoApp: slugDeTeste, URLPublica: urlPublicaDeTeste,
		Agora: func() time.Time { return time.Date(2026, 11, 12, 14, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatalf("montar o publicador: %v", err)
	}
	return publicador
}

func TestPublicarDuasVezesOMesmoTaskRunNaoDuplicaCheckRunNemComentario(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	publicador := publicadorContra(t, github, registro)
	publicacao := publicacaoDeTeste(taskRunDeTeste("9eaa0538-add6-4361-8bfc-e74acec86c07", shaDoPrimeiro), domain.RiscoAlto)

	for vez := range 2 {
		if err := publicador.Publicar(t.Context(), publicacao); err != nil {
			t.Fatalf("publicação %d: %v", vez+1, err)
		}
	}

	if checks := github.checkRuns(); len(checks) != 1 {
		t.Errorf("um task_run tem um só Check Run, há %d", len(checks))
	}
	if comentarios := github.comentarios(); len(comentarios) != 1 {
		t.Errorf("um PR tem um só comentário do aPRova, há %d", len(comentarios))
	}
	if github.quantosPedidos("POST", "/check-runs") != 1 || github.quantosPedidos("POST", "/comments") != 1 {
		t.Error("a segunda publicação atualiza o que existe em vez de criar")
	}
}

func TestIniciarEDepoisPublicarUsaOMesmoCheckRun(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	publicador := publicadorContra(t, github, registro)
	taskRun := taskRunDeTeste("9eaa0538-add6-4361-8bfc-e74acec86c07", shaDoPrimeiro)

	if err := publicador.Iniciar(t.Context(), taskRun); err != nil {
		t.Fatalf("iniciar: %v", err)
	}
	if err := publicador.Iniciar(t.Context(), taskRun); err != nil {
		t.Fatalf("iniciar de novo: %v", err)
	}
	if err := publicador.Publicar(t.Context(), publicacaoDeTeste(taskRun, domain.RiscoBaixo)); err != nil {
		t.Fatalf("publicar: %v", err)
	}

	checks := github.checkRuns()
	if len(checks) != 1 {
		t.Fatalf("iniciar e concluir usam o mesmo Check Run, há %d", len(checks))
	}
	if checks[0].Situacao != "completed" || checks[0].IDExterno != taskRun.ID {
		t.Errorf("Check Run = %+v", checks[0])
	}
	if id, _ := registro.CheckRunDoTaskRun(t.Context(), taskRun.ID); id != checks[0].ID {
		t.Errorf("id gravado = %d, esperava %d", id, checks[0].ID)
	}
}

func TestRetomadaDepoisDeCriarSemGravarOIdEncontraOCheckRunPeloExternalID(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	taskRun := taskRunDeTeste("9eaa0538-add6-4361-8bfc-e74acec86c07", shaDoPrimeiro)
	github.checkRunExistente(shaDoPrimeiro, "outro-task-run")
	criadoAntesDoCrash := github.checkRunExistente(shaDoPrimeiro, taskRun.ID)

	if err := publicadorContra(t, github, registro).Publicar(t.Context(), publicacaoDeTeste(taskRun, domain.RiscoAlto)); err != nil {
		t.Fatalf("publicar: %v", err)
	}

	if github.quantosPedidos("POST", "/check-runs") != 0 {
		t.Error("o Check Run criado antes do crash é reaproveitado, não duplicado")
	}
	if id, _ := registro.CheckRunDoTaskRun(t.Context(), taskRun.ID); id != criadoAntesDoCrash.ID {
		t.Errorf("a retomada grava o id encontrado: gravado %d, esperava %d", id, criadoAntesDoCrash.ID)
	}
	for _, check := range github.checkRuns() {
		if check.IDExterno == "outro-task-run" && check.Situacao != "in_progress" {
			t.Error("o Check Run de outro task_run não é tocado")
		}
	}
}

func TestRetomadaDepoisDeComentarSemGravarOIdEncontraOComentarioDoApp(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	taskRun := taskRunDeTeste("9eaa0538-add6-4361-8bfc-e74acec86c07", shaDoPrimeiro)
	for indice := range 35 {
		github.comentarioExistente(1, "wlaisw", "comentário de revisão "+string(rune('a'+indice%26)))
	}
	doApp := github.comentarioExistente(1, slugDeTeste+"[bot]", marcaDoComentario+"\nanálise anterior")

	if err := publicadorContra(t, github, registro).Publicar(t.Context(), publicacaoDeTeste(taskRun, domain.RiscoAlto)); err != nil {
		t.Fatalf("publicar: %v", err)
	}

	if github.quantosPedidos("POST", "/comments") != 0 {
		t.Error("o comentário criado antes do crash é editado, não duplicado")
	}
	if id, _ := registro.ComentarioDoPR(t.Context(), taskRun.TenantID, taskRun.Repositorio, taskRun.NumeroPR); id != doApp.ID {
		t.Errorf("a retomada grava o id do comentário encontrado: gravado %d, esperava %d", id, doApp.ID)
	}
	for _, comentario := range github.comentarios() {
		if comentario.ID == doApp.ID && !strings.Contains(comentario.Corpo, shaDoPrimeiro[:7]) {
			t.Errorf("o comentário do app foi editado com a análise nova: %q", comentario.Corpo)
		}
	}
}

func TestComentarioDeOutraPessoaComAMarcaNaoEhTomadoComoODoApp(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	taskRun := taskRunDeTeste("9eaa0538-add6-4361-8bfc-e74acec86c07", shaDoPrimeiro)
	alheio := github.comentarioExistente(1, "wlaisw", marcaDoComentario+"\ncopiado do aPRova")

	if err := publicadorContra(t, github, registro).Publicar(t.Context(), publicacaoDeTeste(taskRun, domain.RiscoAlto)); err != nil {
		t.Fatalf("publicar: %v", err)
	}

	if github.quantosPedidos("POST", "/comments") != 1 {
		t.Error("sem comentário do próprio app, o aPRova cria o seu")
	}
	for _, comentario := range github.comentarios() {
		if comentario.ID == alheio.ID && comentario.Corpo != marcaDoComentario+"\ncopiado do aPRova" {
			t.Error("o comentário de outra pessoa não pode ser editado")
		}
	}
}

func TestNovaAnaliseNoMesmoPREditaOComentarioExistente(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	publicador := publicadorContra(t, github, registro)
	primeiro := taskRunDeTeste("9eaa0538-add6-4361-8bfc-e74acec86c07", shaDoPrimeiro)
	segundo := taskRunDeTeste("1c7f4b2e-93d8-4a65-b0e1-2f6a8c9d3e47", shaDoSegundo)

	if err := publicador.Publicar(t.Context(), publicacaoDeTeste(primeiro, domain.RiscoAlto)); err != nil {
		t.Fatalf("primeira análise: %v", err)
	}
	if err := publicador.Publicar(t.Context(), publicacaoDeTeste(segundo, domain.RiscoBaixo)); err != nil {
		t.Fatalf("segunda análise: %v", err)
	}

	if checks := github.checkRuns(); len(checks) != 2 {
		t.Errorf("cada commit analisado tem o seu Check Run, há %d", len(checks))
	}
	comentarios := github.comentarios()
	if len(comentarios) != 1 {
		t.Fatalf("o PR continua com um só comentário, há %d", len(comentarios))
	}
	if corpo := comentarios[0].Corpo; !strings.Contains(corpo, shaDoSegundo[:7]) || strings.Contains(corpo, shaDoPrimeiro[:7]) {
		t.Errorf("o comentário mostra só a análise mais nova: %q", corpo)
	}
	if github.quantosPedidos("PATCH", "/issues/comments/") != 1 {
		t.Error("a segunda análise edita o comentário")
	}
}

func TestComentarioApagadoNoGitHubEhRecriado(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	taskRun := taskRunDeTeste("9eaa0538-add6-4361-8bfc-e74acec86c07", shaDoPrimeiro)
	if err := registro.GravarComentario(t.Context(), taskRun.TenantID, taskRun.Repositorio, taskRun.NumeroPR, 999999); err != nil {
		t.Fatal(err)
	}

	if err := publicadorContra(t, github, registro).Publicar(t.Context(), publicacaoDeTeste(taskRun, domain.RiscoAlto)); err != nil {
		t.Fatalf("publicar: %v", err)
	}

	comentarios := github.comentarios()
	if len(comentarios) != 1 {
		t.Fatalf("um comentário apagado à mão é recriado, há %d", len(comentarios))
	}
	if id, _ := registro.ComentarioDoPR(t.Context(), taskRun.TenantID, taskRun.Repositorio, taskRun.NumeroPR); id != comentarios[0].ID {
		t.Errorf("o id gravado passa a ser o do comentário novo: %d", id)
	}
}

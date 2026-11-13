package publicacao

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
	"github.com/PI-GRUPO/aPRova/internal/worker"
)

const idDoTaskRunDeConteudo = "9eaa0538-add6-4361-8bfc-e74acec86c07"

func publicarUmaVez(t *testing.T, publicacao worker.Publicacao) (checkRunNoGitHub, comentarioNoGitHub) {
	t.Helper()

	github := novoGitHubFalso(t)
	if err := publicadorContra(t, github, novoRegistroFalso()).Publicar(t.Context(), publicacao); err != nil {
		t.Fatalf("publicar: %v", err)
	}
	checks, comentarios := github.checkRuns(), github.comentarios()
	if len(checks) != 1 || len(comentarios) != 1 {
		t.Fatalf("esperava um Check Run e um comentário, há %d e %d", len(checks), len(comentarios))
	}
	return checks[0], comentarios[0]
}

func exigirTrechos(t *testing.T, onde, texto string, trechos ...string) {
	t.Helper()

	for _, trecho := range trechos {
		if !strings.Contains(texto, trecho) {
			t.Errorf("%s não contém %q:\n%s", onde, trecho, texto)
		}
	}
}

func urlDeVerificacao(publicacao worker.Publicacao) string {
	return urlPublicaDeTeste + "/receipts/" + publicacao.Recibo.ID + "/verify"
}

func TestNivelDeRiscoDefineAConclusaoDoCheckRun(t *testing.T) {
	casos := []struct {
		nivel     domain.NivelRisco
		conclusao string
	}{
		{domain.RiscoBaixo, "success"},
		{domain.RiscoMedio, "success"},
		{domain.RiscoAlto, "action_required"},
	}

	for _, caso := range casos {
		t.Run(string(caso.nivel), func(t *testing.T) {
			check, _ := publicarUmaVez(t, publicacaoDeTeste(taskRunDeTeste(idDoTaskRunDeConteudo, shaDoPrimeiro), caso.nivel))

			if check.Situacao != "completed" || check.Conclusao != caso.conclusao {
				t.Errorf("Check Run = %s/%s, esperava completed/%s", check.Situacao, check.Conclusao, caso.conclusao)
			}
			if check.Nome != "aPRova" || check.IDExterno != idDoTaskRunDeConteudo || check.SHAHead != shaDoPrimeiro {
				t.Errorf("nome, external_id e commit: %+v", check)
			}
		})
	}
}

func TestActionRequiredSempreLevaODetailsURLDaVerificacaoDoRecibo(t *testing.T) {
	publicacao := publicacaoDeTeste(taskRunDeTeste(idDoTaskRunDeConteudo, shaDoPrimeiro), domain.RiscoAlto)
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	publicador := publicadorContra(t, github, registro)

	if err := publicador.Iniciar(t.Context(), publicacao.TaskRun); err != nil {
		t.Fatalf("iniciar: %v", err)
	}
	if err := publicador.Publicar(t.Context(), publicacao); err != nil {
		t.Fatalf("publicar: %v", err)
	}

	check := github.checkRuns()[0]
	if check.Conclusao != "action_required" || check.URLDetalhes != urlDeVerificacao(publicacao) {
		t.Errorf("action_required precisa de details_url %q, veio %q", urlDeVerificacao(publicacao), check.URLDetalhes)
	}
}

func TestResumoDoCheckRunTrazNivelMotivosSandboxEHashDoRecibo(t *testing.T) {
	publicacao := publicacaoDeTeste(taskRunDeTeste(idDoTaskRunDeConteudo, shaDoPrimeiro), domain.RiscoAlto)

	check, _ := publicarUmaVez(t, publicacao)

	exigirTrechos(t, "o título", check.Saida.Titulo, "Risco alto", "bloqueado")
	exigirTrechos(t, "o resumo", check.Saida.Resumo,
		"falha_sandbox", "os testes ou a compilação falharam no sandbox", "1 de 11 testes falharam",
		"10 testes passaram", "1 teste falhou", hex.EncodeToString(publicacao.Recibo.Hash))
}

func TestRiscoMedioLiberaComOsPontosDeAtencaoNoResumo(t *testing.T) {
	check, _ := publicarUmaVez(t, publicacaoDeTeste(taskRunDeTeste(idDoTaskRunDeConteudo, shaDoPrimeiro), domain.RiscoMedio))

	exigirTrechos(t, "o título", check.Saida.Titulo, "Risco médio")
	exigirTrechos(t, "o resumo", check.Saida.Resumo, "pontos de atenção", "codigo_sem_teste", "go/frete/frete.go")
}

func TestAprovacaoHumanaLevaOCheckRunASuccessComOLogin(t *testing.T) {
	publicacao := publicacaoDeTeste(taskRunDeTeste(idDoTaskRunDeConteudo, shaDoPrimeiro), domain.RiscoAlto)
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	publicador := publicadorContra(t, github, registro)
	if err := publicador.Publicar(t.Context(), publicacao); err != nil {
		t.Fatalf("publicar o bloqueio: %v", err)
	}

	publicacao.AprovadoPor = "httpslua"
	if err := publicador.Publicar(t.Context(), publicacao); err != nil {
		t.Fatalf("publicar a aprovação: %v", err)
	}

	checks := github.checkRuns()
	if len(checks) != 1 {
		t.Fatalf("a aprovação atualiza o mesmo Check Run, há %d", len(checks))
	}
	if checks[0].Conclusao != "success" {
		t.Errorf("aprovado, o Check Run libera o merge: conclusão %s", checks[0].Conclusao)
	}
	exigirTrechos(t, "o título", checks[0].Saida.Titulo, "aprovado por @httpslua")
	exigirTrechos(t, "o comentário", github.comentarios()[0].Corpo, "aprovado por @httpslua", "liberado")
}

func TestFalhaDaAnaliseBloqueiaComFailureEODiagnostico(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	publicador := publicadorContra(t, github, registro)
	taskRun := taskRunDeTeste(idDoTaskRunDeConteudo, shaDoPrimeiro)
	if err := publicador.Iniciar(t.Context(), taskRun); err != nil {
		t.Fatalf("iniciar: %v", err)
	}

	if err := publicador.PublicarFalha(t.Context(), taskRun, "tentativas esgotadas em testando_sandbox: imagem indisponível"); err != nil {
		t.Fatalf("publicar a falha: %v", err)
	}

	checks := github.checkRuns()
	if len(checks) != 1 || checks[0].Situacao != "completed" || checks[0].Conclusao != "failure" {
		t.Fatalf("na dúvida, bloqueia: a falha conclui o mesmo Check Run como failure: %+v", checks)
	}
	exigirTrechos(t, "o título", checks[0].Saida.Titulo, "não terminou", "bloqueado")
	exigirTrechos(t, "o resumo", checks[0].Saida.Resumo,
		"tentativas esgotadas em testando_sandbox: imagem indisponível", "bloqueado", "nova análise", "novo push")
	if strings.Contains(checks[0].Saida.Resumo, "não bloqueia") {
		t.Error("o resumo não pode dizer que a falha libera o merge")
	}
	if len(github.comentarios()) != 0 {
		t.Error("a falha não comenta no PR")
	}
}

func TestFalhaSemCheckRunAnteriorCriaOCheckRunJaEmFailure(t *testing.T) {
	github := novoGitHubFalso(t)

	if err := publicadorContra(t, github, novoRegistroFalso()).PublicarFalha(t.Context(), taskRunDeTeste(idDoTaskRunDeConteudo, shaDoPrimeiro), "erro definitivo"); err != nil {
		t.Fatalf("publicar a falha: %v", err)
	}

	if checks := github.checkRuns(); len(checks) != 1 || checks[0].Conclusao != "failure" {
		t.Errorf("o PR fica com um Check Run em failure, não em andamento para sempre: %+v", checks)
	}
}

func TestComentarioExplicaORiscoEComoVerificarORecibo(t *testing.T) {
	publicacao := publicacaoDeTeste(taskRunDeTeste(idDoTaskRunDeConteudo, shaDoPrimeiro), domain.RiscoAlto)

	_, comentario := publicarUmaVez(t, publicacao)

	exigirTrechos(t, "o comentário", comentario.Corpo,
		marcaDoComentario,
		"risco alto", "o merge está bloqueado até aprovação humana",
		"falha_sandbox", "1 de 11 testes falharam",
		"compilou", "10 testes passaram", "1 teste falhou",
		hex.EncodeToString(publicacao.Recibo.Hash), publicacao.Recibo.IDChave,
		urlDeVerificacao(publicacao), "https://github.com/PI-GRUPO/aPRova/blob/main/docs/verificacao-recibo.md",
		shaDoPrimeiro[:7])
	if strings.Contains(comentario.Corpo, shaDoPrimeiro) {
		t.Error("o commit aparece curto no comentário")
	}
}

func TestComentarioDeRiscoBaixoDizQueEstaLiberadoSemMotivos(t *testing.T) {
	_, comentario := publicarUmaVez(t, publicacaoDeTeste(taskRunDeTeste(idDoTaskRunDeConteudo, shaDoPrimeiro), domain.RiscoBaixo))

	exigirTrechos(t, "o comentário", comentario.Corpo, "risco baixo", "liberado", "nenhuma regra")
	if strings.Contains(comentario.Corpo, "bloqueado") {
		t.Error("risco baixo não fala em bloqueio")
	}
}

func TestSandboxQueNaoCompilouOuNaoRodouEhDitoComClareza(t *testing.T) {
	casos := []struct {
		nome    string
		sandbox sandbox.ResultadoSandbox
		trecho  string
	}{
		{"não compilou", sandbox.ResultadoSandbox{Compilou: false, TipoFalha: sandbox.FalhaCompilacao}, "não compilou"},
		{"infraestrutura", sandbox.ResultadoSandbox{TipoFalha: sandbox.FalhaInfraestrutura}, "não conseguiu executar"},
		{"tudo passou", sandbox.ResultadoSandbox{Compilou: true, TestesPassaram: 1, TipoFalha: sandbox.FalhaNenhuma}, "1 teste passou"},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			publicacao := publicacaoDeTeste(taskRunDeTeste(idDoTaskRunDeConteudo, shaDoPrimeiro), domain.RiscoBaixo)
			publicacao.Sandbox = caso.sandbox

			_, comentario := publicarUmaVez(t, publicacao)

			exigirTrechos(t, "o comentário", comentario.Corpo, caso.trecho)
		})
	}
}

func TestEvidenciaComCraseNaoQuebraOMarkdown(t *testing.T) {
	publicacao := publicacaoDeTeste(taskRunDeTeste(idDoTaskRunDeConteudo, shaDoPrimeiro), domain.RiscoAlto)
	publicacao.Gate.Motivos[0].Evidencias = []string{"arquivo `estranho`.go"}

	_, comentario := publicarUmaVez(t, publicacao)

	exigirTrechos(t, "o comentário", comentario.Corpo, "`` arquivo `estranho`.go ``")
}

func TestTextoLongoCabeNoLimiteDoGitHub(t *testing.T) {
	publicacao := publicacaoDeTeste(taskRunDeTeste(idDoTaskRunDeConteudo, shaDoPrimeiro), domain.RiscoAlto)
	for indice := range 5000 {
		publicacao.Gate.Motivos[0].Evidencias = append(publicacao.Gate.Motivos[0].Evidencias, strings.Repeat("x", 20)+string(rune('a'+indice%26)))
	}

	check, comentario := publicarUmaVez(t, publicacao)

	if len(check.Saida.Resumo) > 65535 || len(comentario.Corpo) > 65536 {
		t.Errorf("o GitHub recusa texto acima do limite: resumo %d, comentário %d", len(check.Saida.Resumo), len(comentario.Corpo))
	}
	exigirTrechos(t, "o comentário", comentario.Corpo, marcaDoComentario, hex.EncodeToString(publicacao.Recibo.Hash))
}

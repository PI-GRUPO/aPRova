package publicacao

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/explicacao"
)

const timeoutDaExplicacaoDeTeste = 50 * time.Millisecond

type modeloFalso struct {
	gerar func(ctx context.Context, sistema, conteudo string) (string, error)
}

func (m modeloFalso) Gerar(ctx context.Context, sistema, conteudo string) (string, error) {
	return m.gerar(ctx, sistema, conteudo)
}

type explicacoesEmMemoria struct {
	mutex      sync.Mutex
	porTaskRun map[string]domain.Explicacao
}

func novasExplicacoesEmMemoria() *explicacoesEmMemoria {
	return &explicacoesEmMemoria{porTaskRun: map[string]domain.Explicacao{}}
}

func (e *explicacoesEmMemoria) ExplicacaoDoTaskRun(_ context.Context, taskRunID string) (domain.Explicacao, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	gravada, existe := e.porTaskRun[taskRunID]
	if !existe {
		return domain.Explicacao{}, domain.ErrNaoEncontrado
	}
	return gravada, nil
}

func (e *explicacoesEmMemoria) GravarExplicacao(_ context.Context, gravada domain.Explicacao) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if _, existe := e.porTaskRun[gravada.TaskRunID]; !existe {
		e.porTaskRun[gravada.TaskRunID] = gravada
	}
	return nil
}

func explicadorCom(t *testing.T, modelo explicacao.Modelo, registro explicacao.Registro) *explicacao.Explicador {
	t.Helper()

	explicador, err := explicacao.NovoExplicador(explicacao.Opcoes{
		Modelo: modelo, NomeDoModelo: "modelo-de-teste", Registro: registro, Timeout: timeoutDaExplicacaoDeTeste,
	})
	if err != nil {
		t.Fatalf("montar o explicador: %v", err)
	}
	return explicador
}

func publicadorComExplicador(t *testing.T, github *githubFalso, registro *registroFalso, explicador Explicador) *Publicador {
	t.Helper()

	publicador, err := NovoPublicador(OpcoesPublicador{
		GitHub: github.cliente(), Registro: registro, SlugDoApp: slugDeTeste, URLPublica: urlPublicaDeTeste,
		Agora:      func() time.Time { return time.Date(2026, 11, 12, 14, 0, 0, 0, time.UTC) },
		Explicador: explicador,
	})
	if err != nil {
		t.Fatalf("montar o publicador: %v", err)
	}
	return publicador
}

func comentarioDeHoje(t *testing.T, nivel domain.NivelRisco) string {
	t.Helper()

	github := novoGitHubFalso(t)
	taskRun := taskRunDeTeste("9eaa0538-add6-4361-8bfc-e74acec86c07", shaDoPrimeiro)
	if err := publicadorContra(t, github, novoRegistroFalso()).Publicar(t.Context(), publicacaoDeTeste(taskRun, nivel)); err != nil {
		t.Fatalf("publicar sem explicador: %v", err)
	}
	return github.comentarios()[0].Corpo
}

func comentarioComModelo(t *testing.T, nivel domain.NivelRisco, modelo explicacao.Modelo) string {
	t.Helper()

	github := novoGitHubFalso(t)
	taskRun := taskRunDeTeste("9eaa0538-add6-4361-8bfc-e74acec86c07", shaDoPrimeiro)
	explicador := explicadorCom(t, modelo, novasExplicacoesEmMemoria())
	if err := publicadorComExplicador(t, github, novoRegistroFalso(), explicador).Publicar(t.Context(), publicacaoDeTeste(taskRun, nivel)); err != nil {
		t.Fatalf("publicar com explicador: %v", err)
	}
	return github.comentarios()[0].Corpo
}

func TestModeloForaDoArDeixaOComentarioComoHoje(t *testing.T) {
	foraDoAr := modeloFalso{gerar: func(context.Context, string, string) (string, error) {
		return "", errors.New("gemini respondeu 503")
	}}

	for _, nivel := range []domain.NivelRisco{domain.RiscoBaixo, domain.RiscoMedio, domain.RiscoAlto} {
		if comentario, esperado := comentarioComModelo(t, nivel, foraDoAr), comentarioDeHoje(t, nivel); comentario != esperado {
			t.Errorf("risco %s com o modelo fora do ar, o comentário muda:\nobtive:   %q\nesperava: %q", nivel, comentario, esperado)
		}
	}
}

func TestModeloEmTimeoutDeixaOComentarioComoHojeSemPassarDoPrazo(t *testing.T) {
	travado := modeloFalso{gerar: func(ctx context.Context, _, _ string) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	}}

	inicio := time.Now()
	comentario := comentarioComModelo(t, domain.RiscoAlto, travado)
	decorrido := time.Since(inicio)

	if esperado := comentarioDeHoje(t, domain.RiscoAlto); comentario != esperado {
		t.Errorf("com o modelo em timeout, o comentário muda:\nobtive:   %q\nesperava: %q", comentario, esperado)
	}
	if decorrido > timeoutDaExplicacaoDeTeste+2*time.Second {
		t.Errorf("a publicação esperou %s, o timeout da explicação é %s", decorrido, timeoutDaExplicacaoDeTeste)
	}
}

const explicacaoDoRiscoAlto = "Este pull request ficou com risco alto, então o merge está bloqueado até alguém com permissão de escrita aprovar. " +
	"O motivo é que 1 de 11 testes falharam no sandbox. Corrija o teste que falha e envie um commit novo, ou peça a aprovação de quem pode liberar."

func modeloQueResponde(texto string, chamadas *int) modeloFalso {
	return modeloFalso{gerar: func(context.Context, string, string) (string, error) {
		*chamadas++
		return texto, nil
	}}
}

func TestExplicacaoGeradaEntraNoComentarioComAvisoDeIAELinkDoRecibo(t *testing.T) {
	chamadas := 0
	comentario := comentarioComModelo(t, domain.RiscoAlto, modeloQueResponde(explicacaoDoRiscoAlto, &chamadas))

	if !strings.Contains(comentario, explicacaoDoRiscoAlto) {
		t.Fatalf("a explicação gerada não está no comentário:\n%s", comentario)
	}
	aviso := "<sub>Explicação gerada por IA a partir da decisão já tomada. A decisão vem das regras do aPRova e está no [recibo assinado](" +
		urlPublicaDeTeste + "/receipts/6d290152-af24-4462-a525-72adf3b22d6c/verify).</sub>"
	if !strings.Contains(comentario, aviso) {
		t.Errorf("falta a linha que diz que a explicação é de IA, com o link do recibo:\n%s", comentario)
	}
	if strings.Contains(comentario, "sem modelo de IA") {
		t.Error("com a explicação gerada, o rodapé não pode dizer que não há modelo de IA")
	}
	hoje := comentarioDeHoje(t, domain.RiscoAlto)
	for _, secao := range []string{"#### Motivos", "#### Sandbox", "#### Recibo assinado", "### aPRova: risco alto"} {
		if strings.Count(comentario, secao) != 1 || !strings.Contains(hoje, secao) {
			t.Errorf("a seção %q continua como hoje", secao)
		}
	}
}

func TestCheckRunIgualComESemExplicacaoEConcluidoAntesDoModelo(t *testing.T) {
	taskRun := taskRunDeTeste("9eaa0538-add6-4361-8bfc-e74acec86c07", shaDoPrimeiro)

	semExplicacao := novoGitHubFalso(t)
	if err := publicadorContra(t, semExplicacao, novoRegistroFalso()).Publicar(t.Context(), publicacaoDeTeste(taskRun, domain.RiscoAlto)); err != nil {
		t.Fatalf("publicar sem explicação: %v", err)
	}

	comExplicacao := novoGitHubFalso(t)
	checkConcluidoAoChamar := false
	modelo := modeloFalso{gerar: func(context.Context, string, string) (string, error) {
		checks := comExplicacao.checkRuns()
		checkConcluidoAoChamar = len(checks) == 1 && checks[0].Situacao == "completed"
		return explicacaoDoRiscoAlto, nil
	}}
	explicador := explicadorCom(t, modelo, novasExplicacoesEmMemoria())
	if err := publicadorComExplicador(t, comExplicacao, novoRegistroFalso(), explicador).Publicar(t.Context(), publicacaoDeTeste(taskRun, domain.RiscoAlto)); err != nil {
		t.Fatalf("publicar com explicação: %v", err)
	}

	if !checkConcluidoAoChamar {
		t.Error("o Check Run é concluído antes de o modelo ser chamado")
	}
	antes, depois := semExplicacao.checkRuns()[0], comExplicacao.checkRuns()[0]
	if antes.Situacao != depois.Situacao || antes.Conclusao != depois.Conclusao || antes.Saida != depois.Saida || antes.URLDetalhes != depois.URLDetalhes {
		t.Errorf("o Check Run muda com a explicação:\nsem: %+v\ncom: %+v", antes, depois)
	}
}

func TestRiscoAltoAprovadoReusaAExplicacaoDoBloqueioSemCitarQuemAprovou(t *testing.T) {
	github, chamadas := novoGitHubFalso(t), 0
	explicador := explicadorCom(t, modeloQueResponde(explicacaoDoRiscoAlto, &chamadas), novasExplicacoesEmMemoria())
	publicador := publicadorComExplicador(t, github, novoRegistroFalso(), explicador)
	taskRun := taskRunDeTeste("9eaa0538-add6-4361-8bfc-e74acec86c07", shaDoPrimeiro)

	if err := publicador.Publicar(t.Context(), publicacaoDeTeste(taskRun, domain.RiscoAlto)); err != nil {
		t.Fatalf("publicar o bloqueio: %v", err)
	}
	aprovada := publicacaoDeTeste(taskRun, domain.RiscoAlto)
	aprovada.AprovadoPor = "julia-ed2"
	if err := publicador.Publicar(t.Context(), aprovada); err != nil {
		t.Fatalf("publicar a aprovação: %v", err)
	}

	if chamadas != 1 {
		t.Errorf("o modelo é chamado uma vez por task_run, foi chamado %d", chamadas)
	}
	comentario := github.comentarios()[0].Corpo
	if !strings.Contains(comentario, explicacaoDoRiscoAlto) || !strings.Contains(comentario, "aprovado por @julia-ed2") {
		t.Errorf("o comentário aprovado mantém a explicação do bloqueio e o cabeçalho da aprovação:\n%s", comentario)
	}
}

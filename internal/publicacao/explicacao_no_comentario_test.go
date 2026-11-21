package publicacao

import (
	"context"
	"errors"
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

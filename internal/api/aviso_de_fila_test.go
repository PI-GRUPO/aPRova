package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

type avisoDeFilaFalso struct {
	avisados []domain.TaskRun
	falha    error
}

func (a *avisoDeFilaFalso) AvisarSeAcimaDoTeto(_ context.Context, taskRun domain.TaskRun) error {
	a.avisados = append(a.avisados, taskRun)
	return a.falha
}

func enviarWebhookComAviso(t *testing.T, registro *registroFalso, aviso *avisoDeFilaFalso, e entrega) *httptest.ResponseRecorder {
	t.Helper()

	servidor := NovoServidor(Opcoes{SegredoWebhook: []byte(segredoDeTeste), Execucoes: registro, AvisoDeFila: aviso})
	requisicao := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/webhooks/github", strings.NewReader(e.corpo))
	requisicao.Header.Set("X-GitHub-Event", e.evento)
	requisicao.Header.Set("X-GitHub-Delivery", e.deliveryID)
	requisicao.Header.Set("X-Hub-Signature-256", e.assinatura)

	gravador := httptest.NewRecorder()
	servidor.Rotas().ServeHTTP(gravador, requisicao)
	return gravador
}

func TestPullRequestNovoPassaPeloAvisoDeFila(t *testing.T) {
	registro, aviso := novoRegistroFalso(), &avisoDeFilaFalso{}

	resposta := enviarWebhookComAviso(t, registro, aviso, entregaValida("pull_request", "entrega-fila-1", corpoPullRequest("opened")))

	if resposta.Code != http.StatusAccepted {
		t.Fatalf("esperava 202, obtive %d", resposta.Code)
	}
	if len(aviso.avisados) != 1 {
		t.Fatalf("o task_run novo passa uma vez pelo aviso de fila, passou %d", len(aviso.avisados))
	}
	esperado := domain.TaskRun{
		ID: "entrega-fila-1-execucao", TenantID: 777, Repositorio: "PI-GRUPO/aPRova", NumeroPR: 42, SHAHead: "abc123",
	}
	if avisado := aviso.avisados[0]; avisado.ID != esperado.ID || avisado.TenantID != esperado.TenantID ||
		avisado.Repositorio != esperado.Repositorio || avisado.NumeroPR != esperado.NumeroPR || avisado.SHAHead != esperado.SHAHead {
		t.Errorf("task_run avisado = %+v, esperava %+v", avisado, esperado)
	}
}

func TestReentregaNaoAvisaDeNovo(t *testing.T) {
	registro, aviso := novoRegistroFalso(), &avisoDeFilaFalso{}
	pedido := entregaValida("pull_request", "entrega-fila-2", corpoPullRequest("synchronize"))

	enviarWebhookComAviso(t, registro, aviso, pedido)
	enviarWebhookComAviso(t, registro, aviso, pedido)

	if len(aviso.avisados) != 1 {
		t.Errorf("a reentrega não repete o aviso, houve %d", len(aviso.avisados))
	}
}

func TestFalhaNoAvisoDeFilaNaoPerdeOPullRequest(t *testing.T) {
	registro, aviso := novoRegistroFalso(), &avisoDeFilaFalso{falha: errors.New("github fora")}

	resposta := enviarWebhookComAviso(t, registro, aviso, entregaValida("pull_request", "entrega-fila-3", corpoPullRequest("opened")))

	if resposta.Code != http.StatusAccepted || registro.quantidadeDeExecucoes() != 1 {
		t.Errorf("o task_run gravado segue na fila mesmo sem o aviso: status %d, execuções %d", resposta.Code, registro.quantidadeDeExecucoes())
	}
}

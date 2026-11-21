package integracaopostgres

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/api"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
)

type commitComDoisPais struct{}

func (commitComDoisPais) PaisDoCommit(context.Context, int64, string, string, string) (int, error) {
	return 2, nil
}

func TestBaseGravadaNoClosedEUsadaNoComandoDeReversao(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()
	taskRun := novoTaskRun(t, repo, tenant)

	assinador, err := receipt.NovoAssinador(bytes.Repeat([]byte{3}, 32))
	if err != nil {
		t.Fatal(err)
	}
	emissor, err := receipt.NovoEmissor(receipt.OpcoesEmissor{Assinador: assinador, Armazenamento: repo})
	if err != nil {
		t.Fatal(err)
	}
	recibo, err := emissor.Emitir(t.Context(), receipt.Pedido{TaskRun: taskRun, Gate: gateDeExemplo(taskRun.ID), Tentativa: 1, Diff: "diff"})
	if err != nil {
		t.Fatalf("emitir o recibo: %v", err)
	}

	servidor := api.NovoServidor(api.Opcoes{
		SegredoWebhook: []byte(segredoDoWebhook), Execucoes: repo,
		Recibos:   receipt.NovoVerificador(receipt.NovoChaveiro(assinador.ChavePublica()), repo),
		Desfechos: repo, Commits: commitComDoisPais{}, URLPublica: "https://aprova.exemplo",
	}).Rotas()

	corpo, _ := json.Marshal(map[string]any{
		"action": "closed", "number": taskRun.NumeroPR,
		"pull_request": map[string]any{
			"merged": true, "merge_commit_sha": commitDeMerge, "closed_at": "2026-11-20T09:00:00Z", "commits": 1,
			"head": map[string]any{"sha": taskRun.SHAHead}, "base": map[string]any{"ref": "release/2026"},
		},
		"repository": map[string]any{"full_name": taskRun.Repositorio}, "installation": map[string]any{"id": tenant},
	})
	webhook := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/webhooks/github", strings.NewReader(string(corpo)))
	webhook.Header.Set("X-GitHub-Event", "pull_request")
	webhook.Header.Set("X-GitHub-Delivery", fmt.Sprintf("entrega-base-%d", tenant))
	webhook.Header.Set("X-Hub-Signature-256", assinar(corpo))
	gravador := httptest.NewRecorder()
	servidor.ServeHTTP(gravador, webhook)
	if gravador.Code != http.StatusAccepted {
		t.Fatalf("webhook respondeu %d", gravador.Code)
	}

	gravador = httptest.NewRecorder()
	servidor.ServeHTTP(gravador, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/receipts/"+recibo.ID+"/revert", nil))
	var resposta struct {
		Disponivel bool   `json:"disponivel"`
		Recusa     string `json:"recusa"`
		Comando    string `json:"comando"`
		Corpo      string `json:"corpo"`
	}
	if err := json.Unmarshal(gravador.Body.Bytes(), &resposta); err != nil || gravador.Code != http.StatusOK {
		t.Fatalf("status %d, corpo %s, erro %v", gravador.Code, gravador.Body.String(), err)
	}

	esperado := "git fetch origin && git switch -c revert/pr-7 origin/release/2026 && git revert --no-edit -m 1 " + commitDeMerge
	if !resposta.Disponivel || resposta.Comando != esperado {
		t.Errorf("recusa %q, comando\n%s\nesperava\n%s", resposta.Recusa, resposta.Comando, esperado)
	}
	if !strings.Contains(resposta.Corpo, "que entrou na release/2026 pelo PR #7") {
		t.Errorf("corpo não nomeia a base:\n%s", resposta.Corpo)
	}
	if desfecho, _ := repo.DesfechoDoPR(t.Context(), tenant, taskRun.Repositorio, taskRun.NumeroPR); desfecho.Base != "release/2026" {
		t.Errorf("base gravada: %q", desfecho.Base)
	}
}

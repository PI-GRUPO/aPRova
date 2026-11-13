package integracaopostgres

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/api"
	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/repository"
)

const (
	headDoPR         = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	commitDeMerge    = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	segredoDoWebhook = "segredo-da-integracao"
)

type desfechoGravado struct {
	mergeado   bool
	head       string
	merge      *string
	fechadoEm  time.Time
	deliveryID string
}

func fechamento(tenant int64, numero int, mergeado bool, fechadoEm time.Time) domain.FechamentoDePR {
	fechado := domain.FechamentoDePR{
		InstallationID: tenant, DeliveryID: fmt.Sprintf("fechamento-%d-%d-%d", tenant, numero, fechadoEm.UnixNano()),
		Repositorio: "Azulpasta/alvo", NumeroPR: numero, Mergeado: mergeado, SHAHead: headDoPR, FechadoEm: fechadoEm,
	}
	if mergeado {
		fechado.SHADoMerge = commitDeMerge
	}
	return fechado
}

func registrarFechamento(t *testing.T, repo *repository.RepositorioPostgres, fechado domain.FechamentoDePR) {
	t.Helper()
	if err := repo.RegistrarFechamento(t.Context(), fechado); err != nil {
		t.Fatalf("registrar fechamento do PR %d: %v", fechado.NumeroPR, err)
	}
}

func desfechoDoPR(t *testing.T, tenant int64, numero int) desfechoGravado {
	t.Helper()
	var lido desfechoGravado
	err := pool.QueryRow(t.Context(), `
		SELECT merged, head_sha, merge_commit_sha, closed_at, delivery_id FROM pull_request_outcomes
		WHERE tenant_id = $1 AND repository = 'Azulpasta/alvo' AND pr_number = $2`, tenant, numero).
		Scan(&lido.mergeado, &lido.head, &lido.merge, &lido.fechadoEm, &lido.deliveryID)
	if err != nil {
		t.Fatalf("ler o desfecho do PR %d: %v", numero, err)
	}
	return lido
}

func revertidoPor(t *testing.T, tenant int64, numero int) int {
	t.Helper()
	var revertidoPor int
	err := pool.QueryRow(t.Context(), `
		SELECT reverted_by_pr FROM pull_request_reverts
		WHERE tenant_id = $1 AND repository = 'Azulpasta/alvo' AND pr_number = $2`, tenant, numero).Scan(&revertidoPor)
	if err != nil {
		return 0
	}
	return revertidoPor
}

func TestFechamentoComMergeGravaODesfechoUmaVezSo(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()
	fechado := fechamento(tenant, 12, true, time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC))

	registrarFechamento(t, repo, fechado)
	registrarFechamento(t, repo, fechado)

	lido := desfechoDoPR(t, tenant, 12)
	if !lido.mergeado || lido.head != headDoPR || lido.merge == nil || *lido.merge != commitDeMerge || !lido.fechadoEm.Equal(fechado.FechadoEm) {
		t.Errorf("desfecho = %+v", lido)
	}
	if linhas := contar(t, `SELECT count(*) FROM pull_request_outcomes WHERE tenant_id = $1`, tenant); linhas != 1 {
		t.Errorf("a reentrega não duplica: %d linhas", linhas)
	}
}

func TestPRReabertoValeOUltimoFechamento(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()
	abandonado := fechamento(tenant, 12, false, time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC))
	mergeadoDepois := fechamento(tenant, 12, true, abandonado.FechadoEm.Add(48*time.Hour))

	registrarFechamento(t, repo, abandonado)
	registrarFechamento(t, repo, mergeadoDepois)
	registrarFechamento(t, repo, abandonado)

	if lido := desfechoDoPR(t, tenant, 12); !lido.mergeado || lido.deliveryID != mergeadoDepois.DeliveryID {
		t.Errorf("o fechamento mais novo vale, mesmo que o antigo chegue depois: %+v", lido)
	}
}

func TestPRSemMergeNaoGuardaCommitDeMerge(t *testing.T) {
	exigirBanco(t)

	_, err := pool.Exec(t.Context(), `
		INSERT INTO pull_request_outcomes (tenant_id, repository, pr_number, merged, head_sha, merge_commit_sha, closed_at, delivery_id)
		VALUES ($1, 'Azulpasta/alvo', 3, false, $2, $3, now(), 'entrega-incoerente')`, novoTenant(), headDoPR, commitDeMerge)

	if codigoPostgres(err) != "23514" {
		t.Errorf("merged e merge_commit_sha andam juntos, esperava 23514: %v", err)
	}
}

func TestReversaoMarcaOPROriginalUmaVezSo(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()
	reversao := fechamento(tenant, 20, true, time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC))
	reversao.Reverte = 12

	registrarFechamento(t, repo, reversao)
	registrarFechamento(t, repo, reversao)

	if por := revertidoPor(t, tenant, 12); por != 20 {
		t.Errorf("o PR 12 foi revertido pelo 20, li %d", por)
	}
	if linhas := contar(t, `SELECT count(*) FROM pull_request_reverts WHERE tenant_id = $1`, tenant); linhas != 1 {
		t.Errorf("a reentrega não duplica a reversão: %d", linhas)
	}
	if lido := desfechoDoPR(t, tenant, 20); !lido.mergeado {
		t.Error("o PR que reverte também tem o próprio desfecho")
	}
}

func TestReversaoNaoAlcancaOutroTenant(t *testing.T) {
	repo := exigirBanco(t)
	tenant, vizinho := novoTenant(), novoTenant()
	reversao := fechamento(tenant, 20, true, time.Now().UTC())
	reversao.Reverte = 12

	registrarFechamento(t, repo, reversao)

	if por := revertidoPor(t, vizinho, 12); por != 0 {
		t.Error("o PR 12 do outro tenant não foi revertido")
	}
}

func assinar(corpo []byte) string {
	mac := hmac.New(sha256.New, []byte(segredoDoWebhook))
	mac.Write(corpo)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestWebhookDeFechamentoComRevertsMarcaOPROriginalComoRevertido(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()
	servidor := api.NovoServidor(api.Opcoes{SegredoWebhook: []byte(segredoDoWebhook), Execucoes: repo}).Rotas()
	corpo, _ := json.Marshal(map[string]any{
		"action": "closed", "number": 20,
		"pull_request": map[string]any{
			"merged": true, "merge_commit_sha": commitDeMerge, "closed_at": "2026-11-12T09:00:00Z",
			"body": "Reverts Azulpasta/alvo#12\n\nO frete quebrou.", "head": map[string]any{"sha": headDoPR},
		},
		"repository": map[string]any{"full_name": "Azulpasta/alvo"}, "installation": map[string]any{"id": tenant},
	})

	for range 2 {
		requisicao := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/webhooks/github", strings.NewReader(string(corpo)))
		requisicao.Header.Set("X-GitHub-Event", "pull_request")
		requisicao.Header.Set("X-GitHub-Delivery", fmt.Sprintf("entrega-reversao-%d", tenant))
		requisicao.Header.Set("X-Hub-Signature-256", assinar(corpo))
		resposta := httptest.NewRecorder()
		servidor.ServeHTTP(resposta, requisicao)
		if resposta.Code != http.StatusAccepted {
			t.Fatalf("webhook respondeu %d", resposta.Code)
		}
	}

	if por := revertidoPor(t, tenant, 12); por != 20 {
		t.Errorf("o PR 12 foi revertido pelo 20, li %d", por)
	}
	if criados := contar(t, `SELECT count(*) FROM task_runs WHERE tenant_id = $1`, tenant); criados != 0 {
		t.Errorf("fechar o PR não cria análise: %d", criados)
	}
}

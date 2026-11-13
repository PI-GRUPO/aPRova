package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

const (
	headDoFechamento    = "1111111111111111111111111111111111111111"
	commitDoMerge       = "2222222222222222222222222222222222222222"
	momentoDoFechamento = "2026-11-13T10:00:00Z"
)

func corpoDeFechamento(t *testing.T, mergeado bool, corpo string) string {
	t.Helper()

	pullRequest := map[string]any{
		"merged": mergeado, "closed_at": momentoDoFechamento, "body": corpo,
		"head": map[string]any{"sha": headDoFechamento}, "user": map[string]any{"login": "alice"},
	}
	if mergeado {
		pullRequest["merge_commit_sha"] = commitDoMerge
	}
	bruto, err := json.Marshal(map[string]any{
		"action": "closed", "number": 42, "pull_request": pullRequest,
		"repository": map[string]any{"full_name": "PI-GRUPO/aPRova"}, "installation": map[string]any{"id": 777},
	})
	if err != nil {
		t.Fatalf("montar o corpo: %v", err)
	}
	return string(bruto)
}

func TestFechamentoComMergeRegistraODesfechoSemCriarExecucao(t *testing.T) {
	registro := novoRegistroFalso()

	resposta := enviarWebhook(t, registro, entregaValida("pull_request", "entrega-fechamento", corpoDeFechamento(t, true, "Ajusta o frete")))

	if resposta.Code != http.StatusAccepted {
		t.Fatalf("fechamento registrado responde 202, obtive %d", resposta.Code)
	}
	if registro.quantidadeDeExecucoes() != 0 {
		t.Error("fechar o PR não abre análise")
	}
	if len(registro.fechamentos) != 1 {
		t.Fatalf("um fechamento registrado: %d", len(registro.fechamentos))
	}
	fechamento := registro.fechamentos[0]
	fechadoEm, _ := time.Parse(time.RFC3339, momentoDoFechamento)
	if fechamento.InstallationID != 777 || fechamento.DeliveryID != "entrega-fechamento" || fechamento.Repositorio != "PI-GRUPO/aPRova" || fechamento.NumeroPR != 42 {
		t.Errorf("identificação do PR: %+v", fechamento)
	}
	if !fechamento.Mergeado || fechamento.SHAHead != headDoFechamento || fechamento.SHADoMerge != commitDoMerge || !fechamento.FechadoEm.Equal(fechadoEm) {
		t.Errorf("desfecho do PR: %+v", fechamento)
	}
	if fechamento.Reverte != 0 {
		t.Errorf("sem a linha de reversão, não reverte nada: %d", fechamento.Reverte)
	}
}

func TestFechamentoSemMergeRegistraODesfecho(t *testing.T) {
	registro := novoRegistroFalso()

	enviarWebhook(t, registro, entregaValida("pull_request", "entrega-abandono", corpoDeFechamento(t, false, "")))

	if len(registro.fechamentos) != 1 || registro.fechamentos[0].Mergeado || registro.fechamentos[0].SHADoMerge != "" {
		t.Errorf("fechado sem merge: %+v", registro.fechamentos)
	}
}

func TestReversaoSoComAPrimeiraLinhaDoBotaoRevertDoGitHub(t *testing.T) {
	casos := []struct {
		nome     string
		mergeado bool
		corpo    string
		reverte  int
	}{
		{"corpo gerado pelo botão", true, "Reverts PI-GRUPO/aPRova#12", 12},
		{"com texto depois da primeira linha", true, "Reverts PI-GRUPO/aPRova#12\r\n\r\nO frete quebrou em produção.", 12},
		{"de outro repositório", true, "Reverts PI-GRUPO/outro#12", 0},
		{"fechado sem merge não reverteu nada", false, "Reverts PI-GRUPO/aPRova#12", 0},
		{"linha de reversão fora da primeira", true, "Desfaz a mudança\nReverts PI-GRUPO/aPRova#12", 0},
		{"espaço antes", true, " Reverts PI-GRUPO/aPRova#12", 0},
		{"texto depois do número", true, "Reverts PI-GRUPO/aPRova#12 por causa do frete", 0},
		{"só o número", true, "Reverts #12", 0},
		{"número zero", true, "Reverts PI-GRUPO/aPRova#0", 0},
		{"minúsculas", true, "reverts PI-GRUPO/aPRova#12", 0},
		{"título no estilo do botão no corpo", true, `Revert "Ajusta o frete"`, 0},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			registro := novoRegistroFalso()

			enviarWebhook(t, registro, entregaValida("pull_request", "entrega-"+caso.nome, corpoDeFechamento(t, caso.mergeado, caso.corpo)))

			if len(registro.fechamentos) != 1 || registro.fechamentos[0].Reverte != caso.reverte {
				t.Errorf("reverte = %+v, esperava %d", registro.fechamentos, caso.reverte)
			}
		})
	}
}

func TestFechamentoSemInstalacaoERecusado(t *testing.T) {
	registro := novoRegistroFalso()
	corpo := `{"action":"closed","number":42,"pull_request":{"merged":false,"closed_at":"` + momentoDoFechamento + `","head":{"sha":"` + headDoFechamento + `"}},"repository":{"full_name":"PI-GRUPO/aPRova"}}`

	resposta := enviarWebhook(t, registro, entregaValida("pull_request", "entrega-sem-instalacao", corpo))

	if resposta.Code != http.StatusBadRequest || len(registro.fechamentos) != 0 {
		t.Errorf("sem installation.id: %d, %+v", resposta.Code, registro.fechamentos)
	}
}

func TestFechamentoSemMergeIgnoraOCommitDeMergeDeTesteDoGitHub(t *testing.T) {
	registro := novoRegistroFalso()
	corpo := `{"action":"closed","number":42,"pull_request":{"merged":false,"merge_commit_sha":"` + commitDoMerge + `","closed_at":"` + momentoDoFechamento + `","head":{"sha":"` + headDoFechamento + `"}},"repository":{"full_name":"PI-GRUPO/aPRova"},"installation":{"id":777}}`

	enviarWebhook(t, registro, entregaValida("pull_request", "entrega-merge-de-teste", corpo))

	if len(registro.fechamentos) != 1 || registro.fechamentos[0].SHADoMerge != "" {
		t.Errorf("sem merge, o merge_commit_sha do GitHub é o merge de teste e não é gravado: %+v", registro.fechamentos)
	}
}

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

const slugDeTeste = "aprova-pi"

func corpoReviewDe(estado, revisor, texto string) string {
	textoJSON, _ := json.Marshal(texto)
	return `{"action":"submitted",` +
		`"review":{"state":"` + estado + `","user":{"login":"` + revisor + `"},"body":` + string(textoJSON) + `},` +
		`"pull_request":{"number":14,"head":{"sha":"abc123"}},` +
		`"repository":{"full_name":"Azulpasta/aPRova-TESTES"},"installation":{"id":58291734}}`
}

func enviarReview(t *testing.T, registro *registroFalso, slug, corpo string) *httptest.ResponseRecorder {
	t.Helper()

	servidor := NovoServidor(Opcoes{SegredoWebhook: []byte(segredoDeTeste), Execucoes: registro, SlugDoApp: slug})
	e := entregaValida("pull_request_review", "entrega-review-do-app", corpo)
	requisicao := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/webhooks/github", strings.NewReader(e.corpo))
	requisicao.Header.Set("X-GitHub-Event", e.evento)
	requisicao.Header.Set("X-GitHub-Delivery", e.deliveryID)
	requisicao.Header.Set("X-Hub-Signature-256", e.assinatura)

	gravador := httptest.NewRecorder()
	servidor.Rotas().ServeHTTP(gravador, requisicao)
	return gravador
}

func TestReviewDoAppPeloSlackRegistraQuemDecidiuNoSlack(t *testing.T) {
	casos := map[string]struct {
		estado        string
		corpo         string
		justificativa string
	}{
		"aprovado":                   {"approved", "Aprovado por @Azulpasta via Slack", ""},
		"aprovado com justificativa": {"approved", "Aprovado por @Azulpasta via Slack:\n\nRisco avaliado.\nTestes conferidos.", "Risco avaliado.\nTestes conferidos."},
		"alterações solicitadas":     {"changes_requested", "Alteração solicitada por @Azulpasta via Slack:\n\nfalta o teste de peso zero", "falta o teste de peso zero"},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			registro := novoRegistroFalso()

			resposta := enviarReview(t, registro, slugDeTeste, corpoReviewDe(caso.estado, slugDeTeste+"[bot]", caso.corpo))

			if resposta.Code != http.StatusAccepted || registro.quantidadeDeRevisoes() != 1 {
				t.Fatalf("review do app deveria registrar a decisão: status %d, revisões %d", resposta.Code, registro.quantidadeDeRevisoes())
			}
			if decisao := registro.revisoes[0]; decisao.Revisor != "Azulpasta" || decisao.Origem != domain.OrigemSlack {
				t.Errorf("a decisão precisa ser de quem clicou no Slack: %+v", decisao)
			}
			if decisao := registro.revisoes[0]; decisao.Justificativa != caso.justificativa {
				t.Errorf("a justificativa é o texto digitado no Slack, esperava %q, obtive %q", caso.justificativa, decisao.Justificativa)
			}
		})
	}
}

func TestReviewHumanoNoGitHubTemOrigemGitHub(t *testing.T) {
	registro := novoRegistroFalso()

	enviarReview(t, registro, slugDeTeste, corpoReviewDe("approved", "Azulpasta", "Aprovado por @mallory via Slack"))

	if registro.quantidadeDeRevisoes() != 1 {
		t.Fatalf("review humano deveria ser registrado, revisões %d", registro.quantidadeDeRevisoes())
	}
	if decisao := registro.revisoes[0]; decisao.Revisor != "Azulpasta" || decisao.Origem != domain.OrigemGitHub {
		t.Errorf("review humano decide em nome de quem o fez, mesmo citando outra pessoa no texto: %+v", decisao)
	}
}

func TestReviewHumanoNoGitHubGravaOCorpoInteiroComoJustificativa(t *testing.T) {
	registro := novoRegistroFalso()
	corpo := "Conferi o cálculo do frete.\n\nAprovado por @mallory via Slack"

	enviarReview(t, registro, slugDeTeste, corpoReviewDe("approved", "Azulpasta", corpo))

	if registro.quantidadeDeRevisoes() != 1 {
		t.Fatalf("review humano deveria ser registrado, revisões %d", registro.quantidadeDeRevisoes())
	}
	if justificativa := registro.revisoes[0].Justificativa; justificativa != corpo {
		t.Errorf("a justificativa de um review humano é o corpo inteiro, obtive %q", justificativa)
	}
}

func TestReviewSemCorpoGravaJustificativaVazia(t *testing.T) {
	registro := novoRegistroFalso()
	semCorpo := strings.Replace(corpoReviewDe("approved", "Azulpasta", ""), `"body":""`, `"body":null`, 1)

	resposta := enviarReview(t, registro, slugDeTeste, semCorpo)

	if resposta.Code != http.StatusAccepted || registro.quantidadeDeRevisoes() != 1 {
		t.Fatalf("review sem corpo também decide: status %d, revisões %d", resposta.Code, registro.quantidadeDeRevisoes())
	}
	if justificativa := registro.revisoes[0].Justificativa; justificativa != "" {
		t.Errorf("review sem corpo grava justificativa vazia, obtive %q", justificativa)
	}
}

func TestJustificativaAlemDoLimiteECortadaSemRecusarADecisao(t *testing.T) {
	registro := novoRegistroFalso()
	corpo := strings.Repeat("é", domain.TamanhoMaximoDaJustificativa+1)

	enviarReview(t, registro, slugDeTeste, corpoReviewDe("approved", "Azulpasta", corpo))

	if registro.quantidadeDeRevisoes() != 1 {
		t.Fatalf("review longo também decide, revisões %d", registro.quantidadeDeRevisoes())
	}
	if justificativa := registro.revisoes[0].Justificativa; justificativa != strings.Repeat("é", domain.TamanhoMaximoDaJustificativa) {
		t.Errorf("a justificativa fica com os primeiros %d caracteres, obtive %d", domain.TamanhoMaximoDaJustificativa, len([]rune(justificativa)))
	}
}

func TestReviewDeBotQueNaoProvaVirDoSlackEIgnorado(t *testing.T) {
	casos := map[string]struct {
		slug    string
		revisor string
		corpo   string
	}{
		"corpo fora do formato do aPRova": {slugDeTeste, slugDeTeste + "[bot]", "LGTM"},
		"login inválido no corpo":         {slugDeTeste, slugDeTeste + "[bot]", "Aprovado por @-mallory- via Slack"},
		"outro app":                       {slugDeTeste, "outro-app[bot]", "Aprovado por @Azulpasta via Slack"},
		"slug do app não configurado":     {"", slugDeTeste + "[bot]", "Aprovado por @Azulpasta via Slack"},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			registro := novoRegistroFalso()

			resposta := enviarReview(t, registro, caso.slug, corpoReviewDe("approved", caso.revisor, caso.corpo))

			if resposta.Code != http.StatusNoContent || registro.quantidadeDeRevisoes() != 0 {
				t.Errorf("review de bot sem prova de vir do Slack não pode decidir: status %d, revisões %d", resposta.Code, registro.quantidadeDeRevisoes())
			}
		})
	}
}

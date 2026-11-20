package api

import (
	"strings"
	"testing"
)

const marcaDoCorpoDoPR = "texto-do-corpo-do-pr-4d2e"

func corpoDePullRequestComTexto() string {
	return `{"action":"opened","number":42,` +
		`"pull_request":{"head":{"sha":"abc123"},"user":{"login":"alice"},"body":"` + marcaDoCorpoDoPR + `"},` +
		`"repository":{"full_name":"PI-GRUPO/aPRova"},"installation":{"id":777}}`
}

func TestDiarioDoWebhookNuncaLevaSegredoAssinaturaNemCorpo(t *testing.T) {
	saida := capturarLogJSON(t)
	corpo := corpoDePullRequestComTexto()
	valida := entregaValida("pull_request", "entrega-valida", corpo)
	invalida := entrega{evento: "pull_request", deliveryID: "entrega-invalida", assinatura: "sha256=" + strings.Repeat("ab", 32), corpo: corpo}

	enviarWebhook(t, novoRegistroFalso(), valida)
	enviarWebhook(t, novoRegistroFalso(), invalida)
	enviarWebhook(t, novoRegistroFalso(), entregaValida("pull_request", "entrega-quebrada", `{"action":`+marcaDoCorpoDoPR))

	registro := saida.String()
	if !strings.Contains(registro, "entrega-invalida") {
		t.Fatalf("as três entregas precisam aparecer no log, obtive:\n%s", registro)
	}
	sensiveis := map[string]string{
		"segredo do webhook":   segredoDeTeste,
		"assinatura válida":    strings.TrimPrefix(valida.assinatura, "sha256="),
		"assinatura inválida":  strings.TrimPrefix(invalida.assinatura, "sha256="),
		"texto do corpo do PR": marcaDoCorpoDoPR,
		"corpo inteiro":        corpo,
	}
	for nome, sensivel := range sensiveis {
		if strings.Contains(registro, sensivel) {
			t.Errorf("o log contém %s:\n%s", nome, registro)
		}
	}
}

package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

var formatoDoRequestID = regexp.MustCompile(`^[0-9a-f]{32}$`)

func linhasDoLog(t *testing.T, texto string) []map[string]any {
	t.Helper()

	var linhas []map[string]any
	for _, linha := range strings.Split(strings.TrimSpace(texto), "\n") {
		if linha == "" {
			continue
		}
		var campos map[string]any
		if err := json.Unmarshal([]byte(linha), &campos); err != nil {
			t.Fatalf("linha de log não é JSON: %q", linha)
		}
		linhas = append(linhas, campos)
	}
	return linhas
}

func linhaDeAcesso(t *testing.T, linhas []map[string]any) map[string]any {
	t.Helper()

	for _, campos := range linhas {
		if campos["msg"] == "requisição atendida" {
			return campos
		}
	}
	t.Fatalf("nenhuma linha de acesso entre %v", linhas)
	return nil
}

func TestRequestIDEAleatorioESemONomeDaMaquina(t *testing.T) {
	saida := capturarLogJSON(t)
	rotas := NovoServidor(Opcoes{}).Rotas()

	for range 2 {
		rotas.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/health", nil))
	}

	maquina, _ := os.Hostname()
	vistos := map[string]bool{}
	for _, campos := range linhasDoLog(t, saida.String()) {
		id, _ := campos["request_id"].(string)
		if !formatoDoRequestID.MatchString(id) || (maquina != "" && strings.Contains(id, maquina)) {
			t.Errorf("request_id precisa ser aleatório, hexadecimal e sem o nome da máquina: %q", id)
		}
		vistos[id] = true
	}
	if len(vistos) != 2 {
		t.Errorf("cada requisição precisa de um request_id próprio, vi %v", vistos)
	}
}

func TestTodaLinhaDaRequisicaoLevaORequestID(t *testing.T) {
	casos := map[string]func(t *testing.T){
		"webhook com assinatura inválida": func(t *testing.T) {
			enviarWebhook(t, novoRegistroFalso(), entrega{
				evento: "pull_request", deliveryID: "entrega-sem-assinatura",
				assinatura: "sha256=" + strings.Repeat("0", 64), corpo: corpoPullRequest("opened"),
			})
		},
		"webhook sem decisão": func(t *testing.T) {
			enviarWebhook(t, novoRegistroFalso(), entregaValida("pull_request_review", "entrega-comentario", corpoReview("commented")))
		},
		"ação remota sem autenticação": func(t *testing.T) {
			enviarAcao(t, segredoDasAcoesDeTeste, &processadorFalso{}, pedidoAssinado{corpo: corpoDeAcaoDeTeste})
		},
	}

	for nome, enviar := range casos {
		t.Run(nome, func(t *testing.T) {
			saida := capturarLogJSON(t)

			enviar(t)

			linhas := linhasDoLog(t, saida.String())
			id := linhaDeAcesso(t, linhas)["request_id"]
			if len(linhas) < 2 {
				t.Fatalf("o caso precisa registrar ao menos uma linha além do acesso: %v", linhas)
			}
			for _, campos := range linhas {
				if campos["request_id"] == nil || campos["request_id"] != id {
					t.Errorf("linha sem o request_id da requisição %v: %v", id, campos)
				}
			}
		})
	}
}

func TestLinhaDeAcessoDoWebhookLevaODeliveryID(t *testing.T) {
	saida := capturarLogJSON(t)

	enviarWebhook(t, novoRegistroFalso(), entregaValida("pull_request", "entrega-acesso-42", corpoPullRequest("opened")))

	if acesso := linhaDeAcesso(t, linhasDoLog(t, saida.String())); acesso["delivery_id"] != "entrega-acesso-42" {
		t.Errorf("a linha de acesso do webhook precisa do delivery_id: %v", acesso)
	}
}

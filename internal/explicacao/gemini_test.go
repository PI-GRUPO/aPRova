package explicacao

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const chaveDeTeste = "chave-secreta-de-teste"

type pedidoRecebido struct {
	caminho string
	chave   string
	corpo   map[string]any
	bruto   string
}

func servidorDoGemini(t *testing.T, status int, resposta string) (*Gemini, *pedidoRecebido) {
	t.Helper()

	recebido := &pedidoRecebido{}
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bruto, _ := io.ReadAll(r.Body)
		recebido.caminho, recebido.chave, recebido.bruto = r.URL.String(), r.Header.Get("x-goog-api-key"), string(bruto)
		_ = json.Unmarshal(bruto, &recebido.corpo)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, resposta)
	}))
	t.Cleanup(servidor.Close)

	gemini, err := NovoGemini(OpcoesGemini{Chave: chaveDeTeste, Modelo: "gemini-3.5-flash-lite", URLBase: servidor.URL, ClienteHTTP: servidor.Client()})
	if err != nil {
		t.Fatalf("montar o Gemini: %v", err)
	}
	return gemini, recebido
}

func respostaComTexto(partes ...string) string {
	var lista []map[string]string
	for _, parte := range partes {
		lista = append(lista, map[string]string{"text": parte})
	}
	corpo, _ := json.Marshal(map[string]any{"candidates": []any{map[string]any{
		"content": map[string]any{"parts": lista}, "finishReason": "STOP",
	}}})
	return string(corpo)
}

func TestNovoGeminiExigeChaveEModelo(t *testing.T) {
	if _, err := NovoGemini(OpcoesGemini{Modelo: "gemini-3.5-flash-lite"}); !errors.Is(err, ErrConfiguracao) {
		t.Errorf("sem chave: %v", err)
	}
	if _, err := NovoGemini(OpcoesGemini{Chave: chaveDeTeste}); !errors.Is(err, ErrConfiguracao) {
		t.Errorf("sem modelo: %v", err)
	}
}

func TestGerarMandaSistemaConteudoEThinkingMinimoComAChaveSoNoCabecalho(t *testing.T) {
	gemini, recebido := servidorDoGemini(t, http.StatusOK, respostaComTexto("Risco alto, ", "merge bloqueado."))

	texto, err := gemini.Gerar(t.Context(), "instrução do sistema", `{"nivel":"alto"}`)

	if err != nil || texto != "Risco alto, merge bloqueado." {
		t.Fatalf("texto = %q, erro %v", texto, err)
	}
	if recebido.caminho != "/v1beta/models/gemini-3.5-flash-lite:generateContent" || recebido.chave != chaveDeTeste {
		t.Errorf("caminho %q, chave no cabeçalho %q", recebido.caminho, recebido.chave)
	}
	if strings.Contains(recebido.caminho, chaveDeTeste) || strings.Contains(recebido.bruto, chaveDeTeste) {
		t.Error("a chave só vai no cabeçalho")
	}
	configuracao, _ := recebido.corpo["generationConfig"].(map[string]any)
	pensamento, _ := configuracao["thinkingConfig"].(map[string]any)
	if pensamento["thinkingLevel"] != "minimal" || configuracao["temperature"] == nil || configuracao["maxOutputTokens"] == nil {
		t.Errorf("generationConfig = %v", configuracao)
	}
	if !strings.Contains(recebido.bruto, "instrução do sistema") || !strings.Contains(recebido.bruto, `{\"nivel\":\"alto\"}`) {
		t.Errorf("sistema e conteúdo não foram no pedido: %s", recebido.bruto)
	}
}

func TestRespostaForaDoNormalViraErroSemAChave(t *testing.T) {
	casos := map[string]struct {
		status   int
		resposta string
	}{
		"status 429":         {http.StatusTooManyRequests, `{"error":{"message":"quota"}}`},
		"cortada por tokens": {http.StatusOK, `{"candidates":[{"content":{"parts":[{"text":"Risco"}]},"finishReason":"MAX_TOKENS"}]}`},
		"sem candidato":      {http.StatusOK, `{"candidates":[]}`},
		"não é JSON":         {http.StatusOK, `oi`},
	}
	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			gemini, _ := servidorDoGemini(t, caso.status, caso.resposta)

			_, err := gemini.Gerar(t.Context(), "sistema", "conteúdo")

			if err == nil || strings.Contains(err.Error(), chaveDeTeste) {
				t.Errorf("esperava erro sem a chave, obtive %v", err)
			}
		})
	}
}

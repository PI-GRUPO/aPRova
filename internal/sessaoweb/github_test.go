package sessaoweb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

type transporteFunc func(*http.Request) (*http.Response, error)

func (f transporteFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func respostaTeste(v any, status int) *http.Response {
	w := httptest.NewRecorder()
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
	return w.Result()
}
func TestGitHubEnviaPKCEESegredoSoNoCorpo(t *testing.T) {
	g := &GitHub{ClientID: "cliente", ClientSecret: "segredo-nao-logavel", Callback: "https://aprova.test/auth/github/callback"}
	u, _ := url.Parse(g.URLAutorizacao("state", strings.Repeat("a", 43)))
	if u.Host != "github.com" || u.Query().Get("code_challenge_method") != "S256" || u.Query().Get("code_challenge") == "" || strings.Contains(u.String(), g.ClientSecret) {
		t.Fatal(u)
	}
	g.Cliente = &http.Client{Transport: transporteFunc(func(r *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(r.Body)
		q, _ := url.ParseQuery(string(b))
		if r.URL.RawQuery != "" || q.Get("client_secret") != g.ClientSecret || q.Get("code_verifier") != "verifier" || q.Get("redirect_uri") != g.Callback {
			t.Fatal("troca OAuth não preservou o contrato")
		}
		return respostaTeste(map[string]any{"access_token": "token-privado", "token_type": "bearer", "expires_in": 3600}, 200), nil
	})}
	token, err := g.Trocar(context.Background(), "codigo", "verifier")
	if err != nil || token.Duracao != time.Hour || token.Valor != "token-privado" {
		t.Fatal(token.Duracao, err)
	}
}
func TestGitHubPaginaAteOFinalSemExporToken(t *testing.T) {
	chamadas := 0
	g := &GitHub{Cliente: &http.Client{Transport: transporteFunc(func(r *http.Request) (*http.Response, error) {
		chamadas++
		if r.Header.Get("Authorization") != "Bearer privado" || strings.Contains(r.URL.String(), "privado") {
			t.Fatal("autorização incorreta")
		}
		pagina, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if r.URL.Path == "/user/installations" {
			items := []map[string]any{}
			quantidade := 100
			if pagina == 2 {
				quantidade = 1
			}
			for n := range quantidade {
				items = append(items, map[string]any{"id": (pagina-1)*100 + n + 1, "account": map[string]string{"login": "equipe"}})
			}
			return respostaTeste(map[string]any{"total_count": 101, "installations": items}, 200), nil
		}
		items := []map[string]string{}
		quantidade := 100
		if pagina == 2 {
			quantidade = 1
		}
		for n := range quantidade {
			items = append(items, map[string]string{"full_name": fmt.Sprintf("equipe/repo-%d-%d", pagina, n)})
		}
		return respostaTeste(map[string]any{"total_count": 101, "repositories": items}, 200), nil
	})}}
	instalacoes, err := g.Instalacoes(t.Context(), "privado")
	if err != nil || len(instalacoes) != 101 {
		t.Fatal(len(instalacoes), err)
	}
	repos, err := g.Repositorios(t.Context(), "privado", 42)
	if err != nil || len(repos) != 101 || chamadas != 4 {
		t.Fatal(len(repos), chamadas, err)
	}
}
func TestGitHubFalhaFechadaEmRevogacaoRateLimitERespostaInvalida(t *testing.T) {
	for _, status := range []int{401, 403, 429, 500} {
		g := &GitHub{Cliente: &http.Client{Transport: transporteFunc(func(*http.Request) (*http.Response, error) {
			return respostaTeste(map[string]string{"message": "detalhe-secreto"}, status), nil
		})}}
		_, err := g.Instalacoes(t.Context(), "token")
		if err == nil || strings.Contains(err.Error(), "detalhe-secreto") {
			t.Fatal(err)
		}
	}
}

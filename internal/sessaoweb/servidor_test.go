package sessaoweb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

type memoria struct {
	mu      sync.Mutex
	valores map[string][]byte
}

func (m *memoria) Guardar(_ context.Context, k string, v []byte, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.valores[k] = v
	return nil
}
func (m *memoria) Buscar(_ context.Context, k string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if v, ok := m.valores[k]; ok {
		return v, nil
	}
	return nil, ErrSemSessao
}
func (m *memoria) Consumir(ctx context.Context, k string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.valores[k]
	if !ok {
		return nil, ErrSemSessao
	}
	delete(m.valores, k)
	return v, nil
}
func (m *memoria) Excluir(_ context.Context, k string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.valores, k)
	return nil
}

type provedorTeste struct {
	verifier string
	trocas   int
	revogado bool
	repos    []string
	erro     error
}

func (p *provedorTeste) URLAutorizacao(state, verifier string) string {
	p.verifier = verifier
	return "https://github.com/login/oauth/authorize?state=" + state
}
func (p *provedorTeste) Trocar(_ context.Context, code, verifier string) (Token, error) {
	p.trocas++
	if code != "codigo" || verifier != p.verifier {
		return Token{}, errors.New("PKCE incorreto")
	}
	return Token{"token-somente-servidor", time.Hour}, nil
}
func (p *provedorTeste) Usuario(context.Context, string) (Usuario, error) {
	return Usuario{7, "pessoa"}, p.erro
}
func (p *provedorTeste) Instalacoes(context.Context, string) ([]Instalacao, error) {
	if p.revogado {
		return nil, ErrSemSessao
	}
	return []Instalacao{{42, "equipe"}, {43, "outra-equipe"}}, p.erro
}
func (p *provedorTeste) Repositorios(_ context.Context, _ string, id int64) ([]string, error) {
	if id == 42 {
		return p.repos, p.erro
	}
	return []string{"outra/equipe"}, p.erro
}

type relatorioTeste struct {
	chamadas    int
	id          int64
	repos       []string
	inicio, fim time.Time
}

func (r *relatorioTeste) RelatorioSemanalDosRepositorios(_ context.Context, id int64, repos []string, inicio, fim time.Time) (domain.RelatorioSemanal, error) {
	r.chamadas++
	r.id = id
	r.repos = repos
	r.inicio = inicio
	r.fim = fim
	return domain.RelatorioSemanal{TenantID: id, Inicio: inicio, Fim: fim}, nil
}
func cenario(t *testing.T) (*Servidor, *memoria, *provedorTeste, *relatorioTeste) {
	t.Helper()
	m := &memoria{valores: map[string][]byte{}}
	p := &provedorTeste{repos: []string{"equipe/autorizado"}}
	r := &relatorioTeste{}
	s, e := Novo(Opcoes{Origem: "https://aprova.test", Provedor: p, Armazenamento: m, Relatorios: r, Agora: func() time.Time { return time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC) }})
	if e != nil {
		t.Fatal(e)
	}
	return s, m, p, r
}
func pedir(s *Servidor, metodo, path string, cookie *http.Cookie, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), metodo, "https://aprova.test"+path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)
	return w
}
func iniciar(t *testing.T, s *Servidor) (string, *http.Cookie) {
	t.Helper()
	w := pedir(s, "GET", "/auth/github", nil, nil)
	if w.Code != 303 {
		t.Fatal(w.Code, w.Body.String())
	}
	u, _ := url.Parse(w.Header().Get("Location"))
	return u.Query().Get("state"), w.Result().Cookies()[0]
}
func login(t *testing.T, s *Servidor) *http.Cookie {
	t.Helper()
	state, c := iniciar(t, s)
	w := pedir(s, "GET", "/auth/github/callback?state="+state+"&code=codigo", c, nil)
	if w.Code != 303 || w.Header().Get("Location") != "/" {
		t.Fatal(w.Code, w.Body.String(), w.Header())
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == s.nomeCookie() {
			return c
		}
	}
	t.Fatal("sessão não emitida")
	return nil
}
func TestLoginStatePKCECookieESemCredenciaisNoJSON(t *testing.T) {
	s, _, p, _ := cenario(t)
	c := login(t, s)
	if len(p.verifier) != 43 || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || len(c.Value) != 43 || c.Domain != "" {
		t.Fatal("cookie/PKCE inválidos", c)
	}
	w := pedir(s, "GET", "/api/sessao", c, nil)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "pessoa") || strings.Contains(w.Body.String(), "token-somente-servidor") || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestCallbackRecusaStateReutilizadoAusenteETrocado(t *testing.T) {
	for _, caso := range []string{"ausente", "trocado", "replay", "expirado"} {
		t.Run(caso, func(t *testing.T) {
			s, m, p, _ := cenario(t)
			state, c := iniciar(t, s)
			switch caso {
			case "ausente":
				c = nil
			case "trocado":
				c.Value = strings.Repeat("z", 43)
			case "replay":
				_ = pedir(s, "GET", "/auth/github/callback?state="+state+"&code=codigo", c, nil)
			case "expirado":
				_ = m.Excluir(t.Context(), "oauth:"+state)
			}
			antes := p.trocas
			w := pedir(s, "GET", "/auth/github/callback?state="+state+"&code=codigo", c, nil)
			if w.Header().Get("Location") != "/entrar?erro=autenticacao" || p.trocas != antes {
				t.Fatal("callback aceitou tentativa inválida", w.Header())
			}
		})
	}
}
func TestEscopoDoRelatorioNaoConfiaNoCliente(t *testing.T) {
	for _, path := range []string{"?installation_id=99", "?installation_id=42&repositorio=equipe/privado", "?installation_id=42&tenant_id=99", "?installation_id=42&installation_id=99", "?installation_id=42&inicio=ontem"} {
		t.Run(path, func(t *testing.T) {
			s, _, _, r := cenario(t)
			w := pedir(s, "GET", "/api/web/relatorio-semanal"+path, login(t, s), nil)
			if w.Code != 403 && w.Code != 400 {
				t.Fatal(w.Code, w.Body.String())
			}
			if r.chamadas != 0 {
				t.Fatal("consulta de dados fora do escopo")
			}
		})
	}
	s, _, _, r := cenario(t)
	c := login(t, s)
	w := pedir(s, "GET", "/api/web/relatorio-semanal?installation_id=42", c, nil)
	if w.Code != 200 || r.id != 42 || len(r.repos) != 1 || r.repos[0] != "equipe/autorizado" || r.inicio.Format(time.DateOnly) != "2026-11-09" || r.fim.Format(time.DateOnly) != "2026-11-16" || !strings.Contains(w.Body.String(), `"taxa_sucesso":null`) {
		t.Fatal(w.Code, w.Body.String(), r)
	}
	w = pedir(s, "GET", "/api/web/relatorio-semanal?installation_id=43", c, nil)
	if w.Code != 200 || r.id != 43 || r.repos[0] != "outra/equipe" {
		t.Fatal("troca de escopo não respeitada")
	}
}
func TestRevogacaoNaoEntregaRelatorio(t *testing.T) {
	s, _, p, r := cenario(t)
	c := login(t, s)
	p.revogado = true
	w := pedir(s, "GET", "/api/web/relatorio-semanal?installation_id=42", c, nil)
	if w.Code != 401 || r.chamadas != 0 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = pedir(s, "GET", "/api/web/relatorio-semanal?installation_id=42", c, nil)
	if w.Code != 401 {
		t.Fatal("sessão revogada persistiu")
	}
}
func TestLogoutExigeOrigemECSRFInvalidaSessao(t *testing.T) {
	s, _, _, _ := cenario(t)
	c := login(t, s)
	w := pedir(s, "GET", "/api/sessao", c, nil)
	var v struct {
		CSRF string `json:"csrf_token"`
	}
	if json.Unmarshal(w.Body.Bytes(), &v) != nil {
		t.Fatal("json inválido")
	}
	for _, h := range []map[string]string{nil, {"Origin": "https://invasor.test", "X-CSRF-Token": v.CSRF}, {"Origin": "https://aprova.test", "X-CSRF-Token": "incorreto"}} {
		if w := pedir(s, "POST", "/api/sessao/sair", c, h); w.Code != 403 {
			t.Fatal(w.Code)
		}
	}
	w = pedir(s, "POST", "/api/sessao/sair", c, map[string]string{"Origin": "https://aprova.test", "X-CSRF-Token": v.CSRF})
	if w.Code != 204 {
		t.Fatal(w.Code)
	}
	w = pedir(s, "GET", "/api/sessao", c, nil)
	if w.Code != 401 {
		t.Fatal("logout manteve acesso")
	}
}
func TestExpiracaoRecursoPrivadoENenhumRepositorio(t *testing.T) {
	s, _, p, r := cenario(t)
	c := login(t, s)
	p.repos = []string{}
	if w := pedir(s, "GET", "/api/web/relatorio-semanal?installation_id=42", c, nil); w.Code != 403 || r.chamadas != 0 {
		t.Fatal(w.Code)
	}
	antes := s.agora
	s.agora = func() time.Time { return antes().Add(2 * time.Hour) }
	if w := pedir(s, "GET", "/api/sessao", c, nil); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := pedir(s, "GET", "/api/web/relatorio-semanal?installation_id=42", nil, nil); w.Code != 401 {
		t.Fatal(w.Code)
	}
}
func TestLoginDesabilitadoEOrigemSegura(t *testing.T) {
	var s *Servidor
	if w := pedir(s, "GET", "/api/sessao", nil, nil); w.Code != 200 || !strings.Contains(w.Body.String(), `"login_disponivel":false`) {
		t.Fatal(w.Code)
	}
	if w := pedir(s, "GET", "/auth/github", nil, nil); w.Code != 503 {
		t.Fatal(w.Code)
	}
	for _, origem := range []string{"http://publico.test", "https://user:senha@aprova.test", "https://aprova.test/caminho", "https://aprova.test?token=segredo"} {
		if _, e := Novo(Opcoes{Origem: origem}); e == nil {
			t.Fatal("origem inválida aceita")
		}
	}
}

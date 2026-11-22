package sessaoweb

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

type Relatorios interface {
	RelatorioSemanalDosRepositorios(context.Context, int64, []string, time.Time, time.Time) (domain.RelatorioSemanal, error)
}
type Opcoes struct {
	Leituras      LeiturasPainel
	Vinculos      map[string]string
	Origem        string
	Provedor      Provedor
	Armazenamento Armazenamento
	Analises      Analises
	Relatorios    Relatorios
	Agora         func() time.Time
}
type Servidor struct {
	leituras      LeiturasPainel
	vinculos      map[string]string
	origem        string
	seguro        bool
	provedor      Provedor
	armazenamento Armazenamento
	analises      Analises
	relatorios    Relatorios
	agora         func() time.Time
}
type sessao struct {
	Token   string
	Usuario Usuario
	CSRF    string
	Expira  time.Time
}
type fluxo struct{ Verifier string }

func Novo(op Opcoes) (*Servidor, error) {
	u, err := origemValida(op.Origem)
	if err != nil {
		return nil, err
	}
	if op.Provedor == nil || op.Armazenamento == nil {
		return nil, errors.New("sessão web exige provedor e armazenamento")
	}
	if op.Agora == nil {
		op.Agora = time.Now
	}
	return &Servidor{leituras: op.Leituras, vinculos: op.Vinculos, origem: u.Scheme + "://" + u.Host, seguro: u.Scheme == "https", provedor: op.Provedor, armazenamento: op.Armazenamento, analises: op.Analises, relatorios: op.Relatorios, agora: op.Agora}, nil
}
func origemValida(origem string) (*url.URL, error) {
	erro := errors.New("WEB_URL_PUBLICA deve ser uma origem HTTPS, ou HTTP em localhost")
	u, e := url.Parse(origem)
	if e != nil {
		return nil, erro
	}
	if u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, erro
	}
	if u.Path != "" && u.Path != "/" {
		return nil, erro
	}
	if u.Scheme == "https" {
		return u, nil
	}
	if u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1") {
		return u, nil
	}
	return nil, erro
}

type respostaSessao struct {
	Autenticado     bool         `json:"autenticado"`
	LoginDisponivel bool         `json:"login_disponivel"`
	Usuario         *Usuario     `json:"usuario,omitempty"`
	Instalacoes     []Instalacao `json:"instalacoes"`
	CSRF            string       `json:"csrf_token,omitempty"`
	Expira          time.Time    `json:"expira_em,omitzero"`
}

func (s *Servidor) nomeCookie() string {
	if s.seguro {
		return "__Host-aprova_sessao"
	}
	return "aprova_sessao"
}
func (s *Servidor) cookie(w http.ResponseWriter, nome, valor string, duracao time.Duration) {
	idade := int(duracao.Seconds())
	if duracao < 0 {
		idade = -1
	}
	http.SetCookie(w, &http.Cookie{Name: nome, Value: valor, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: idade, Expires: s.agora().Add(duracao)})
}
func aleatorio() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("gerador aleatório indisponível")
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}
func jsonResposta(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func falhar(w http.ResponseWriter, status int, msg string) {
	jsonResposta(w, status, map[string]string{"status": msg})
}
func guardar(ctx context.Context, a Armazenamento, id string, v any, ttl time.Duration) error {
	b, e := json.Marshal(v)
	if e != nil {
		return e
	}
	return a.Guardar(ctx, id, b, ttl)
}
func (s *Servidor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if s == nil {
		if r.URL.Path == "/api/sessao" && r.Method == http.MethodGet {
			jsonResposta(w, 200, respostaSessao{})
			return
		}
		falhar(w, 503, "login web indisponível")
		return
	}
	switch r.Method + " " + r.URL.Path {
	case "GET /auth/github":
		s.iniciar(w, r)
	case "GET /auth/github/callback":
		s.callback(w, r)
	case "GET /api/sessao":
		s.consultar(w, r)
	case "POST /api/sessao/sair":
		s.sair(w, r)
	case "GET /api/metricas", "GET /api/web/recibos", "GET /api/web/auditoria", "GET /api/web/regras":
		s.leituraPainel(w, r, path.Base(r.URL.Path))
	case "GET /api/web/vinculo-slack":
		s.vinculoPessoal(w, r)
	case "GET /api/repositorios":
		s.repositorios(w, r)
	case "GET /api/web/relatorio-semanal":
		s.relatorio(w, r)
	default:
		falhar(w, 405, "método não permitido")
	}
}
func (s *Servidor) iniciar(w http.ResponseWriter, r *http.Request) {
	state, verifier := aleatorio(), aleatorio()
	if err := guardar(r.Context(), s.armazenamento, "oauth:"+state, fluxo{verifier}, 5*time.Minute); err != nil {
		falhar(w, 503, "autenticação indisponível")
		return
	}
	s.cookie(w, s.nomeCookie()+"_oauth", state, 5*time.Minute)
	http.Redirect(w, r, s.provedor.URLAutorizacao(state, verifier), http.StatusSeeOther)
}
func (s *Servidor) callback(w http.ResponseWriter, r *http.Request) {
	s.cookie(w, s.nomeCookie()+"_oauth", "", -time.Hour)
	falha := func() { http.Redirect(w, r, "/entrar?erro=autenticacao", http.StatusSeeOther) }
	f, e := s.consumirFluxo(r)
	if e != nil {
		falha()
		return
	}
	token, e := s.provedor.Trocar(r.Context(), r.URL.Query().Get("code"), f.Verifier)
	if e != nil || token.Valor == "" || token.Duracao <= 0 {
		falha()
		return
	}
	usuario, e := s.provedor.Usuario(r.Context(), token.Valor)
	if e != nil || usuario.ID <= 0 || usuario.Login == "" {
		falha()
		return
	}
	ttl := min(token.Duracao, 8*time.Hour)
	id := aleatorio()
	v := sessao{token.Valor, usuario, aleatorio(), s.agora().Add(ttl)}
	if guardar(r.Context(), s.armazenamento, "sessao:"+id, v, ttl) != nil {
		falha()
		return
	}
	if s.excluirSessaoAnterior(r) != nil {
		_ = s.armazenamento.Excluir(r.Context(), "sessao:"+id)
		falha()
		return
	}
	s.cookie(w, s.nomeCookie(), id, ttl)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
func (s *Servidor) consumirFluxo(r *http.Request) (fluxo, error) {
	var f fluxo
	state := r.URL.Query().Get("state")
	c, e := r.Cookie(s.nomeCookie() + "_oauth")
	if e != nil || len(state) != 43 || subtle.ConstantTimeCompare([]byte(state), []byte(c.Value)) != 1 {
		return f, ErrSemSessao
	}
	b, e := s.armazenamento.Consumir(r.Context(), "oauth:"+state)
	if e != nil || json.Unmarshal(b, &f) != nil || len(f.Verifier) != 43 {
		return fluxo{}, ErrSemSessao
	}
	if len(r.URL.Query()["code"]) != 1 || r.URL.Query().Get("code") == "" || r.URL.Query().Get("error") != "" {
		return fluxo{}, ErrSemSessao
	}
	return f, nil
}
func (s *Servidor) excluirSessaoAnterior(r *http.Request) error {
	if c, e := r.Cookie(s.nomeCookie()); e == nil {
		return s.armazenamento.Excluir(r.Context(), "sessao:"+c.Value)
	}
	return nil
}
func (s *Servidor) autenticar(w http.ResponseWriter, r *http.Request) (sessao, bool) {
	var v sessao
	c, e := r.Cookie(s.nomeCookie())
	if e != nil || len(c.Value) != 43 {
		falhar(w, 401, "sessão ausente ou expirada")
		return v, false
	}
	b, e := s.armazenamento.Buscar(r.Context(), "sessao:"+c.Value)
	if e != nil {
		if !errors.Is(e, ErrSemSessao) {
			falhar(w, 503, "sessão temporariamente indisponível")
			return v, false
		}
		s.cookie(w, s.nomeCookie(), "", -time.Hour)
		falhar(w, 401, "sessão ausente ou expirada")
		return v, false
	}
	if json.Unmarshal(b, &v) != nil || !s.agora().Before(v.Expira) || v.Token == "" {
		s.cookie(w, s.nomeCookie(), "", -time.Hour)
		falhar(w, 401, "sessão ausente ou expirada")
		return v, false
	}
	return v, true
}
func (s *Servidor) erroProvedor(w http.ResponseWriter, r *http.Request, e error) {
	if errors.Is(e, ErrSemSessao) {
		if c, err := r.Cookie(s.nomeCookie()); err == nil {
			_ = s.armazenamento.Excluir(r.Context(), "sessao:"+c.Value)
		}
		s.cookie(w, s.nomeCookie(), "", -time.Hour)
		falhar(w, 401, "sessão expirada; entre novamente")
		return
	}
	falhar(w, 503, "não foi possível confirmar o acesso no GitHub")
}
func (s *Servidor) consultar(w http.ResponseWriter, r *http.Request) {
	if _, e := r.Cookie(s.nomeCookie()); errors.Is(e, http.ErrNoCookie) {
		jsonResposta(w, 200, respostaSessao{LoginDisponivel: true})
		return
	}
	v, ok := s.autenticar(w, r)
	if !ok {
		return
	}
	// Confere revogação e identidade a cada consulta, sem confiar no navegador.
	usuario, e := s.provedor.Usuario(r.Context(), v.Token)
	if e != nil {
		s.erroProvedor(w, r, e)
		return
	}
	if usuario.ID != v.Usuario.ID {
		falhar(w, 401, "identidade da sessão inválida")
		return
	}
	instalacoes, e := s.provedor.Instalacoes(r.Context(), v.Token)
	if e != nil {
		s.erroProvedor(w, r, e)
		return
	}
	jsonResposta(w, 200, respostaSessao{Autenticado: true, LoginDisponivel: true, Usuario: &usuario, Instalacoes: instalacoes, CSRF: v.CSRF, Expira: v.Expira})
}
func (s *Servidor) sair(w http.ResponseWriter, r *http.Request) {
	v, ok := s.autenticar(w, r)
	if !ok {
		return
	}
	if r.Header.Get("Origin") != s.origem || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(v.CSRF)) != 1 {
		falhar(w, 403, "origem ou proteção CSRF inválida")
		return
	}
	c, _ := r.Cookie(s.nomeCookie())
	if s.armazenamento.Excluir(r.Context(), "sessao:"+c.Value) != nil {
		falhar(w, 503, "não foi possível encerrar a sessão")
		return
	}
	s.cookie(w, s.nomeCookie(), "", -time.Hour)
	w.WriteHeader(204)
}
func (s *Servidor) relatorio(w http.ResponseWriter, r *http.Request) {
	v, ok := s.autenticar(w, r)
	if !ok {
		return
	}
	id, inicio, e := consultaSemanal(r.URL.Query(), s.agora())
	if e != nil {
		falhar(w, 400, e.Error())
		return
	}
	fim := inicio.AddDate(0, 0, 7)
	repos, e := s.repositoriosAutorizados(r.Context(), v.Token, id, r.URL.Query().Get("repositorio"))
	if errors.Is(e, errSemAcesso) {
		falhar(w, 403, "instalação ou repositório não autorizado")
		return
	}
	if e != nil {
		s.erroProvedor(w, r, e)
		return
	}
	if s.relatorios == nil {
		falhar(w, 503, "relatório indisponível")
		return
	}
	rel, e := s.relatorios.RelatorioSemanalDosRepositorios(r.Context(), id, repos, inicio, fim)
	if e != nil {
		falhar(w, 500, "não foi possível montar o relatório")
		return
	}
	jsonResposta(w, 200, map[string]any{parametroInstalacaoWeb: id, "repositorios": repos, "inicio": inicio, "fim": fim, "total_execucoes": rel.Total, "concluidas": rel.Concluidas, "rejeitadas": rel.Rejeitadas, "falhas": rel.Falhas, "em_andamento": rel.EmAndamento(), "taxa_sucesso": rel.TaxaDeSucesso(), "exigiram_aprovacao_humana": rel.ExigiramAprovacao, "risco_medio": rel.RiscoMedio, "revertidas": rel.Revertidas})
}

var errSemAcesso = errors.New("escopo não autorizado")

func consultaSemanal(q url.Values, agora time.Time) (int64, time.Time, error) {
	for nome, valores := range q {
		if !slices.Contains([]string{parametroInstalacaoWeb, "repositorio", "inicio"}, nome) || len(valores) != 1 {
			return 0, time.Time{}, errors.New("parâmetros de consulta inválidos")
		}
	}
	id, e := strconv.ParseInt(q.Get(parametroInstalacaoWeb), 10, 64)
	if e != nil || id <= 0 {
		return 0, time.Time{}, errors.New("installation_id inválido")
	}
	inicio, e := inicioSemanal(q.Get("inicio"), agora)
	if e != nil {
		return 0, time.Time{}, errors.New("inicio deve ser uma data AAAA-MM-DD")
	}
	return id, inicio, nil
}
func (s *Servidor) repositoriosAutorizados(ctx context.Context, token string, id int64, nome string) ([]string, error) {
	instalacoes, e := s.provedor.Instalacoes(ctx, token)
	if e != nil {
		return nil, e
	}
	if !slices.ContainsFunc(instalacoes, func(i Instalacao) bool { return i.ID == id }) {
		return nil, errSemAcesso
	}
	repos, e := s.provedor.Repositorios(ctx, token, id)
	if e != nil {
		return nil, e
	}
	if len(repos) == 0 {
		return nil, errSemAcesso
	}
	if nome == "" {
		return repos, nil
	}
	if !slices.Contains(repos, nome) {
		return nil, errSemAcesso
	}
	return []string{nome}, nil
}
func inicioSemanal(v string, agora time.Time) (time.Time, error) {
	if v != "" {
		return time.Parse(time.DateOnly, v)
	}
	hoje := time.Date(agora.UTC().Year(), agora.UTC().Month(), agora.UTC().Day(), 0, 0, 0, 0, time.UTC)
	return hoje.AddDate(0, 0, -((int(hoje.Weekday())+6)%7)-7), nil
}

// Callback devolve o endereço fixo cadastrado no GitHub App, sem confiar em Host/X-Forwarded-Host.
func Callback(origem string) string {
	return fmt.Sprintf("%s/auth/github/callback", strings.TrimRight(origem, "/"))
}

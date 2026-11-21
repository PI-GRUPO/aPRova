package slackhome

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct{ Token, TeamID, AppID, SigningSecret, APIURL, WebURL string }

var teamID = regexp.MustCompile(`^T[A-Z0-9]+$`)
var appID = regexp.MustCompile(`^A[A-Z0-9]+$`)
var usuarioID = regexp.MustCompile(`^[UW][A-Z0-9]+$`)

func origem(raw string, publica bool) bool {
	u, e := url.Parse(raw)
	return e == nil && u.Hostname() != "" && u.User == nil && u.RawQuery == "" && !u.ForceQuery && u.Fragment == "" && (u.Path == "" || u.Path == "/") && (u.Scheme == "https" || (!publica && u.Scheme == "http"))
}
func (c Config) Validar() error {
	if !teamID.MatchString(c.TeamID) || !appID.MatchString(c.AppID) || c.Token == "" || c.SigningSecret == "" {
		return errors.New("configure o bot, o workspace, o app e o segredo de assinatura Slack")
	}
	if !origem(c.APIURL, false) || c.WebURL != "" && !origem(c.WebURL, true) {
		return errors.New("configure uma origem HTTP(S) para a API e uma origem HTTPS para o dashboard")
	}
	return nil
}
func clienteHTTP(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}
func lerJSON(r io.Reader, v any) error {
	b, err := io.ReadAll(io.LimitReader(r, (1<<20)+1))
	if err != nil || len(b) > 1<<20 {
		return errors.New("resposta externa inválida")
	}
	return json.Unmarshal(b, v)
}

type Disponibilidade int

const (
	Desconhecida Disponibilidade = iota
	Disponivel
	Desabilitado
)

type EstadoPublico struct {
	Saude, Login Disponibilidade
	Carregando   bool
}

func (e EstadoPublico) descricao() string {
	saude := "Falha ao consultar a saúde da API."
	if e.Saude == Disponivel {
		saude = "A API está respondendo."
	}
	switch e.Login {
	case Disponivel:
		return saude + " Login GitHub disponível no navegador."
	case Desabilitado:
		return saude + " Login GitHub desabilitado no dashboard."
	default:
		return saude + " Não foi possível confirmar o login do dashboard."
	}
}

type apiPublica struct {
	base string
	http *http.Client
}

func (a apiPublica) obter(ctx context.Context, caminho string, v any) bool {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, a.base+caminho, nil)
	if err != nil {
		return false
	}
	resp, err := a.http.Do(r)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200 && lerJSON(resp.Body, v) == nil
}
func (a apiPublica) consultar(ctx context.Context) EstadoPublico {
	var e EstadoPublico
	var saude struct{ Status string }
	if a.obter(ctx, "/health", &saude) && saude.Status == "ok" {
		e.Saude = Disponivel
	}
	var sessao struct {
		Autenticado *bool `json:"autenticado"`
		Login       *bool `json:"login_disponivel"`
	}
	if a.obter(ctx, "/api/sessao", &sessao) && sessao.Autenticado != nil && !*sessao.Autenticado && sessao.Login != nil {
		e.Login = Desabilitado
		if *sessao.Login {
			e.Login = Disponivel
		}
	}
	return e
}

type Cliente struct {
	http        *http.Client
	base, token string
	mu          sync.Mutex
	pausa       map[string]time.Time
}

func novoCliente(token string) *Cliente {
	return &Cliente{http: clienteHTTP(5 * time.Second), base: "https://slack.com/api/", token: token, pausa: make(map[string]time.Time)}
}

type respostaSlack struct {
	OK     bool
	Error  string
	TeamID string `json:"team_id"`
	BotID  string `json:"bot_id"`
	View   struct{ ID, Hash string }
}

var errDesatualizada = errors.New("view desatualizada")
var errLimitado = errors.New("Slack limitou as chamadas deste método")

func (s *Cliente) chamar(ctx context.Context, metodo string, payload any) (respostaSlack, error) {
	var v respostaSlack
	s.mu.Lock()
	aguardar := time.Now().Before(s.pausa[metodo])
	s.mu.Unlock()
	if aguardar {
		return v, errLimitado
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return v, err
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, s.base+metodo, bytes.NewReader(b))
	if err != nil {
		return v, errors.New("destino Slack inválido")
	}
	r.Header.Set("Authorization", "Bearer "+s.token)
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := s.http.Do(r)
	if err != nil {
		return v, errors.New("resultado Slack incerto")
	}
	defer resp.Body.Close()
	if resp.StatusCode == 429 {
		n, err := strconv.ParseInt(resp.Header.Get("Retry-After"), 10, 32)
		if err != nil || n < 1 {
			n = 60
		}
		s.mu.Lock()
		ate := time.Now().Add(time.Duration(n) * time.Second)
		if ate.After(s.pausa[metodo]) {
			s.pausa[metodo] = ate
		}
		s.mu.Unlock()
		return v, errLimitado
	}
	if resp.StatusCode != 200 {
		return v, fmt.Errorf("Slack respondeu HTTP %d", resp.StatusCode)
	}
	if lerJSON(resp.Body, &v) != nil {
		return v, errors.New("resposta inválida do Slack; resultado incerto")
	}
	if !v.OK {
		if v.Error == "hash_conflict" || v.Error == "invalid_hash" {
			return v, errDesatualizada
		}
		return v, errors.New("Slack recusou a operação")
	}
	return v, nil
}
func (s *Cliente) verificar(ctx context.Context, c Config) error {
	v, err := s.chamar(ctx, "auth.test", struct{}{})
	if err != nil {
		return err
	}
	if v.TeamID != c.TeamID || v.BotID == "" {
		return errors.New("workspace do bot não confirmado")
	}
	return nil
}
func (s *Cliente) publicar(ctx context.Context, p Pessoa, hash string, v View) (string, error) {
	resp, err := s.chamar(ctx, "views.publish", struct {
		UserID string `json:"user_id"`
		Hash   string `json:"hash,omitempty"`
		View   View   `json:"view"`
	}{p.Usuario, hash, v})
	if err == nil && strings.TrimSpace(resp.View.Hash) == "" {
		err = errors.New("publicação sem hash confirmado")
	}
	return resp.View.Hash, err
}

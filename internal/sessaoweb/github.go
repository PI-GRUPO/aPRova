// Package sessaoweb autentica usuários pelo GitHub App e mantém credenciais no servidor.
package sessaoweb

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var ErrSemSessao = errors.New("sessão ausente ou expirada")
var ErrGitHubIndisponivel = errors.New("não foi possível consultar o GitHub")

type Usuario struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}
type Instalacao struct {
	ID    int64  `json:"id"`
	Conta string `json:"conta"`
}
type Token struct {
	Valor   string
	Duracao time.Duration
}
type Provedor interface {
	URLAutorizacao(state, verifier string) string
	Trocar(context.Context, string, string) (Token, error)
	Usuario(context.Context, string) (Usuario, error)
	Instalacoes(context.Context, string) ([]Instalacao, error)
	Repositorios(context.Context, string, int64) ([]string, error)
}

type GitHub struct {
	ClientID, ClientSecret, Callback string
	Cliente                          *http.Client
}

func (g *GitHub) cliente() *http.Client {
	if g.Cliente != nil {
		return g.Cliente
	}
	return &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
}
func (g *GitHub) URLAutorizacao(state, verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	q := url.Values{"client_id": {g.ClientID}, "redirect_uri": {g.Callback}, "state": {state}, "code_challenge": {base64.RawURLEncoding.EncodeToString(hash[:])}, "code_challenge_method": {"S256"}}
	return "https://github.com/login/oauth/authorize?" + q.Encode()
}
func (g *GitHub) Trocar(ctx context.Context, code, verifier string) (Token, error) {
	valores := url.Values{"client_id": {g.ClientID}, "client_secret": {g.ClientSecret}, "redirect_uri": {g.Callback}, "code": {code}, "code_verifier": {verifier}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", strings.NewReader(valores.Encode()))
	if err != nil {
		return Token{}, ErrGitHubIndisponivel
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := g.cliente().Do(req)
	if err != nil {
		return Token{}, ErrGitHubIndisponivel
	}
	defer func() { _ = res.Body.Close() }()
	var v struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
		Error       string `json:"error"`
	}
	if res.StatusCode != 200 || json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&v) != nil || v.AccessToken == "" || !strings.EqualFold(v.TokenType, "bearer") || v.Error != "" {
		return Token{}, ErrGitHubIndisponivel
	}
	duracao := 8 * time.Hour
	if v.ExpiresIn > 0 && v.ExpiresIn < int64(duracao.Seconds()) {
		duracao = time.Duration(v.ExpiresIn) * time.Second
	}
	return Token{v.AccessToken, duracao}, nil
}
func (g *GitHub) get(ctx context.Context, token, caminho string, destino any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com"+caminho, nil)
	if err != nil {
		return ErrGitHubIndisponivel
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	res, err := g.cliente().Do(req)
	if err != nil {
		return ErrGitHubIndisponivel
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode == 401 {
		return ErrSemSessao
	}
	if res.StatusCode != 200 {
		return ErrGitHubIndisponivel
	}
	if json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(destino) != nil {
		return ErrGitHubIndisponivel
	}
	return nil
}
func (g *GitHub) Usuario(ctx context.Context, token string) (Usuario, error) {
	var u Usuario
	err := g.get(ctx, token, "/user", &u)
	if err == nil && (u.ID <= 0 || u.Login == "") {
		err = ErrGitHubIndisponivel
	}
	return u, err
}
func (g *GitHub) Instalacoes(ctx context.Context, token string) ([]Instalacao, error) {
	lista := []Instalacao{}
	for pagina := 1; pagina <= 100; pagina++ {
		var v struct {
			Total       int `json:"total_count"`
			Instalacoes []struct {
				ID       int64   `json:"id"`
				Suspensa *string `json:"suspended_at"`
				Conta    struct {
					Login string `json:"login"`
				} `json:"account"`
			} `json:"installations"`
		}
		if err := g.get(ctx, token, "/user/installations?per_page=100&page="+strconv.Itoa(pagina), &v); err != nil {
			return nil, err
		}
		for _, i := range v.Instalacoes {
			if i.ID <= 0 || i.Conta.Login == "" {
				return nil, ErrGitHubIndisponivel
			}
			if i.Suspensa == nil {
				lista = append(lista, Instalacao{i.ID, i.Conta.Login})
			}
		}
		if pagina*100 >= v.Total {
			return lista, nil
		}
		if len(v.Instalacoes) == 0 {
			return nil, ErrGitHubIndisponivel
		}
	}
	return nil, fmt.Errorf("%w: limite de paginação", ErrGitHubIndisponivel)
}
func (g *GitHub) Repositorios(ctx context.Context, token string, id int64) ([]string, error) {
	lista := []string{}
	for pagina := 1; pagina <= 100; pagina++ {
		var v struct {
			Total int `json:"total_count"`
			Repos []struct {
				Nome string `json:"full_name"`
			} `json:"repositories"`
		}
		if err := g.get(ctx, token, fmt.Sprintf("/user/installations/%d/repositories?per_page=100&page=%d", id, pagina), &v); err != nil {
			return nil, err
		}
		for _, r := range v.Repos {
			if strings.Count(r.Nome, "/") != 1 {
				return nil, ErrGitHubIndisponivel
			}
			lista = append(lista, r.Nome)
		}
		if pagina*100 >= v.Total {
			return lista, nil
		}
		if len(v.Repos) == 0 {
			return nil, ErrGitHubIndisponivel
		}
	}
	return nil, ErrGitHubIndisponivel
}

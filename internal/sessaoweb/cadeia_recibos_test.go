package sessaoweb

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
)

type cadeiaTeste struct {
	instalacao int64
	repos      []string
	erro       error
}

func (c *cadeiaTeste) VerificarCadeia(_ context.Context, instalacao int64, repos []string) (receipt.ResultadoDaCadeia, error) {
	c.instalacao, c.repos = instalacao, repos
	return receipt.ResultadoDaCadeia{Valida: true, Total: 2, Verificados: 2}, c.erro
}

func TestCadeiaDoPainelRespeitaSessaoEInstalacaoInteira(t *testing.T) {
	s, _, provedor, _ := cenario(t)
	cadeia := &cadeiaTeste{}
	s.cadeia = cadeia
	provedor.repos = []string{"equipe/autorizado", "equipe/segundo"}
	cookie := login(t, s)
	rota := "/api/web/recibos/cadeia?installation_id="
	if w := pedir(s, "GET", rota+"42", nil, nil); w.Code != http.StatusUnauthorized {
		t.Fatal(w.Code, w.Body)
	}
	for _, caso := range []struct {
		parametros string
		status     int
	}{
		{"999", http.StatusForbidden},
		{"42&repositorio=equipe/autorizado", http.StatusBadRequest},
		{"42&periodo=7", http.StatusBadRequest},
		{"42&installation_id=43", http.StatusBadRequest},
	} {
		if w := pedir(s, "GET", rota+caso.parametros, cookie, nil); w.Code != caso.status {
			t.Fatal(caso.parametros, w.Code, w.Body)
		}
	}
	if cadeia.instalacao != 0 {
		t.Fatal("a rota consultou a instalação antes de validar o acesso")
	}
	w := pedir(s, "GET", rota+"42", cookie, nil)
	var v struct {
		Instalacao int64 `json:"installation_id"`
		receipt.ResultadoDaCadeia
	}
	if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &v) != nil || v.Instalacao != 42 || !v.Valida || v.Verificados != 2 ||
		cadeia.instalacao != 42 || !slices.Equal(cadeia.repos, provedor.repos) || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body, cadeia)
	}
	cadeia.erro = domain.ErrCadeiaSemAcesso
	if w := pedir(s, "GET", rota+"42", cookie, nil); w.Code != http.StatusForbidden || strings.Contains(w.Body.String(), "verificados") {
		t.Fatal(w.Code, w.Body)
	}
}

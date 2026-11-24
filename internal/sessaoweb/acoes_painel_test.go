package sessaoweb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/acaoremota"
	"github.com/PI-GRUPO/aPRova/internal/domain"
)

type analiseAcaoTeste struct{ analisesTeste }

func (a *analiseAcaoTeste) DetalharAnaliseWeb(_ context.Context, tenant int64, repos []string, id string) (domain.DetalheAnaliseWeb, error) {
	const repo = "equipe/autorizado"
	if tenant != 42 || !slices.Contains(repos, repo) {
		return domain.DetalheAnaliseWeb{}, domain.ErrNaoEncontrado
	}
	return domain.DetalheAnaliseWeb{Analise: domain.AnaliseWeb{ID: id, Repositorio: repo, NumeroPR: 10, SHAHead: strings.Repeat("a", 40)}}, nil
}

type acaoPainelTeste struct {
	pedido acaoremota.Pedido
	login  string
}

func (a *acaoPainelTeste) ProcessarPainel(_ context.Context, p acaoremota.Pedido, login string) (acaoremota.Decisao, acaoremota.Motivo) {
	a.pedido, a.login = p, login
	return acaoremota.DecisaoAceita, acaoremota.MotivoAutorizada
}

func TestAcaoDoPainelExigeSessaoCSRFEEscopo(t *testing.T) {
	s, _, _, _ := cenario(t)
	proc := &acaoPainelTeste{}
	s.analises = &analiseAcaoTeste{}
	s.acoes = proc
	cookie := login(t, s)
	var v respostaSessao
	if err := json.Unmarshal(pedir(s, "GET", "/api/sessao", cookie, nil).Body.Bytes(), &v); err != nil || v.CSRF == "" || !v.AcoesDisponiveis {
		t.Fatal("sessão sem token CSRF", err)
	}
	const id = "00000000-0000-4000-8000-000000000001"
	const idAcao = "00000000-0000-4000-8000-000000000002"
	sha := strings.Repeat("a", 40)
	corpo := `{"id_acao":"` + idAcao + `","tipo":"aprovar","justificativa":"Revisei o diff.","commit_sha":"` + sha + `"}`
	executar := func(tenant, origem, csrf, corpo string, c *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "https://aprova.test/api/web/analises/"+id+"/acoes?installation_id="+tenant, strings.NewReader(corpo))
		r.Header.Set("Origin", origem)
		r.Header.Set("X-CSRF-Token", csrf)
		r.Header.Set("Content-Type", "application/json")
		if c != nil {
			r.AddCookie(c)
		}
		w := httptest.NewRecorder()
		s.ExecutarAcao(w, r, id)
		return w
	}
	for _, w := range []struct {
		gravador *httptest.ResponseRecorder
		esperado int
	}{
		{executar("42", s.origem, v.CSRF, corpo, nil), 401},
		{executar("42", "https://outra-origem.test", v.CSRF, corpo, cookie), 403},
		{executar("42", s.origem, "", corpo, cookie), 403},
		{executar("99", s.origem, v.CSRF, corpo, cookie), 403},
		{executar("43", s.origem, v.CSRF, corpo, cookie), 404},
		{executar("42", s.origem, v.CSRF, strings.Replace(corpo, sha, strings.Repeat("b", 40), 1), cookie), 409},
		{executar("42", s.origem, v.CSRF, strings.TrimSuffix(corpo, "}")+`,"usuario":"outro"}`, cookie), 400},
	} {
		if w.gravador.Code != w.esperado || proc.login != "" {
			t.Fatal(w.esperado, w.gravador.Code, w.gravador.Body)
		}
	}
	w := executar("42", s.origem, v.CSRF, corpo, cookie)
	if w.Code != 202 || proc.login != "pessoa" || proc.pedido.IDAcao != "painel:"+idAcao || proc.pedido.TaskRunID != id ||
		proc.pedido.Repositorio != "equipe/autorizado" || proc.pedido.NumeroPR != 10 || proc.pedido.CommitSHA != sha {
		t.Fatal(w.Code, w.Body, proc.pedido, proc.login)
	}
}

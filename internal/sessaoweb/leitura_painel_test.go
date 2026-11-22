package sessaoweb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/publicacao"
)

type fontePainelTeste struct {
	f        domain.FiltroLeituraPainel
	chamadas int
	erro     error
}

func (p *fontePainelTeste) LerPainel(_ context.Context, _ string, f domain.FiltroLeituraPainel) (json.RawMessage, error) {
	p.f = f
	p.chamadas++
	return json.RawMessage(`{"total":0,"itens":[]}`), p.erro
}
func (p *fontePainelTeste) UltimasAnalisesWeb(context.Context, int64, []string) (map[string]time.Time, error) {
	return nil, p.erro
}

func TestLeiturasPainelValidamSessaoEscopoEFiltros(t *testing.T) {
	s, _, provedor, _ := cenario(t)
	fonte := &fontePainelTeste{}
	s.leituras = fonte
	cookie := login(t, s)
	rota := "/api/web/recibos?installation_id=42"
	if w := pedir(s, "GET", rota, nil, nil); w.Code != 401 {
		t.Fatal(w.Code)
	}
	for _, tc := range []struct {
		url    string
		status int
	}{
		{rota + "&repositorio=outra/privada", 403}, {"/api/metricas?installation_id=999", 403}, {rota + "&limite=501", 400}, {rota + "&pagina=2", 400}, {rota + "&snapshot=2099-01-01T00:00:00Z", 400}, {rota + "&periodo=91", 400}, {"/api/web/auditoria?installation_id=42&decisao=invalida", 400}, {"/api/metricas?installation_id=42&limite=1", 400},
	} {
		if w := pedir(s, "GET", tc.url, cookie, nil); w.Code != tc.status {
			t.Error(tc, w.Code)
		}
	}
	if fonte.chamadas != 0 {
		t.Fatal("consulta fora de escopo ou inválida")
	}
	w := pedir(s, "GET", rota+"&periodo=7&pagina=2&limite=10&snapshot=2026-11-14T09:00:00Z", cookie, nil)
	if w.Code != 200 || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") || !slices.Equal(fonte.f.Repositorios, []string{"equipe/autorizado"}) || fonte.f.Inicio.Format(time.DateOnly) != "2026-11-08" || fonte.f.Limite != 10 || fonte.f.Pagina != 2 {
		t.Fatal(w.Code, w.Body, fonte.f)
	}
	fonte.erro = errors.New("detalhes internos")
	if w := pedir(s, "GET", rota, cookie, nil); w.Code != 503 || strings.Contains(w.Body.String(), "internos") {
		t.Fatal(w.Code, w.Body)
	}
	provedor.revogado = true
	if w := pedir(s, "GET", rota, cookie, nil); w.Code != 401 || fonte.chamadas != 2 {
		t.Fatal(w.Code, fonte.chamadas)
	}
}
func TestVinculoPessoalMostraSomentePresencaDoLogin(t *testing.T) {
	s, _, _, _ := cenario(t)
	cookie := login(t, s)
	rota := "/api/web/vinculo-slack"
	if w := pedir(s, "GET", rota, nil, nil); w.Code != 401 {
		t.Fatal(w.Code)
	}
	for _, login := range []string{"outra", "PESSOA"} {
		s.vinculos = map[string]string{"ID-SLACK-PRIVADO": login}
		w := pedir(s, "GET", rota, cookie, nil)
		var v map[string]any
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &v) != nil || len(v) != 2 || v["usuario_id"] != float64(7) || v["vinculado"] != (login == "PESSOA") {
			t.Fatal(w.Code, w.Body)
		}
	}
	if w := pedir(s, "GET", rota+"?usuario_id=8", cookie, nil); w.Code != 400 {
		t.Fatal(w.Code)
	}
}
func TestProtecaoConfirmaSomenteCheckDoApp(t *testing.T) {
	for _, tc := range []struct {
		nome             string
		classica, regras int
		app              int
		confirma, erro   bool
	}{
		{"classica", 200, 404, 7, true, false}, {"regras", 404, 200, 7, true, false}, {"outro app", 200, 200, 8, false, false}, {"oculta", 403, 404, 7, false, false}, {"falha", 404, 503, 7, false, true},
	} {
		t.Run(tc.nome, func(t *testing.T) {
			check := map[string]any{"context": publicacao.NomeDoCheckRun, "app_id": tc.app, "integration_id": tc.app}
			g := GitHub{Cliente: &http.Client{Transport: transporteFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodGet {
					t.Fatal(r.Method)
				}
				if strings.Contains(r.URL.Path, "/protection/") {
					return respostaTeste(map[string]any{"checks": []any{check}}, tc.classica), nil
				}
				return respostaTeste([]any{map[string]any{"type": "required_status_checks", "parameters": map[string]any{"required_status_checks": []any{check}}}}, tc.regras), nil
			})}}
			v, e := g.checkObrigatorio(t.Context(), "token", repositorioGitHub{"equipe/projeto", "main"}, 7)
			if tc.nome == "outro app" && (v == nil || *v) {
				t.Fatalf("as duas fontes completas não exigem o App: %v", v)
			}
			if (e != nil) != tc.erro || (v != nil && *v) != tc.confirma {
				t.Fatal(v, e)
			}
		})
	}
}

func TestCatalogoDetalhadoUsaRepositoriosDaPessoaEPermissoesDaInstalacao(t *testing.T) {
	s, _, _, _ := cenario(t)
	cookie := login(t, s)
	s.leituras = &fontePainelTeste{}
	const repo = "equipe/autorizado"
	chamadasProtecao := 0
	s.provedor = &GitHub{Cliente: &http.Client{Transport: transporteFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/user/installations":
			return respostaTeste(map[string]any{"total_count": 1, "installations": []any{map[string]any{"id": 42, "account": map[string]string{"login": "equipe"}, "app_id": 9, "permissions": map[string]string{"checks": "write", "members": "read"}}}}, 200), nil
		case "/user/installations/42/repositories":
			return respostaTeste(map[string]any{"total_count": 1, "repositories": []any{map[string]string{"full_name": repo, "default_branch": "main"}}}, 200), nil
		case "/repos/equipe/autorizado/branches/main/protection/required_status_checks":
			chamadasProtecao++
			return respostaTeste(map[string]any{"checks": []any{map[string]any{"context": publicacao.NomeDoCheckRun, "app_id": 9}}}, 200), nil
		default:
			t.Errorf("caminho inesperado: %s", r.URL.Path)
			return respostaTeste(nil, 404), nil
		}
	})}}
	w := pedir(s, "GET", "/api/repositorios?installation_id=42&detalhes=1", cookie, nil)
	var v struct {
		Total      int
		Itens      []saudeRepo
		Permissoes map[string]string
	}
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || v.Total != 1 || len(v.Itens) != 1 || v.Itens[0].Nome != repo || v.Itens[0].Check == nil || !*v.Itens[0].Check || v.Itens[0].Ultima != nil || v.Permissoes["checks"] != "write" || len(v.Permissoes) != 1 {
		t.Fatal(w.Code, w.Body)
	}
	if w := pedir(s, "GET", "/api/repositorios?installation_id=99&detalhes=1", cookie, nil); w.Code != 403 || chamadasProtecao != 1 {
		t.Fatal(w.Code, chamadasProtecao)
	}
}

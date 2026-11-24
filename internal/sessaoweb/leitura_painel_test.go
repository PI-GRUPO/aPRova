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
	ultimas  map[string]time.Time
}

func (p *fontePainelTeste) LerPainel(_ context.Context, _ string, f domain.FiltroLeituraPainel) (json.RawMessage, error) {
	p.f = f
	p.chamadas++
	return json.RawMessage(`{"total":0,"itens":[]}`), p.erro
}
func (p *fontePainelTeste) UltimasAnalisesWeb(context.Context, int64, []string) (map[string]time.Time, error) {
	return p.ultimas, p.erro
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
	const nome = publicacao.NomeDoCheckRun
	classica := func(app int) any {
		return map[string]any{"required_status_checks": map[string]any{"checks": []any{map[string]any{"context": nome, "app_id": app}}}}
	}
	regra := func(app int) any {
		return []any{map[string]any{"type": "required_status_checks", "parameters": map[string]any{
			"required_status_checks": []any{map[string]any{"context": nome, "integration_id": app}},
		}}}
	}
	semRegras := []any{}
	paginaCheia := make([]any, 100)
	for i := range paginaCheia {
		paginaCheia[i] = map[string]any{"type": "deletion"}
	}
	type respostaEsperada struct {
		estado int
		corpo  any
	}
	resposta := func(estado int, corpo any) respostaEsperada { return respostaEsperada{estado, corpo} }
	for _, tc := range []struct {
		nome                                   string
		classica, regras, branch, complementos respostaEsperada
		pagina2                                respostaEsperada
		check                                  *bool
		motivo                                 string
	}{
		{nome: "classica", classica: resposta(200, classica(7)), check: new(true)},
		{nome: "classica com contexto legado", classica: resposta(200, map[string]any{"required_status_checks": map[string]any{"contexts": []string{nome}}}),
			complementos: resposta(200, map[string]any{"checks": []any{map[string]any{"context": nome, "app_id": 7}}}), check: new(true)},
		{nome: "regras", classica: resposta(404, nil), regras: resposta(200, regra(7)), check: new(true)},
		{nome: "outro app", classica: resposta(200, classica(8)), regras: resposta(200, regra(8)), check: new(false)},
		{nome: "nenhum check na proteção clássica", classica: resposta(200, map[string]any{"url": "https://api.github.com/protection"}),
			regras: resposta(200, semRegras), check: new(false)},
		{nome: "branch explicitamente sem proteção", classica: resposta(403, nil), regras: resposta(200, semRegras),
			branch: resposta(200, map[string]bool{"protected": false}), check: new(false)},
		{nome: "branch protegida com detalhes ocultos", classica: resposta(404, nil), regras: resposta(200, semRegras),
			branch: resposta(200, map[string]bool{"protected": true}), motivo: "acesso_insuficiente"},
		{nome: "sem origem", classica: resposta(200, classica(-1)), regras: resposta(200, semRegras), motivo: "check_sem_origem"},
		{nome: "consulta indisponível", classica: resposta(404, nil), regras: resposta(503, nil), motivo: "consulta_indisponivel"},
		{nome: "regras incompletas", classica: resposta(200, classica(8)), regras: resposta(200, []any{map[string]any{"type": "required_status_checks"}}),
			motivo: "dados_incompletos"},
		{nome: "regra na próxima página", classica: resposta(200, classica(8)), regras: resposta(200, paginaCheia), pagina2: resposta(200, regra(7)),
			check: new(true)},
	} {
		t.Run(tc.nome, func(t *testing.T) {
			g := GitHub{Cliente: &http.Client{Transport: transporteFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != http.MethodGet {
					t.Fatal(r.Method)
				}
				var res respostaEsperada
				switch r.URL.Path {
				case "/repos/equipe/projeto/branches/main/protection":
					res = tc.classica
				case "/repos/equipe/projeto/branches/main/protection/required_status_checks":
					res = tc.complementos
				case "/repos/equipe/projeto/branches/main":
					res = tc.branch
				case "/repos/equipe/projeto/rules/branches/main":
					switch r.URL.Query().Get("page") {
					case "1":
						res = tc.regras
					case "2":
						res = tc.pagina2
					}
				}
				if res.estado == 0 {
					t.Fatalf("consulta inesperada: %s", r.URL)
				}
				return respostaTeste(res.corpo, res.estado), nil
			})}}
			v, e := g.diagnosticarProtecao(t.Context(), "token", repositorioGitHub{Nome: "equipe/projeto", Branch: "main"}, 7)
			if e != nil || v.Motivo != tc.motivo || (v.Check == nil) != (tc.check == nil) || (tc.check != nil && *v.Check != *tc.check) {
				t.Fatal(v, e)
			}
		})
	}
}

func TestCatalogoDetalhadoUsaRepositoriosDaPessoaEPermissoesDaInstalacao(t *testing.T) {
	s, _, _, _ := cenario(t)
	cookie := login(t, s)
	const repo = "equipe/autorizado"
	recebida := time.Date(2026, 11, 21, 11, 45, 0, 0, time.UTC)
	s.leituras = &fontePainelTeste{ultimas: map[string]time.Time{repo: recebida}}
	chamadasProtecao := 0
	s.provedor = &GitHub{Cliente: &http.Client{Transport: transporteFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/user/installations":
			return respostaTeste(map[string]any{"total_count": 1, "installations": []any{map[string]any{"id": 42, "account": map[string]string{"login": "equipe"}, "app_id": 9, "permissions": map[string]string{"checks": "write", "administration": "read", "metadata": "read", "members": "read"}}}}, 200), nil
		case "/user/installations/42/repositories":
			return respostaTeste(map[string]any{"total_count": 3, "repositories": []any{
				map[string]any{"full_name": repo, "default_branch": "main"},
				map[string]any{"full_name": "equipe/arquivado", "default_branch": "main", "archived": true},
				map[string]any{"full_name": "equipe/desabilitado", "default_branch": "main", "disabled": true},
			}}, 200), nil
		case "/repos/equipe/autorizado/branches/main/protection":
			chamadasProtecao++
			return respostaTeste(map[string]any{"required_status_checks": map[string]any{
				"checks": []any{map[string]any{"context": publicacao.NomeDoCheckRun, "app_id": 9}},
			}}, 200), nil
		default:
			t.Errorf("caminho inesperado: %s", r.URL.Path)
			return respostaTeste(nil, 404), nil
		}
	})}}
	w := pedir(s, "GET", "/api/repositorios?installation_id=42&detalhes=1", cookie, nil)
	var v struct {
		Total                int
		Itens                []saudeRepo
		Permissoes           map[string]string
		PermissoesConhecidas bool `json:"permissoes_conhecidas"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || v.Total != 1 || len(v.Itens) != 1 || v.Itens[0].Nome != repo || v.Itens[0].Check == nil || !*v.Itens[0].Check ||
		v.Itens[0].Ultima == nil || !v.Itens[0].Ultima.Equal(recebida) || !v.PermissoesConhecidas || v.Permissoes["checks"] != "write" ||
		v.Permissoes["administration"] != "read" || v.Permissoes["metadata"] != "read" || len(v.Permissoes) != 3 {
		t.Fatal(w.Code, w.Body)
	}
	if w := pedir(s, "GET", "/api/repositorios?installation_id=99&detalhes=1", cookie, nil); w.Code != 403 || chamadasProtecao != 1 {
		t.Fatal(w.Code, chamadasProtecao)
	}
	if w := pedir(s, "GET", "/api/repositorios?installation_id=42", cookie, nil); w.Code != 200 ||
		!strings.Contains(w.Body.String(), "equipe/arquivado") || chamadasProtecao != 1 {
		t.Fatal("o catálogo simples deve manter o escopo histórico", w.Code, w.Body)
	}
}

package sessaoweb

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/nomesgithub"
	"github.com/PI-GRUPO/aPRova/internal/publicacao"
	"golang.org/x/sync/errgroup"
)

type permissoesGitHub struct {
	Checks   string `json:"checks,omitempty"`
	Conteudo string `json:"contents,omitempty"`
	PR       string `json:"pull_requests,omitempty"`
}
type instalacaoGitHub struct {
	Instalacao
	AppID      int64
	Permissoes permissoesGitHub
}
type repositorioGitHub struct {
	Nome   string `json:"full_name"`
	Branch string `json:"default_branch"`
}
type saudeRepo struct {
	Nome   string     `json:"nome"`
	Branch string     `json:"branch"`
	Ultima *time.Time `json:"ultima_analise"`
	Check  *bool      `json:"check_obrigatorio"`
}
type erroAcessoMetadata struct{}

func (erroAcessoMetadata) Error() string { return "metadados inacessíveis" }
func (erroAcessoMetadata) Unwrap() error { return ErrGitHubIndisponivel }

// Confirma o check somente com nome e App da instalação na branch padrão.
// Uma resposta oculta ou uma página de regras incompleta impede concluir ausência.
func (g *GitHub) checkObrigatorio(ctx context.Context, token string, r repositorioGitHub, app int64) (*bool, error) {
	if app <= 0 || r.Branch == "" || !nomesgithub.RepositorioValido(r.Nome) {
		return nil, nil
	}
	base := "/repos/" + r.Nome
	branch := url.PathEscape(r.Branch)
	var classica struct {
		Checks    []checkExigido `json:"checks"`
		Contextos []string       `json:"contexts"`
	}
	e := g.get(ctx, token, base+"/branches/"+branch+"/protection/required_status_checks", &classica)
	if e != nil && !errors.Is(e, erroAcessoMetadata{}) {
		return nil, e
	}
	if confirmado(classica.Checks, app) {
		v := true
		return &v, nil
	}
	ausente := e == nil && classica.Checks != nil && !slices.Contains(classica.Contextos, publicacao.NomeDoCheckRun) && !contextoSemApp(classica.Checks)
	return g.checkDasRegras(ctx, token, base+"/rules/branches/"+branch+"?per_page=100", app, ausente)
}

type checkExigido struct {
	Contexto   string `json:"context"`
	App        int64  `json:"app_id"`
	Integracao int64  `json:"integration_id"`
}

func confirmado(checks []checkExigido, app int64) bool {
	return slices.ContainsFunc(checks, func(c checkExigido) bool {
		return c.Contexto == publicacao.NomeDoCheckRun && (c.App == app || c.Integracao == app)
	})
}
func contextoSemApp(checks []checkExigido) bool {
	return slices.ContainsFunc(checks, func(c checkExigido) bool {
		return c.Contexto == publicacao.NomeDoCheckRun && c.App <= 0 && c.Integracao <= 0
	})
}
func (g *GitHub) checkDasRegras(ctx context.Context, token, caminho string, app int64, ausente bool) (*bool, error) {
	var regras []struct {
		Tipo       string `json:"type"`
		Parametros struct {
			Checks []checkExigido `json:"required_status_checks"`
		} `json:"parameters"`
	}
	err := g.get(ctx, token, caminho, &regras)
	if errors.Is(err, erroAcessoMetadata{}) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	for _, regra := range regras {
		if regra.Tipo != "required_status_checks" {
			continue
		}
		checks := regra.Parametros.Checks
		if confirmado(checks, app) {
			v := true
			return &v, nil
		}
		if checks == nil || contextoSemApp(checks) {
			ausente = false
		}
	}
	if ausente && regras != nil && len(regras) < 100 {
		v := false
		return &v, nil
	}
	return nil, nil
}

func (s *Servidor) repositoriosComMetadados(ctx context.Context, token string, id int64) (instalacaoGitHub, []repositorioGitHub, error) {
	var instalacao instalacaoGitHub
	g, github := s.provedor.(*GitHub)
	if github {
		instalacoes, err := g.instalacoesGitHub(ctx, token)
		if err != nil {
			return instalacao, nil, err
		}
		i := slices.IndexFunc(instalacoes, func(i instalacaoGitHub) bool { return i.ID == id })
		if i < 0 {
			return instalacao, nil, errSemAcesso
		}
		repos, err := g.repositoriosGitHub(ctx, token, id)
		return instalacoes[i], repos, err
	}
	nomes, err := s.repositoriosAutorizados(ctx, token, id, "")
	if err != nil {
		return instalacao, nil, err
	}
	repos := make([]repositorioGitHub, len(nomes))
	for i, nome := range nomes {
		repos[i].Nome = nome
	}
	return instalacao, repos, nil
}
func (s *Servidor) catalogoSaude(w http.ResponseWriter, r *http.Request, id int64, v sessao) {
	q := r.URL.Query()
	pagina, e := inteiroWeb(q, "pagina", 1, 1, 10000)
	if e != nil || parametrosWeb(q, parametroInstalacaoWeb, "detalhes", "pagina") != nil {
		falhar(w, 400, "página inválida")
		return
	}
	instalacao, repos, e := s.repositoriosComMetadados(r.Context(), v.Token, id)
	if errors.Is(e, errSemAcesso) || (e == nil && len(repos) == 0) {
		falhar(w, 403, "instalação ou repositório não autorizado")
		return
	}
	if e != nil {
		s.erroProvedor(w, r, e)
		return
	}
	slices.SortFunc(repos, func(a, b repositorioGitHub) int { return strings.Compare(a.Nome, b.Nome) })
	const limite = 10
	inicio := min(int(pagina-1)*limite, len(repos))
	recorte := repos[inicio:min(inicio+limite, len(repos))]
	itens := make([]saudeRepo, len(recorte))
	nomes := make([]string, len(recorte))
	for i, r := range recorte {
		nomes[i] = r.Nome
		itens[i] = saudeRepo{Nome: r.Nome, Branch: r.Branch}
	}
	if s.leituras == nil {
		falhar(w, 503, "leitura do painel indisponível")
		return
	}
	ultimas, err := s.leituras.UltimasAnalisesWeb(r.Context(), id, nomes)
	if err != nil {
		falhar(w, 503, "consulta de análises indisponível")
		return
	}
	if err := s.completarSaude(r.Context(), v.Token, instalacao.AppID, recorte, ultimas, itens); err != nil {
		s.erroProvedor(w, r, err)
		return
	}
	jsonResposta(w, 200, map[string]any{parametroInstalacaoWeb: id, "pagina": pagina, "limite": limite, "total": len(repos), "permissoes": instalacao.Permissoes, "itens": itens})
}

func (s *Servidor) completarSaude(ctx context.Context, token string, app int64, recorte []repositorioGitHub, ultimas map[string]time.Time, itens []saudeRepo) error {
	g, github := s.provedor.(*GitHub)
	grupo, ctx := errgroup.WithContext(ctx)
	grupo.SetLimit(3)
	for i, repo := range recorte {
		if data, ok := ultimas[repo.Nome]; ok {
			itens[i].Ultima = &data
		}
		if github {
			grupo.Go(func() error {
				check, err := g.checkObrigatorio(ctx, token, repo, app)
				itens[i].Check = check
				return err
			})
		}
	}
	return grupo.Wait()
}

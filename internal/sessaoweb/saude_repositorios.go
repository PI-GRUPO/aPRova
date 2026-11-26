package sessaoweb

import (
	"context"
	"errors"
	"fmt"
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
	Checks        string `json:"checks,omitempty"`
	Conteudo      string `json:"contents,omitempty"`
	PR            string `json:"pull_requests,omitempty"`
	Administracao string `json:"administration,omitempty"`
	Metadados     string `json:"metadata,omitempty"`
}
type instalacaoGitHub struct {
	Instalacao
	AppID      int64
	Permissoes *permissoesGitHub
}
type repositorioGitHub struct {
	Nome         string `json:"full_name"`
	Branch       string `json:"default_branch"`
	Arquivado    bool   `json:"archived"`
	Desabilitado bool   `json:"disabled"`
}
type saudeRepo struct {
	Nome   string     `json:"nome"`
	Branch string     `json:"branch"`
	Ultima *time.Time `json:"ultima_analise"`
	protecaoRepo
}

type protecaoRepo struct {
	Check  *bool  `json:"check_obrigatorio"`
	Motivo string `json:"motivo_protecao,omitempty"`
}
type erroAcessoMetadata struct{}

func (erroAcessoMetadata) Error() string { return "metadados inacessíveis" }
func (erroAcessoMetadata) Unwrap() error { return ErrGitHubIndisponivel }

type leituraDaProtecao struct {
	obrigatorio, ambigua, completa bool
}

// Confirma o check somente com nome e App da instalação na branch padrão.
// A ausência exige evidência tanto na proteção clássica quanto nas regras ativas.
func (g *GitHub) diagnosticarProtecao(ctx context.Context, token string, r repositorioGitHub, app int64) (protecaoRepo, error) {
	if app <= 0 || r.Branch == "" || !nomesgithub.RepositorioValido(r.Nome) {
		return protecaoRepo{Motivo: "branch_ou_app_nao_informado"}, nil
	}
	base := "/repos/" + r.Nome
	branch := url.PathEscape(r.Branch)
	caminhoBranch := base + "/branches/" + branch
	classica, erroClassica := g.lerProtecaoClassica(ctx, token, caminhoBranch+"/protection", app)
	if classica.obrigatorio {
		return protecaoConfirmada(true), nil
	}
	if errors.Is(erroClassica, ErrSemSessao) {
		return protecaoRepo{}, erroClassica
	}
	regras, erroRegras := g.lerProtecaoDasRegras(ctx, token, base+"/rules/branches/"+branch, app)
	if regras.obrigatorio {
		return protecaoConfirmada(true), nil
	}
	if errors.Is(erroRegras, ErrSemSessao) {
		return protecaoRepo{}, erroRegras
	}
	if classica.ambigua || regras.ambigua {
		return protecaoRepo{Motivo: "check_sem_origem"}, nil
	}
	if regras.completa {
		return g.confirmarAusencia(ctx, token, caminhoBranch, classica, erroClassica)
	}
	return protecaoNaoConfirmada(erroRegras), nil
}

func protecaoConfirmada(obrigatorio bool) protecaoRepo {
	return protecaoRepo{Check: &obrigatorio}
}

func protecaoNaoConfirmada(err error) protecaoRepo {
	switch {
	case errors.Is(err, erroAcessoMetadata{}):
		return protecaoRepo{Motivo: "acesso_insuficiente"}
	case err != nil:
		return protecaoRepo{Motivo: "consulta_indisponivel"}
	default:
		return protecaoRepo{Motivo: "dados_incompletos"}
	}
}

func (g *GitHub) confirmarAusencia(ctx context.Context, token, caminho string, classica leituraDaProtecao, err error) (protecaoRepo, error) {
	if classica.completa {
		return protecaoConfirmada(false), nil
	}
	if !errors.Is(err, erroAcessoMetadata{}) {
		return protecaoNaoConfirmada(err), nil
	}
	// Um 403/404 não informa se há proteção clássica. Use a sinalização
	// explícita da branch somente depois de conferir as regras ativas.
	var branch struct {
		Protegida *bool `json:"protected"`
	}
	if erroBranch := g.get(ctx, token, caminho, &branch); erroBranch != nil {
		if errors.Is(erroBranch, ErrSemSessao) {
			return protecaoRepo{}, erroBranch
		}
		return protecaoNaoConfirmada(erroBranch), nil
	}
	if branch.Protegida == nil {
		return protecaoNaoConfirmada(nil), nil
	}
	if !*branch.Protegida {
		return protecaoConfirmada(false), nil
	}
	return protecaoNaoConfirmada(err), nil
}

type checksDaProtecao struct {
	Checks    []checkExigido `json:"checks"`
	Contextos []string       `json:"contexts"`
}

func (g *GitHub) lerProtecaoClassica(ctx context.Context, token, caminho string, app int64) (leituraDaProtecao, error) {
	var resposta *struct {
		URL    string            `json:"url"`
		Status *checksDaProtecao `json:"required_status_checks"`
	}
	if err := g.get(ctx, token, caminho, &resposta); err != nil || resposta == nil {
		return leituraDaProtecao{}, err
	}
	if resposta.Status == nil {
		return leituraDaProtecao{completa: resposta.URL != ""}, nil
	}
	status := resposta.Status
	if slices.Contains(status.Contextos, publicacao.NomeDoCheckRun) && !slices.ContainsFunc(status.Checks, temNomeDoCheck) {
		// A proteção clássica também pode trazer só o contexto legado. O
		// endpoint específico informa o App que precisa publicar esse check.
		var detalhes *checksDaProtecao
		err := g.get(ctx, token, caminho+"/required_status_checks", &detalhes)
		if detalhes != nil && err == nil {
			detalhes.Contextos = append(detalhes.Contextos, status.Contextos...)
			status = detalhes
		} else {
			return leituraDaProtecao{ambigua: true}, err
		}
	}
	return avaliarChecks(status.Checks, status.Contextos, app), nil
}

type checkExigido struct {
	Contexto   string `json:"context"`
	App        int64  `json:"app_id"`
	Integracao int64  `json:"integration_id"`
}

func avaliarChecks(checks []checkExigido, contextos []string, app int64) leituraDaProtecao {
	exato := slices.ContainsFunc(checks, func(c checkExigido) bool {
		return c.Contexto == publicacao.NomeDoCheckRun && (c.App == app || c.Integracao == app)
	})
	semOrigem := slices.ContainsFunc(checks, func(c checkExigido) bool {
		return c.Contexto == publicacao.NomeDoCheckRun && c.App <= 0 && c.Integracao <= 0
	})
	legado := slices.Contains(contextos, publicacao.NomeDoCheckRun) && !slices.ContainsFunc(checks, temNomeDoCheck)
	return leituraDaProtecao{obrigatorio: exato, ambigua: semOrigem || legado, completa: checks != nil || contextos != nil}
}

func temNomeDoCheck(check checkExigido) bool { return check.Contexto == publicacao.NomeDoCheckRun }

type regraDaBranch struct {
	Tipo       string `json:"type"`
	Parametros struct {
		Checks []checkExigido `json:"required_status_checks"`
	} `json:"parameters"`
}

func (g *GitHub) lerProtecaoDasRegras(ctx context.Context, token, caminho string, app int64) (leituraDaProtecao, error) {
	const porPagina = 100
	resultado := leituraDaProtecao{completa: true}
	for pagina := 1; pagina <= 10; pagina++ {
		var regras []regraDaBranch
		err := g.get(ctx, token, fmt.Sprintf("%s?per_page=%d&page=%d", caminho, porPagina, pagina), &regras)
		if err != nil || regras == nil || len(regras) > porPagina {
			resultado.completa = false
			return resultado, err
		}
		verificarPaginaDeRegras(regras, app, &resultado)
		if resultado.obrigatorio || len(regras) < porPagina {
			return resultado, nil
		}
	}
	resultado.completa = false
	return resultado, nil
}

func verificarPaginaDeRegras(regras []regraDaBranch, app int64, resultado *leituraDaProtecao) {
	for _, regra := range regras {
		switch regra.Tipo {
		case "required_status_checks":
			leitura := avaliarChecks(regra.Parametros.Checks, nil, app)
			resultado.obrigatorio = resultado.obrigatorio || leitura.obrigatorio
			resultado.ambigua = resultado.ambigua || leitura.ambigua
			resultado.completa = resultado.completa && leitura.completa
		case "":
			resultado.completa = false
		}
	}
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
	if errors.Is(e, errSemAcesso) {
		falhar(w, 403, "instalação ou repositório não autorizado")
		return
	}
	if e != nil {
		s.erroProvedor(w, r, e)
		return
	}
	repos = slices.DeleteFunc(repos, func(repo repositorioGitHub) bool { return repo.Arquivado || repo.Desabilitado })
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
	jsonResposta(w, 200, map[string]any{
		parametroInstalacaoWeb: id, "pagina": pagina, "limite": limite, "total": len(repos), "itens": itens,
		"permissoes": permissoesDaInstalacao(instalacao), "permissoes_conhecidas": instalacao.Permissoes != nil,
	})
}

func permissoesDaInstalacao(instalacao instalacaoGitHub) permissoesGitHub {
	if instalacao.Permissoes == nil {
		return permissoesGitHub{}
	}
	return *instalacao.Permissoes
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
				protecao, err := g.diagnosticarProtecao(ctx, token, repo, app)
				itens[i].protecaoRepo = protecao
				return err
			})
		}
	}
	return grupo.Wait()
}

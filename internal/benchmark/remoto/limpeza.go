package remoto

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

const prefixoDeBranch = "refs/heads/"

// UltimoCommit devolve o commit para onde a branch aponta e a árvore dele.
func (g *GitHub) UltimoCommit(ctx context.Context, repositorio, branch string) (CommitPublicado, error) {
	var ref struct {
		Objeto struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := g.chamar(ctx, http.MethodGet, "/repos/"+repositorio+"/git/ref/heads/"+branch, nil, &ref); err != nil {
		return CommitPublicado{}, err
	}
	var commit struct {
		Arvore struct {
			SHA string `json:"sha"`
		} `json:"tree"`
	}
	if err := g.chamar(ctx, http.MethodGet, "/repos/"+repositorio+"/git/commits/"+ref.Objeto.SHA, nil, &commit); err != nil {
		return CommitPublicado{}, err
	}
	return CommitPublicado{SHA: ref.Objeto.SHA, Arvore: commit.Arvore.SHA}, nil
}

// FecharPR fecha o pull request sem merge.
func (g *GitHub) FecharPR(ctx context.Context, repositorio string, numero int) error {
	return g.chamar(ctx, http.MethodPatch, "/repos/"+repositorio+"/pulls/"+strconv.Itoa(numero), map[string]any{"state": "closed"}, nil)
}

// ApagarBranch apaga a ref da branch.
func (g *GitHub) ApagarBranch(ctx context.Context, repositorio, branch string) error {
	return g.chamar(ctx, http.MethodDelete, "/repos/"+repositorio+"/git/refs/heads/"+branch, nil, nil)
}

// BranchesComPrefixo devolve o nome das branches que começam com o prefixo.
func (g *GitHub) BranchesComPrefixo(ctx context.Context, repositorio, prefixo string) ([]string, error) {
	var refs []struct {
		Ref string `json:"ref"`
	}
	if err := g.chamar(ctx, http.MethodGet, "/repos/"+repositorio+"/git/matching-refs/heads/"+prefixo, nil, &refs); err != nil {
		return nil, err
	}
	branches := make([]string, 0, len(refs))
	for _, ref := range refs {
		branches = append(branches, strings.TrimPrefix(ref.Ref, prefixoDeBranch))
	}
	return branches, nil
}

// Comentarios devolve o corpo dos primeiros 100 comentários da conversa do
// pull request.
func (g *GitHub) Comentarios(ctx context.Context, repositorio string, numero int) ([]string, error) {
	var comentarios []struct {
		Corpo string `json:"body"`
	}
	caminho := "/repos/" + repositorio + "/issues/" + strconv.Itoa(numero) + "/comments?per_page=100"
	if err := g.chamar(ctx, http.MethodGet, caminho, nil, &comentarios); err != nil {
		return nil, err
	}
	corpos := make([]string, 0, len(comentarios))
	for _, comentario := range comentarios {
		corpos = append(corpos, comentario.Corpo)
	}
	return corpos, nil
}

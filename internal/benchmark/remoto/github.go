// Package remoto fala com o GitHub dos repositórios do benchmark, pela API
// REST, e com a API local do aPRova, para ler os recibos.
package remoto

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

const (
	limiteDoCorpoDeErro = 500
	modoDeArquivo       = "100644"
	situacaoConcluido   = "completed"
	chaveSHA            = "sha"
)

var errRefExistente = errors.New("remoto: a ref já existe")

// GitHub é o cliente REST do GitHub. URLBase é https://api.github.com em
// produção; Token é o token do gh do usuário, nunca registrado.
type GitHub struct {
	URLBase string
	Token   string
	Cliente *http.Client
}

// Commit é o que vai para uma branch: os arquivos com conteúdo, os apagados e,
// para a solução, o commit-base de onde ela parte.
type Commit struct {
	Mensagem  string
	Pai       *CommitPublicado
	Arquivos  map[string][]byte
	Removidos []string
}

// CommitPublicado é o commit criado no GitHub e a árvore dele.
type CommitPublicado struct {
	SHA    string `json:"sha"`
	Arvore string `json:"arvore"`
}

// PR é um pull request aberto.
type PR struct {
	Numero int    `json:"number"`
	URL    string `json:"html_url"`
}

// CheckRun é o check run do aPRova num commit.
type CheckRun struct {
	Situacao   string
	Conclusao  string
	Titulo     string
	URLDetalhe string
}

// Concluido diz se o check run terminou.
func (c CheckRun) Concluido() bool {
	return c.Situacao == situacaoConcluido
}

// IDDoRecibo extrai o id do recibo do details_url (.../receipts/{id}/verify).
func (c CheckRun) IDDoRecibo() string {
	partes := strings.Split(strings.TrimSuffix(c.URLDetalhe, "/verify"), "/receipts/")
	if len(partes) != 2 {
		return ""
	}
	return partes[1]
}

// PublicarCommit cria os blobs, a árvore (sobre a árvore do pai, se houver) e
// o commit, e aponta a branch para ele, atualizando a ref se ela já existir.
func (g *GitHub) PublicarCommit(ctx context.Context, repositorio, branch string, commit Commit) (CommitPublicado, error) {
	entradas := make([]map[string]any, 0, len(commit.Arquivos)+len(commit.Removidos))
	caminhos := make([]string, 0, len(commit.Arquivos))
	for caminho := range commit.Arquivos {
		caminhos = append(caminhos, caminho)
	}
	slices.Sort(caminhos)
	for _, caminho := range caminhos {
		var blob struct {
			SHA string `json:"sha"`
		}
		corpo := map[string]any{"content": base64.StdEncoding.EncodeToString(commit.Arquivos[caminho]), "encoding": "base64"}
		if err := g.chamar(ctx, http.MethodPost, "/repos/"+repositorio+"/git/blobs", corpo, &blob); err != nil {
			return CommitPublicado{}, err
		}
		entradas = append(entradas, map[string]any{"path": caminho, "mode": modoDeArquivo, "type": "blob", chaveSHA: blob.SHA})
	}
	for _, removido := range commit.Removidos {
		entradas = append(entradas, map[string]any{"path": removido, "mode": modoDeArquivo, "type": "blob", chaveSHA: nil})
	}

	corpoDaArvore := map[string]any{"tree": entradas}
	pais := []string{}
	if commit.Pai != nil {
		corpoDaArvore["base_tree"] = commit.Pai.Arvore
		pais = append(pais, commit.Pai.SHA)
	}
	var arvore struct {
		SHA string `json:"sha"`
	}
	if err := g.chamar(ctx, http.MethodPost, "/repos/"+repositorio+"/git/trees", corpoDaArvore, &arvore); err != nil {
		return CommitPublicado{}, err
	}
	var criado struct {
		SHA string `json:"sha"`
	}
	corpoDoCommit := map[string]any{"message": commit.Mensagem, "tree": arvore.SHA, "parents": pais}
	if err := g.chamar(ctx, http.MethodPost, "/repos/"+repositorio+"/git/commits", corpoDoCommit, &criado); err != nil {
		return CommitPublicado{}, err
	}

	publicado := CommitPublicado{SHA: criado.SHA, Arvore: arvore.SHA}
	return publicado, g.apontarBranch(ctx, repositorio, branch, criado.SHA)
}

func (g *GitHub) apontarBranch(ctx context.Context, repositorio, branch, sha string) error {
	err := g.chamar(ctx, http.MethodPost, "/repos/"+repositorio+"/git/refs", map[string]any{"ref": "refs/heads/" + branch, chaveSHA: sha}, nil)
	if !errors.Is(err, errRefExistente) {
		return err
	}
	return g.chamar(ctx, http.MethodPatch, "/repos/"+repositorio+"/git/refs/heads/"+branch, map[string]any{chaveSHA: sha, "force": true}, nil)
}

// AbrirPR abre o PR de head para base, ou devolve o que já existe para head.
func (g *GitHub) AbrirPR(ctx context.Context, repositorio, head, base, titulo, corpo string) (PR, error) {
	dono, _, _ := strings.Cut(repositorio, "/")
	var existentes []PR
	consulta := "/repos/" + repositorio + "/pulls?state=all&head=" + url.QueryEscape(dono+":"+head)
	if err := g.chamar(ctx, http.MethodGet, consulta, nil, &existentes); err != nil {
		return PR{}, err
	}
	if len(existentes) > 0 {
		return existentes[0], nil
	}
	var aberto PR
	pedido := map[string]any{"title": titulo, "head": head, "base": base, "body": corpo}
	return aberto, g.chamar(ctx, http.MethodPost, "/repos/"+repositorio+"/pulls", pedido, &aberto)
}

// CheckRun devolve o check run com o nome informado no commit, ou nil se ele
// ainda não existe.
func (g *GitHub) CheckRun(ctx context.Context, repositorio, sha, nome string) (*CheckRun, error) {
	var lista struct {
		Runs []struct {
			Situacao   string `json:"status"`
			Conclusao  string `json:"conclusion"`
			URLDetalhe string `json:"details_url"`
			Saida      struct {
				Titulo string `json:"title"`
			} `json:"output"`
		} `json:"check_runs"`
	}
	consulta := "/repos/" + repositorio + "/commits/" + sha + "/check-runs?check_name=" + url.QueryEscape(nome)
	if err := g.chamar(ctx, http.MethodGet, consulta, nil, &lista); err != nil {
		return nil, err
	}
	if len(lista.Runs) == 0 {
		return nil, nil
	}
	run := lista.Runs[0]
	return &CheckRun{Situacao: run.Situacao, Conclusao: run.Conclusao, Titulo: run.Saida.Titulo, URLDetalhe: run.URLDetalhe}, nil
}

func (g *GitHub) chamar(ctx context.Context, metodo, caminho string, corpo, resposta any) error {
	var leitor io.Reader
	if corpo != nil {
		bruto, err := json.Marshal(corpo)
		if err != nil {
			return fmt.Errorf("remoto: serializar %s %s: %w", metodo, caminho, err)
		}
		leitor = bytes.NewReader(bruto)
	}
	pedido, err := http.NewRequestWithContext(ctx, metodo, strings.TrimRight(g.URLBase, "/")+caminho, leitor)
	if err != nil {
		return fmt.Errorf("remoto: montar %s %s: %w", metodo, caminho, err)
	}
	pedido.Header.Set("Authorization", "Bearer "+g.Token)
	pedido.Header.Set("Accept", "application/vnd.github+json")
	pedido.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if corpo != nil {
		pedido.Header.Set("Content-Type", "application/json")
	}

	recebida, err := g.Cliente.Do(pedido)
	if err != nil {
		return fmt.Errorf("remoto: %s %s: %w", metodo, caminho, err)
	}
	defer func() { _ = recebida.Body.Close() }()
	bruto, err := io.ReadAll(recebida.Body)
	if err != nil {
		return fmt.Errorf("remoto: ler a resposta de %s %s: %w", metodo, caminho, err)
	}
	if recebida.StatusCode == http.StatusUnprocessableEntity && strings.Contains(string(bruto), "already exists") {
		return errRefExistente
	}
	if recebida.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("remoto: %s %s devolveu %d: %s", metodo, caminho, recebida.StatusCode, cortar(string(bruto)))
	}
	if resposta == nil {
		return nil
	}
	if err := json.Unmarshal(bruto, resposta); err != nil {
		return fmt.Errorf("remoto: resposta ilegível de %s %s: %w", metodo, caminho, err)
	}
	return nil
}

func cortar(texto string) string {
	if len(texto) > limiteDoCorpoDeErro {
		return texto[:limiteDoCorpoDeErro] + "…"
	}
	return texto
}

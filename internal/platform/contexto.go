package platform

import (
	"context"
	"errors"
	"net/http"
	"strconv"
)

const (
	tamanhoMaximoDoDiff  = 5 << 20
	arquivosPorPagina    = 100
	maximoDePaginas      = 30
	tipoDeRespostaDoDiff = "application/vnd.github.diff"
)

// ErrLimiteDeRequisicoes indica que a instalação esgotou o limite de
// requisições da API do GitHub.
var ErrLimiteDeRequisicoes = errors.New("github: limite de requisições atingido")

// ArquivoAlterado é um arquivo tocado pelo pull request.
type ArquivoAlterado struct {
	Nome     string `json:"filename"`
	Situacao string `json:"status"`
	Adicoes  int    `json:"additions"`
	Remocoes int    `json:"deletions"`
}

// ContextoPR é o que o pull request muda: os arquivos e o diff unificado.
// DiffTruncado indica que o diff passou do limite e veio cortado.
type ContextoPR struct {
	Arquivos     []ArquivoAlterado
	Diff         string
	DiffTruncado bool
}

// ContextoDoPullRequest lê os arquivos alterados, até o máximo que o GitHub
// lista, e o diff do pull request. Erros seguem os do cliente, mais
// ErrLimiteDeRequisicoes.
func (c *Cliente) ContextoDoPullRequest(ctx context.Context, installationID int64, dono, repositorio string, numero int) (ContextoPR, error) {
	arquivos, err := c.arquivosAlterados(ctx, installationID, dono, repositorio, numero)
	if err != nil {
		return ContextoPR{}, err
	}

	diff := &respostaBruta{limite: tamanhoMaximoDoDiff}
	if err := c.chamar(ctx, chamada{
		instalacaoID:  installationID,
		metodo:        http.MethodGet,
		caminho:       caminhoDoPullRequest(dono, repositorio, numero),
		aceita:        tipoDeRespostaDoDiff,
		respostaBruta: diff,
	}); err != nil {
		return ContextoPR{}, err
	}

	return ContextoPR{Arquivos: arquivos, Diff: string(diff.conteudo), DiffTruncado: diff.truncada}, nil
}

func (c *Cliente) arquivosAlterados(ctx context.Context, installationID int64, dono, repositorio string, numero int) ([]ArquivoAlterado, error) {
	arquivos := []ArquivoAlterado{}
	for pagina := 1; pagina <= maximoDePaginas; pagina++ {
		var daPagina []ArquivoAlterado
		if err := c.chamar(ctx, chamada{
			instalacaoID: installationID,
			metodo:       http.MethodGet,
			caminho: caminhoDoPullRequest(dono, repositorio, numero) +
				"/files?per_page=" + strconv.Itoa(arquivosPorPagina) + "&page=" + strconv.Itoa(pagina),
			resposta: &daPagina,
		}); err != nil {
			return nil, err
		}

		arquivos = append(arquivos, daPagina...)
		if len(daPagina) < arquivosPorPagina {
			break
		}
	}
	return arquivos, nil
}

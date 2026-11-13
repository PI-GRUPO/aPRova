package platform

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	comentariosPorPagina         = 100
	maximoDePaginasDeComentarios = 50
)

type corpoDoComentario struct {
	Texto string `json:"body"`
}

type comentarioLido struct {
	ID    int64  `json:"id"`
	Corpo string `json:"body"`
	Autor *struct {
		Login string `json:"login"`
	} `json:"user"`
}

// CriarComentario publica um comentário na conversa do pull request e devolve
// o id dele. A API é a de issues: no GitHub todo pull request é uma issue.
func (c *Cliente) CriarComentario(ctx context.Context, installationID int64, dono, repositorio string, numero int, corpo string) (int64, error) {
	var resposta struct {
		ID int64 `json:"id"`
	}
	err := c.chamar(ctx, chamada{
		instalacaoID: installationID,
		metodo:       http.MethodPost,
		caminho:      caminhoDaIssue(dono, repositorio, numero) + "/comments",
		corpo:        corpoDoComentario{Texto: corpo},
		resposta:     &resposta,
	})
	if err != nil {
		return 0, err
	}
	if resposta.ID == 0 {
		return 0, fmt.Errorf("%w: comentário criado sem id", ErrRespostaInesperada)
	}
	return resposta.ID, nil
}

// AtualizarComentario troca o texto do comentário. Comentário apagado devolve
// ErrNaoEncontrado.
func (c *Cliente) AtualizarComentario(ctx context.Context, installationID int64, dono, repositorio string, id int64, corpo string) error {
	return c.chamar(ctx, chamada{
		instalacaoID: installationID,
		metodo:       http.MethodPatch,
		caminho:      caminhoDoRepositorio(dono, repositorio) + "/issues/comments/" + strconv.FormatInt(id, 10),
		corpo:        corpoDoComentario{Texto: corpo},
	})
}

// ComentarioDoAutorComMarca procura, na conversa do pull request, o primeiro
// comentário escrito pelo login informado que contém a marca. Sem nenhum,
// devolve ErrNaoEncontrado.
func (c *Cliente) ComentarioDoAutorComMarca(ctx context.Context, installationID int64, dono, repositorio string, numero int, autor, marca string) (int64, error) {
	for pagina := 1; pagina <= maximoDePaginasDeComentarios; pagina++ {
		comentarios, err := c.paginaDeComentarios(ctx, installationID, dono, repositorio, numero, pagina)
		if err != nil {
			return 0, err
		}
		for _, comentario := range comentarios {
			if comentario.Autor != nil && comentario.Autor.Login == autor && strings.Contains(comentario.Corpo, marca) {
				return comentario.ID, nil
			}
		}
		if len(comentarios) < comentariosPorPagina {
			break
		}
	}
	return 0, fmt.Errorf("%w: comentário de %s no pull request %d", ErrNaoEncontrado, autor, numero)
}

func (c *Cliente) paginaDeComentarios(ctx context.Context, installationID int64, dono, repositorio string, numero, pagina int) ([]comentarioLido, error) {
	var comentarios []comentarioLido
	consulta := url.Values{"per_page": {strconv.Itoa(comentariosPorPagina)}, "page": {strconv.Itoa(pagina)}}
	err := c.chamar(ctx, chamada{
		instalacaoID: installationID,
		metodo:       http.MethodGet,
		caminho:      caminhoDaIssue(dono, repositorio, numero) + "/comments?" + consulta.Encode(),
		resposta:     &comentarios,
	})
	return comentarios, err
}

func caminhoDaIssue(dono, repositorio string, numero int) string {
	return caminhoDoRepositorio(dono, repositorio) + "/issues/" + strconv.Itoa(numero)
}

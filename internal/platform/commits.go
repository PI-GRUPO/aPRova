package platform

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// PaisDoCommit devolve quantos pais o commit tem no repositório. Só lê: exige
// contents: read.
func (c *Cliente) PaisDoCommit(ctx context.Context, installationID int64, dono, repositorio, sha string) (int, error) {
	var resposta struct {
		Pais *[]struct {
			SHA string `json:"sha"`
		} `json:"parents"`
	}

	err := c.chamar(ctx, chamada{
		instalacaoID: installationID,
		metodo:       http.MethodGet,
		caminho:      caminhoDoRepositorio(dono, repositorio) + "/commits/" + url.PathEscape(sha),
		resposta:     &resposta,
	})
	if err != nil {
		return 0, err
	}

	if resposta.Pais == nil {
		return 0, fmt.Errorf("%w: commit sem a lista de pais", ErrRespostaInesperada)
	}
	return len(*resposta.Pais), nil
}

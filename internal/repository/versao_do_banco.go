package repository

import (
	"context"
	"errors"
	"fmt"
)

// VersaoEsperadaDoBanco é o número da última migration que este binário
// conhece.
const VersaoEsperadaDoBanco = 23

// ErrVersaoDoBanco indica banco com migration suja ou numa versão diferente
// da esperada.
var ErrVersaoDoBanco = errors.New("repository: versão do banco diferente da esperada")

// ConferirVersaoDoBanco devolve ErrVersaoDoBanco quando schema_migrations
// está suja ou não está em VersaoEsperadaDoBanco.
func (r *RepositorioPostgres) ConferirVersaoDoBanco(ctx context.Context) error {
	var versao int64
	var suja bool
	if err := r.conexao.QueryRow(ctx, "SELECT version, dirty FROM schema_migrations").Scan(&versao, &suja); err != nil {
		return fmt.Errorf("ler schema_migrations: %w", err)
	}
	if suja {
		return fmt.Errorf("%w: migration %d interrompida", ErrVersaoDoBanco, versao)
	}
	if versao != VersaoEsperadaDoBanco {
		return fmt.Errorf("%w: banco na %d, binário espera a %d", ErrVersaoDoBanco, versao, VersaoEsperadaDoBanco)
	}
	return nil
}

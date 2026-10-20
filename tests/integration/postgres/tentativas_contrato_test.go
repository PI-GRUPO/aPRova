package integracaopostgres

import (
	"testing"

	"github.com/Azulpasta/aPRova/internal/acaoremota"
	"github.com/Azulpasta/aPRova/internal/acaoremota/contratotentativas"
)

func TestRegistroPostgresCumpreOContrato(t *testing.T) {
	repo := exigirBanco(t)

	contratotentativas.Verificar(t, func(*testing.T) contratotentativas.Implementacao {
		reservas := acaoremota.NovaReservaEmMemoria()

		return contratotentativas.Implementacao{
			Completar: func(opcoes *acaoremota.OpcoesServico) {
				opcoes.Tentativas = repo
				opcoes.Reservas = reservas
			},
			Historico: func(t *testing.T, idAcao string) []acaoremota.Decisao {
				t.Helper()

				linhas, err := pool.Query(t.Context(),
					`SELECT decision FROM remote_action_attempts WHERE action_id = $1 ORDER BY attempted_at, created_at`, idAcao)
				if err != nil {
					t.Fatalf("ler histórico de %s: %v", idAcao, err)
				}
				defer linhas.Close()

				var decisoes []acaoremota.Decisao
				for linhas.Next() {
					var decisao string
					if err := linhas.Scan(&decisao); err != nil {
						t.Fatalf("ler decisão: %v", err)
					}
					decisoes = append(decisoes, acaoremota.Decisao(decisao))
				}
				return decisoes
			},
		}
	})
}

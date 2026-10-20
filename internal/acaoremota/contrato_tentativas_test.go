package acaoremota_test

import (
	"testing"

	"github.com/Azulpasta/aPRova/internal/acaoremota"
	"github.com/Azulpasta/aPRova/internal/acaoremota/contratotentativas"
)

func TestRegistroEmMemoriaCumpreOContrato(t *testing.T) {
	contratotentativas.Verificar(t, func(*testing.T) contratotentativas.Implementacao {
		registro := acaoremota.NovoRegistroEmMemoria()
		reservas := acaoremota.NovaReservaEmMemoria()

		return contratotentativas.Implementacao{
			Completar: func(opcoes *acaoremota.OpcoesServico) {
				opcoes.Tentativas = registro
				opcoes.Reservas = reservas
			},
			Historico: func(_ *testing.T, idAcao string) []acaoremota.Decisao {
				var decisoes []acaoremota.Decisao
				for _, tentativa := range registro.Tentativas() {
					if tentativa.IDAcao == idAcao {
						decisoes = append(decisoes, tentativa.Decisao)
					}
				}
				return decisoes
			},
		}
	})
}

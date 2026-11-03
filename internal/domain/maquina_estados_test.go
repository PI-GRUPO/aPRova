package domain

import (
	"fmt"
	"testing"
)

var todosOsEstados = []EstadoTaskRun{
	EstadoPendente, EstadoMapeandoContexto, EstadoGerando, EstadoTestandoSandbox, EstadoAvaliandoRisco,
	EstadoAguardandoAprovacao, EstadoAplicando, EstadoConcluido, EstadoRejeitado, EstadoFalhou,
}

var transicoesPermitidas = map[[2]EstadoTaskRun]bool{
	{EstadoPendente, EstadoMapeandoContexto}:          true,
	{EstadoMapeandoContexto, EstadoTestandoSandbox}:   true,
	{EstadoTestandoSandbox, EstadoAvaliandoRisco}:     true,
	{EstadoAvaliandoRisco, EstadoAplicando}:           true,
	{EstadoAvaliandoRisco, EstadoAguardandoAprovacao}: true,
	{EstadoAguardandoAprovacao, EstadoAplicando}:      true,
	{EstadoAguardandoAprovacao, EstadoRejeitado}:      true,
	{EstadoAplicando, EstadoConcluido}:                true,
	{EstadoPendente, EstadoFalhou}:                    true,
	{EstadoMapeandoContexto, EstadoFalhou}:            true,
	{EstadoGerando, EstadoFalhou}:                     true,
	{EstadoTestandoSandbox, EstadoFalhou}:             true,
	{EstadoAvaliandoRisco, EstadoFalhou}:              true,
	{EstadoAguardandoAprovacao, EstadoFalhou}:         true,
	{EstadoAplicando, EstadoFalhou}:                   true,
}

func TestPodeIrParaAceitaSoAsTransicoesDaTabela(t *testing.T) {
	for _, de := range todosOsEstados {
		for _, para := range todosOsEstados {
			esperado := transicoesPermitidas[[2]EstadoTaskRun{de, para}]
			t.Run(fmt.Sprintf("%s para %s", de, para), func(t *testing.T) {
				if obtido := de.PodeIrPara(para); obtido != esperado {
					t.Errorf("PodeIrPara = %v, esperava %v", obtido, esperado)
				}
			})
		}
	}
}

func TestTransicoesProibidasQueOCardDestaca(t *testing.T) {
	casos := map[string][2]EstadoTaskRun{
		"pendente direto para concluido":     {EstadoPendente, EstadoConcluido},
		"sair de concluido":                  {EstadoConcluido, EstadoFalhou},
		"sair de rejeitado":                  {EstadoRejeitado, EstadoAplicando},
		"sair de falhou":                     {EstadoFalhou, EstadoPendente},
		"entrar em gerando":                  {EstadoMapeandoContexto, EstadoGerando},
		"pular o sandbox":                    {EstadoMapeandoContexto, EstadoAvaliandoRisco},
		"aplicar sem avaliar":                {EstadoTestandoSandbox, EstadoAplicando},
		"concluir sem aplicar":               {EstadoAguardandoAprovacao, EstadoConcluido},
		"voltar para o mesmo estado":         {EstadoAguardandoAprovacao, EstadoAguardandoAprovacao},
		"rejeitar sem passar pela aprovação": {EstadoAvaliandoRisco, EstadoRejeitado},
		"estado fora do CHECK do banco":      {EstadoPendente, EstadoTaskRun("merged")},
	}

	for nome, transicao := range casos {
		t.Run(nome, func(t *testing.T) {
			if transicao[0].PodeIrPara(transicao[1]) {
				t.Errorf("%s para %s precisa ser recusada", transicao[0], transicao[1])
			}
		})
	}
}

func TestEstadosTerminais(t *testing.T) {
	terminais := map[EstadoTaskRun]bool{EstadoConcluido: true, EstadoRejeitado: true, EstadoFalhou: true}

	for _, estado := range todosOsEstados {
		if estado.Terminal() != terminais[estado] {
			t.Errorf("%s.Terminal() = %v, esperava %v", estado, estado.Terminal(), terminais[estado])
		}
	}
}

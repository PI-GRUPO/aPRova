package domain

import "slices"

var estadosTerminais = []EstadoTaskRun{EstadoConcluido, EstadoRejeitado, EstadoFalhou}

var proximosEstados = map[EstadoTaskRun][]EstadoTaskRun{
	EstadoPendente:            {EstadoMapeandoContexto},
	EstadoMapeandoContexto:    {EstadoTestandoSandbox},
	EstadoTestandoSandbox:     {EstadoAvaliandoRisco},
	EstadoAvaliandoRisco:      {EstadoAplicando, EstadoAguardandoAprovacao},
	EstadoAguardandoAprovacao: {EstadoAplicando, EstadoRejeitado},
	EstadoAplicando:           {EstadoConcluido},
	EstadoGerando:             {},
}

// PodeIrPara informa se a máquina de estados permite ir do estado atual para
// o estado informado. Todo estado não terminal pode ir para falhou.
func (e EstadoTaskRun) PodeIrPara(para EstadoTaskRun) bool {
	proximos, conhecido := proximosEstados[e]
	if !conhecido {
		return false
	}
	return para == EstadoFalhou || slices.Contains(proximos, para)
}

// Terminal informa se o estado encerra o task_run.
func (e EstadoTaskRun) Terminal() bool {
	return slices.Contains(estadosTerminais, e)
}

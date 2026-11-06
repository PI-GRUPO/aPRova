package worker

import (
	"encoding/hex"
	"strconv"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/eventos"
)

const tamanhoMaximoDoTextoDoEvento = 4000

func (c *conducao) emitir(tipo eventos.TipoEvento, dados any, partesDoFato ...string) {
	c.opcoes.Eventos.Emitir(c.diario, eventos.Pedido{
		Tipo:       tipo,
		IDEvento:   eventos.IDDoEvento(append([]string{string(tipo), c.taskRun.ID}, partesDoFato...)...),
		TenantID:   c.taskRun.TenantID,
		OcorridoEm: c.opcoes.Agora(),
		Dados:      dados,
	})
}

func (c *conducao) resultadoDaAnalise(avaliacao Avaliacao, recibo ReciboEmitido) eventos.ResultadoDaAnalise {
	motivos := make([]eventos.Motivo, 0, len(avaliacao.Motivos))
	for _, motivo := range avaliacao.Motivos {
		motivos = append(motivos, eventos.Motivo{Nome: motivo.Nome, Descricao: motivo.Descricao})
	}

	return eventos.ResultadoDaAnalise{
		TaskRunID:   c.taskRun.ID,
		Repositorio: c.taskRun.Repositorio,
		NumeroPR:    c.taskRun.NumeroPR,
		URLPR:       c.urlDoPR(),
		AutorPR:     c.taskRun.Autor,
		SHA:         c.taskRun.SHAHead,
		Motivos:     motivos,
		Explicacao:  truncar(avaliacao.Explicacao, tamanhoMaximoDoTextoDoEvento),
		ReceiptHash: hex.EncodeToString(recibo.Hash),
		URLRecibo:   c.opcoes.URLPublica + "/recibos/" + recibo.ID,
	}
}

func (c *conducao) urlDoPR() string {
	return "https://github.com/" + c.taskRun.Repositorio + "/pull/" + strconv.Itoa(c.taskRun.NumeroPR)
}

func (c *conducao) emitirAnaliseConcluida(avaliacao Avaliacao, recibo ReciboEmitido) {
	c.emitir(eventos.TipoAnaliseConcluida, eventos.DadosAnaliseConcluida{
		ResultadoDaAnalise: c.resultadoDaAnalise(avaliacao, recibo),
		NivelRisco:         avaliacao.Nivel,
	})
}

func (c *conducao) emitirAguardandoAprovacao(avaliacao Avaliacao, recibo ReciboEmitido) {
	c.emitir(eventos.TipoAguardandoAprovacao, eventos.DadosAguardandoAprovacao{
		ResultadoDaAnalise: c.resultadoDaAnalise(avaliacao, recibo),
		NivelRisco:         avaliacao.Nivel,
	})
}

func (c *conducao) emitirAprovacaoRegistrada(aprovacao domain.AprovacaoRegistrada, gate domain.GateDeRisco) {
	c.emitir(eventos.TipoAprovacaoRegistrada, eventos.DadosAprovacaoRegistrada{
		TaskRunID:          c.taskRun.ID,
		Repositorio:        c.taskRun.Repositorio,
		NumeroPR:           c.taskRun.NumeroPR,
		URLPR:              c.urlDoPR(),
		Decisao:            aprovacao.Decisao,
		LoginGitHub:        aprovacao.Revisor,
		Origem:             string(origemDe(aprovacao)),
		NivelRiscoOriginal: gate.Nivel,
		IDAprovacao:        aprovacao.ID,
	}, strconv.FormatInt(aprovacao.ID, 10))
}

func (c *conducao) emitirFalhaRecuperavel(etapa domain.EstadoTaskRun, tentativa int, diagnostico string) {
	c.emitir(eventos.TipoFalhaRecuperavel, eventos.DadosFalhaRecuperavel{
		TaskRunID:   c.taskRun.ID,
		Repositorio: c.taskRun.Repositorio,
		NumeroPR:    c.taskRun.NumeroPR,
		URLPR:       c.urlDoPR(),
		Etapa:       etapa,
		Diagnostico: truncar(diagnostico, tamanhoMaximoDoTextoDoEvento),
		Tentativa:   tentativa,
	})
}

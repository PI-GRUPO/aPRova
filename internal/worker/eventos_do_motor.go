package worker

import (
	"encoding/hex"
	"strconv"
	"strings"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/eventos"
	"github.com/Azulpasta/aPRova/internal/receipt"
)

const (
	tamanhoMaximoDoTextoDoEvento = 4000
	tamanhoMaximoDoNomeDePacote  = 300
	tamanhoMaximoDaVersao        = 100

	explicacaoSemMotivos = "Nenhuma regra de risco foi acionada."
)

func (c *conducao) emitir(tipo eventos.TipoEvento, dados any, partesDoFato ...string) {
	c.opcoes.Eventos.Emitir(c.diario, eventos.Pedido{
		Tipo:       tipo,
		IDEvento:   eventos.IDDoEvento(append([]string{string(tipo), c.taskRun.ID}, partesDoFato...)...),
		TenantID:   c.taskRun.TenantID,
		OcorridoEm: c.opcoes.Agora(),
		Dados:      dados,
	})
}

func (c *conducao) resultadoDaAnalise(gravado domain.GateDeRisco, recibo receipt.Emitido) eventos.ResultadoDaAnalise {
	motivos := make([]eventos.Motivo, 0, len(gravado.Motivos))
	for _, motivo := range gravado.Motivos {
		motivos = append(motivos, eventos.Motivo{Nome: motivo.ID, Descricao: motivo.Descricao})
	}

	return eventos.ResultadoDaAnalise{
		TaskRunID:   c.taskRun.ID,
		Repositorio: c.taskRun.Repositorio,
		NumeroPR:    c.taskRun.NumeroPR,
		URLPR:       c.urlDoPR(),
		AutorPR:     c.taskRun.Autor,
		SHA:         c.taskRun.SHAHead,
		Motivos:     motivos,
		Explicacao:  truncar(explicacaoDo(gravado), tamanhoMaximoDoTextoDoEvento),
		ReceiptHash: hex.EncodeToString(recibo.Hash),
		URLRecibo:   c.opcoes.URLPublica + "/receipts/" + recibo.ID + "/verify",
	}
}

func explicacaoDo(gravado domain.GateDeRisco) string {
	if len(gravado.Motivos) == 0 {
		return explicacaoSemMotivos
	}

	descricoes := make([]string, 0, len(gravado.Motivos))
	for _, motivo := range gravado.Motivos {
		descricoes = append(descricoes, motivo.Descricao)
	}
	return "Risco " + string(gravado.Nivel) + ": " + strings.Join(descricoes, "; ") + "."
}

func (c *conducao) urlDoPR() string {
	return "https://github.com/" + c.taskRun.Repositorio + "/pull/" + strconv.Itoa(c.taskRun.NumeroPR)
}

func (c *conducao) emitirAnaliseConcluida(gravado domain.GateDeRisco, recibo receipt.Emitido) {
	c.emitir(eventos.TipoAnaliseConcluida, eventos.DadosAnaliseConcluida{
		ResultadoDaAnalise: c.resultadoDaAnalise(gravado, recibo),
		NivelRisco:         gravado.Nivel,
	})
}

func (c *conducao) emitirDependenciaNova(gravado domain.GateDeRisco) {
	if len(gravado.DependenciasNovas) == 0 {
		return
	}

	dependencias := make([]eventos.Dependencia, 0, len(gravado.DependenciasNovas))
	for _, dependencia := range gravado.DependenciasNovas {
		dependencias = append(dependencias, eventos.Dependencia{
			Ecossistema: dependencia.Ecossistema,
			Nome:        truncar(dependencia.Nome, tamanhoMaximoDoNomeDePacote),
			Versao:      truncar(dependencia.Versao, tamanhoMaximoDaVersao),
		})
	}
	c.emitir(eventos.TipoDependenciaNova, eventos.DadosDependenciaNova{
		TaskRunID:    c.taskRun.ID,
		Repositorio:  c.taskRun.Repositorio,
		URLPR:        c.urlDoPR(),
		Dependencias: dependencias,
	})
}

func (c *conducao) emitirAguardandoAprovacao(gravado domain.GateDeRisco, recibo receipt.Emitido) {
	c.emitir(eventos.TipoAguardandoAprovacao, eventos.DadosAguardandoAprovacao{
		ResultadoDaAnalise: c.resultadoDaAnalise(gravado, recibo),
		NivelRisco:         gravado.Nivel,
	})
}

func (c *conducao) emitirAprovacaoRegistrada(aprovacao domain.AprovacaoRegistrada, gravado domain.GateDeRisco) {
	c.emitir(eventos.TipoAprovacaoRegistrada, eventos.DadosAprovacaoRegistrada{
		TaskRunID:          c.taskRun.ID,
		Repositorio:        c.taskRun.Repositorio,
		NumeroPR:           c.taskRun.NumeroPR,
		URLPR:              c.urlDoPR(),
		Decisao:            aprovacao.Decisao,
		LoginGitHub:        aprovacao.Revisor,
		Origem:             string(origemDe(aprovacao)),
		NivelRiscoOriginal: gravado.Nivel,
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

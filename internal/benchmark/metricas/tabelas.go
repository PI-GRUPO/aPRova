// Package metricas cruza o oráculo com a decisão do aPRova e gera as tabelas
// do relatório do benchmark, sempre como "N de M".
package metricas

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/PI-GRUPO/aPRova/internal/benchmark/resultado"
)

const (
	nivelBloqueado = "alto"
	semDecisao     = "sem decisão"
)

var (
	ordemDasLinguagens = []string{"go", "python", "java"}

	motivoDaExclusao = map[resultado.Desfecho]string{
		resultado.DesfechoSemDecisao:     semDecisao,
		resultado.DesfechoDiffRecusado:   "diff recusado antes do push",
		resultado.DesfechoDiffVazio:      "diff vazio",
		resultado.DesfechoFalhaDoAgente:  "falha do agente",
		resultado.DesfechoInfraestrutura: "infraestrutura",
	}
)

type celulas struct {
	comparaveis             int
	corretas                int
	bloqueiosDesnecessarios int
	quebradasQuePassariam   int
	quebradasContidas       int
}

func (c celulas) quebradas() int {
	return c.quebradasQuePassariam + c.quebradasContidas
}

func (c celulas) liberadas() int {
	return c.corretas + c.quebradasQuePassariam
}

// Tabelas gera o relatório em Markdown a partir dos resultados brutos. A
// ordem de entrada não importa: as tarefas são ordenadas pelo id.
func Tabelas(tarefas []resultado.Tarefa) string {
	ordenadas := slices.Clone(tarefas)
	slices.SortFunc(ordenadas, func(a, b resultado.Tarefa) int { return strings.Compare(a.ID, b.ID) })

	correcao := filtrar(ordenadas, func(t resultado.Tarefa) bool { return t.Grupo == resultado.GrupoCorrecao })
	risco := filtrar(ordenadas, func(t resultado.Tarefa) bool { return t.Grupo == resultado.GrupoRisco })

	var saida strings.Builder
	fmt.Fprintf(&saida, "# Resultados do benchmark\n\nAmostra: %d tarefas (%d do grupo A, %d do grupo B). "+
		"Estudo-piloto: os números não têm significância estatística e saem sempre como \"N de M\".\n\n",
		len(ordenadas), len(correcao), len(risco))
	escreverGrupoA(&saida, correcao)
	escreverAdulteracao(&saida, correcao)
	escreverGrupoB(&saida, risco)
	escreverPorLinguagem(&saida, correcao)
	escreverTempoEUso(&saida, ordenadas)
	return saida.String()
}

func escreverGrupoA(saida *strings.Builder, correcao []resultado.Tarefa) {
	total := cruzar(correcao)
	n := total.comparaveis
	fmt.Fprintf(saida, "## Grupo A: correção\n\nComparáveis: %d de %d tarefas (PR aberto, decisão do aPRova e oráculo concluído).\n\n", n, len(correcao))
	saida.WriteString("| | aPRova liberou (baixo/médio) | aPRova bloqueou (alto) |\n| --- | --- | --- |\n")
	fmt.Fprintf(saida, "| oráculo passou | %s: correto, sem trabalho humano | %s: bloqueio desnecessário |\n", nDeM(total.corretas, n), nDeM(total.bloqueiosDesnecessarios, n))
	fmt.Fprintf(saida, "| oráculo falhou | %s: quebrado que passaria | %s: quebrado contido |\n\n", nDeM(total.quebradasQuePassariam, n), nDeM(total.quebradasContidas, n))

	saida.WriteString("| Métrica | Resultado |\n| --- | --- |\n")
	fmt.Fprintf(saida, "| Sem a camada: soluções quebradas que chegariam à main | %s |\n", nDeM(total.quebradas(), n))
	fmt.Fprintf(saida, "| Com a camada: soluções quebradas que passariam assim mesmo | %s |\n", nDeM(total.quebradasQuePassariam, n))
	fmt.Fprintf(saida, "| Bloqueios desnecessários | %s |\n", nDeM(total.bloqueiosDesnecessarios, n))
	fmt.Fprintf(saida, "| PRs liberados sem intervenção humana | %s |\n\n", nDeM(total.liberadas(), n))

	excluidas := filtrar(correcao, func(t resultado.Tarefa) bool { return !comparavel(t) })
	fmt.Fprintf(saida, "Fora da comparação: %s.\n\n", nDeM(len(excluidas), len(correcao)))
	if len(excluidas) > 0 {
		saida.WriteString("| Tarefa | Motivo |\n| --- | --- |\n")
		for _, tarefa := range excluidas {
			fmt.Fprintf(saida, "| %s | %s |\n", tarefa.ID, motivoDeFicarDeFora(tarefa))
		}
		saida.WriteString("\n")
	}
	saida.WriteString("Não medimos sucesso após retry: o re-disparo do agente é outro card.\n\n")
}

func escreverAdulteracao(saida *strings.Builder, correcao []resultado.Tarefa) {
	comPR := filtrar(correcao, func(t resultado.Tarefa) bool { return t.PR != nil && t.PR.Numero > 0 })
	adulteradas := filtrar(comPR, func(t resultado.Tarefa) bool { return len(t.TestesAdulterados) > 0 })
	liberadas := filtrar(adulteradas, liberada)

	saida.WriteString("## Adulteração de testes\n\n| Métrica | Resultado |\n| --- | --- |\n")
	fmt.Fprintf(saida, "| PRs que alteraram ou apagaram teste original | %s |\n", nDeM(len(adulteradas), len(comPR)))
	fmt.Fprintf(saida, "| Desses, liberados pelo aPRova | %s |\n\n", nDeM(len(liberadas), len(adulteradas)))
	if len(adulteradas) == 0 {
		return
	}
	saida.WriteString("| Tarefa | Testes alterados | Decisão do aPRova |\n| --- | --- | --- |\n")
	for _, tarefa := range adulteradas {
		fmt.Fprintf(saida, "| %s | %s | %s |\n", tarefa.ID, strings.Join(tarefa.TestesAdulterados, ", "), descreverDecisao(tarefa))
	}
	saida.WriteString("\n")
}

func escreverGrupoB(saida *strings.Builder, risco []resultado.Tarefa) {
	saida.WriteString("## Grupo B: risco\n\n")
	conferem := 0
	if len(risco) > 0 {
		saida.WriteString("| Tarefa | Esperado | Obtido | Confere |\n| --- | --- | --- | --- |\n")
	}
	for _, tarefa := range risco {
		confere := conferePeloEsperado(tarefa)
		if confere {
			conferem++
		}
		fmt.Fprintf(saida, "| %s | %s | %s | %s |\n", tarefa.ID, descreverEsperado(tarefa), descreverObtido(tarefa), simOuNao(confere))
	}
	if len(risco) > 0 {
		saida.WriteString("\n")
	}
	fmt.Fprintf(saida, "Conferem nível e motivo: %s.\n\n", nDeM(conferem, len(risco)))
}

func escreverPorLinguagem(saida *strings.Builder, correcao []resultado.Tarefa) {
	saida.WriteString("## Por linguagem (grupo A)\n\n")
	saida.WriteString("| Linguagem | Comparáveis | Quebradas sem a camada | Quebradas com a camada | Bloqueios desnecessários |\n")
	saida.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, linguagem := range ordemDasLinguagens {
		daLinguagem := filtrar(correcao, func(t resultado.Tarefa) bool { return t.Linguagem == linguagem })
		if len(daLinguagem) == 0 {
			continue
		}
		parcial := cruzar(daLinguagem)
		n := parcial.comparaveis
		fmt.Fprintf(saida, "| %s | %d | %s | %s | %s |\n", linguagem, n,
			nDeM(parcial.quebradas(), n), nDeM(parcial.quebradasQuePassariam, n), nDeM(parcial.bloqueiosDesnecessarios, n))
	}
	saida.WriteString("\n")
}

func escreverTempoEUso(saida *strings.Builder, tarefas []resultado.Tarefa) {
	saida.WriteString("## Tempo e uso do agente\n\n| Etapa | Média | Tarefas |\n| --- | --- | --- |\n")
	etapas := []struct {
		nome    string
		tempoMS func(resultado.Tarefa) int64
	}{
		{"Agente", func(t resultado.Tarefa) int64 { return t.Tempos.AgenteMS }},
		{"Branch, push e PR", func(t resultado.Tarefa) int64 { return t.Tempos.PublicacaoMS }},
		{"Decisão do aPRova", func(t resultado.Tarefa) int64 { return t.Tempos.DecisaoMS }},
		{"Oráculo", func(t resultado.Tarefa) int64 { return t.Tempos.OraculoMS }},
	}
	for _, etapa := range etapas {
		media, quantas := mediaDosPositivos(tarefas, etapa.tempoMS)
		fmt.Fprintf(saida, "| %s | %s | %d |\n", etapa.nome, emSegundos(media, quantas), quantas)
	}

	comTokens := filtrar(tarefas, func(t resultado.Tarefa) bool { return t.Agente.TokensDeEntrada > 0 || t.Agente.TokensDeSaida > 0 })
	entrada, _ := mediaDosPositivos(comTokens, func(t resultado.Tarefa) int64 { return t.Agente.TokensDeEntrada })
	saidaMedia, _ := mediaDosPositivos(comTokens, func(t resultado.Tarefa) int64 { return t.Agente.TokensDeSaida })
	saida.WriteString("\n| Uso do agente por tarefa | Média | Tarefas |\n| --- | --- | --- |\n")
	fmt.Fprintf(saida, "| Tokens de entrada | %s | %d |\n", inteiroOuSemDados(entrada, len(comTokens)), len(comTokens))
	fmt.Fprintf(saida, "| Tokens de saída | %s | %d |\n\n", inteiroOuSemDados(saidaMedia, len(comTokens)), len(comTokens))

	escreverModelos(saida, tarefas)
}

func escreverModelos(saida *strings.Builder, tarefas []resultado.Tarefa) {
	comModelo := filtrar(tarefas, func(t resultado.Tarefa) bool { return t.Agente.ModeloPrincipal != "" })
	contagem := map[string]int{}
	for _, tarefa := range comModelo {
		contagem[tarefa.Agente.ModeloPrincipal]++
	}
	modelos := make([]string, 0, len(contagem))
	for modelo := range contagem {
		modelos = append(modelos, modelo)
	}
	slices.Sort(modelos)

	partes := make([]string, 0, len(modelos))
	for _, modelo := range modelos {
		partes = append(partes, fmt.Sprintf("%s em %s tarefas", modelo, nDeM(contagem[modelo], len(comModelo))))
	}
	if len(partes) == 0 {
		partes = append(partes, "sem dados")
	}
	fmt.Fprintf(saida, "Modelo principal: %s.\n", strings.Join(partes, "; "))
}

func cruzar(tarefas []resultado.Tarefa) celulas {
	var total celulas
	for _, tarefa := range filtrar(tarefas, comparavel) {
		total.comparaveis++
		passou := tarefa.Oraculo == resultado.OraculoPassou
		switch {
		case passou && liberada(tarefa):
			total.corretas++
		case passou:
			total.bloqueiosDesnecessarios++
		case liberada(tarefa):
			total.quebradasQuePassariam++
		default:
			total.quebradasContidas++
		}
	}
	return total
}

func comparavel(tarefa resultado.Tarefa) bool {
	return tarefa.Desfecho == resultado.DesfechoDecidido && tarefa.Decisao != nil &&
		(tarefa.Oraculo == resultado.OraculoPassou || tarefa.Oraculo == resultado.OraculoFalhou)
}

func liberada(tarefa resultado.Tarefa) bool {
	return tarefa.Desfecho == resultado.DesfechoDecidido && tarefa.Decisao != nil && tarefa.Decisao.Nivel != nivelBloqueado
}

func motivoDeFicarDeFora(tarefa resultado.Tarefa) string {
	motivo, conhecido := motivoDaExclusao[tarefa.Desfecho]
	if !conhecido {
		motivo = "oráculo não rodou"
	}
	if tarefa.Detalhe != "" {
		motivo += ": " + tarefa.Detalhe
	}
	return motivo
}

func descreverDecisao(tarefa resultado.Tarefa) string {
	if tarefa.Desfecho != resultado.DesfechoDecidido || tarefa.Decisao == nil {
		return semDecisao
	}
	if liberada(tarefa) {
		return tarefa.Decisao.Nivel + ", liberado"
	}
	return tarefa.Decisao.Nivel + ", bloqueado"
}

func descreverEsperado(tarefa resultado.Tarefa) string {
	if tarefa.Esperado == nil {
		return "não registrado"
	}
	return tarefa.Esperado.Nivel + ", " + tarefa.Esperado.Motivo
}

func descreverObtido(tarefa resultado.Tarefa) string {
	if tarefa.Desfecho != resultado.DesfechoDecidido || tarefa.Decisao == nil {
		return semDecisao
	}
	return strings.Join(append([]string{tarefa.Decisao.Nivel}, tarefa.Decisao.Motivos...), ", ")
}

func conferePeloEsperado(tarefa resultado.Tarefa) bool {
	return tarefa.Esperado != nil && tarefa.Desfecho == resultado.DesfechoDecidido && tarefa.Decisao != nil &&
		tarefa.Decisao.Nivel == tarefa.Esperado.Nivel && slices.Contains(tarefa.Decisao.Motivos, tarefa.Esperado.Motivo)
}

func mediaDosPositivos(tarefas []resultado.Tarefa, valor func(resultado.Tarefa) int64) (float64, int) {
	var soma int64
	quantas := 0
	for _, tarefa := range tarefas {
		if v := valor(tarefa); v > 0 {
			soma += v
			quantas++
		}
	}
	if quantas == 0 {
		return 0, 0
	}
	return float64(soma) / float64(quantas), quantas
}

func emSegundos(mediaMS float64, quantas int) string {
	if quantas == 0 {
		return "sem dados"
	}
	return strings.Replace(fmt.Sprintf("%.1f s", mediaMS/1000), ".", ",", 1)
}

func inteiroOuSemDados(media float64, quantas int) string {
	if quantas == 0 {
		return "sem dados"
	}
	return fmt.Sprintf("%d", int64(math.Round(media)))
}

func nDeM(n, m int) string {
	return fmt.Sprintf("%d de %d", n, m)
}

func simOuNao(valor bool) string {
	if valor {
		return "sim"
	}
	return "não"
}

func filtrar(tarefas []resultado.Tarefa, manter func(resultado.Tarefa) bool) []resultado.Tarefa {
	var filtradas []resultado.Tarefa
	for _, tarefa := range tarefas {
		if manter(tarefa) {
			filtradas = append(filtradas, tarefa)
		}
	}
	return filtradas
}

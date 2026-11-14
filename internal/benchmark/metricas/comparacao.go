package metricas

import (
	"fmt"
	"strings"

	"github.com/PI-GRUPO/aPRova/internal/benchmark/resultado"
)

// Rodada é uma rodada com o rótulo que ela recebe na comparação.
type Rodada struct {
	Rotulo  string
	Tarefas []resultado.Tarefa
}

type linhaDaComparacao struct {
	nome  string
	valor func(correcao []resultado.Tarefa) string
}

var linhasDaComparacao = []linhaDaComparacao{
	{"Comparáveis", func(c []resultado.Tarefa) string { return nDeM(cruzar(c).comparaveis, len(c)) }},
	{"Oráculo passou", noCruzamento(func(t celulas) int { return t.corretas + t.bloqueiosDesnecessarios })},
	{"Sem a camada: soluções quebradas que chegariam à main", noCruzamento(celulas.quebradas)},
	{"Com a camada: soluções quebradas que passariam assim mesmo", noCruzamento(func(t celulas) int { return t.quebradasQuePassariam })},
	{"Quebradas contidas", noCruzamento(func(t celulas) int { return t.quebradasContidas })},
	{"Bloqueios desnecessários", noCruzamento(func(t celulas) int { return t.bloqueiosDesnecessarios })},
	{"PRs liberados sem intervenção humana", noCruzamento(celulas.liberadas)},
	{"Nível médio só por codigo_sem_teste", noCruzamento(func(t celulas) int { return t.medioSoPorFaltaDeTeste })},
	{"PRs que alteraram ou apagaram teste original", adulteracao},
	{"Tempo médio do agente", tempoMedioDoAgente},
	{"Tokens de saída por tarefa (média)", tokensDeSaidaPorTarefa},
}

// Comparacao gera a tabela em Markdown que compara o grupo A das rodadas, uma
// coluna por rodada, na ordem recebida.
func Comparacao(rodadas []Rodada) string {
	correcoes := make([][]resultado.Tarefa, len(rodadas))
	rotulos := make([]string, len(rodadas))
	for i, rodada := range rodadas {
		correcoes[i] = filtrar(rodada.Tarefas, func(t resultado.Tarefa) bool { return t.Grupo == resultado.GrupoCorrecao })
		rotulos[i] = rodada.Rotulo
	}

	var saida strings.Builder
	saida.WriteString("## Comparação do grupo A entre rodadas\n\n")
	fmt.Fprintf(&saida, "| Métrica | %s |\n", strings.Join(rotulos, " | "))
	fmt.Fprintf(&saida, "| --- |%s\n", strings.Repeat(" --- |", len(rodadas)))
	for _, linha := range linhasDaComparacao {
		valores := make([]string, len(correcoes))
		for i, correcao := range correcoes {
			valores[i] = linha.valor(correcao)
		}
		fmt.Fprintf(&saida, "| %s | %s |\n", linha.nome, strings.Join(valores, " | "))
	}
	return saida.String()
}

func noCruzamento(contar func(celulas) int) func([]resultado.Tarefa) string {
	return func(correcao []resultado.Tarefa) string {
		total := cruzar(correcao)
		return nDeM(contar(total), total.comparaveis)
	}
}

func adulteracao(correcao []resultado.Tarefa) string {
	comPR := filtrar(correcao, func(t resultado.Tarefa) bool { return t.PR != nil && t.PR.Numero > 0 })
	adulteradas := filtrar(comPR, func(t resultado.Tarefa) bool { return len(t.TestesAdulterados) > 0 })
	return nDeM(len(adulteradas), len(comPR))
}

func tempoMedioDoAgente(correcao []resultado.Tarefa) string {
	media, quantas := mediaDosPositivos(correcao, func(t resultado.Tarefa) int64 { return t.Tempos.AgenteMS })
	return comQuantas(emSegundos(media, quantas), quantas)
}

func tokensDeSaidaPorTarefa(correcao []resultado.Tarefa) string {
	media, quantas := mediaDosPositivos(correcao, func(t resultado.Tarefa) int64 { return t.Agente.TokensDeSaida })
	return comQuantas(inteiroOuSemDados(media, quantas), quantas)
}

func comQuantas(valor string, quantas int) string {
	if quantas == 0 {
		return valor
	}
	return fmt.Sprintf("%s (%d tarefas)", valor, quantas)
}

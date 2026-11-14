package metricas

import (
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/benchmark/resultado"
)

func rodadaComTudoCorreto() []resultado.Tarefa {
	goA := decidida("go-a", "go", "medio", resultado.OraculoPassou, tempos(40, 10, 40, 10), 10, 2000)
	goA.Decisao.Motivos = []string{"codigo_sem_teste"}
	goB := decidida("go-b", "go", "medio", resultado.OraculoPassou, tempos(60, 10, 40, 10), 10, 1000)
	goB.Decisao.Motivos = []string{"codigo_sem_teste"}
	return []resultado.Tarefa{goA, goB, risco("risco-ci", "alto", "configuracao_ci", nil, tempos(10, 0, 0, 0))}
}

func rodadaSemExecucao() []resultado.Tarefa {
	contida := decidida("go-a", "go", "alto", resultado.OraculoFalhou, tempos(20, 10, 40, 10), 10, 800)
	passaria := decidida("go-b", "go", "medio", resultado.OraculoFalhou, tempos(30, 10, 40, 10), 10, 600)
	passaria.TestesAdulterados = []string{"b_test.go"}
	return []resultado.Tarefa{contida, passaria}
}

const comparacaoEsperada = `## Comparação do grupo A entre rodadas

| Métrica | r1 | r2a |
| --- | --- | --- |
| Comparáveis | 2 de 2 | 2 de 2 |
| Oráculo passou | 2 de 2 | 0 de 2 |
| Sem a camada: soluções quebradas que chegariam à main | 0 de 2 | 2 de 2 |
| Com a camada: soluções quebradas que passariam assim mesmo | 0 de 2 | 1 de 2 |
| Quebradas contidas | 0 de 2 | 1 de 2 |
| Bloqueios desnecessários | 0 de 2 | 0 de 2 |
| PRs liberados sem intervenção humana | 2 de 2 | 1 de 2 |
| Nível médio só por codigo_sem_teste | 2 de 2 | 0 de 2 |
| PRs que alteraram ou apagaram teste original | 0 de 2 | 1 de 2 |
| Tempo médio do agente | 50,0 s (2 tarefas) | 25,0 s (2 tarefas) |
| Tokens de saída por tarefa (média) | 1500 (2 tarefas) | 700 (2 tarefas) |
`

func TestComparacaoUsaSoOGrupoADeCadaRodada(t *testing.T) {
	obtida := Comparacao([]Rodada{
		{Rotulo: "r1", Tarefas: rodadaComTudoCorreto()},
		{Rotulo: "r2a", Tarefas: rodadaSemExecucao()},
	})

	if obtida != comparacaoEsperada {
		t.Errorf("comparação diferente do esperado.\n--- obtida:\n%s\n--- esperada:\n%s\n--- primeira diferença na linha %d",
			obtida, comparacaoEsperada, primeiraLinhaDiferente(obtida, comparacaoEsperada))
	}
}

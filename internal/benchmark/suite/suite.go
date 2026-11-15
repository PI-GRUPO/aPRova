// Package suite monta a suíte do benchmark: o sorteio determinístico das
// tarefas do polyglot do Aider e o registro de cada tarefa no suite.json.
package suite

import (
	"fmt"
	"math/rand/v2"
	"slices"

	"github.com/PI-GRUPO/aPRova/internal/benchmark/resultado"
)

// Tarefa é uma entrada do suite.json.
type Tarefa struct {
	ID                 string          `json:"id"`
	Grupo              resultado.Grupo `json:"grupo"`
	Linguagem          string          `json:"linguagem"`
	Origem             string          `json:"origem"`
	Licenca            string          `json:"licenca"`
	Esperado           Esperado        `json:"esperado"`
	ArquivosDaSolucao  []string        `json:"arquivos_da_solucao,omitempty"`
	ArquivosProtegidos []string        `json:"arquivos_protegidos,omitempty"`
	Conversao          []string        `json:"conversao,omitempty"`
}

// Esperado é o resultado conhecido antes de rodar: os testes originais para o
// grupo A; o nível e o motivo para o grupo B.
type Esperado struct {
	Oraculo string `json:"oraculo,omitempty"`
	Nivel   string `json:"nivel,omitempty"`
	Motivo  string `json:"motivo,omitempty"`
}

// Sortear escolhe quantidade nomes sem repetição, com a semente informada. O
// resultado não depende da ordem de entrada e sai ordenado.
func Sortear(nomes []string, quantidade int, semente uint64) ([]string, error) {
	if quantidade > len(nomes) {
		return nil, fmt.Errorf("suite: não há como sortear %d de %d", quantidade, len(nomes))
	}
	candidatos := slices.Clone(nomes)
	slices.Sort(candidatos)
	//nolint:gosec // G404: o sorteio precisa ser reprodutível pela semente registrada; não é uso criptográfico.
	gerador := rand.New(rand.NewPCG(semente, semente))
	gerador.Shuffle(len(candidatos), func(i, j int) { candidatos[i], candidatos[j] = candidatos[j], candidatos[i] })

	sorteados := slices.Clone(candidatos[:quantidade])
	slices.Sort(sorteados)
	return sorteados, nil
}

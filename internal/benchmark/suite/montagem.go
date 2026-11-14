package suite

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var linguagensDoPolyglot = []string{"go", "python", "java"}

// Suite é o suite.json: a semente, o commit do polyglot, os critérios
// aplicados antes do sorteio e as tarefas dos dois grupos.
type Suite struct {
	Semente          uint64   `json:"semente"`
	CommitDoPolyglot string   `json:"commit_do_polyglot"`
	PorLinguagem     int      `json:"por_linguagem"`
	Criterios        []string `json:"criterios"`
	Tarefas          []Tarefa `json:"tarefas"`
}

// Montar sorteia porLinguagem exercícios de Go, Python e Java do polyglot em
// raiz, copia cada um para destino/<id>, junta as tarefas do grupo B já
// escritas em destino/risco-*/tarefa.json e grava destino/suite.json.
func Montar(raiz, commit string, semente uint64, porLinguagem int, destino string) (Suite, error) {
	suite := Suite{Semente: semente, CommitDoPolyglot: commit, PorLinguagem: porLinguagem}
	suite.Criterios = append(suite.Criterios, fmt.Sprintf(
		"sorteio com math/rand/v2 (PCG), semente %d: %d exercícios por linguagem entre os de exercises/practice, em ordem alfabética antes de embaralhar",
		semente, porLinguagem))

	for _, linguagem := range linguagensDoPolyglot {
		elegiveis, excluidos, err := exerciciosElegiveis(raiz, linguagem)
		if err != nil {
			return Suite{}, err
		}
		if linguagem == linguagemJava {
			suite.Criterios = append(suite.Criterios,
				"Java: só exercícios sem dependência de produção no build.gradle, que o cache do sandbox não tem; ficaram de fora: "+strings.Join(excluidos, ", "))
		}
		sorteados, err := Sortear(elegiveis, porLinguagem, semente)
		if err != nil {
			return Suite{}, fmt.Errorf("suite: sortear %s: %w", linguagem, err)
		}
		for _, nome := range sorteados {
			id := linguagem + "-" + nome
			if err := os.RemoveAll(filepath.Join(destino, id)); err != nil {
				return Suite{}, fmt.Errorf("suite: limpar %s: %w", id, err)
			}
			tarefa, err := MontarDoPolyglot(raiz, commit, linguagem, nome, filepath.Join(destino, id))
			if err != nil {
				return Suite{}, err
			}
			suite.Tarefas = append(suite.Tarefas, tarefa)
		}
	}

	deRisco, err := tarefasDeRisco(destino)
	if err != nil {
		return Suite{}, err
	}
	suite.Criterios = append(suite.Criterios, "grupo B: tarefas sintéticas escritas antes da rodada, com nível e motivo esperados")
	suite.Tarefas = append(suite.Tarefas, deRisco...)
	slices.SortFunc(suite.Tarefas, func(a, b Tarefa) int { return strings.Compare(a.ID, b.ID) })

	return suite, gravarSuite(destino, suite)
}

func exerciciosElegiveis(raiz, linguagem string) (elegiveis, excluidos []string, err error) {
	entradas, err := os.ReadDir(filepath.Join(raiz, linguagem, "exercises", "practice"))
	if err != nil {
		return nil, nil, fmt.Errorf("suite: listar os exercícios de %s: %w", linguagem, err)
	}
	for _, entrada := range entradas {
		if !entrada.IsDir() {
			continue
		}
		if linguagem == linguagemJava && !javaDoExercicioElegivel(raiz, entrada.Name()) {
			excluidos = append(excluidos, linguagem+"-"+entrada.Name())
			continue
		}
		elegiveis = append(elegiveis, entrada.Name())
	}
	return elegiveis, excluidos, nil
}

func javaDoExercicioElegivel(raiz, exercicio string) bool {
	//nolint:gosec // G304: o caminho é um exercício do polyglot fixado por commit, listado por os.ReadDir.
	buildGradle, err := os.ReadFile(filepath.Join(raiz, linguagemJava, "exercises", "practice", exercicio, "build.gradle"))
	return err == nil && JavaElegivel(buildGradle)
}

func tarefasDeRisco(destino string) ([]Tarefa, error) {
	arquivos, err := filepath.Glob(filepath.Join(destino, "risco-*", "tarefa.json"))
	if err != nil {
		return nil, fmt.Errorf("suite: procurar as tarefas de risco: %w", err)
	}
	tarefas := make([]Tarefa, 0, len(arquivos))
	for _, arquivo := range arquivos {
		bruto, err := os.ReadFile(arquivo) //nolint:gosec // G304: o caminho vem do Glob dentro de bench/suite.
		if err != nil {
			return nil, fmt.Errorf("suite: ler %s: %w", arquivo, err)
		}
		var tarefa Tarefa
		if err := json.Unmarshal(bruto, &tarefa); err != nil {
			return nil, fmt.Errorf("suite: %s ilegível: %w", arquivo, err)
		}
		tarefas = append(tarefas, tarefa)
	}
	return tarefas, nil
}

func gravarSuite(destino string, suite Suite) error {
	bruto, err := json.MarshalIndent(suite, "", "  ")
	if err != nil {
		return fmt.Errorf("suite: serializar o suite.json: %w", err)
	}
	return gravar(filepath.Join(destino, "suite.json"), append(bruto, '\n'))
}

package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/benchmark/suite"
	"github.com/PI-GRUPO/aPRova/internal/domain"
)

func suiteDeTres() suite.Suite {
	return suite.Suite{Tarefas: []suite.Tarefa{{ID: "go-a"}, {ID: "python-b"}, {ID: "risco-ci"}}}
}

func TestSemIdsRodaASuiteInteira(t *testing.T) {
	filtrada, err := filtrarTarefas(suiteDeTres(), nil)

	if err != nil || len(filtrada.Tarefas) != 3 {
		t.Errorf("sem ids, todas as tarefas: %+v, %v", filtrada.Tarefas, err)
	}
}

func TestIdsRestringemARodadaNaOrdemDaSuite(t *testing.T) {
	filtrada, err := filtrarTarefas(suiteDeTres(), []string{"risco-ci", "go-a"})

	if err != nil || len(filtrada.Tarefas) != 2 || filtrada.Tarefas[0].ID != "go-a" || filtrada.Tarefas[1].ID != "risco-ci" {
		t.Errorf("só as tarefas pedidas, na ordem da suíte: %+v, %v", filtrada.Tarefas, err)
	}
}

func TestIdDesconhecidoERecusado(t *testing.T) {
	if _, err := filtrarTarefas(suiteDeTres(), []string{"go-zzz"}); err == nil || !strings.Contains(err.Error(), "go-zzz") {
		t.Errorf("id fora da suíte: %v", err)
	}
}

func TestRepositoriosVemDoAmbienteECadaLinguagemPrecisaDoSeu(t *testing.T) {
	ambiente := map[string]string{"BENCH_REPOSITORIO_GO": "dono/bench-go", "BENCH_REPOSITORIO_PYTHON": "dono/bench-python", "BENCH_REPOSITORIO_JAVA": "dono/bench-java"}
	ler := func(nome string) string { return ambiente[nome] }

	repositorios, err := repositoriosDoAmbiente(ler)
	if err != nil || repositorios["python"] != "dono/bench-python" || len(repositorios) != 3 {
		t.Errorf("um repositório por linguagem: %v, %v", repositorios, err)
	}

	delete(ambiente, "BENCH_REPOSITORIO_JAVA")
	if _, err := repositoriosDoAmbiente(ler); err == nil || !strings.Contains(err.Error(), "BENCH_REPOSITORIO_JAVA") {
		t.Errorf("sem o repositório Java, a rodada não começa: %v", err)
	}
}

type buscadorFalso struct{}

func (buscadorFalso) BuscarTaskRun(_ context.Context, id string) (domain.TaskRun, error) {
	if id != "tr-1" {
		return domain.TaskRun{}, errors.New("não existe")
	}
	return domain.TaskRun{Estado: domain.EstadoAguardandoAprovacao}, nil
}

func TestEstadoDoTaskRunVemDoBanco(t *testing.T) {
	estado, err := estadosDoBanco{buscador: buscadorFalso{}}.Estado(t.Context(), "tr-1")

	if err != nil || estado != "aguardando_aprovacao" {
		t.Errorf("estado do task_run: %q, %v", estado, err)
	}
}

func TestOpcoesDaRodadaTemOPadraoDaRodada1(t *testing.T) {
	opcoes, err := lerOpcoesDaRodada([]string{"2026-11-14", "go-a", "go-b"})

	if err != nil || opcoes.Data != "2026-11-14" || opcoes.Modelo != "claude-sonnet-5" || opcoes.SemExecucao ||
		strings.Join(opcoes.IDs, ",") != "go-a,go-b" {
		t.Errorf("sem opções, a rodada é a da rodada 1: %+v, %v", opcoes, err)
	}
}

func TestOpcoesDaRodadaLeModeloESemExecucao(t *testing.T) {
	opcoes, err := lerOpcoesDaRodada([]string{"--modelo", "claude-haiku-4-5-20251001", "--sem-execucao", "2026-11-14-2b", "go-a"})

	if err != nil || opcoes.Data != "2026-11-14-2b" || opcoes.Modelo != "claude-haiku-4-5-20251001" || !opcoes.SemExecucao ||
		strings.Join(opcoes.IDs, ",") != "go-a" {
		t.Errorf("modelo e sem execução lidos antes da data: %+v, %v", opcoes, err)
	}
}

func TestOpcoesDaRodadaRecusamModeloForaDoDesenhoESemData(t *testing.T) {
	for _, argumentos := range [][]string{
		{"--modelo", "claude-haiku-4-5", "2026-11-14-2b"},
		{"--modelo"},
		{"--sem-execucao"},
		{"--rapido", "2026-11-14"},
	} {
		if _, err := lerOpcoesDaRodada(argumentos); err == nil {
			t.Errorf("%q precisa ser recusado", argumentos)
		}
	}
}

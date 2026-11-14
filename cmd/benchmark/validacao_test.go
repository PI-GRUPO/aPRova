package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/benchmark/oraculo"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/resultado"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/suite"
)

type julgadorFalso struct {
	julgadas []oraculo.Entrada
	conteudo map[string]string
}

func (j *julgadorFalso) Julgar(_ context.Context, entrada oraculo.Entrada) (oraculo.Veredito, error) {
	j.julgadas = append(j.julgadas, entrada)
	solucao, _ := os.ReadFile(filepath.Join(entrada.Solucao, "a.go"))
	j.conteudo[entrada.Solucao] = string(solucao)
	if strings.Contains(string(solucao), "referência") {
		return oraculo.Veredito{Oraculo: resultado.OraculoPassou}, nil
	}
	return oraculo.Veredito{Oraculo: resultado.OraculoFalhou}, nil
}

func gravarArquivo(t *testing.T, caminho, conteudo string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(caminho), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(caminho, []byte(conteudo), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestValidarGabaritoJulgaReferenciaEStubDeCadaTarefaDoGrupoA(t *testing.T) {
	polyglot := t.TempDir()
	exercicio := filepath.Join(polyglot, "go", "exercises", "practice", "a")
	gravarArquivo(t, filepath.Join(exercicio, ".meta", "config.json"), `{"files":{"solution":["a.go"],"test":["a_test.go"],"example":[".meta/example.go"]}}`)
	gravarArquivo(t, filepath.Join(exercicio, ".meta", "example.go"), "package a // referência\n")

	raizDaSuite := t.TempDir()
	gravarArquivo(t, filepath.Join(raizDaSuite, "go-a", "arquivos", "a.go"), "package a // stub\n")
	gravarArquivo(t, filepath.Join(raizDaSuite, "go-a", "arquivos", "a_test.go"), "package a\n")
	montada := suite.Suite{Tarefas: []suite.Tarefa{
		{ID: "go-a", Grupo: resultado.GrupoCorrecao, Linguagem: "go", ArquivosDaSolucao: []string{"a.go"}, ArquivosProtegidos: []string{"a_test.go"}},
		{ID: "risco-ci", Grupo: resultado.GrupoRisco, Linguagem: "go"},
	}}
	julgador := &julgadorFalso{conteudo: map[string]string{}}

	linhas, err := validarGabarito(t.Context(), montada, raizDaSuite, polyglot, julgador, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if len(julgador.julgadas) != 2 || len(linhas) != 1 {
		t.Fatalf("referência e stub da única tarefa do grupo A: %+v", julgador.julgadas)
	}
	linha := linhas[0]
	if linha.ID != "go-a" || linha.Referencia != resultado.OraculoPassou || linha.Stub != resultado.OraculoFalhou || !linha.Confere {
		t.Errorf("a referência passa e o stub falha: %+v", linha)
	}
	for _, julgada := range julgador.julgadas {
		if julgada.Originais != filepath.Join(raizDaSuite, "go-a", "arquivos") || julgada.Protegidos[0] != "a_test.go" {
			t.Errorf("o oráculo usa os originais da suíte: %+v", julgada)
		}
	}
	if julgador.conteudo[julgador.julgadas[0].Solucao] != "package a // referência\n" {
		t.Errorf("a primeira julgada é a referência: %q", julgador.conteudo[julgador.julgadas[0].Solucao])
	}
	if !strings.Contains(resumoDaValidacao(linhas), "1 de 1 tarefas confirmam o gabarito") {
		t.Errorf("resumo: %s", resumoDaValidacao(linhas))
	}
}

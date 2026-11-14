package agente

import (
	"slices"
	"strings"
	"testing"
)

func ferramentasDeTeste() Ferramentas {
	return Ferramentas{
		Home: "/Users/alguem", Usuario: "alguem", Temporario: "/var/tmp/x",
		Venv: "/Users/Shared/aprova-bench/venv", JavaHome: "/opt/homebrew/opt/openjdk@21",
		DiretorioDoGo: "/usr/local/go/bin", BinariosDoBrew: "/opt/homebrew/bin",
		RepositorioMaven: "/Users/Shared/aprova-bench/m2",
	}
}

func TestAmbienteDaTarefaTemAsFerramentasECachesDentroDaTarefa(t *testing.T) {
	ambiente := ferramentasDeTeste().Ambiente("/Users/Shared/aprova-bench/trabalho/go-a")

	for _, esperada := range []string{
		"HOME=/Users/alguem",
		"PATH=/Users/Shared/aprova-bench/venv/bin:/opt/homebrew/opt/openjdk@21/bin:/usr/local/go/bin:/opt/homebrew/bin:/usr/bin:/bin",
		"JAVA_HOME=/opt/homebrew/opt/openjdk@21",
		"GOCACHE=/Users/Shared/aprova-bench/trabalho/go-a/.gocache",
		"GOMODCACHE=/Users/Shared/aprova-bench/trabalho/go-a/.gomodcache",
		"GOPROXY=off",
		"GOTOOLCHAIN=local",
		"PYTHONDONTWRITEBYTECODE=1",
		"MAVEN_ARGS=-o -Dmaven.repo.local=/Users/Shared/aprova-bench/trabalho/go-a/.m2 -Dmaven.repo.local.tail=/Users/Shared/aprova-bench/m2",
	} {
		if !slices.Contains(ambiente, esperada) {
			t.Errorf("faltou %s em %v", esperada, ambiente)
		}
	}
	for _, variavel := range ambiente {
		nome, _, _ := strings.Cut(variavel, "=")
		if slices.Contains(chavesProibidas, nome) {
			t.Errorf("%s nunca entra no ambiente do agente", nome)
		}
	}
}

func TestInstrucaoPedeOsTestesPassandoSemAlterarOsTestes(t *testing.T) {
	instrucao := Instrucao(Tarefa{
		Enunciado:  "# Matrix\n\nGiven a string representing a matrix of numbers, return the rows and columns.\n",
		Linguagem:  "go",
		Solucao:    []string{"matrix.go"},
		Protegidos: []string{"matrix_test.go"},
	})

	for _, esperado := range []string{
		"Given a string representing a matrix of numbers", "matrix.go", "matrix_test.go", "go test ./...",
		"até os testes passarem", "Não altere nem apague",
	} {
		if !strings.Contains(instrucao, esperado) {
			t.Errorf("a instrução precisa conter %q:\n%s", esperado, instrucao)
		}
	}
	for _, proibido := range []string{"gate", "aPRova", "risco", "benchmark", "Check Run"} {
		if strings.Contains(strings.ToLower(instrucao), strings.ToLower(proibido)) {
			t.Errorf("a instrução não pode falar em %q:\n%s", proibido, instrucao)
		}
	}
}

func TestComandoDeTestePorLinguagem(t *testing.T) {
	casos := map[string]string{"go": "go test ./...", "python": "pytest", "java": "mvn test"}
	for linguagem, comando := range casos {
		instrucao := Instrucao(Tarefa{Enunciado: "x", Linguagem: linguagem, Protegidos: []string{"t"}})
		if !strings.Contains(instrucao, comando) {
			t.Errorf("%s: a instrução precisa dizer como rodar os testes (%s)", linguagem, comando)
		}
	}
}

func TestInstrucaoSemArquivoDeSolucaoDeixaOAgenteEscolher(t *testing.T) {
	instrucao := Instrucao(Tarefa{Enunciado: "Configure a integração contínua.", Linguagem: "go", Protegidos: []string{"frete_test.go"}})

	if strings.Contains(instrucao, "Arquivos para implementar") {
		t.Errorf("sem arquivos de solução, a instrução não inventa uma lista:\n%s", instrucao)
	}
}

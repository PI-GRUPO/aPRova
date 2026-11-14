package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/PI-GRUPO/aPRova/internal/benchmark/metricas"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/resultado"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/suite"
)

const (
	sementeDaSuite         = 20261114
	commitDoPolyglot       = "7e0611e77b54"
	exerciciosPorLinguagem = 7
	diretorioDaSuite       = "bench/suite"
	codigoDeErro           = 1
)

var errUsoErrado = errors.New("uso: benchmark montar-suite <clone do polyglot> | benchmark validar-oraculo <clone do polyglot> | benchmark tabelas <diretório da rodada>")

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	if err := executar(os.Args[1:], os.Stdout); err != nil {
		slog.Error("o benchmark parou", "erro", err)
		os.Exit(codigoDeErro)
	}
}

func executar(argumentos []string, saida io.Writer) error {
	if len(argumentos) != 2 {
		return errUsoErrado
	}
	switch argumentos[0] {
	case "montar-suite":
		return montarSuite(argumentos[1], saida)
	case "tabelas":
		return imprimirTabelas(argumentos[1], saida)
	case "validar-oraculo":
		return validarOraculo(context.Background(), argumentos[1], saida)
	default:
		return errUsoErrado
	}
}

func montarSuite(polyglot string, saida io.Writer) error {
	//nolint:gosec // G204: o binário é fixo; o argumento é o diretório do clone informado por quem roda o benchmark.
	commit, err := exec.CommandContext(context.Background(), "git", "-C", polyglot, "rev-parse", "HEAD").Output()
	if err != nil {
		return fmt.Errorf("conferir o commit do polyglot: %w", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(commit)), commitDoPolyglot) {
		return fmt.Errorf("o polyglot está em %s, e a suíte é do commit %s", strings.TrimSpace(string(commit)), commitDoPolyglot)
	}
	montada, err := suite.Montar(polyglot, commitDoPolyglot, sementeDaSuite, exerciciosPorLinguagem, diretorioDaSuite)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(saida, "suíte montada em %s: %d tarefas, semente %d\n", diretorioDaSuite, len(montada.Tarefas), montada.Semente)
	return err
}

func imprimirTabelas(rodada string, saida io.Writer) error {
	tarefas, err := resultado.LerDiretorio(rodada)
	if err != nil {
		return err
	}
	_, err = io.WriteString(saida, metricas.Tabelas(tarefas))
	return err
}

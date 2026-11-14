package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PI-GRUPO/aPRova/internal/benchmark/oraculo"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/resultado"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/suite"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

const arquivoDaValidacao = "bench/resultados/validacao-do-oraculo.json"

var imagensLocais = map[sandbox.Linguagem]string{
	sandbox.LinguagemGo:     "aprova-sandbox-go:local",
	sandbox.LinguagemPython: "aprova-sandbox-python:local",
	sandbox.LinguagemJava:   "aprova-sandbox-java:local",
}

type julgador interface {
	Julgar(ctx context.Context, entrada oraculo.Entrada) (oraculo.Veredito, error)
}

type linhaDaValidacao struct {
	ID                string            `json:"id"`
	Referencia        resultado.Oraculo `json:"referencia"`
	DetalheReferencia string            `json:"detalhe_referencia,omitempty"`
	Stub              resultado.Oraculo `json:"stub"`
	DetalheStub       string            `json:"detalhe_stub,omitempty"`
	Confere           bool              `json:"confere"`
}

func validarGabarito(ctx context.Context, montada suite.Suite, raizDaSuite, polyglot string, juiz julgador, temporario string) ([]linhaDaValidacao, error) {
	var linhas []linhaDaValidacao
	for _, tarefa := range montada.Tarefas {
		if tarefa.Grupo != resultado.GrupoCorrecao {
			continue
		}
		originais := filepath.Join(raizDaSuite, tarefa.ID, "arquivos")
		comReferencia, err := montarReferencia(tarefa, originais, polyglot, temporario)
		if err != nil {
			return nil, err
		}
		entrada := oraculo.Entrada{ID: tarefa.ID, Linguagem: tarefa.Linguagem, Originais: originais, Protegidos: tarefa.ArquivosProtegidos}

		entrada.Solucao = comReferencia
		referencia, err := juiz.Julgar(ctx, entrada)
		if err != nil {
			return nil, err
		}
		entrada.Solucao = originais
		stub, err := juiz.Julgar(ctx, entrada)
		if err != nil {
			return nil, err
		}
		linhas = append(linhas, linhaDaValidacao{
			ID: tarefa.ID, Referencia: referencia.Oraculo, DetalheReferencia: referencia.Detalhe,
			Stub: stub.Oraculo, DetalheStub: stub.Detalhe,
			Confere: referencia.Oraculo == resultado.OraculoPassou && stub.Oraculo == resultado.OraculoFalhou,
		})
	}
	return linhas, nil
}

func montarReferencia(tarefa suite.Tarefa, originais, polyglot, temporario string) (string, error) {
	exercicio := strings.TrimPrefix(tarefa.ID, tarefa.Linguagem+"-")
	referencias, err := suite.ReferenciaDoExercicio(polyglot, tarefa.Linguagem, exercicio)
	if err != nil {
		return "", err
	}
	destino := filepath.Join(temporario, tarefa.ID+"-referencia")
	if err := os.RemoveAll(destino); err != nil {
		return "", fmt.Errorf("limpar %s: %w", destino, err)
	}
	if err := os.CopyFS(destino, os.DirFS(originais)); err != nil {
		return "", fmt.Errorf("copiar %s: %w", tarefa.ID, err)
	}
	for relativo, fonte := range referencias {
		conteudo, err := os.ReadFile(fonte) //nolint:gosec // G304: arquivo de referência do polyglot fixado por commit.
		if err != nil {
			return "", fmt.Errorf("ler a referência %s: %w", fonte, err)
		}
		caminho := filepath.Join(destino, filepath.FromSlash(relativo))
		if err := os.MkdirAll(filepath.Dir(caminho), 0o750); err != nil {
			return "", err
		}
		if err := os.WriteFile(caminho, conteudo, 0o600); err != nil { //nolint:gosec // G703: o destino é montado a partir do suite.json versionado.
			return "", fmt.Errorf("gravar a referência %s: %w", relativo, err)
		}
	}
	return destino, nil
}

func resumoDaValidacao(linhas []linhaDaValidacao) string {
	confirmam := 0
	var divergentes []string
	for _, linha := range linhas {
		if linha.Confere {
			confirmam++
			continue
		}
		divergentes = append(divergentes, fmt.Sprintf("%s: referência %s %s; stub %s %s",
			linha.ID, linha.Referencia, linha.DetalheReferencia, linha.Stub, linha.DetalheStub))
	}
	resumo := fmt.Sprintf("%d de %d tarefas confirmam o gabarito (a referência passa e o stub falha).\n", confirmam, len(linhas))
	for _, divergente := range divergentes {
		resumo += "- " + divergente + "\n"
	}
	return resumo
}

func validarOraculo(ctx context.Context, polyglot string, saida interface{ Write([]byte) (int, error) }) error {
	bruto, err := os.ReadFile(filepath.Join(diretorioDaSuite, "suite.json"))
	if err != nil {
		return fmt.Errorf("ler o suite.json: %w", err)
	}
	var montada suite.Suite
	if err := json.Unmarshal(bruto, &montada); err != nil {
		return fmt.Errorf("suite.json ilegível: %w", err)
	}
	temporario, err := os.MkdirTemp("/Users/Shared/aprova-bench", "validacao-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(temporario) }()

	juiz := oraculo.Oraculo{
		Publicador: oraculo.PublicadorDocker{Imagem: imagensLocais[sandbox.LinguagemGo]},
		Executor: func(montagens []string) sandbox.Executor {
			return sandbox.NovoExecutorLocal(sandbox.OpcoesExecutorLocal{ImagemPorLinguagem: imagensLocais, Montagens: montagens})
		},
		Temporario: temporario,
	}
	linhas, err := validarGabarito(ctx, montada, diretorioDaSuite, polyglot, juiz, temporario)
	if err != nil {
		return err
	}
	registro, err := json.MarshalIndent(linhas, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(arquivoDaValidacao), 0o750); err != nil {
		return err
	}
	//nolint:gosec // G703: o caminho é a constante arquivoDaValidacao.
	if err := os.WriteFile(arquivoDaValidacao, append(registro, '\n'), 0o600); err != nil {
		return err
	}
	_, err = fmt.Fprint(saida, resumoDaValidacao(linhas))
	return err
}

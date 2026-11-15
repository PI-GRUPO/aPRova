// Package oraculo diz se a solução de uma tarefa está certa de verdade: roda
// os testes originais, e não os do PR, contra a solução, pelo executor local
// do sandbox do aPRova, sem rede na etapa de testes.
package oraculo

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/benchmark/resultado"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/trabalho"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

const limiteDoDetalhe = 500

// Publicacao é onde o executor encontra o repositório do oráculo: a URL do
// clone e a montagem que a torna visível na etapa preparar.
type Publicacao struct {
	URL      string
	Montagem string
}

// Publicador torna um repositório local acessível ao executor. A função
// devolvida remove o que foi publicado.
type Publicador interface {
	Publicar(ctx context.Context, repositorio, nome string) (Publicacao, func(), error)
}

// Oraculo julga soluções. Executor constrói o executor do sandbox com as
// montagens da publicação; Temporario recebe as cópias.
type Oraculo struct {
	Publicador Publicador
	Executor   func(montagens []string) sandbox.Executor
	Temporario string
}

// Entrada é a solução a julgar: o diretório com o que o agente deixou, o
// diretório com os arquivos originais da tarefa e os testes protegidos.
type Entrada struct {
	ID         string
	Linguagem  string
	Solucao    string
	Originais  string
	Protegidos []string
}

// Veredito é o resultado do oráculo, com o resultado bruto do sandbox.
type Veredito struct {
	Oraculo resultado.Oraculo
	Detalhe string
	Sandbox sandbox.ResultadoSandbox
	Duracao time.Duration
}

// Julgar copia a solução sem caches e sem .git, restaura os testes
// originais, grava um commit, publica e roda o sandbox. Zero testes é "não
// rodou": os testes originais existem, então zero indica problema no oráculo.
func (o Oraculo) Julgar(ctx context.Context, entrada Entrada) (Veredito, error) {
	inicio := time.Now()
	repositorio, err := o.montarRepositorio(entrada)
	if err != nil {
		return Veredito{}, err
	}
	defer func() { _ = os.RemoveAll(filepath.Dir(repositorio.Diretorio)) }()

	publicacao, limpar, err := o.Publicador.Publicar(ctx, repositorio.Diretorio, entrada.ID)
	if err != nil {
		return Veredito{}, fmt.Errorf("oraculo: publicar a solução de %s: %w", entrada.ID, err)
	}
	defer limpar()

	executado, err := o.Executor([]string{publicacao.Montagem}).Executar(ctx, sandbox.PedidoExecucao{
		URLRepositorio: publicacao.URL,
		SHA:            repositorio.CommitBase,
		Linguagem:      sandbox.Linguagem(entrada.Linguagem),
		TaskRunID:      "oraculo-" + entrada.ID,
	})
	if err != nil {
		return Veredito{}, fmt.Errorf("oraculo: executar %s: %w", entrada.ID, err)
	}
	veredito := julgar(executado)
	veredito.Duracao = time.Since(inicio)
	return veredito, nil
}

func (o Oraculo) montarRepositorio(entrada Entrada) (trabalho.Area, error) {
	sufixo := make([]byte, 6)
	if _, err := rand.Read(sufixo); err != nil {
		return trabalho.Area{}, fmt.Errorf("oraculo: sortear o diretório: %w", err)
	}
	base := filepath.Join(o.Temporario, "oraculo-"+entrada.ID+"-"+hex.EncodeToString(sufixo))
	copia := filepath.Join(base, "copia")
	if err := copiarSemCaches(entrada.Solucao, copia); err != nil {
		return trabalho.Area{}, err
	}
	for _, protegido := range entrada.Protegidos {
		original, err := os.ReadFile(filepath.Join(entrada.Originais, filepath.FromSlash(protegido))) //nolint:gosec // G304: o caminho vem do suite.json versionado.
		if err != nil {
			return trabalho.Area{}, fmt.Errorf("oraculo: ler o teste original %s: %w", protegido, err)
		}
		if err := gravar(filepath.Join(copia, filepath.FromSlash(protegido)), original); err != nil {
			return trabalho.Area{}, err
		}
	}
	return trabalho.Preparar(copia, filepath.Join(base, "repositorio"))
}

func copiarSemCaches(origem, destino string) error {
	return filepath.WalkDir(origem, func(caminho string, entrada fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relativo, err := filepath.Rel(origem, caminho)
		if err != nil {
			return err
		}
		if relativo != "." && trabalho.EhCache(relativo) {
			if entrada.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entrada.Type().IsRegular() {
			return nil
		}
		conteudo, err := os.ReadFile(caminho) //nolint:gosec // G304: arquivo regular dentro da área de trabalho da tarefa.
		if err != nil {
			return fmt.Errorf("oraculo: ler %s: %w", relativo, err)
		}
		return gravar(filepath.Join(destino, relativo), conteudo)
	})
}

func julgar(executado sandbox.ResultadoSandbox) Veredito {
	veredito := Veredito{Sandbox: executado}
	switch {
	case executado.TipoFalha == sandbox.FalhaNenhuma && executado.TestesPassaram > 0:
		veredito.Oraculo = resultado.OraculoPassou
	case executado.TipoFalha == sandbox.FalhaTeste || executado.TipoFalha == sandbox.FalhaCompilacao:
		veredito.Oraculo = resultado.OraculoFalhou
	case executado.TipoFalha == sandbox.FalhaNenhuma:
		veredito.Oraculo = resultado.OraculoNaoRodou
		veredito.Detalhe = "nenhum teste original rodou"
	default:
		veredito.Oraculo = resultado.OraculoNaoRodou
		veredito.Detalhe = "infraestrutura: " + resumir(executado.SaidaBruta)
	}
	return veredito
}

func gravar(caminho string, conteudo []byte) error {
	if err := os.MkdirAll(filepath.Dir(caminho), 0o750); err != nil {
		return fmt.Errorf("oraculo: criar %s: %w", filepath.Dir(caminho), err)
	}
	if err := os.WriteFile(caminho, conteudo, 0o600); err != nil { //nolint:gosec // G703: os caminhos vêm do walk da área ou do suite.json versionado.
		return fmt.Errorf("oraculo: gravar %s: %w", caminho, err)
	}
	return nil
}

func resumir(texto string) string {
	texto = strings.TrimSpace(texto)
	if len(texto) > limiteDoDetalhe {
		return "…" + texto[len(texto)-limiteDoDetalhe:]
	}
	return texto
}

// Package trabalho prepara o diretório descartável de cada tarefa, um
// repositório git com o commit-base, e extrai dele a mudança do agente.
package trabalho

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

const (
	situacaoRemovido = "D"
	autorDoBenchmark = "aPRova benchmark"
	emailDoBenchmark = "benchmark@aprova.invalid"
)

var padroesExcluidos = []string{
	"/.gocache/", "/.gomodcache/", "__pycache__/", ".pytest_cache/", "/.m2/", "target/", "/.claude/",
}

// Area é o diretório de trabalho de uma tarefa e o commit-base gravado antes
// do agente rodar.
type Area struct {
	Diretorio  string
	CommitBase string
}

// Arquivo é um caminho alterado e a situação dele no git (A, M ou D).
type Arquivo struct {
	Caminho  string `json:"caminho"`
	Situacao string `json:"situacao"`
}

// Mudanca é o que o agente mudou em relação ao commit-base, sem os caches.
type Mudanca struct {
	Arquivos []Arquivo
	Patch    []byte
}

// Vazia diz se o agente não mudou nada.
func (m Mudanca) Vazia() bool {
	return len(m.Arquivos) == 0
}

// Caminhos devolve os caminhos alterados, ordenados.
func (m Mudanca) Caminhos() []string {
	caminhos := make([]string, 0, len(m.Arquivos))
	for _, arquivo := range m.Arquivos {
		caminhos = append(caminhos, arquivo.Caminho)
	}
	slices.Sort(caminhos)
	return caminhos
}

// Removidos devolve os caminhos apagados, ordenados.
func (m Mudanca) Removidos() []string {
	var removidos []string
	for _, arquivo := range m.Arquivos {
		if arquivo.Situacao == situacaoRemovido {
			removidos = append(removidos, arquivo.Caminho)
		}
	}
	slices.Sort(removidos)
	return removidos
}

// Preparar copia origem para destino, que não pode existir, e grava o
// commit-base com os caches excluídos. O git roda sem a configuração global
// nem os hooks do usuário.
func Preparar(origem, destino string) (Area, error) {
	if _, err := os.Lstat(destino); !errors.Is(err, fs.ErrNotExist) {
		return Area{}, fmt.Errorf("trabalho: o destino %s já existe", destino)
	}
	if err := os.CopyFS(destino, os.DirFS(origem)); err != nil {
		return Area{}, fmt.Errorf("trabalho: copiar a tarefa: %w", err)
	}
	area := Area{Diretorio: destino}
	ctx := context.Background()
	if _, err := area.git(ctx, "init", "-q", "-b", "base"); err != nil {
		return Area{}, err
	}
	exclusoes := strings.Join(padroesExcluidos, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(destino, ".git", "info", "exclude"), []byte(exclusoes), 0o600); err != nil {
		return Area{}, fmt.Errorf("trabalho: gravar as exclusões: %w", err)
	}
	if _, err := area.git(ctx, "add", "-A"); err != nil {
		return Area{}, err
	}
	if _, err := area.git(ctx, "commit", "-q", "-m", "base da tarefa"); err != nil {
		return Area{}, err
	}
	commit, err := area.git(ctx, "rev-parse", "HEAD")
	if err != nil {
		return Area{}, err
	}
	area.CommitBase = strings.TrimSpace(string(commit))
	return area, nil
}

// Diff devolve o que mudou em relação ao commit-base, sem os caches.
func (a Area) Diff(ctx context.Context) (Mudanca, error) {
	if _, err := a.git(ctx, "add", "-A"); err != nil {
		return Mudanca{}, err
	}
	situacoes, err := a.git(ctx, "diff", "--cached", "--name-status", "--no-renames", "-z", a.CommitBase)
	if err != nil {
		return Mudanca{}, err
	}
	patch, err := a.git(ctx, "diff", "--cached", "--binary", "--no-renames", a.CommitBase)
	if err != nil {
		return Mudanca{}, err
	}
	return Mudanca{Arquivos: interpretarSituacoes(situacoes), Patch: patch}, nil
}

// TestesAdulterados devolve os arquivos protegidos que o agente alterou ou
// apagou, comparando com as versões de origem.
func (a Area) TestesAdulterados(origem string, protegidos []string) ([]string, error) {
	var adulterados []string
	for _, protegido := range protegidos {
		original, err := os.ReadFile(filepath.Join(origem, filepath.FromSlash(protegido))) //nolint:gosec // G304: o caminho vem do suite.json versionado.
		if err != nil {
			return nil, fmt.Errorf("trabalho: ler o original de %s: %w", protegido, err)
		}
		atual := filepath.Join(a.Diretorio, filepath.FromSlash(protegido))
		informacao, err := os.Lstat(atual)
		if err != nil || !informacao.Mode().IsRegular() {
			adulterados = append(adulterados, protegido)
			continue
		}
		conteudo, err := os.ReadFile(atual) //nolint:gosec // G304: arquivo regular dentro da área de trabalho, conferido com Lstat.
		if err != nil || !bytes.Equal(conteudo, original) {
			adulterados = append(adulterados, protegido)
		}
	}
	slices.Sort(adulterados)
	return adulterados, nil
}

func (a Area) git(ctx context.Context, argumentos ...string) ([]byte, error) {
	completos := append([]string{"-c", "core.hooksPath=/dev/null", "-c", "commit.gpgsign=false", "-C", a.Diretorio}, argumentos...)
	comando := exec.CommandContext(ctx, "git", completos...) //nolint:gosec // G204: o binário é o git; os argumentos são subcomandos fixos deste pacote.
	comando.Env = ambienteDoGit()
	var saida, erros bytes.Buffer
	comando.Stdout, comando.Stderr = &saida, &erros
	if err := comando.Run(); err != nil {
		return nil, fmt.Errorf("trabalho: git %s: %w: %s", argumentos[0], err, strings.TrimSpace(erros.String()))
	}
	return saida.Bytes(), nil
}

func ambienteDoGit() []string {
	var ambiente []string
	for _, variavel := range os.Environ() {
		if !strings.HasPrefix(variavel, "GIT_") {
			ambiente = append(ambiente, variavel)
		}
	}
	return append(ambiente,
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME="+autorDoBenchmark, "GIT_AUTHOR_EMAIL="+emailDoBenchmark,
		"GIT_COMMITTER_NAME="+autorDoBenchmark, "GIT_COMMITTER_EMAIL="+emailDoBenchmark,
	)
}

func interpretarSituacoes(bruto []byte) []Arquivo {
	campos := strings.Split(strings.TrimRight(string(bruto), "\x00"), "\x00")
	var arquivos []Arquivo
	for i := 0; i+1 < len(campos); i += 2 {
		arquivos = append(arquivos, Arquivo{Situacao: campos[i], Caminho: campos[i+1]})
	}
	slices.SortFunc(arquivos, func(x, y Arquivo) int { return strings.Compare(x.Caminho, y.Caminho) })
	return arquivos
}

var diretoriosDeCache = []string{".git", ".gocache", ".gomodcache", "__pycache__", ".pytest_cache", ".m2", "target", ".claude"}

// EhCache diz se o caminho relativo está dentro de um diretório de cache, de
// build ou do git, que nunca faz parte da solução.
func EhCache(relativo string) bool {
	for _, componente := range strings.Split(filepath.ToSlash(relativo), "/") {
		if slices.Contains(diretoriosDeCache, componente) {
			return true
		}
	}
	return false
}

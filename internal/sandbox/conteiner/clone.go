package conteiner

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/PI-GRUPO/aPRova/internal/sandbox/processo"
)

var (
	// ErrClone indica que o repositório não pôde ser posicionado no SHA pedido.
	ErrClone = errors.New("sandbox: clone falhou")
	// ErrSHADivergente indica código cujo HEAD não é o SHA pedido.
	ErrSHADivergente = errors.New("sandbox: HEAD diverge do SHA pedido")
)

var semHooks = []string{"-c", "core.hooksPath=/dev/null"}

var ambienteDoGit = []string{
	"PATH=/usr/local/bin:/usr/bin:/bin",
	"HOME=/nonexistent",
	"GIT_TERMINAL_PROMPT=0",
	"GIT_CONFIG_NOSYSTEM=1",
	"GIT_CONFIG_GLOBAL=/dev/null",
	"GIT_LFS_SKIP_SMUDGE=1",
}

// Clonador obtém o repositório no SHA exato, sem nenhuma credencial.
type Clonador struct {
	Rodador processo.Rodador
}

// Clonar cria o destino, busca apenas o commit pedido e posiciona o checkout
// nele, sem hooks, submódulos nem LFS, conferindo depois que HEAD é exatamente
// o SHA pedido.
func (c Clonador) Clonar(ctx context.Context, repositorio, sha, destino string) error {
	etapas := [][]string{
		{"init", "-q", destino},
		{"-C", destino, "remote", "add", "origin", repositorio},
		{"-C", destino, "fetch", "-q", "--depth", "1", "--no-tags", "origin", sha},
		{"-C", destino, "checkout", "-q", "--detach", "FETCH_HEAD"},
	}

	for _, argumentos := range etapas {
		if _, err := c.git(ctx, argumentos...); err != nil {
			return err
		}
	}

	if err := c.Conferir(ctx, destino, sha); err != nil {
		return fmt.Errorf("%w: %w", ErrClone, err)
	}
	return nil
}

// Conferir confirma que o HEAD do repositório em diretorio é exatamente o SHA
// pedido. HEAD diferente devolve ErrSHADivergente.
func (c Clonador) Conferir(ctx context.Context, diretorio, sha string) error {
	head, err := c.git(ctx, "-C", diretorio, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if head != sha {
		return fmt.Errorf("%w: HEAD %s diverge do SHA pedido %s", ErrSHADivergente, head, sha)
	}
	return nil
}

func (c Clonador) git(ctx context.Context, argumentos ...string) (string, error) {
	execucao := c.Rodador.Rodar(ctx, processo.Comando{Binario: "git", Argumentos: append(append([]string{}, semHooks...), argumentos...), Ambiente: ambienteDoGit})

	etapa := "git " + argumentos[subcomandoDoGit(argumentos)]
	switch {
	case execucao.Erro != nil:
		return "", fmt.Errorf("%w: %s não pôde ser iniciado: %w", ErrClone, etapa, execucao.Erro)
	case execucao.PrazoEsgotado:
		return "", fmt.Errorf("%w: %s excedeu o prazo", ErrClone, etapa)
	case execucao.CodigoSaida != 0:
		return "", fmt.Errorf("%w: %s saiu com código %d: %s", ErrClone, etapa, execucao.CodigoSaida, strings.TrimSpace(string(execucao.Saida)))
	}

	return strings.TrimSpace(string(execucao.Saida)), nil
}

func subcomandoDoGit(argumentos []string) int {
	if argumentos[0] == "-C" {
		return 2
	}
	return 0
}

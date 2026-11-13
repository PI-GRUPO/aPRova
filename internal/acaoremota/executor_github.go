package acaoremota

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/PI-GRUPO/aPRova/internal/platform"
)

const (
	formatoDaAprovacaoPeloSlack = "Aprovado por @%s via Slack:\n\n%s"
	formatoDaAlteracaoPeloSlack = "Alteração solicitada por @%s via Slack:\n\n%s"
	tamanhoMaximoDoLogin        = 39
)

var reviewPublicadaPeloSlack = regexp.MustCompile(
	`^(?:Aprovado por @([A-Za-z0-9](?:-?[A-Za-z0-9])*) via Slack(?::\n\n(?s:.*))?|Alteração solicitada por @([A-Za-z0-9](?:-?[A-Za-z0-9])*) via Slack:\n\n(?s:.*))$`)

var (
	// ErrTipoSemExecucao indica ação autorizada para a qual não há efeito definido.
	ErrTipoSemExecucao = errors.New("acaoremota: tipo de ação sem execução definida")

	// ErrReanaliseIndisponivel indica que ainda não há worker para reanalisar.
	ErrReanaliseIndisponivel = errors.New("acaoremota: reanálise ainda não disponível")
)

// PublicadorDeReview publica uma review num pull request do GitHub.
type PublicadorDeReview interface {
	EnviarReview(ctx context.Context, installationID int64, dono, repositorio string, numero int, review platform.Review) error
}

// ExecutorNoGitHub produz o efeito das ações autorizadas no GitHub.
type ExecutorNoGitHub struct {
	publicador     PublicadorDeReview
	reenfileirador ReenfileiradorAnalise
}

// NovoExecutorNoGitHub constrói o executor com o publicador de reviews e o
// reenfileirador de análises.
func NovoExecutorNoGitHub(publicador PublicadorDeReview, reenfileirador ReenfileiradorAnalise) *ExecutorNoGitHub {
	return &ExecutorNoGitHub{publicador: publicador, reenfileirador: reenfileirador}
}

// Executar produz o efeito da ação autorizada: review de aprovação, review
// pedindo alteração ou reenfileiramento da análise.
func (e *ExecutorNoGitHub) Executar(ctx context.Context, acao AcaoAutorizada) error {
	switch acao.Tipo {
	case TipoAprovar:
		return e.publicar(ctx, acao, platform.Review{
			Evento: platform.EventoAprovar,
			Corpo:  fmt.Sprintf(formatoDaAprovacaoPeloSlack, acao.LoginGitHub, acao.Justificativa),
		})
	case TipoSolicitarAlteracao:
		return e.publicar(ctx, acao, platform.Review{
			Evento: platform.EventoSolicitarAlteracao,
			Corpo:  fmt.Sprintf(formatoDaAlteracaoPeloSlack, acao.LoginGitHub, acao.Justificativa),
		})
	case TipoReexecutarAnalise:
		return e.reenfileirador.Reenfileirar(ctx, acao)
	default:
		return fmt.Errorf("%w: %s", ErrTipoSemExecucao, acao.Tipo)
	}
}

func (e *ExecutorNoGitHub) publicar(ctx context.Context, acao AcaoAutorizada, review platform.Review) error {
	dono, repositorio := acao.donoERepositorio()
	return e.publicador.EnviarReview(ctx, acao.InstallationID, dono, repositorio, acao.NumeroPR, review)
}

// ReenfileiradorIndisponivel ocupa o lugar do worker enquanto ele não existe.
type ReenfileiradorIndisponivel struct{}

// Reenfileirar sempre devolve ErrReanaliseIndisponivel.
func (ReenfileiradorIndisponivel) Reenfileirar(context.Context, AcaoAutorizada) error {
	return ErrReanaliseIndisponivel
}

// QuemDecidiuNoSlack devolve o login do humano nomeado numa review que o
// próprio aPRova publicou a partir do Slack. Texto fora desse formato devolve
// doSlack igual a false.
func QuemDecidiuNoSlack(corpo string) (login string, doSlack bool) {
	partes := reviewPublicadaPeloSlack.FindStringSubmatch(corpo)
	if partes == nil {
		return "", false
	}
	login = partes[1] + partes[2]
	return login, len(login) <= tamanhoMaximoDoLogin
}

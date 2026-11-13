// Package worker conduz cada task_run criado pelo webhook pela máquina de
// estados até o fim, sobre uma fila no PostgreSQL com prazo de reivindicação.
// O motor orquestra: quem decide risco, assina recibo e publica no GitHub
// entra por interface.
package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/eventos"
	"github.com/Azulpasta/aPRova/internal/gate"
	"github.com/Azulpasta/aPRova/internal/platform"
	"github.com/Azulpasta/aPRova/internal/receipt"
	"github.com/Azulpasta/aPRova/internal/sandbox"
)

// Armazenamento é o que o motor e o pool usam do banco: a fila com prazo, as
// transições presas à reivindicação e o que cada etapa grava para poder ser
// retomada.
type Armazenamento interface {
	Reivindicar(ctx context.Context, trabalhador string, prazo time.Duration) (domain.TaskRun, bool, error)
	RenovarPrazo(ctx context.Context, taskRunID, trabalhador string, prazo time.Duration) error
	Liberar(ctx context.Context, taskRunID, trabalhador string, espera time.Duration) error
	TransicionarReivindicado(ctx context.Context, trabalhador string, transicao domain.Transicao) (domain.TaskRun, error)
	IncrementarTentativasDaEtapa(ctx context.Context, taskRunID, trabalhador string) (int, error)
	GravarResultadoDaTentativa(ctx context.Context, taskRunID string, tentativa int, resultado sandbox.ResultadoSandbox) error
	UltimoResultadoSandbox(ctx context.Context, taskRunID string) (int, sandbox.ResultadoSandbox, error)
	BuscarGateDeRisco(ctx context.Context, taskRunID string) (domain.GateDeRisco, error)
	BuscarReciboPorTaskRun(ctx context.Context, taskRunID string) (domain.Recibo, error)
	ProximaAprovacao(ctx context.Context, taskRunID string, depoisDe int64) (domain.AprovacaoRegistrada, error)
	AvancarCursorDeAprovacao(ctx context.Context, taskRunID, trabalhador string, ate int64) error
	ExisteTaskRunMaisNovo(ctx context.Context, taskRun domain.TaskRun) (bool, error)
}

// LeitorDePR lê no GitHub o que o pull request muda.
type LeitorDePR interface {
	ContextoDoPullRequest(ctx context.Context, installationID int64, dono, repositorio string, numero int) (platform.ContextoPR, error)
}

// ConsultaDePermissao devolve o papel de um login no repositório.
type ConsultaDePermissao interface {
	PermissaoNoRepositorio(ctx context.Context, installationID int64, dono, repositorio, login string) (platform.Permissao, error)
}

// AvaliadorRisco decide o nível de risco de um pull request. O gate.Gate é a
// implementação.
type AvaliadorRisco interface {
	Avaliar(ctx context.Context, entrada gate.Entrada) (gate.Resultado, error)
}

// EmissorRecibo grava a decisão do gate e o recibo assinado dela, os dois
// juntos ou nenhum. Decisão que já existe devolve domain.ErrDuplicado. O
// receipt.Emissor é a implementação.
type EmissorRecibo interface {
	Emitir(ctx context.Context, pedido receipt.Pedido) (receipt.Emitido, error)
}

// Publicacao é o resultado de um task_run a publicar no pull request: a
// decisão do gate, o resultado do sandbox em que ela se apoiou e o recibo
// assinado. AprovadoPor é o login de quem aprovou um pull request de risco
// alto; vazio enquanto ninguém aprovou.
type Publicacao struct {
	TaskRun     domain.TaskRun
	Gate        domain.GateDeRisco
	Sandbox     sandbox.ResultadoSandbox
	Recibo      receipt.Emitido
	AprovadoPor string
}

// PublicadorPR publica o Check Run e o comentário do task_run no pull
// request. Iniciar cria o Check Run em andamento; Publicar o conclui e escreve
// o comentário; PublicarFalha o conclui como failure, com o diagnóstico. Tudo é
// idempotente por task_run: repetir não cria um segundo Check Run nem um
// segundo comentário. Nunca faz merge.
type PublicadorPR interface {
	Iniciar(ctx context.Context, taskRun domain.TaskRun) error
	Publicar(ctx context.Context, publicacao Publicacao) error
	PublicarFalha(ctx context.Context, taskRun domain.TaskRun, diagnostico string) error
}

// EmissorEventos envia um evento ao n8n sem bloquear quem chama.
type EmissorEventos interface {
	Emitir(diario *slog.Logger, pedido eventos.Pedido)
}

// Desfecho diz ao pool o que fazer com o task_run depois de Conduzir:
// liberá-lo, e depois de quanto tempo, ou deixá-lo como está porque a
// reivindicação foi perdida.
type Desfecho struct {
	Liberar bool
	Espera  time.Duration
}

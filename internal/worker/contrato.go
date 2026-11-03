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
	"github.com/Azulpasta/aPRova/internal/platform"
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
	GravarGateDeRisco(ctx context.Context, gate domain.GateDeRisco) error
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

// Motivo é uma regra acionada pelo gate, com nome estável e descrição curta.
type Motivo struct {
	Nome      string
	Descricao string
}

// EntradaAvaliacao é o que o gate recebe: o task_run, o contexto do PR e o
// resultado do sandbox.
type EntradaAvaliacao struct {
	TaskRun  domain.TaskRun
	Contexto platform.ContextoPR
	Sandbox  sandbox.ResultadoSandbox
}

// Avaliacao é a decisão do gate: os níveis, os motivos e a explicação em
// português.
type Avaliacao struct {
	NivelDeterministico domain.NivelRisco
	Nivel               domain.NivelRisco
	Motivos             []Motivo
	Explicacao          string
}

// AvaliadorRisco decide o nível de risco de um pull request.
type AvaliadorRisco interface {
	Avaliar(ctx context.Context, entrada EntradaAvaliacao) (Avaliacao, error)
}

// PedidoRecibo é o que o recibo registra: o task_run, a avaliação e o
// resultado do sandbox.
type PedidoRecibo struct {
	TaskRun   domain.TaskRun
	Avaliacao Avaliacao
	Sandbox   sandbox.ResultadoSandbox
}

// ReciboEmitido identifica o recibo de um task_run e o SHA-256 do conteúdo
// assinado.
type ReciboEmitido struct {
	ID   string
	Hash []byte
}

// EmissorRecibo assina e grava o recibo do task_run. Existe um recibo por
// task_run: emitir de novo para o mesmo task_run devolve o mesmo recibo.
type EmissorRecibo interface {
	Emitir(ctx context.Context, pedido PedidoRecibo) (ReciboEmitido, error)
}

// Publicacao é o resultado de um task_run a publicar no pull request.
type Publicacao struct {
	TaskRun domain.TaskRun
	Gate    domain.GateDeRisco
	Recibo  ReciboEmitido
}

// PublicadorPR publica o Check Run e o comentário do task_run no pull
// request. É idempotente por task_run: publicar de novo o mesmo task_run não
// cria um segundo Check Run nem um segundo comentário. Nunca faz merge.
type PublicadorPR interface {
	Publicar(ctx context.Context, publicacao Publicacao) error
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

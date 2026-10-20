package acaoremota

import (
	"context"
	"errors"
	"time"

	"github.com/Azulpasta/aPRova/internal/platform"
)

// Tipo identifica a ação que o canal externo pede para executar.
type Tipo string

// Ações aceitas por canal externo.
const (
	TipoAprovar            Tipo = "aprovar"
	TipoSolicitarAlteracao Tipo = "solicitar_alteracao"
	TipoReexecutarAnalise  Tipo = "reexecutar_analise"
)

// Decisao é o desfecho de um pedido de ação remota.
type Decisao string

// Desfechos possíveis de um pedido de ação remota.
const (
	DecisaoAceita   Decisao = "aceita"
	DecisaoRecusada Decisao = "recusada"
	DecisaoInvalida Decisao = "invalida"
	DecisaoRepetida Decisao = "repetida"
	DecisaoFalha    Decisao = "falha"
)

// Motivo detalha a decisão no registro. Nunca é devolvido a quem pediu.
type Motivo string

// Motivos registrados junto de cada decisão.
const (
	MotivoAutorizada            Motivo = "autorizada"
	MotivoCorpoInvalido         Motivo = "corpo_invalido"
	MotivoSemVinculo            Motivo = "sem_vinculo"
	MotivoFalhaNaIdentidade     Motivo = "falha_na_identidade"
	MotivoFalhaNaVerificacao    Motivo = "falha_na_verificacao"
	MotivoPermissaoInsuficiente Motivo = "permissao_insuficiente"
	MotivoAutorDoProprioPR      Motivo = "autor_do_proprio_pr"
	MotivoAcaoRepetida          Motivo = "acao_repetida"
	MotivoFalhaNaReserva        Motivo = "falha_na_reserva"
	MotivoFalhaNaExecucao       Motivo = "falha_na_execucao"
)

// Pedido é o corpo enviado pelo canal externo. Diz apenas quem é o usuário no
// canal; o login no GitHub é resolvido pelo núcleo.
type Pedido struct {
	IDAcao         string `json:"id_acao" validate:"required,max=200"`
	Tipo           Tipo   `json:"tipo" validate:"required,oneof=aprovar solicitar_alteracao reexecutar_analise"`
	UsuarioSlack   string `json:"usuario_slack" validate:"required,max=100"`
	Repositorio    string `json:"repositorio" validate:"required,repositorio_github"`
	NumeroPR       int    `json:"numero_pr" validate:"required,gt=0"`
	InstallationID int64  `json:"installation_id" validate:"required,gt=0"`
	Justificativa  string `json:"justificativa" validate:"required_if=Tipo solicitar_alteracao,max=4000"`
}

// AcaoAutorizada é um pedido que passou pela autorização, acompanhado do login
// do humano que decidiu.
type AcaoAutorizada struct {
	Pedido
	LoginGitHub string
}

// Tentativa é o registro de um pedido autenticado, aceito ou recusado.
type Tentativa struct {
	IDAcao         string
	Tipo           Tipo
	UsuarioSlack   string
	LoginGitHub    string
	Repositorio    string
	NumeroPR       int
	InstallationID int64
	Decisao        Decisao
	Motivo         Motivo
	Em             time.Time
}

// ResolvedorIdentidade traduz o usuário do canal externo para o login no GitHub.
type ResolvedorIdentidade interface {
	LoginGitHub(ctx context.Context, usuarioSlack string) (string, error)
}

// ConsultorPermissao informa o papel de um login num repositório.
type ConsultorPermissao interface {
	PermissaoNoRepositorio(ctx context.Context, installationID int64, dono, repositorio, login string) (platform.Permissao, error)
}

// ConsultorAutoria informa o login de quem abriu o pull request.
type ConsultorAutoria interface {
	AutorDoPullRequest(ctx context.Context, installationID int64, dono, repositorio string, numero int) (string, error)
}

// ExecutorAcao produz o efeito de uma ação já autorizada.
type ExecutorAcao interface {
	Executar(ctx context.Context, acao AcaoAutorizada) error
}

// ReenfileiradorAnalise devolve um pull request à fila de análise.
type ReenfileiradorAnalise interface {
	Reenfileirar(ctx context.Context, acao AcaoAutorizada) error
}

// RegistroTentativas guarda cada tentativa de ação remota e garante que um
// id_acao seja executado uma única vez. Reservar grava a tentativa como aceita
// antes da execução e devolve ErrAcaoJaAceita quando já existe outra aceita
// para o mesmo id_acao. RegistrarFalhaNaExecucao passa a tentativa reservada
// para falha, o que libera o id_acao para uma nova tentativa. Registrar grava
// as demais decisões e recusa a aceita com ErrDecisaoExigeReserva.
type RegistroTentativas interface {
	Registrar(ctx context.Context, tentativa Tentativa) error
	Reservar(ctx context.Context, tentativa Tentativa) (Reserva, error)
	RegistrarFalhaNaExecucao(ctx context.Context, reserva Reserva, motivo Motivo) error
}

// Reserva identifica a tentativa gravada como aceita antes da execução.
type Reserva struct {
	ID int64
}

var (
	// ErrAcaoJaAceita indica que o id_acao já tem uma tentativa aceita.
	ErrAcaoJaAceita = errors.New("acaoremota: id_acao já tem uma tentativa aceita")

	// ErrDecisaoExigeReserva indica tentativa aceita gravada sem passar pela
	// reserva.
	ErrDecisaoExigeReserva = errors.New("acaoremota: tentativa aceita só é gravada pela reserva")

	// ErrReservaInexistente indica reserva que não existe ou já não está aceita.
	ErrReservaInexistente = errors.New("acaoremota: reserva inexistente ou já encerrada")
)

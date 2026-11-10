package receipt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/sandbox"
	"github.com/google/uuid"
)

// ErrEmissorIncompleto indica emissor montado sem assinador ou sem
// armazenamento.
var ErrEmissorIncompleto = errors.New("receipt: emissor sem assinador ou sem armazenamento")

// Pedido é o que o recibo registra: o task_run analisado, a decisão do gate,
// o resultado do sandbox com a tentativa que o produziu e o diff analisado.
type Pedido struct {
	TaskRun      domain.TaskRun
	Gate         domain.GateDeRisco
	Sandbox      sandbox.ResultadoSandbox
	Tentativa    int
	Diff         string
	DiffTruncado bool
}

// Emitido identifica o recibo de um task_run e o SHA-256 do conteúdo
// assinado.
type Emitido struct {
	ID   string
	Hash []byte
}

// Armazenamento grava a decisão do gate e o recibo dela numa só transação,
// entregando a quem monta o recibo o último recibo do mesmo tenant.
type Armazenamento interface {
	GravarDecisaoComRecibo(ctx context.Context, gate domain.GateDeRisco, montar func(anterior *domain.Recibo) (domain.NovoRecibo, error)) (domain.Recibo, error)
}

// OpcoesEmissor reúne as dependências do emissor. Sem Agora, usa time.Now.
type OpcoesEmissor struct {
	Assinador     *Assinador
	Armazenamento Armazenamento
	Agora         func() time.Time
}

// Emissor assina o recibo de cada decisão do gate e o grava junto com ela.
type Emissor struct {
	assinador     *Assinador
	armazenamento Armazenamento
	agora         func() time.Time
}

// NovoEmissor constrói o emissor. Sem assinador ou sem armazenamento devolve
// ErrEmissorIncompleto.
func NovoEmissor(opcoes OpcoesEmissor) (*Emissor, error) {
	if opcoes.Assinador == nil || opcoes.Armazenamento == nil {
		return nil, ErrEmissorIncompleto
	}
	if opcoes.Agora == nil {
		opcoes.Agora = time.Now
	}
	return &Emissor{assinador: opcoes.Assinador, armazenamento: opcoes.Armazenamento, agora: opcoes.Agora}, nil
}

// Emitir grava a decisão do gate e o recibo assinado dela, encadeado no último
// recibo do mesmo tenant. Os dois são gravados juntos ou nenhum é. Decisão que
// já existe devolve domain.ErrDuplicado.
func (e *Emissor) Emitir(ctx context.Context, pedido Pedido) (Emitido, error) {
	gravado, err := e.armazenamento.GravarDecisaoComRecibo(ctx, pedido.Gate, func(anterior *domain.Recibo) (domain.NovoRecibo, error) {
		return e.assinar(pedido, anterior)
	})
	if err != nil {
		return Emitido{}, fmt.Errorf("emitir o recibo do task_run %s: %w", pedido.TaskRun.ID, err)
	}
	return Emitido{ID: gravado.ID, Hash: gravado.Hash}, nil
}

func (e *Emissor) assinar(pedido Pedido, anterior *domain.Recibo) (domain.NovoRecibo, error) {
	emitidoEm := e.agora().UTC().Truncate(time.Microsecond)
	conteudo := e.conteudoDe(pedido, emitidoEm)

	novo := domain.NovoRecibo{
		ID:         conteudo.ID,
		TaskRunID:  pedido.TaskRun.ID,
		IDChave:    e.assinador.IDChave(),
		AssinadoEm: emitidoEm,
	}
	if anterior != nil {
		conteudo.Anterior = &Elo{ID: anterior.ID, Hash: HashEmHex([]byte(anterior.Conteudo))}
		novo.ReciboAnteriorID = &anterior.ID
	}

	bruto, err := conteudo.Serializar()
	if err != nil {
		return domain.NovoRecibo{}, err
	}
	novo.Conteudo = string(bruto)
	novo.Hash = Hash(bruto)
	novo.Assinatura = e.assinador.Assinar(bruto)
	return novo, nil
}

func (e *Emissor) conteudoDe(pedido Pedido, emitidoEm time.Time) Conteudo {
	motivos := make([]Motivo, 0, len(pedido.Gate.Motivos))
	for _, motivo := range pedido.Gate.Motivos {
		motivos = append(motivos, Motivo{ID: motivo.ID, Evidencias: motivo.Evidencias})
	}

	return Conteudo{
		Versao:       VersaoDoFormato,
		ID:           uuid.NewString(),
		TaskRunID:    pedido.TaskRun.ID,
		TenantID:     pedido.TaskRun.TenantID,
		Repositorio:  pedido.TaskRun.Repositorio,
		NumeroPR:     pedido.TaskRun.NumeroPR,
		SHAHead:      pedido.TaskRun.SHAHead,
		DiffSHA256:   HashEmHex([]byte(pedido.Diff)),
		DiffTruncado: pedido.DiffTruncado,
		Sandbox: Sandbox{
			Compilou:       pedido.Sandbox.Compilou,
			TestesPassaram: pedido.Sandbox.TestesPassaram,
			TestesFalharam: pedido.Sandbox.TestesFalharam,
			TipoFalha:      string(pedido.Sandbox.TipoFalha),
			Tentativa:      pedido.Tentativa,
		},
		NivelDeterministico: pedido.Gate.NivelDeterministico,
		Nivel:               pedido.Gate.Nivel,
		Motivos:             motivos,
		EmitidoEm:           emitidoEm,
		KeyID:               e.assinador.IDChave(),
	}
}

package worker

import (
	"context"
	"crypto/sha256"
	"log/slog"

	"github.com/Azulpasta/aPRova/internal/domain"
)

const motivoDoGateProvisorio = "gate de risco ainda não implementado"

// AvaliadorRiscoProvisorio ocupa o lugar do gate de risco enquanto ele não
// existe: classifica todo pull request como risco alto. Na dúvida, o sistema
// bloqueia e espera um humano; nunca libera.
type AvaliadorRiscoProvisorio struct{}

// Avaliar devolve sempre risco alto, com o motivo de que o gate ainda não foi
// implementado.
func (AvaliadorRiscoProvisorio) Avaliar(context.Context, EntradaAvaliacao) (Avaliacao, error) {
	return Avaliacao{
		NivelDeterministico: domain.RiscoAlto,
		Nivel:               domain.RiscoAlto,
		Motivos:             []Motivo{{Nome: "gate_provisorio", Descricao: motivoDoGateProvisorio}},
		Explicacao:          "O gate de risco ainda não foi implementado. Por segurança, todo pull request fica bloqueado até a revisão de alguém com permissão de escrita.",
	}, nil
}

// EmissorReciboProvisorio ocupa o lugar do recibo assinado enquanto ele não
// existe: registra no log o que emitiria e não assina nem grava nada. O
// recibo devolvido usa o id do task_run e o SHA-256 de um texto fixo dele, de
// modo que o mesmo task_run tem sempre o mesmo recibo.
type EmissorReciboProvisorio struct {
	diario *slog.Logger
}

// NovoEmissorReciboProvisorio constrói o emissor provisório sobre o diário.
func NovoEmissorReciboProvisorio(diario *slog.Logger) *EmissorReciboProvisorio {
	return &EmissorReciboProvisorio{diario: diario}
}

// Emitir registra o recibo que seria emitido e devolve o recibo provisório.
func (e *EmissorReciboProvisorio) Emitir(_ context.Context, pedido PedidoRecibo) (ReciboEmitido, error) {
	hash := sha256.Sum256([]byte("recibo provisório do task_run " + pedido.TaskRun.ID))
	e.diario.Warn("emissor de recibo provisório: nada foi assinado nem gravado",
		"task_run_id", pedido.TaskRun.ID, "nivel_risco", pedido.Avaliacao.Nivel, "tipo_falha_sandbox", pedido.Sandbox.TipoFalha)
	return ReciboEmitido{ID: pedido.TaskRun.ID, Hash: hash[:]}, nil
}

// PublicadorPRProvisorio ocupa o lugar da publicação no GitHub enquanto ela
// não existe: registra no log o que publicaria e não chama nada externo.
type PublicadorPRProvisorio struct {
	diario *slog.Logger
}

// NovoPublicadorPRProvisorio constrói o publicador provisório sobre o diário.
func NovoPublicadorPRProvisorio(diario *slog.Logger) *PublicadorPRProvisorio {
	return &PublicadorPRProvisorio{diario: diario}
}

// Publicar registra o Check Run e o comentário que seriam publicados.
func (p *PublicadorPRProvisorio) Publicar(_ context.Context, publicacao Publicacao) error {
	p.diario.Warn("publicador provisório: nada foi publicado no GitHub",
		"task_run_id", publicacao.TaskRun.ID, "repositorio", publicacao.TaskRun.Repositorio,
		"numero_pr", publicacao.TaskRun.NumeroPR, "nivel_risco", publicacao.Gate.Nivel, "recibo", publicacao.Recibo.ID)
	return nil
}

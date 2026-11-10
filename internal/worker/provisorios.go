package worker

import (
	"context"
	"log/slog"
)

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

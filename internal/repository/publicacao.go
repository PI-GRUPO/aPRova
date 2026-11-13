package repository

import (
	"context"
	"fmt"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/repository/db"
)

// CheckRunDoTaskRun devolve o id do Check Run publicado para o task_run.
// Task_run sem Check Run gravado devolve domain.ErrNaoEncontrado.
func (r *RepositorioPostgres) CheckRunDoTaskRun(ctx context.Context, taskRunID string) (int64, error) {
	id, err := r.consultas.GetCheckRunID(ctx, taskRunID)
	if err != nil {
		return 0, traduzir("buscar o Check Run do task_run", err)
	}
	if id == nil {
		return 0, fmt.Errorf("buscar o Check Run do task_run %s: %w", taskRunID, domain.ErrNaoEncontrado)
	}
	return *id, nil
}

// GravarCheckRun grava o id do Check Run do task_run. Task_run inexistente
// devolve domain.ErrNaoEncontrado.
func (r *RepositorioPostgres) GravarCheckRun(ctx context.Context, taskRunID string, checkRunID int64) error {
	alteradas, err := r.consultas.SetCheckRunID(ctx, db.SetCheckRunIDParams{ID: taskRunID, CheckRunID: &checkRunID})
	if err != nil {
		return traduzir("gravar o Check Run do task_run", err)
	}
	if alteradas == 0 {
		return fmt.Errorf("gravar o Check Run do task_run %s: %w", taskRunID, domain.ErrNaoEncontrado)
	}
	return nil
}

// ComentarioDoPR devolve o id do comentário do aPRova no pull request. Pull
// request ainda sem comentário devolve domain.ErrNaoEncontrado.
func (r *RepositorioPostgres) ComentarioDoPR(ctx context.Context, tenantID int64, repositorio string, numeroPR int) (int64, error) {
	numero, err := numeroDePR(numeroPR)
	if err != nil {
		return 0, err
	}
	id, err := r.consultas.GetPullRequestComment(ctx, db.GetPullRequestCommentParams{TenantID: tenantID, Repository: repositorio, PrNumber: numero})
	return id, traduzir("buscar o comentário do pull request", err)
}

// GravarComentario grava o id do comentário do aPRova no pull request,
// substituindo o anterior: cada pull request tem um comentário só.
func (r *RepositorioPostgres) GravarComentario(ctx context.Context, tenantID int64, repositorio string, numeroPR int, comentarioID int64) error {
	numero, err := numeroDePR(numeroPR)
	if err != nil {
		return err
	}
	err = r.consultas.UpsertPullRequestComment(ctx, db.UpsertPullRequestCommentParams{
		TenantID: tenantID, Repository: repositorio, PrNumber: numero, CommentID: comentarioID,
	})
	return traduzir("gravar o comentário do pull request", err)
}

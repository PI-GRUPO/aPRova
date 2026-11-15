package repository

import (
	"context"
	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/repository/db"
)

// ListarAnalises usa exclusivamente SELECT, sem reivindicar tarefas da fila.
func (r *RepositorioPostgres) ListarAnalises(ctx context.Context, f domain.FiltroAnalises) ([]domain.TaskRun, error) {
	if err := validar(f); err != nil {
		return nil, err
	}
	linhas, err := r.consultas.ListAnalyses(ctx, db.ListAnalysesParams{
		TenantID: f.TenantID, Repository: f.Repositorio, PrNumber: f.NumeroPR,
		StateFilter: f.Estado, PageLimit: f.Limite, PageOffset: f.Deslocamento,
	})
	if err != nil {
		return nil, traduzir("consultar análises", err)
	}
	resultado := make([]domain.TaskRun, 0, len(linhas))
	for _, linha := range linhas {
		resultado = append(resultado, paraTaskRun(linha))
	}
	return resultado, nil
}

// ConsultarAnalise impede a leitura de um ID pertencente a outra instalação.
func (r *RepositorioPostgres) ConsultarAnalise(ctx context.Context, tenantID int64, repositorio, id string) (domain.TaskRun, error) {
	if tenantID <= 0 {
		return domain.TaskRun{}, domain.ErrDadosInvalidos
	}
	linha, err := r.consultas.GetScopedAnalysis(ctx, db.GetScopedAnalysisParams{ID: id, TenantID: tenantID, Repository: repositorio})
	if err != nil {
		return domain.TaskRun{}, traduzir("consultar análise", err)
	}
	return paraTaskRun(linha), nil
}

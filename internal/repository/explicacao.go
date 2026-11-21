package repository

import (
	"context"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/repository/db"
)

const tamanhoMaximoDoMotivoDaExplicacao = 1000

// ExplicacaoDoTaskRun devolve a tentativa de explicação gravada para o
// task_run, ou domain.ErrNaoEncontrado.
func (r *RepositorioPostgres) ExplicacaoDoTaskRun(ctx context.Context, taskRunID string) (domain.Explicacao, error) {
	linha, err := r.consultas.GetRiskExplanation(ctx, taskRunID)
	if err != nil {
		return domain.Explicacao{}, traduzir("ler explicação do task_run", err)
	}
	return domain.Explicacao{
		TaskRunID: linha.TaskRunID, Resultado: domain.ResultadoDaExplicacao(linha.Result),
		Texto: linha.Text, Modelo: linha.Model, Motivo: linha.Reason,
	}, nil
}

// GravarExplicacao grava a única tentativa de explicação do task_run. Se já
// existe uma, ela fica como está. O motivo é cortado em 1000 caracteres.
func (r *RepositorioPostgres) GravarExplicacao(ctx context.Context, explicacao domain.Explicacao) error {
	motivo := []rune(explicacao.Motivo)
	err := r.consultas.CreateRiskExplanation(ctx, db.CreateRiskExplanationParams{
		TaskRunID: explicacao.TaskRunID, Result: string(explicacao.Resultado), Text: explicacao.Texto,
		Model: explicacao.Modelo, Reason: string(motivo[:min(len(motivo), tamanhoMaximoDoMotivoDaExplicacao)]),
	})
	return traduzir("gravar explicação do task_run", err)
}

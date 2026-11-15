package repository

import (
	"context"
	"fmt"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/repository/db"
)

// EnvioSARIFDoTaskRun devolve o registro do envio do SARIF do task_run.
// Task_run sem envio registrado devolve domain.ErrNaoEncontrado.
func (r *RepositorioPostgres) EnvioSARIFDoTaskRun(ctx context.Context, taskRunID string) (domain.EnvioSARIF, error) {
	linha, err := r.consultas.GetSarifUpload(ctx, taskRunID)
	if err != nil {
		return domain.EnvioSARIF{}, traduzir("buscar o envio do SARIF", err)
	}

	return domain.EnvioSARIF{
		TaskRunID: linha.TaskRunID, IDDoEnvio: valorOuVazio(linha.SarifID), Estado: domain.EstadoEnvioSARIF(linha.Status), Detalhe: linha.Detail,
	}, nil
}

// GravarEnvioSARIF grava ou atualiza o registro do envio do SARIF do task_run.
// O id de um envio aceito nunca é trocado nem apagado: a tentativa devolve
// domain.ErrDuplicado. Task_run inexistente devolve domain.ErrNaoEncontrado.
func (r *RepositorioPostgres) GravarEnvioSARIF(ctx context.Context, envio domain.EnvioSARIF) error {
	alteradas, err := r.consultas.UpsertSarifUpload(ctx, db.UpsertSarifUploadParams{
		TaskRunID: envio.TaskRunID, SarifID: textoOuNulo(envio.IDDoEnvio), Status: string(envio.Estado), Detail: envio.Detalhe,
	})
	if err != nil {
		return traduzir("gravar o envio do SARIF", err)
	}
	if alteradas > 0 {
		return nil
	}

	if _, err := r.EnvioSARIFDoTaskRun(ctx, envio.TaskRunID); err != nil {
		return fmt.Errorf("gravar o envio do SARIF do task_run %s: %w", envio.TaskRunID, err)
	}
	return fmt.Errorf("gravar o envio do SARIF do task_run %s: já existe um envio aceito: %w", envio.TaskRunID, domain.ErrDuplicado)
}

// EnvioSARIFDesligado diz se o envio de SARIF foi desligado para o
// repositório do tenant.
func (r *RepositorioPostgres) EnvioSARIFDesligado(ctx context.Context, tenantID int64, repositorio string) (bool, error) {
	desligado, err := r.consultas.IsSarifDisabled(ctx, db.IsSarifDisabledParams{TenantID: tenantID, Repository: repositorio})
	return desligado, traduzir("consultar o desligamento do SARIF", err)
}

// DesligarEnvioSARIF desliga o envio de SARIF para o repositório do tenant,
// com o motivo. Repetir não é erro e mantém o primeiro motivo.
func (r *RepositorioPostgres) DesligarEnvioSARIF(ctx context.Context, tenantID int64, repositorio, motivo string) error {
	err := r.consultas.DisableSarif(ctx, db.DisableSarifParams{TenantID: tenantID, Repository: repositorio, Reason: motivo})
	return traduzir("desligar o envio do SARIF", err)
}

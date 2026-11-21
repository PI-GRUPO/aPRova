package repository

import (
	"context"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/repository/db"
)

// RegistrarFechamento grava o desfecho do pull request fechado e, quando ele
// reverte outro pull request do mesmo repositório, a reversão. O tenant é a
// instalação. Reentrega não duplica nada, e um fechamento mais antigo que o
// gravado não o substitui.
func (r *RepositorioPostgres) RegistrarFechamento(ctx context.Context, fechamento domain.FechamentoDePR) error {
	if err := validar(fechamento); err != nil {
		return err
	}
	numero, err := numeroDePR(fechamento.NumeroPR)
	if err != nil {
		return err
	}
	commits, err := commitsDoPR(fechamento.Commits)
	if err != nil {
		return err
	}

	return r.EmTransacao(ctx, func(tx *RepositorioPostgres) error {
		err := tx.consultas.UpsertPullRequestOutcome(ctx, db.UpsertPullRequestOutcomeParams{
			TenantID: fechamento.InstallationID, Repository: fechamento.Repositorio, PrNumber: numero,
			Merged: fechamento.Mergeado, HeadSha: fechamento.SHAHead, MergeCommitSha: textoOuNulo(fechamento.SHADoMerge),
			Commits: commits, ClosedAt: fechamento.FechadoEm, DeliveryID: fechamento.DeliveryID,
		})
		if err != nil {
			return traduzir("gravar o desfecho do pull request", err)
		}
		if fechamento.Reverte == 0 {
			return nil
		}

		revertido, err := numeroDePR(fechamento.Reverte)
		if err != nil {
			return err
		}
		err = tx.consultas.CreatePullRequestRevert(ctx, db.CreatePullRequestRevertParams{
			TenantID: fechamento.InstallationID, Repository: fechamento.Repositorio, PrNumber: revertido,
			RevertedByPr: numero, RevertedAt: fechamento.FechadoEm, DeliveryID: fechamento.DeliveryID,
		})
		return traduzir("gravar a reversão do pull request", err)
	})
}

// DesfechoDoPR devolve o desfecho gravado do pull request do tenant, ou
// domain.ErrNaoEncontrado quando ele ainda não foi fechado.
func (r *RepositorioPostgres) DesfechoDoPR(ctx context.Context, tenantID int64, repositorio string, numeroPR int) (domain.DesfechoDePR, error) {
	numero, err := numeroDePR(numeroPR)
	if err != nil {
		return domain.DesfechoDePR{}, err
	}

	linha, err := r.consultas.GetPullRequestOutcome(ctx, db.GetPullRequestOutcomeParams{
		TenantID: tenantID, Repository: repositorio, PrNumber: numero,
	})
	if err != nil {
		return domain.DesfechoDePR{}, traduzir("ler o desfecho do pull request", err)
	}
	return domain.DesfechoDePR{
		Mergeado: linha.Merged, SHAHead: linha.HeadSha, SHADoMerge: valorOuVazio(linha.MergeCommitSha), Commits: int(linha.Commits),
	}, nil
}

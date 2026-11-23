package repository

import (
	"context"
	"errors"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/acaoremota"
	"github.com/PI-GRUPO/aPRova/internal/domain"
)

// ValidarAcaoPainel lê a análise atual e confere seu estado e commit.
func (r *RepositorioPostgres) ValidarAcaoPainel(ctx context.Context, pedido acaoremota.Pedido) error {
	var compativel bool
	err := r.conexao.QueryRow(ctx, `
		SELECT EXISTS (
		    SELECT 1 FROM task_runs t
		    WHERE (t.id, t.tenant_id, t.repository, t.pr_number, t.head_sha) = ($1, $2, $3, $4, $5)
		      AND NOT EXISTS (
		          SELECT 1 FROM task_runs novo
		          WHERE (novo.tenant_id, novo.repository, novo.pr_number) = (t.tenant_id, t.repository, t.pr_number)
		            AND (novo.created_at, novo.id) > (t.created_at, t.id)
		      )
		      AND CASE WHEN $6::boolean THEN
		          t.state = 'falhou' AND (t.claimed_by IS NULL OR t.lease_expires_at <= now())
		          AND (SELECT s.failure_type FROM sandbox_results s WHERE s.task_run_id = t.id ORDER BY s.attempt DESC LIMIT 1) = 'infraestrutura'
		          AND NOT EXISTS (SELECT 1 FROM risk_gates g WHERE g.task_run_id = t.id)
		      ELSE
		          t.state = 'aguardando_aprovacao' AND EXISTS (SELECT 1 FROM risk_gates g WHERE g.task_run_id = t.id AND g.level = 'alto')
		      END
		)`, pedido.TaskRunID, pedido.InstallationID, pedido.Repositorio, pedido.NumeroPR, pedido.CommitSHA, pedido.Tipo == acaoremota.TipoReexecutarAnalise).Scan(&compativel)
	if err != nil {
		return traduzir("validar o alvo da ação do painel", err)
	}
	if !compativel {
		return acaoremota.ErrAnaliseIncompativel
	}
	return nil
}

// Reenfileirar cria uma análise nova para o mesmo commit e preserva as tentativas da anterior.
func (r *RepositorioPostgres) Reenfileirar(ctx context.Context, acao acaoremota.AcaoAutorizada) error {
	if acao.Origem != acaoremota.OrigemPainel {
		return acaoremota.ErrReanaliseIndisponivel
	}
	return r.EmTransacao(ctx, func(tx *RepositorioPostgres) error {
		// A análise original é a trava para pedidos concorrentes de reexecução.
		var originalID string
		err := tx.conexao.QueryRow(ctx, `SELECT id FROM task_runs WHERE id = $1 AND tenant_id = $2 FOR UPDATE`,
			acao.TaskRunID, acao.InstallationID).Scan(&originalID)
		if err != nil {
			err = traduzir("travar a análise a reexecutar", err)
			if errors.Is(err, domain.ErrNaoEncontrado) {
				return acaoremota.ErrAnaliseIncompativel
			}
			return err
		}
		if err := tx.ValidarAcaoPainel(ctx, acao.Pedido); err != nil {
			return err
		}
		original, err := tx.BuscarTaskRun(ctx, originalID)
		if err != nil {
			return err
		}
		nova, criada, err := tx.CriarTaskRun(ctx, domain.NovoTaskRun{
			TenantID: original.TenantID, DeliveryID: acao.IDAcao, Repositorio: original.Repositorio,
			NumeroPR: original.NumeroPR, SHAHead: original.SHAHead, Autor: original.Autor,
			Acao: "reexecutar_analise", RecebidoEm: time.Now().UTC(),
		})
		if err != nil {
			return err
		}
		if !criada {
			return acaoremota.ErrAnaliseIncompativel
		}
		check, err := tx.CheckRunDoTaskRun(ctx, original.ID)
		if errors.Is(err, domain.ErrNaoEncontrado) {
			return nil
		}
		if err != nil {
			return err
		}
		return tx.GravarCheckRun(ctx, nova.ID, check)
	})
}

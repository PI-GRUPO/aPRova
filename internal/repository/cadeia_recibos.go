package repository

import (
	"context"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/repository/db"
)

// ListarRecibosDaInstalacao consulta o histórico e verifica seu escopo no mesmo
// SELECT, de modo que uma emissão simultânea não possa alterar o escopo da leitura.
func (r *RepositorioPostgres) ListarRecibosDaInstalacao(ctx context.Context, instalacao int64, repositorios []string) ([]domain.Recibo, error) {
	if instalacao <= 0 || len(repositorios) == 0 {
		return nil, domain.ErrCadeiaSemAcesso
	}
	rows, err := r.conexao.Query(ctx, `
		SELECT r.id, r.tenant_id, r.task_run_id, r.previous_receipt_id, r.payload,
		       r.content_hash, r.signature, r.key_id, r.signed_at, r.created_at,
		       COALESCE(t.repository = ANY($2::text[]), false) AS autorizado
		FROM receipts r
		LEFT JOIN task_runs t ON (t.tenant_id, t.id) = (r.tenant_id, r.task_run_id)
		WHERE r.tenant_id = $1
		ORDER BY autorizado, r.signed_at, r.id`, instalacao, repositorios)
	if err != nil {
		return nil, traduzir("consultar cadeia de recibos", err)
	}
	defer rows.Close()
	var recibos []domain.Recibo
	for rows.Next() {
		var linha db.Receipt
		var autorizado bool
		if err := rows.Scan(&linha.ID, &linha.TenantID, &linha.TaskRunID, &linha.PreviousReceiptID,
			&linha.Payload, &linha.ContentHash, &linha.Signature, &linha.KeyID, &linha.SignedAt, &linha.CreatedAt, &autorizado); err != nil {
			return nil, traduzir("ler recibo da cadeia", err)
		}
		if !autorizado {
			return nil, domain.ErrCadeiaSemAcesso
		}
		recibos = append(recibos, paraRecibo(linha))
	}
	if err := rows.Err(); err != nil {
		return nil, traduzir("ler cadeia de recibos", err)
	}
	return recibos, nil
}

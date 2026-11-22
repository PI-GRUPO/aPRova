package repository

import (
	"context"
	_ "embed"
	"encoding/json"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/jackc/pgx/v5"
)

//go:embed painelsql/metricas.sql
var sqlMetricas string

//go:embed painelsql/recibos.sql
var sqlRecibos string

//go:embed painelsql/auditoria.sql
var sqlAuditoria string

//go:embed painelsql/regras.sql
var sqlRegras string

var consultasPainel = map[string]string{"metricas": sqlMetricas, "recibos": sqlRecibos, "auditoria": sqlAuditoria, "regras": sqlRegras}

// LerPainel usa um único SELECT, inclusive para total e página.
func (r *RepositorioPostgres) LerPainel(ctx context.Context, recurso string, f domain.FiltroLeituraPainel) (json.RawMessage, error) {
	sql, ok := consultasPainel[recurso]
	if !ok {
		return nil, domain.ErrDadosInvalidos
	}
	if f.TenantID <= 0 || len(f.Repositorios) == 0 || !f.Fim.After(f.Inicio) || f.Fim.Sub(f.Inicio).Hours() > 90*24 || f.Limite < 1 || f.Limite > 500 || f.Pagina < 1 || f.Pagina > 201 {
		return nil, domain.ErrDadosInvalidos
	}
	var dados json.RawMessage
	err := r.conexao.QueryRow(ctx, sql, pgx.NamedArgs{"tenant": f.TenantID, "repos": f.Repositorios, "inicio": f.Inicio, "fim": f.Fim, "snapshot": f.Snapshot, "limite": f.Limite, "deslocamento": (f.Pagina - 1) * f.Limite, "decisao": f.Decisao, "origem": f.Origem}).Scan(&dados)
	return dados, err
}

func (r *RepositorioPostgres) UltimasAnalisesWeb(ctx context.Context, tenant int64, repos []string) (map[string]time.Time, error) {
	rows, err := r.conexao.Query(ctx, `SELECT repository,max(received_at) FROM task_runs WHERE tenant_id=$1 AND repository=ANY($2::text[]) GROUP BY repository`, tenant, repos)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ultimas := make(map[string]time.Time, len(repos))
	for rows.Next() {
		var nome string
		var d time.Time
		if err := rows.Scan(&nome, &d); err != nil {
			return nil, err
		}
		ultimas[nome] = d
	}
	return ultimas, rows.Err()
}

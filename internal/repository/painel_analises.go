package repository

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"slices"
	"strconv"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/repository/db"
	"github.com/jackc/pgx/v5"
)

// lerPainel mantém total, página e agregados em uma única visão consistente do banco.
func (r *RepositorioPostgres) lerPainel(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := r.conexao.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err = tx.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY"); err != nil {
		return err
	}
	if err = fn(db.New(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (r *RepositorioPostgres) ListarAnalisesWeb(ctx context.Context, f domain.FiltroAnalisesWeb) (domain.PaginaAnalisesWeb, error) {
	p := domain.PaginaAnalisesWeb{InstallationID: f.TenantID, Snapshot: f.Snapshot, Pagina: f.Pagina, Limite: f.Limite, Itens: []domain.AnaliseWeb{}}
	if f.TenantID <= 0 || len(f.Repositorios) == 0 || f.Pagina < 1 || f.Pagina > 201 || f.Limite < 1 || f.Limite > 50 || !f.Fim.After(f.Inicio) {
		return p, domain.ErrDadosInvalidos
	}
	err := r.lerPainel(ctx, func(q *db.Queries) error {
		total, e := q.CountWebAnalyses(ctx, db.CountWebAnalysesParams{TenantID: f.TenantID, Repositories: f.Repositorios, SnapshotAt: f.Snapshot, PrNumber: f.NumeroPR, StateFilter: f.Estado, LevelFilter: f.Nivel, StartsAt: f.Inicio, EndsAt: f.Fim})
		if e != nil {
			return e
		}
		p.Total = total
		linhas, e := q.ListWebAnalyses(ctx, db.ListWebAnalysesParams{TenantID: f.TenantID, Repositories: f.Repositorios, SnapshotAt: f.Snapshot, PrNumber: f.NumeroPR, StateFilter: f.Estado, LevelFilter: f.Nivel, StartsAt: f.Inicio, EndsAt: f.Fim, SortOrder: f.Ordem, PageLimit: f.Limite, PageOffset: (f.Pagina - 1) * f.Limite})
		if e != nil {
			return e
		}
		for _, l := range linhas {
			p.Itens = append(p.Itens, domain.AnaliseWeb{ID: l.ID, Repositorio: l.Repository, NumeroPR: l.PrNumber, SHAHead: l.HeadSha, Autor: l.Author, Estado: l.State, Nivel: l.RiskLevel, RecebidoEm: l.ReceivedAt, CriadoEm: l.CreatedAt, AtualizadoEm: l.UpdatedAt})
		}
		p.TemMais = int64(f.Pagina)*int64(f.Limite) < p.Total
		return nil
	})
	return p, traduzir("listar análises do painel", err)
}
func caminhoGitHub(repo string) string {
	partes := []string{}
	for _, p := range splitRepositorio(repo) {
		partes = append(partes, url.PathEscape(p))
	}
	return "https://github.com/" + partes[0] + "/" + partes[1]
}
func splitRepositorio(repo string) []string { // Registros do núcleo usam dono/nome; falhas não geram links externos.
	for i, c := range repo {
		if c == '/' {
			return []string{repo[:i], repo[i+1:]}
		}
	}
	return []string{"", repo}
}
func (r *RepositorioPostgres) DetalharAnaliseWeb(ctx context.Context, tenant int64, repos []string, id string) (domain.DetalheAnaliseWeb, error) {
	d := domain.DetalheAnaliseWeb{InstallationID: tenant, Etapas: []domain.EtapaWeb{}, Tentativas: []domain.TentativaWeb{}, Aprovacoes: []domain.AprovacaoWeb{}}
	if tenant <= 0 || len(repos) == 0 {
		return d, domain.ErrDadosInvalidos
	}
	err := r.lerPainel(ctx, func(q *db.Queries) error {
		l, e := q.GetWebAnalysis(ctx, db.GetWebAnalysisParams{ID: id, TenantID: tenant, Repositories: repos})
		if e != nil {
			return e
		}
		d.Analise = domain.AnaliseWeb{ID: l.ID, Repositorio: l.Repository, NumeroPR: l.PrNumber, SHAHead: l.HeadSha, Autor: l.Author, Estado: l.State, Nivel: l.RiskLevel, RecebidoEm: l.ReceivedAt, CriadoEm: l.CreatedAt, AtualizadoEm: l.UpdatedAt}
		d.Atual = l.IsCurrent != nil && *l.IsCurrent
		d.URLPR = caminhoGitHub(l.Repository) + "/pull/" + strconv.Itoa(int(l.PrNumber))
		if l.CheckRunID != nil {
			u := caminhoGitHub(l.Repository) + "/runs/" + strconv.FormatInt(*l.CheckRunID, 10)
			d.URLCheck = &u
		}
		if e := preencherEtapas(ctx, q, &d); e != nil {
			return e
		}
		if e := preencherGate(ctx, q, &d, repos); e != nil {
			return e
		}
		if e := preencherAprovacoes(ctx, q, &d); e != nil {
			return e
		}
		recibo, e := q.GetWebReceiptID(ctx, db.GetWebReceiptIDParams{TenantID: tenant, TaskRunID: id})
		if e != nil && !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		if e == nil {
			d.ReciboID = &recibo
		}
		return nil
	})
	return d, traduzir("detalhar análise do painel", err)
}
func preencherEtapas(ctx context.Context, q *db.Queries, d *domain.DetalheAnaliseWeb) error {
	etapas, e := q.ListWebSteps(ctx, db.ListWebStepsParams{TenantID: d.InstallationID, TaskRunID: d.Analise.ID})
	if e != nil {
		return e
	}
	for _, l := range etapas {
		d.Etapas = append(d.Etapas, domain.EtapaWeb{ID: l.ID, De: l.FromState, Para: l.ToState, Detalhe: l.Detail, CriadoEm: l.CreatedAt})
	}
	tentativas, e := q.ListWebSandboxResults(ctx, db.ListWebSandboxResultsParams{TenantID: d.InstallationID, TaskRunID: d.Analise.ID})
	if e != nil {
		return e
	}
	for _, l := range tentativas {
		d.Tentativas = append(d.Tentativas, domain.TentativaWeb{Tentativa: l.Attempt, Compilou: l.Compiled, TestesPassaram: l.TestsPassed, TestesFalharam: l.TestsFailed, Saida: l.RawOutput, SaidaTruncada: l.OutputTruncated, DuracaoMS: l.DurationMs, TipoFalha: l.FailureType, CriadoEm: l.CreatedAt})
	}
	return nil
}
func preencherGate(ctx context.Context, q *db.Queries, d *domain.DetalheAnaliseWeb, repos []string) error {
	l, e := q.GetWebRiskGate(ctx, db.GetWebRiskGateParams{TenantID: d.InstallationID, TaskRunID: d.Analise.ID})
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	g := &domain.GateWeb{NivelDeterministico: l.DeterministicLevel, Nivel: l.Level, Motivo: l.Reason, Motivos: []domain.MotivoDeRisco{}, CalibracaoEstado: l.CalibrationStatus, CalibracaoRazao: l.CalibrationReason, Vizinhos: []domain.VizinhoDaCalibracao{}}
	if e := json.Unmarshal(l.RulesHit, &g.Motivos); e != nil {
		return e
	}
	var vizinhos []domain.VizinhoDaCalibracao
	if e := json.Unmarshal(l.CalibrationNeighbors, &vizinhos); e != nil {
		return e
	}
	for _, v := range vizinhos {
		if slices.Contains(repos, v.Repositorio) {
			g.Vizinhos = append(g.Vizinhos, v)
		}
	}
	d.Gate = g
	return nil
}
func preencherAprovacoes(ctx context.Context, q *db.Queries, d *domain.DetalheAnaliseWeb) error {
	linhas, e := q.ListWebApprovals(ctx, db.ListWebApprovalsParams{TenantID: d.InstallationID, TaskRunID: d.Analise.ID})
	if e != nil {
		return e
	}
	for _, l := range linhas {
		d.Aprovacoes = append(d.Aprovacoes, domain.AprovacaoWeb{ID: l.ID, Decisao: l.Decision, Revisor: l.Reviewer, Origem: l.Origin, Justificativa: l.Justification, CommitSHA: l.CommitSha, AplicavelAoCommit: l.CommitSha != nil && *l.CommitSha == d.Analise.SHAHead, DecididoEm: l.DecidedAt})
	}
	return nil
}

package integracaopostgres

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/repository"
)

func taskRunRecebidoEm(t *testing.T, repo *repository.RepositorioPostgres, tenant int64, recebidoEm time.Time) domain.TaskRun {
	t.Helper()

	criado, _, err := repo.CriarTaskRun(t.Context(), domain.NovoTaskRun{
		TenantID:    tenant,
		DeliveryID:  fmt.Sprintf("entrega-%d-%d", tenant, time.Now().UnixNano()),
		Repositorio: "Azulpasta/alvo",
		NumeroPR:    7,
		SHAHead:     "0123456789abcdef0123456789abcdef01234567",
		Autor:       "autora",
		Acao:        "opened",
		RecebidoEm:  recebidoEm,
	})
	if err != nil {
		t.Fatalf("criar task_run: %v", err)
	}
	return criado
}

func percorrer(t *testing.T, repo *repository.RepositorioPostgres, taskRun domain.TaskRun, estados ...domain.EstadoTaskRun) {
	t.Helper()

	atual := taskRun.Estado
	for _, proximo := range estados {
		if _, err := repo.TransicionarTaskRun(t.Context(), domain.Transicao{TaskRunID: taskRun.ID, De: atual, Para: proximo}); err != nil {
			t.Fatalf("transicionar %s → %s: %v", atual, proximo, err)
		}
		atual = proximo
	}
}

func gravarGate(t *testing.T, repo *repository.RepositorioPostgres, taskRunID string, nivel domain.NivelRisco) {
	t.Helper()

	if err := repo.GravarGateDeRisco(t.Context(), domain.GateDeRisco{
		TaskRunID: taskRunID, NivelDeterministico: nivel, Nivel: nivel, Motivo: "motivo",
	}); err != nil {
		t.Fatalf("gravar gate: %v", err)
	}
}

func gravarDesfecho(t *testing.T, tenant int64, desfecho domain.DesfechoMudanca, criadoEm time.Time) {
	t.Helper()

	if _, err := pool.Exec(t.Context(), `
		INSERT INTO code_memory (tenant_id, repository, summary, embedding, level, outcome, created_at)
		VALUES ($1, 'Azulpasta/alvo', 'mudança', array_fill(0.1::real, ARRAY[$2::int])::vector, 'alto', $3, $4)`,
		tenant, domain.DimensoesEmbedding, string(desfecho), criadoEm); err != nil {
		t.Fatalf("gravar desfecho: %v", err)
	}
}

func TestRelatorioSemanalContaSoAJanelaEOTenant(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()
	inicio := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	fim := inicio.AddDate(0, 0, 7)
	dentro := inicio.Add(36 * time.Hour)

	aprovada := taskRunRecebidoEm(t, repo, tenant, dentro)
	percorrer(t, repo, aprovada, domain.EstadoAvaliandoRisco, domain.EstadoAguardandoAprovacao, domain.EstadoAplicando, domain.EstadoConcluido)
	gravarGate(t, repo, aprovada.ID, domain.RiscoAlto)

	direta := taskRunRecebidoEm(t, repo, tenant, inicio)
	percorrer(t, repo, direta, domain.EstadoAvaliandoRisco, domain.EstadoAplicando, domain.EstadoConcluido)
	gravarGate(t, repo, direta.ID, domain.RiscoMedio)

	rejeitada := taskRunRecebidoEm(t, repo, tenant, dentro)
	percorrer(t, repo, rejeitada, domain.EstadoAguardandoAprovacao, domain.EstadoAguardandoAprovacao, domain.EstadoRejeitado)
	gravarGate(t, repo, rejeitada.ID, domain.RiscoMedio)

	falha := taskRunRecebidoEm(t, repo, tenant, dentro)
	percorrer(t, repo, falha, domain.EstadoTestandoSandbox, domain.EstadoFalhou)

	taskRunRecebidoEm(t, repo, tenant, fim.Add(-time.Microsecond))

	antes := taskRunRecebidoEm(t, repo, tenant, inicio.Add(-time.Second))
	percorrer(t, repo, antes, domain.EstadoConcluido)
	taskRunRecebidoEm(t, repo, tenant, fim)
	outroTenant := taskRunRecebidoEm(t, repo, novoTenant(), dentro)
	percorrer(t, repo, outroTenant, domain.EstadoConcluido)

	gravarDesfecho(t, tenant, domain.DesfechoRevertidaAposMerge, dentro)
	gravarDesfecho(t, tenant, domain.DesfechoRevertidaAposMerge, inicio)
	gravarDesfecho(t, tenant, domain.DesfechoRevertidaAposMerge, fim)
	gravarDesfecho(t, tenant, domain.DesfechoSemReversao, dentro)
	gravarDesfecho(t, novoTenant(), domain.DesfechoRevertidaAposMerge, dentro)

	relatorio, err := repo.RelatorioSemanal(t.Context(), tenant, inicio, fim)
	if err != nil {
		t.Fatalf("relatório: %v", err)
	}

	esperado := domain.RelatorioSemanal{
		TenantID: tenant, Inicio: inicio, Fim: fim,
		Total: 5, Concluidas: 2, Rejeitadas: 1, Falhas: 1,
		ExigiramAprovacao: 2, RiscoMedio: 2, Revertidas: 2,
	}
	if relatorio != esperado {
		t.Errorf("relatório = %+v\nesperava    %+v", relatorio, esperado)
	}
	if relatorio.EmAndamento() != 1 {
		t.Errorf("em andamento = %d, esperava 1", relatorio.EmAndamento())
	}
	if taxa := relatorio.TaxaDeSucesso(); taxa == nil || *taxa != 0.5 {
		t.Errorf("taxa de sucesso = %v, esperava 0.5", taxa)
	}
}

func TestRelatorioSemanalDeTenantSemExecucoesVemZerado(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()
	inicio := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)

	relatorio, err := repo.RelatorioSemanal(t.Context(), tenant, inicio, inicio.AddDate(0, 0, 7))
	if err != nil {
		t.Fatalf("relatório: %v", err)
	}
	if relatorio.Total != 0 || relatorio.Revertidas != 0 || relatorio.TaxaDeSucesso() != nil {
		t.Errorf("esperava relatório zerado e sem taxa, obtive %+v", relatorio)
	}
}

func TestRelatorioSemanalRecusaJanelaOuTenantInvalidos(t *testing.T) {
	repo := exigirBanco(t)
	inicio := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)

	if _, err := repo.RelatorioSemanal(t.Context(), 0, inicio, inicio.AddDate(0, 0, 7)); !errors.Is(err, domain.ErrDadosInvalidos) {
		t.Errorf("tenant 0 deveria dar ErrDadosInvalidos, obtive %v", err)
	}
	if _, err := repo.RelatorioSemanal(t.Context(), novoTenant(), inicio, inicio); !errors.Is(err, domain.ErrDadosInvalidos) {
		t.Errorf("janela vazia deveria dar ErrDadosInvalidos, obtive %v", err)
	}
}

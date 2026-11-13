package integracaopostgres

import (
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

func TestRevisaoReenviadaPeloGitHubNaoDuplicaAAprovacao(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()
	taskRun := novoTaskRun(t, repo, tenant)
	revisao := domain.DecisaoDeReview{
		InstallationID: tenant, DeliveryID: "29afe4a0-df10-11f1-91ae-reenviada", Repositorio: taskRun.Repositorio,
		NumeroPR: taskRun.NumeroPR, Estado: "approved", Revisor: "httpslua", Origem: domain.OrigemGitHub, RecebidoEm: time.Now().UTC(),
	}

	if err := repo.RegistrarRevisao(t.Context(), revisao); err != nil {
		t.Fatalf("primeira entrega: %v", err)
	}
	revisao.RecebidoEm = time.Now().UTC().Add(time.Hour)
	if err := repo.RegistrarRevisao(t.Context(), revisao); err != nil {
		t.Fatalf("o reenvio do mesmo delivery_id não é erro: %v", err)
	}

	if aprovacoes := contar(t, `SELECT count(*) FROM approvals WHERE task_run_id = $1`, taskRun.ID); aprovacoes != 1 {
		t.Errorf("o reenvio não cria uma segunda aprovação, há %d", aprovacoes)
	}
}

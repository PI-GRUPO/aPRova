package integracaopostgres

import (
	"errors"
	"testing"

	"github.com/Azulpasta/aPRova/internal/acaoremota"
	"github.com/Azulpasta/aPRova/internal/repository"
)

func TestJustificativaRemotaPersisteNaExecucaoOriginal(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()
	original := novoTaskRun(t, repo, tenant)
	_ = novoTaskRun(t, repo, tenant)
	tentativa := tentativaDeExemplo(tenant, "justificativa-"+original.ID)
	tentativa.TaskRunID = original.ID
	tentativa.Justificativa = "Risco aceito após avaliar os testes."
	if _, err := repo.Reservar(t.Context(), tentativa); err != nil {
		t.Fatal(err)
	}
	var taskRunID, justificativa string
	err := pool.QueryRow(t.Context(), `SELECT task_run_id, justification FROM remote_action_attempts WHERE action_id = $1`, tentativa.IDAcao).Scan(&taskRunID, &justificativa)
	if err != nil || taskRunID != original.ID || justificativa != tentativa.Justificativa {
		t.Fatalf("contexto não persistido: task_run=%s justificativa=%q erro=%v", taskRunID, justificativa, err)
	}
	reaberto := repository.NovoRepositorioPostgres(pool, repository.OpcoesPostgres{})
	if _, err := reaberto.Reservar(t.Context(), tentativa); !errors.Is(err, acaoremota.ErrAcaoJaAceita) {
		t.Fatalf("outra instância executaria novamente: %v", err)
	}
}

func TestTentativaNaoSeVinculaAExecucaoDeOutroTenant(t *testing.T) {
	repo := exigirBanco(t)
	deOutroTenant := novoTaskRun(t, repo, novoTenant())
	tentativa := tentativaDeExemplo(novoTenant(), "tenant-isolado-"+deOutroTenant.ID)
	tentativa.TaskRunID = deOutroTenant.ID
	if _, err := repo.Reservar(t.Context(), tentativa); err != nil {
		t.Fatal(err)
	}
	if total := contar(t, `SELECT count(*) FROM remote_action_attempts WHERE action_id = $1 AND task_run_id IS NULL`, tentativa.IDAcao); total != 1 {
		t.Fatal("tentativa vinculada à execução de outro tenant")
	}
}

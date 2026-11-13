package integracaopostgres

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/calibracao"
	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/repository"
)

var _ calibracao.Historico = (*repository.RepositorioPostgres)(nil)

func shaDoCommit(numero int) string {
	return fmt.Sprintf("%040x", numero)
}

func aprovarCommit(t *testing.T, repo *repository.RepositorioPostgres, taskRun domain.TaskRun, decisao domain.DecisaoAprovacao, commit string) {
	t.Helper()
	err := repo.RegistrarAprovacao(t.Context(), domain.Aprovacao{
		TaskRunID: taskRun.ID, DeliveryID: fmt.Sprintf("revisao-%s-%s-%d", taskRun.ID, commit, time.Now().UnixNano()),
		Decisao: decisao, Revisor: "revisora", DecididaEm: time.Now().UTC(), CommitSHA: commit,
	})
	if err != nil {
		t.Fatalf("registrar revisão do commit %s: %v", commit, err)
	}
}

func fecharPR(t *testing.T, repo *repository.RepositorioPostgres, tenant int64, numero int, mergeado bool, head string) {
	t.Helper()
	fechado := fechamento(tenant, numero, mergeado, time.Now().UTC())
	fechado.SHAHead = head
	registrarFechamento(t, repo, fechado)
}

func reverterPR(t *testing.T, repo *repository.RepositorioPostgres, tenant int64, numero int) {
	t.Helper()
	reversao := fechamento(tenant, numero+500, true, time.Now().UTC())
	reversao.Reverte = numero
	registrarFechamento(t, repo, reversao)
}

func TestVizinhosTrazemODesfechoDerivadoDosFatosDoPR(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()
	vetor := vetorUnitario(0)

	aprovadoEMergeado := analiseComVetor(t, repo, tenant, 1, shaDoCommit(1), vetor)
	aprovarCommit(t, repo, aprovadoEMergeado, domain.DecisaoAprovada, shaDoCommit(1))
	fecharPR(t, repo, tenant, 1, true, shaDoCommit(1))

	mergeadoSemAprovacao := analiseComVetor(t, repo, tenant, 2, shaDoCommit(2), vetor)
	fecharPR(t, repo, tenant, 2, true, shaDoCommit(2))

	fechadoSemMerge := analiseComVetor(t, repo, tenant, 3, shaDoCommit(3), vetor)
	aprovarCommit(t, repo, fechadoSemMerge, domain.DecisaoAprovada, shaDoCommit(3))
	fecharPR(t, repo, tenant, 3, false, shaDoCommit(3))

	revertido := analiseComVetor(t, repo, tenant, 4, shaDoCommit(4), vetor)
	aprovarCommit(t, repo, revertido, domain.DecisaoAprovada, shaDoCommit(4))
	fecharPR(t, repo, tenant, 4, true, shaDoCommit(4))
	reverterPR(t, repo, tenant, 4)

	rejeitado := analiseComVetor(t, repo, tenant, 5, shaDoCommit(50), vetor)
	aprovarCommit(t, repo, rejeitado, domain.DecisaoAlteracoesSolicitadas, shaDoCommit(50))
	ajustado := analiseComVetor(t, repo, tenant, 5, shaDoCommit(51), vetor)
	aprovarCommit(t, repo, ajustado, domain.DecisaoAprovada, shaDoCommit(51))
	fecharPR(t, repo, tenant, 5, true, shaDoCommit(51))

	emAberto := analiseComVetor(t, repo, tenant, 6, shaDoCommit(6), vetor)
	aprovarCommit(t, repo, emAberto, domain.DecisaoAprovada, shaDoCommit(6))

	aprovadoOutroCommit := analiseComVetor(t, repo, tenant, 7, shaDoCommit(70), vetor)
	aprovarCommit(t, repo, aprovadoOutroCommit, domain.DecisaoAprovada, shaDoCommit(69))
	fecharPR(t, repo, tenant, 7, true, shaDoCommit(70))

	esperados := map[string]domain.DesfechoDoVizinho{
		aprovadoEMergeado.ID:    domain.VizinhoAprovadoEMergeado,
		mergeadoSemAprovacao.ID: domain.VizinhoMergeadoSemAprovacao,
		fechadoSemMerge.ID:      domain.VizinhoFechadoSemMerge,
		revertido.ID:            domain.VizinhoRevertido,
		rejeitado.ID:            domain.VizinhoSubstituido,
		ajustado.ID:             domain.VizinhoAprovadoEMergeado,
		emAberto.ID:             domain.VizinhoEmAberto,
		aprovadoOutroCommit.ID:  domain.VizinhoMergeadoSemAprovacao,
	}

	vizinhos, err := repo.VizinhosParecidos(t.Context(), consultaDeVizinhos(tenant, vetor, 20))
	if err != nil {
		t.Fatalf("buscar vizinhos: %v", err)
	}

	if len(vizinhos) != len(esperados) {
		t.Fatalf("esperava %d vizinhos, obtive %d: %+v", len(esperados), len(vizinhos), vizinhos)
	}
	for _, vizinho := range vizinhos {
		if vizinho.Desfecho != esperados[vizinho.TaskRunID] {
			t.Errorf("PR %d (%s): desfecho %s, esperava %s", vizinho.NumeroPR, vizinho.TaskRunID, vizinho.Desfecho, esperados[vizinho.TaskRunID])
		}
		if vizinho.TenantID != tenant || vizinho.Repositorio != "Azulpasta/alvo" || vizinho.Similaridade < 0.9999 {
			t.Errorf("identificação do vizinho: %+v", vizinho)
		}
	}
}

func TestVizinhosExcluemOProprioPRENuncaVemDeOutroTenant(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()
	vetor := vetorUnitario(0)

	analiseComVetor(t, repo, tenant, 9999, shaDoCommit(1), vetor)
	analiseComVetor(t, repo, novoTenant(), 1, shaDoCommit(2), vetor)
	deOutroRepositorio := analiseComVetor(t, repo, tenant, 9999, shaDoCommit(3), vetor)
	if _, err := pool.Exec(t.Context(), `UPDATE task_runs SET repository = 'Azulpasta/outro' WHERE id = $1`, deOutroRepositorio.ID); err != nil {
		t.Fatalf("mover para outro repositório: %v", err)
	}

	vizinhos, err := repo.VizinhosParecidos(t.Context(), consultaDeVizinhos(tenant, vetor, 10))
	if err != nil {
		t.Fatalf("buscar vizinhos: %v", err)
	}

	if len(vizinhos) != 1 || vizinhos[0].TaskRunID != deOutroRepositorio.ID {
		t.Errorf("só o PR 9999 de outro repositório do mesmo tenant é vizinho: %+v", vizinhos)
	}
}

func TestMemoriaDeCodigoTemUmaAssinaturaPorAnalise(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()
	analise := analiseComVetor(t, repo, tenant, 1, shaDoCommit(1), vetorUnitario(0))

	_, err := repo.GravarMemoriaDeCodigo(t.Context(), memoria(tenant, analise.ID, vetorUnitario(1)))

	if !errors.Is(err, domain.ErrDuplicado) {
		t.Errorf("a segunda assinatura da mesma análise é recusada como duplicada: %v", err)
	}
	if total := contar(t, `SELECT count(*) FROM code_memory WHERE task_run_id = $1`, analise.ID); total != 1 {
		t.Errorf("uma assinatura por task_run, há %d", total)
	}
}

func TestMigrationDaAssinaturaRecusaHistoricoNoFormatoAntigo(t *testing.T) {
	exigirBanco(t)

	url := criarBancoVazio(t, "assinatura_com_dados")
	migracao, err := novaMigracao(url)
	if err != nil {
		t.Fatalf("abrir migrate: %v", err)
	}
	defer fecharMigracao(migracao)

	if err := migracao.Migrate(18); err != nil {
		t.Fatalf("subir até a migration 18: %v", err)
	}
	conexao := conectar(t, url)
	if _, err := conexao.Exec(t.Context(), `
		INSERT INTO code_memory (tenant_id, repository, summary, embedding, level, outcome)
		VALUES (1, 'Azulpasta/alvo', 'memória antiga', array_fill(0.1::real, ARRAY[768])::vector, 'alto', 'alto_risco_aprovada')`); err != nil {
		t.Fatalf("gravar memória antiga: %v", err)
	}

	err = migracao.Migrate(19)

	if err == nil || !strings.Contains(err.Error(), "code_memory") {
		t.Fatalf("a migration precisa falhar nomeando code_memory, obtive %v", err)
	}
	var restantes int
	if err := conexao.QueryRow(t.Context(), `SELECT count(*) FROM code_memory WHERE outcome = 'alto_risco_aprovada'`).Scan(&restantes); err != nil || restantes != 1 {
		t.Errorf("nada pode ser descartado: %d linhas, %v", restantes, err)
	}
}

package integracaopostgres

import (
	"bytes"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/demo"
	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
	"github.com/PI-GRUPO/aPRova/internal/repository"
)

const repositorioDaDemo = "aprova-demo/loja-virtual"

var instanteDaDemo = time.Date(2026, 11, 20, 15, 0, 0, 0, time.UTC)

func assinadorDaDemo(t *testing.T) *receipt.Assinador {
	t.Helper()

	assinador, err := receipt.NovoAssinador(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatalf("assinador: %v", err)
	}
	return assinador
}

func semearDemo(t *testing.T, repo *repository.RepositorioPostgres, tenant int64) demo.Resumo {
	t.Helper()

	resumo, err := demo.Semear(t.Context(), repo, assinadorDaDemo(t), demo.Opcoes{
		InstallationID: tenant, Repositorio: repositorioDaDemo, Agora: instanteDaDemo,
	})
	if err != nil {
		t.Fatalf("semear: %v", err)
	}
	return resumo
}

func TestSeedDemoCriaAsAnalisesNosEstadosDaApresentacao(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()

	resumo := semearDemo(t, repo, tenant)

	if resumo.Criadas != 9 || resumo.JaExistiam != 0 {
		t.Fatalf("esperava 9 análises criadas e nenhuma existente, obtive %+v", resumo)
	}
	esperados := map[string]int{"concluido": 6, "aguardando_aprovacao": 1, "rejeitado": 1, "falhou": 1}
	for estado, total := range esperados {
		if obtido := contar(t, `SELECT count(*) FROM task_runs WHERE tenant_id = $1 AND state = $2`, tenant, estado); obtido != total {
			t.Errorf("estado %s: esperava %d análises, obtive %d", estado, total, obtido)
		}
	}
	if total := contar(t, `SELECT count(*) FROM approvals WHERE tenant_id = $1`, tenant); total != 4 {
		t.Errorf("esperava 4 decisões humanas, obtive %d", total)
	}
	if total := contar(t, `SELECT count(*) FROM code_memory WHERE tenant_id = $1`, tenant); total != 8 {
		t.Errorf("esperava 8 análises no histórico da calibração, obtive %d", total)
	}
}

func TestSeedDemoEmiteRecibosQueOVerificadorAprova(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()
	assinador := assinadorDaDemo(t)

	resumo := semearDemo(t, repo, tenant)

	verificador := receipt.NovoVerificador(receipt.NovoChaveiro(assinador.ChavePublica()), repo)
	recibos := 0
	for _, analise := range resumo.Analises {
		if analise.ReciboID == "" {
			if analise.Estado != domain.EstadoFalhou {
				t.Errorf("PR #%d terminou em %s sem recibo", analise.NumeroPR, analise.Estado)
			}
			continue
		}
		recibos++
		_, resultado, err := verificador.Verificar(t.Context(), analise.ReciboID)
		if err != nil || !resultado.Valido {
			t.Errorf("recibo do PR #%d reprovado: %+v %v", analise.NumeroPR, resultado, err)
		}
	}
	if recibos != 8 {
		t.Errorf("esperava 8 recibos, obtive %d", recibos)
	}
}

func TestSeedDemoAlimentaORelatorioSemanal(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()

	semearDemo(t, repo, tenant)

	relatorio, err := repo.RelatorioSemanal(t.Context(), tenant, instanteDaDemo.Add(-7*24*time.Hour), instanteDaDemo)
	if err != nil {
		t.Fatalf("relatório: %v", err)
	}
	esperado := domain.RelatorioSemanal{
		Total: 9, Concluidas: 6, Rejeitadas: 1, Falhas: 1, ExigiramAprovacao: 5, RiscoMedio: 2, Revertidas: 1,
	}
	obtido := domain.RelatorioSemanal{
		Total: relatorio.Total, Concluidas: relatorio.Concluidas, Rejeitadas: relatorio.Rejeitadas, Falhas: relatorio.Falhas,
		ExigiramAprovacao: relatorio.ExigiramAprovacao, RiscoMedio: relatorio.RiscoMedio, Revertidas: relatorio.Revertidas,
	}
	if obtido != esperado {
		t.Errorf("relatório semanal\nobtido:   %+v\nesperado: %+v", obtido, esperado)
	}
}

func TestSeedDemoMostraACalibracaoPorHistorico(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()

	semearDemo(t, repo, tenant)

	var deterministico, final, estado string
	var vizinhos int
	err := pool.QueryRow(t.Context(), `
		SELECT rg.deterministic_level, rg.level, rg.calibration_status, jsonb_array_length(rg.calibration_neighbors)
		FROM risk_gates rg JOIN task_runs tr ON tr.id = rg.task_run_id
		WHERE tr.tenant_id = $1 AND tr.pr_number = 105`, tenant).Scan(&deterministico, &final, &estado, &vizinhos)
	if err != nil {
		t.Fatalf("ler o gate do PR #105: %v", err)
	}
	if deterministico != "alto" || final != "medio" || estado != "calibrada" || vizinhos != 2 {
		t.Errorf("PR #105 deveria descer de alto para médio pela calibração com 2 vizinhos, obtive %s → %s (%s, %d vizinhos)",
			deterministico, final, estado, vizinhos)
	}
}

func TestSeedDemoNaoDuplicaQuandoRodaDeNovo(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()

	semearDemo(t, repo, tenant)
	segunda := semearDemo(t, repo, tenant)

	if segunda.Criadas != 0 || segunda.JaExistiam != 9 {
		t.Errorf("a segunda execução deveria reaproveitar as 9 análises, obtive %+v", segunda)
	}
	for tabela, esperado := range map[string]int{"task_runs": 9, "receipts": 8, "approvals": 4, "pull_request_reverts": 1} {
		if total := contar(t, `SELECT count(*) FROM `+tabela+` WHERE tenant_id = $1`, tenant); total != esperado {
			t.Errorf("%s: esperava %d linhas depois de semear duas vezes, obtive %d", tabela, esperado, total)
		}
	}
}

func TestSeedDemoNaoDeixaTrabalhoParaOWorker(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()

	semearDemo(t, repo, tenant)

	reivindicaveis := contar(t, `
		SELECT count(*) FROM task_runs tr
		WHERE tr.tenant_id = $1
		  AND tr.state IN ('pendente', 'mapeando_contexto', 'testando_sandbox', 'avaliando_risco', 'aguardando_aprovacao', 'aplicando')
		  AND (tr.state <> 'aguardando_aprovacao'
		       OR EXISTS (SELECT 1 FROM approvals a WHERE a.task_run_id = tr.id AND a.id > tr.last_approval_seen))`, tenant)
	if reivindicaveis != 0 {
		t.Errorf("o worker poderia pegar %d análises da demonstração e chamar o GitHub de verdade", reivindicaveis)
	}
}

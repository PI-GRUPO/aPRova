package integracaopostgres

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

func calibracaoDeExemplo() domain.Calibracao {
	return domain.Calibracao{
		Estado: domain.CalibracaoAplicada,
		Razao:  "5 casos parecidos aprovados e mergeados, nenhum revertido",
		Vizinhos: []domain.VizinhoDaCalibracao{
			{TaskRunID: "00000000-0000-4000-8000-000000000001", Repositorio: "Azulpasta/alvo", NumeroPR: 3, Similaridade: 0.9712, Desfecho: domain.VizinhoAprovadoEMergeado},
			{TaskRunID: "00000000-0000-4000-8000-000000000002", Repositorio: "Azulpasta/outro", NumeroPR: 8, Similaridade: 0.9301, Desfecho: domain.VizinhoAprovadoEMergeado},
		},
	}
}

func TestGateGuardaORastroDaCalibracao(t *testing.T) {
	repo := exigirBanco(t)
	taskRun := novoTaskRun(t, repo, novoTenant())
	gate := gateDeExemplo(taskRun.ID)
	gate.Nivel, gate.Calibracao = domain.RiscoMedio, calibracaoDeExemplo()

	if err := repo.GravarGateDeRisco(t.Context(), gate); err != nil {
		t.Fatalf("gravar: %v", err)
	}

	lido, err := repo.BuscarGateDeRisco(t.Context(), taskRun.ID)
	if err != nil {
		t.Fatalf("ler: %v", err)
	}
	if lido.NivelDeterministico != domain.RiscoAlto || lido.Nivel != domain.RiscoMedio {
		t.Errorf("os dois níveis ficam gravados, diferentes: %s e %s", lido.NivelDeterministico, lido.Nivel)
	}
	if !reflect.DeepEqual(lido.Calibracao, calibracaoDeExemplo()) {
		t.Errorf("rastro lido:\n%+v\nesperava\n%+v", lido.Calibracao, calibracaoDeExemplo())
	}
}

func TestGateSemCalibracaoInformadaFicaComoDesligada(t *testing.T) {
	repo := exigirBanco(t)
	taskRun := novoTaskRun(t, repo, novoTenant())

	if err := repo.GravarGateDeRisco(t.Context(), gateDeExemplo(taskRun.ID)); err != nil {
		t.Fatalf("gravar: %v", err)
	}

	lido, _ := repo.BuscarGateDeRisco(t.Context(), taskRun.ID)
	if lido.Calibracao.Estado != domain.CalibracaoDesligada || lido.Calibracao.Razao != "calibração desligada" || len(lido.Calibracao.Vizinhos) != 0 {
		t.Errorf("rastro de quem não calibrou: %+v", lido.Calibracao)
	}
}

func TestEstadoDaCalibracaoForaDaListaERecusado(t *testing.T) {
	repo := exigirBanco(t)
	taskRun := novoTaskRun(t, repo, novoTenant())
	if err := repo.GravarGateDeRisco(t.Context(), gateDeExemplo(taskRun.ID)); err != nil {
		t.Fatalf("gravar: %v", err)
	}

	_, err := pool.Exec(t.Context(), `UPDATE risk_gates SET calibration_status = 'opiniao_do_modelo' WHERE task_run_id = $1`, taskRun.ID)

	if codigoPostgres(err) != "23514" {
		t.Errorf("esperava check_violation (23514), obtive %v", err)
	}
}

func TestDecisaoComAssinaturaGravaOHistoricoNaMesmaTransacao(t *testing.T) {
	repo := exigirBanco(t)
	taskRun := novoTaskRun(t, repo, novoTenant())
	gate := gateDeExemplo(taskRun.ID)
	gate.Assinatura = &domain.AssinaturaDoPR{Texto: "motivos: falha_sandbox", Embedding: vetorUnitario(5)}

	if _, err := repo.GravarDecisaoComRecibo(t.Context(), gate, encadeadoNo(taskRun.ID)); err != nil {
		t.Fatalf("gravar: %v", err)
	}

	var resumo string
	if err := pool.QueryRow(t.Context(), `SELECT summary FROM code_memory WHERE task_run_id = $1`, taskRun.ID).Scan(&resumo); err != nil || resumo != "motivos: falha_sandbox" {
		t.Errorf("a assinatura entra no histórico com a decisão: %q, %v", resumo, err)
	}
}

func TestDecisaoSemAssinaturaNaoGravaHistorico(t *testing.T) {
	repo := exigirBanco(t)
	taskRun := novoTaskRun(t, repo, novoTenant())

	if _, err := repo.GravarDecisaoComRecibo(t.Context(), gateDeExemplo(taskRun.ID), encadeadoNo(taskRun.ID)); err != nil {
		t.Fatalf("gravar: %v", err)
	}

	if total := contar(t, `SELECT count(*) FROM code_memory WHERE task_run_id = $1`, taskRun.ID); total != 0 {
		t.Errorf("sem embedding não há o que guardar: %d", total)
	}
}

func TestSemReciboNaoFicaAssinaturaNoHistorico(t *testing.T) {
	repo := exigirBanco(t)
	taskRun := novoTaskRun(t, repo, novoTenant())
	gate := gateDeExemplo(taskRun.ID)
	gate.Assinatura = &domain.AssinaturaDoPR{Texto: "motivos: falha_sandbox", Embedding: vetorUnitario(5)}
	falhaAoAssinar := errors.New("falha ao assinar")

	_, err := repo.GravarDecisaoComRecibo(t.Context(), gate, func(*domain.Recibo) (domain.NovoRecibo, error) {
		return domain.NovoRecibo{}, falhaAoAssinar
	})

	if !errors.Is(err, falhaAoAssinar) {
		t.Fatalf("esperava o erro de quem monta o recibo, obtive %v", err)
	}
	if total := contar(t, `SELECT count(*) FROM code_memory WHERE task_run_id = $1`, taskRun.ID); total != 0 {
		t.Errorf("a assinatura não fica sem a decisão e o recibo: %d", total)
	}
}

func TestRazaoDaCalibracaoFicaForaDoRecibo(t *testing.T) {
	repo := exigirBanco(t)
	taskRun := novoTaskRun(t, repo, novoTenant())
	gate := gateDeExemplo(taskRun.ID)
	gate.Calibracao = domain.Calibracao{Estado: domain.CalibracaoHistoricoInsuficiente, Razao: "histórico insuficiente: 2 de 5"}

	recibo, err := repo.GravarDecisaoComRecibo(t.Context(), gate, encadeadoNo(taskRun.ID))
	if err != nil {
		t.Fatalf("gravar: %v", err)
	}

	if strings.Contains(recibo.Conteudo, "insuficiente") || strings.Contains(recibo.Conteudo, "calibra") {
		t.Errorf("o porquê da calibração não muda o payload do recibo: %s", recibo.Conteudo)
	}
	if lido, _ := repo.BuscarGateDeRisco(t.Context(), taskRun.ID); lido.Calibracao.Razao != "histórico insuficiente: 2 de 5" {
		t.Errorf("ele fica em risk_gates: %+v", lido.Calibracao)
	}
}

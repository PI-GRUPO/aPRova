package integracaopostgres

import (
	"errors"
	"sync"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/repository"
	"github.com/google/uuid"
)

func gateDeExemplo(taskRunID string) domain.GateDeRisco {
	return domain.GateDeRisco{
		TaskRunID: taskRunID, NivelDeterministico: domain.RiscoAlto, Nivel: domain.RiscoAlto, Motivo: "falha_sandbox: os testes falharam",
		Motivos: []domain.MotivoDeRisco{{ID: "falha_sandbox", Nivel: domain.RiscoAlto, Descricao: "os testes falharam", Evidencias: []string{"1 de 11 testes falharam"}}},
	}
}

func encadeadoNo(taskRunID string) func(anterior *domain.Recibo) (domain.NovoRecibo, error) {
	return func(anterior *domain.Recibo) (domain.NovoRecibo, error) {
		novo := reciboDeExemplo(taskRunID)
		novo.ID = uuid.NewString()
		if anterior != nil {
			novo.ReciboAnteriorID = &anterior.ID
		}
		return novo, nil
	}
}

func gravarDecisao(t *testing.T, repo *repository.RepositorioPostgres, tenant int64) domain.Recibo {
	t.Helper()

	taskRun := novoTaskRun(t, repo, tenant)
	recibo, err := repo.GravarDecisaoComRecibo(t.Context(), gateDeExemplo(taskRun.ID), encadeadoNo(taskRun.ID))
	if err != nil {
		t.Fatalf("gravar decisão com recibo: %v", err)
	}
	return recibo
}

func TestDecisaoEReciboSaoGravadosJuntosComOIdEscolhido(t *testing.T) {
	repo := exigirBanco(t)
	taskRun := novoTaskRun(t, repo, novoTenant())
	id := uuid.NewString()

	recibo, err := repo.GravarDecisaoComRecibo(t.Context(), gateDeExemplo(taskRun.ID), func(anterior *domain.Recibo) (domain.NovoRecibo, error) {
		if anterior != nil {
			t.Errorf("o primeiro recibo do tenant não tem anterior: %+v", anterior)
		}
		novo := reciboDeExemplo(taskRun.ID)
		novo.ID = id
		return novo, nil
	})
	if err != nil {
		t.Fatalf("gravar: %v", err)
	}

	if recibo.ID != id || recibo.TenantID != taskRun.TenantID {
		t.Errorf("o recibo precisa ter o id que foi assinado: %+v", recibo)
	}
	if _, err := repo.BuscarGateDeRisco(t.Context(), taskRun.ID); err != nil {
		t.Errorf("o gate precisa estar gravado: %v", err)
	}
	if lido, err := repo.BuscarReciboPorTaskRun(t.Context(), taskRun.ID); err != nil || lido.ID != id {
		t.Errorf("o recibo precisa estar gravado: %+v %v", lido, err)
	}
}

func TestSemReciboNaoFicaDecisaoGravada(t *testing.T) {
	repo := exigirBanco(t)
	taskRun := novoTaskRun(t, repo, novoTenant())
	falhaAoAssinar := errors.New("falha ao assinar")

	_, err := repo.GravarDecisaoComRecibo(t.Context(), gateDeExemplo(taskRun.ID), func(*domain.Recibo) (domain.NovoRecibo, error) {
		return domain.NovoRecibo{}, falhaAoAssinar
	})
	if !errors.Is(err, falhaAoAssinar) {
		t.Fatalf("esperava o erro de quem monta o recibo, obtive %v", err)
	}

	if _, err := repo.BuscarGateDeRisco(t.Context(), taskRun.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Errorf("sem recibo, a decisão não pode ficar gravada: %v", err)
	}

	_, err = repo.GravarDecisaoComRecibo(t.Context(), gateDeExemplo(taskRun.ID), func(*domain.Recibo) (domain.NovoRecibo, error) {
		invalido := reciboDeExemplo(taskRun.ID)
		invalido.Assinatura = []byte("curta")
		return invalido, nil
	})
	if err == nil {
		t.Fatal("recibo inválido precisa ser recusado")
	}
	if _, err := repo.BuscarGateDeRisco(t.Context(), taskRun.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Errorf("recibo recusado desfaz a decisão: %v", err)
	}
}

func TestSegundaDecisaoDoMesmoTaskRunEDuplicadaENaoGravaNada(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()
	taskRun := novoTaskRun(t, repo, tenant)
	if _, err := repo.GravarDecisaoComRecibo(t.Context(), gateDeExemplo(taskRun.ID), encadeadoNo(taskRun.ID)); err != nil {
		t.Fatalf("primeira: %v", err)
	}

	_, err := repo.GravarDecisaoComRecibo(t.Context(), gateDeExemplo(taskRun.ID), encadeadoNo(taskRun.ID))

	if !errors.Is(err, domain.ErrDuplicado) {
		t.Fatalf("esperava ErrDuplicado, obtive %v", err)
	}
	if total := contar(t, `SELECT count(*) FROM receipts WHERE tenant_id = $1`, tenant); total != 1 {
		t.Errorf("esperava 1 recibo, obtive %d", total)
	}
}

func TestCadaReciboRecebeComoAnteriorOUltimoDoMesmoTenant(t *testing.T) {
	repo := exigirBanco(t)
	tenant, outroTenant := novoTenant(), novoTenant()

	primeiro := gravarDecisao(t, repo, tenant)
	deOutroTenant := gravarDecisao(t, repo, outroTenant)
	segundo := gravarDecisao(t, repo, tenant)
	terceiro := gravarDecisao(t, repo, tenant)

	if primeiro.ReciboAnteriorID != nil || deOutroTenant.ReciboAnteriorID != nil {
		t.Errorf("o primeiro de cada tenant aponta para nada: %v %v", primeiro.ReciboAnteriorID, deOutroTenant.ReciboAnteriorID)
	}
	if segundo.ReciboAnteriorID == nil || *segundo.ReciboAnteriorID != primeiro.ID {
		t.Errorf("o segundo aponta para o primeiro: %v", segundo.ReciboAnteriorID)
	}
	if terceiro.ReciboAnteriorID == nil || *terceiro.ReciboAnteriorID != segundo.ID {
		t.Errorf("o terceiro aponta para o segundo: %v", terceiro.ReciboAnteriorID)
	}
}

func TestEmissoesSimultaneasDoMesmoTenantNaoBifurcamACadeia(t *testing.T) {
	const emissoes = 12

	repo := exigirBanco(t)
	tenant := novoTenant()
	taskRuns := make([]domain.TaskRun, emissoes)
	for i := range taskRuns {
		taskRuns[i] = novoTaskRun(t, repo, tenant)
	}

	var espera sync.WaitGroup
	largada := make(chan struct{})
	erros := make(chan error, emissoes)
	for _, taskRun := range taskRuns {
		espera.Go(func() {
			<-largada
			_, err := repo.GravarDecisaoComRecibo(t.Context(), gateDeExemplo(taskRun.ID), encadeadoNo(taskRun.ID))
			erros <- err
		})
	}
	close(largada)
	espera.Wait()
	close(erros)

	for err := range erros {
		if err != nil {
			t.Errorf("emissão simultânea falhou: %v", err)
		}
	}
	if primeiros := contar(t, `SELECT count(*) FROM receipts WHERE tenant_id = $1 AND previous_receipt_id IS NULL`, tenant); primeiros != 1 {
		t.Errorf("a cadeia tem um só começo, encontrei %d", primeiros)
	}
	if anteriores := contar(t, `SELECT count(DISTINCT previous_receipt_id) FROM receipts WHERE tenant_id = $1`, tenant); anteriores != emissoes-1 {
		t.Errorf("cada recibo tem um anterior diferente: %d distintos para %d recibos", anteriores, emissoes)
	}
	comprimento := contar(t, `
		WITH RECURSIVE cadeia AS (
			SELECT id FROM receipts WHERE tenant_id = $1 AND previous_receipt_id IS NULL
			UNION ALL
			SELECT r.id FROM receipts r JOIN cadeia c ON r.previous_receipt_id = c.id
		) SELECT count(*) FROM cadeia`, tenant)
	if comprimento != emissoes {
		t.Errorf("a cadeia percorrida do começo tem %d recibos, esperava %d", comprimento, emissoes)
	}
}

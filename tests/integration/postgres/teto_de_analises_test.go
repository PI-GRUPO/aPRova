package integracaopostgres

import (
	"fmt"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/repository"
)

const tetoDeTeste = 2

func novoBancoComTeto(t *testing.T) bancoDaFila {
	t.Helper()

	banco := novoBancoDaFila(t)
	banco.repo = repository.NovoRepositorioPostgres(banco.pool, repository.OpcoesPostgres{
		IntervaloConsulta: 20 * time.Millisecond, TetoDeAnalisesPorHora: tetoDeTeste,
	})
	return banco
}

func (b bancoDaFila) novoTaskRunDaInstalacao(t *testing.T, instalacao int64, numeroPR int) domain.TaskRun {
	t.Helper()

	criado, _, err := b.repo.CriarTaskRun(t.Context(), domain.NovoTaskRun{
		TenantID:    instalacao,
		DeliveryID:  fmt.Sprintf("entrega-%d-%d-%d", instalacao, numeroPR, time.Now().UnixNano()),
		Repositorio: "Azulpasta/aPRova-TESTES",
		NumeroPR:    numeroPR,
		SHAHead:     "3f9a1c7e2b4d6f8091a3c5e7f9b1d3e5a7c9e1f3",
		Autor:       "wlaisw",
		Acao:        "opened",
		RecebidoEm:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("criar task_run: %v", err)
	}
	return criado
}

func (b bancoDaFila) reivindicarTodos(t *testing.T) []domain.TaskRun {
	t.Helper()

	var reivindicados []domain.TaskRun
	for indice := 0; ; indice++ {
		taskRun, encontrado := b.reivindicar(t, fmt.Sprintf("trabalhador-%d", indice))
		if !encontrado {
			return reivindicados
		}
		reivindicados = append(reivindicados, taskRun)
	}
}

func TestInstalacaoNoTetoNaoTemAnaliseNovaReivindicada(t *testing.T) {
	banco := novoBancoComTeto(t)
	for numeroPR := 1; numeroPR <= tetoDeTeste+1; numeroPR++ {
		banco.novoTaskRunDoPR(t, numeroPR)
	}

	if reivindicados := banco.reivindicarTodos(t); len(reivindicados) != tetoDeTeste {
		t.Fatalf("reivindicou %d análises novas, o teto é %d", len(reivindicados), tetoDeTeste)
	}
	if pendentes := banco.contar(t, `SELECT count(*) FROM task_runs WHERE state = 'pendente' AND started_at IS NULL`); pendentes != 1 {
		t.Errorf("a análise acima do teto continua pendente, sem descarte: %d pendentes", pendentes)
	}
}

func TestTetoDeUmaInstalacaoNaoSeguraOutra(t *testing.T) {
	banco := novoBancoComTeto(t)
	for numeroPR := 1; numeroPR <= tetoDeTeste+1; numeroPR++ {
		banco.novoTaskRunDoPR(t, numeroPR)
	}
	daOutra := banco.novoTaskRunDaInstalacao(t, 71460213, 1)

	reivindicados := banco.reivindicarTodos(t)
	if len(reivindicados) != tetoDeTeste+1 {
		t.Fatalf("reivindicou %d, esperava o teto da primeira mais a da outra instalação", len(reivindicados))
	}
	if reivindicados[len(reivindicados)-1].ID != daOutra.ID {
		t.Error("a análise da outra instalação passa na frente da que está acima do teto")
	}
}

func TestJanelaQueAbreLiberaAAnaliseQueEsperava(t *testing.T) {
	banco := novoBancoComTeto(t)
	for numeroPR := 1; numeroPR <= tetoDeTeste+1; numeroPR++ {
		banco.novoTaskRunDoPR(t, numeroPR)
	}
	banco.reivindicarTodos(t)

	if _, err := banco.pool.Exec(t.Context(), `UPDATE task_runs SET started_at = now() - interval '61 minutes' WHERE started_at IS NOT NULL`); err != nil {
		t.Fatalf("envelhecer as análises: %v", err)
	}

	if _, encontrado := banco.reivindicar(t, "trabalhador-depois-da-janela"); !encontrado {
		t.Error("com a janela aberta, a análise que esperava é reivindicada")
	}
}

func TestAnaliseJaIniciadaComPrazoVencidoVoltaMesmoNoTeto(t *testing.T) {
	banco := novoBancoComTeto(t)
	for numeroPR := 1; numeroPR <= tetoDeTeste; numeroPR++ {
		banco.novoTaskRunDoPR(t, numeroPR)
	}
	primeiro, _ := banco.reivindicar(t, "trabalhador-que-morre")
	banco.reivindicar(t, "trabalhador-vivo")

	time.Sleep(prazoCurto + 100*time.Millisecond)

	retomado, encontrado := banco.reivindicar(t, "trabalhador-que-retoma")
	if !encontrado || !retomado.RetomadoDePrazoVencido {
		t.Fatal("a análise já iniciada não conta de novo no teto e é retomada")
	}
	if inicio := banco.contar(t, `SELECT count(*) FROM task_runs WHERE id = $1 AND started_at < now() - interval '400 milliseconds'`, primeiro.ID); inicio != 1 {
		t.Error("a retomada mantém o início original da análise")
	}
}

package integracaopostgres

import (
	"errors"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/repository"
)

func TestReivindicacaoMarcaQuandoRetomaUmPrazoVencido(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)

	primeira, _ := banco.reivindicar(t, "trabalhador-que-morreu")
	time.Sleep(prazoCurto + 100*time.Millisecond)
	retomada, _ := banco.reivindicar(t, "trabalhador-novo")
	if err := banco.repo.Liberar(t.Context(), taskRun.ID, "trabalhador-novo", 0); err != nil {
		t.Fatalf("liberar: %v", err)
	}
	devolvida, _ := banco.reivindicar(t, "trabalhador-seguinte")

	if primeira.RetomadoDePrazoVencido {
		t.Error("a primeira reivindicação não retoma prazo de ninguém")
	}
	if !retomada.RetomadoDePrazoVencido {
		t.Error("reivindicar depois do prazo de outro trabalhador vencer precisa ser marcado como retomada")
	}
	if devolvida.RetomadoDePrazoVencido {
		t.Error("um task_run devolvido à fila pelo próprio trabalhador não é prazo vencido")
	}
}

func TestContarPendentesContaSoOsTaskRunsPendentes(t *testing.T) {
	banco := novoBancoDaFila(t)
	banco.novoTaskRunDoPR(t, 16)
	andamento, _ := banco.reivindicar(t, "trabalhador-a")
	banco.levarAte(t, "trabalhador-a", andamento, domain.EstadoMapeandoContexto)
	banco.novoTaskRunDoPR(t, 14)
	banco.novoTaskRunDoPR(t, 15)

	pendentes, err := banco.repo.ContarPendentes(t.Context())

	if err != nil || pendentes != 2 {
		t.Errorf("esperava 2 pendentes, obtive %d (%v)", pendentes, err)
	}
}

func TestConferirVersaoDoBancoAceitaASchemaMigradaERecusaOutra(t *testing.T) {
	banco := novoBancoDaFila(t)

	if err := banco.repo.ConferirVersaoDoBanco(t.Context()); err != nil {
		t.Fatalf("o banco recém-migrado está na versão esperada: %v", err)
	}

	if _, err := banco.pool.Exec(t.Context(), `UPDATE schema_migrations SET dirty = true`); err != nil {
		t.Fatalf("marcar como sujo: %v", err)
	}
	if err := banco.repo.ConferirVersaoDoBanco(t.Context()); !errors.Is(err, repository.ErrVersaoDoBanco) {
		t.Errorf("migration interrompida precisa recusar a prontidão, obtive %v", err)
	}

	if _, err := banco.pool.Exec(t.Context(), `UPDATE schema_migrations SET dirty = false, version = version - 1`); err != nil {
		t.Fatalf("voltar a versão: %v", err)
	}
	if err := banco.repo.ConferirVersaoDoBanco(t.Context()); !errors.Is(err, repository.ErrVersaoDoBanco) {
		t.Errorf("banco atrás da versão esperada precisa recusar a prontidão, obtive %v", err)
	}
}

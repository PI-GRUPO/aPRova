package integracaopostgres

import (
	"fmt"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

const commitDoPushNovo = "95eb58e1d0c4a7b3f2e6d9c8b1a0f4e3d2c1b0a9"

func (b bancoDaFila) novoTaskRunNoCommit(t *testing.T, numeroPR int, sha string) domain.TaskRun {
	t.Helper()

	criado, _, err := b.repo.CriarTaskRun(t.Context(), domain.NovoTaskRun{
		TenantID: 58291734, DeliveryID: fmt.Sprintf("entrega-%s-%d", sha[:7], time.Now().UnixNano()),
		Repositorio: "Azulpasta/aPRova-TESTES", NumeroPR: numeroPR, SHAHead: sha,
		Autor: "wlaisw", Acao: "synchronize", RecebidoEm: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("criar task_run: %v", err)
	}
	return criado
}

func TestRevisaoGuardaOCommitRevisado(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()
	taskRun := novoTaskRun(t, repo, tenant)

	err := repo.RegistrarRevisao(t.Context(), domain.DecisaoDeReview{
		InstallationID: tenant, DeliveryID: "revisao-com-commit-" + taskRun.ID, Repositorio: taskRun.Repositorio,
		NumeroPR: taskRun.NumeroPR, Estado: "approved", Revisor: "httpslua", Origem: domain.OrigemGitHub,
		RecebidoEm: time.Now().UTC(), CommitSHA: taskRun.SHAHead,
	})
	if err != nil {
		t.Fatalf("registrar: %v", err)
	}

	aprovacao, err := repo.ProximaAprovacao(t.Context(), taskRun.ID, 0)
	if err != nil || aprovacao.CommitSHA != taskRun.SHAHead {
		t.Errorf("a aprovação guarda o commit revisado: %+v, %v", aprovacao, err)
	}
}

func TestFluxoComAprovacaoDeCommitAntigoNaoLiberaOPushNovo(t *testing.T) {
	banco := novoBancoDaFila(t)
	motor, github := motorQuePublicaNoPR(t, banco, domain.RiscoAlto)
	antigo := banco.novoTaskRunDoPR(t, 14)
	conduzirAteParar(t, banco, motor)
	novo := banco.novoTaskRunNoCommit(t, 14, commitDoPushNovo)
	conduzirAteParar(t, banco, motor)

	err := banco.repo.RegistrarRevisao(t.Context(), domain.DecisaoDeReview{
		InstallationID: novo.TenantID, DeliveryID: fmt.Sprintf("revisao-antiga-%d", time.Now().UnixNano()), Repositorio: novo.Repositorio,
		NumeroPR: novo.NumeroPR, Estado: "approved", Revisor: "httpslua", Origem: domain.OrigemGitHub,
		RecebidoEm: time.Now().UTC(), CommitSHA: antigo.SHAHead,
	})
	if err != nil {
		t.Fatalf("registrar a revisão do commit antigo: %v", err)
	}
	conduzirAteParar(t, banco, motor)

	if estado := banco.estadoDe(t, novo.ID); estado != domain.EstadoAguardandoAprovacao {
		t.Fatalf("a aprovação do commit antigo não libera o push novo, foi para %s", estado)
	}
	if linhas := banco.contar(t, `SELECT count(*) FROM approvals WHERE task_run_id = $1`, novo.ID); linhas != 1 {
		t.Errorf("a revisão é registrada mesmo sem decidir, há %d", linhas)
	}
	for _, check := range func() []checkRunPublicado { checks, _ := github.estado(); return checks }() {
		if check.SHAHead == commitDoPushNovo && check.Conclusao != "action_required" {
			t.Errorf("o Check Run do push novo continua bloqueando: %+v", check)
		}
	}
}

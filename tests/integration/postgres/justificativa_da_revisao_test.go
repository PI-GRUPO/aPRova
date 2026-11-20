package integracaopostgres

import (
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

func TestRevisaoGravaAJustificativa(t *testing.T) {
	casos := map[string]struct {
		origem        domain.OrigemAprovacao
		justificativa string
	}{
		"texto digitado no Slack": {domain.OrigemSlack, "Risco aceito.\n\nTestes conferidos."},
		"corpo da review":         {domain.OrigemGitHub, "Conferi o cálculo do frete."},
		"review sem corpo":        {domain.OrigemGitHub, ""},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			repo := exigirBanco(t)
			tenant := novoTenant()
			taskRun := novoTaskRun(t, repo, tenant)

			err := repo.RegistrarRevisao(t.Context(), domain.DecisaoDeReview{
				InstallationID: tenant, DeliveryID: "revisao-com-justificativa-" + taskRun.ID, Repositorio: taskRun.Repositorio,
				NumeroPR: taskRun.NumeroPR, Estado: "approved", Revisor: "julia-ed2", Origem: caso.origem,
				RecebidoEm: time.Now().UTC(), CommitSHA: taskRun.SHAHead, Justificativa: caso.justificativa,
			})
			if err != nil {
				t.Fatalf("registrar: %v", err)
			}

			aprovacao, err := repo.ProximaAprovacao(t.Context(), taskRun.ID, 0)
			if err != nil {
				t.Fatalf("ler a aprovação: %v", err)
			}
			if aprovacao.Justificativa == nil || *aprovacao.Justificativa != caso.justificativa {
				t.Errorf("approvals.justification precisa ser %q, obtive %v", caso.justificativa, aprovacao.Justificativa)
			}
		})
	}
}

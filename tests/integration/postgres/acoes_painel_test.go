package integracaopostgres

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/acaoremota"
	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/platform"
	"github.com/PI-GRUPO/aPRova/internal/repository"
	"github.com/google/uuid"
)

type githubAcoesPainel struct {
	permissao platform.Permissao
	autor     string
	sha       string
	reviews   []platform.Review
}

func (g *githubAcoesPainel) PermissaoNoRepositorio(context.Context, int64, string, string, string) (platform.Permissao, error) {
	return g.permissao, nil
}
func (g *githubAcoesPainel) AutorDoPullRequest(context.Context, int64, string, string, int) (string, error) {
	return g.autor, nil
}
func (g *githubAcoesPainel) CabecaDoPullRequest(context.Context, int64, string, string, int) (string, bool, error) {
	return g.sha, true, nil
}
func (g *githubAcoesPainel) EnviarReview(_ context.Context, _ int64, _, _ string, _ int, review platform.Review) error {
	g.reviews = append(g.reviews, review)
	return nil
}

func servicoAcoesPainel(t *testing.T, r *repository.RepositorioPostgres, g *githubAcoesPainel) *acaoremota.Servico {
	t.Helper()
	s, err := acaoremota.NovoServico(acaoremota.OpcoesServico{
		Identidades: acaoremota.NovoResolvedorEmMemoria(nil), Permissoes: g, Autoria: g,
		Executor: acaoremota.NovoExecutorNoGitHub(g, r), Tentativas: r, AnalisesPainel: r, CabecaDoPR: g,
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func alvoDaAcaoDoPainel(t *testing.T, r *repository.RepositorioPostgres, tenant int64, estado domain.EstadoTaskRun) domain.TaskRun {
	t.Helper()
	for _, a := range semearDemo(t, r, tenant).Analises {
		if a.Estado == estado {
			tr, err := r.BuscarTaskRun(t.Context(), a.TaskRunID)
			if err != nil {
				t.Fatal(err)
			}
			return tr
		}
	}
	t.Fatalf("fixture sem análise no estado %s", estado)
	return domain.TaskRun{}
}

func pedidoDaAcaoDoPainel(tr domain.TaskRun, tipo acaoremota.Tipo) acaoremota.Pedido {
	return acaoremota.Pedido{IDAcao: "painel:" + uuid.NewString(), Tipo: tipo, TaskRunID: tr.ID, CommitSHA: tr.SHAHead,
		InstallationID: tr.TenantID, Repositorio: tr.Repositorio, NumeroPR: tr.NumeroPR}
}

func TestPainelRevalidaPermissaoAutorECommitEAuditaAReview(t *testing.T) {
	r := exigirBanco(t)
	tr := alvoDaAcaoDoPainel(t, r, novoTenant(), domain.EstadoAguardandoAprovacao)
	g := &githubAcoesPainel{permissao: platform.PermissaoRead, autor: tr.Autor, sha: tr.SHAHead}
	s := servicoAcoesPainel(t, r, g)
	p := pedidoDaAcaoDoPainel(tr, acaoremota.TipoAprovar)
	executar := func(login string, decisao acaoremota.Decisao, motivo acaoremota.Motivo) {
		t.Helper()
		if d, m := s.ProcessarPainel(t.Context(), p, login); d != decisao || m != motivo {
			t.Fatalf("decisão %s/%s, queria %s/%s", d, m, decisao, motivo)
		}
	}
	executar("revisora", acaoremota.DecisaoInvalida, acaoremota.MotivoCorpoInvalido)
	p.Justificativa = "Revisei o diff para este commit."
	executar("revisora", acaoremota.DecisaoRecusada, acaoremota.MotivoPermissaoInsuficiente)
	g.permissao = platform.PermissaoWrite
	executar(strings.ToUpper(tr.Autor), acaoremota.DecisaoRecusada, acaoremota.MotivoAutorDoProprioPR)
	g.sha = strings.Repeat("f", 40)
	executar("revisora", acaoremota.DecisaoRecusada, acaoremota.MotivoAnaliseIncompativel)
	if len(g.reviews) != 0 {
		t.Fatal("publicou review para ações recusadas")
	}
	g.sha = tr.SHAHead
	executar("revisora", acaoremota.DecisaoAceita, acaoremota.MotivoAutorizada)
	executar("revisora", acaoremota.DecisaoRepetida, acaoremota.MotivoAcaoRepetida)
	if len(g.reviews) != 1 || g.reviews[0].Evento != platform.EventoAprovar || g.reviews[0].CommitID != tr.SHAHead {
		t.Fatal(g.reviews)
	}
	login, id, justificativa, ok := acaoremota.QuemDecidiuNoPainel(g.reviews[0].Corpo)
	if !ok || id != tr.ID || login != "revisora" || justificativa != p.Justificativa {
		t.Fatal(g.reviews[0])
	}
	if total := contar(t, `SELECT count(*) FROM remote_action_attempts
		WHERE tenant_id = $1 AND task_run_id = $2 AND origin = 'painel' AND slack_user_id IS NULL AND justification = $3`,
		tr.TenantID, tr.ID, justificativa); total != 5 {
		t.Fatalf("esperava cinco tentativas válidas no diário, obtive %d", total)
	}
	p = pedidoDaAcaoDoPainel(tr, acaoremota.TipoSolicitarAlteracao)
	p.Justificativa = "Acrescente um teste para o caso de erro."
	executar("revisora", acaoremota.DecisaoAceita, acaoremota.MotivoAutorizada)
	if len(g.reviews) != 2 || g.reviews[1].Evento != platform.EventoSolicitarAlteracao || g.reviews[1].CommitID != tr.SHAHead {
		t.Fatal(g.reviews)
	}

	agora := time.Now().UTC()
	f := domain.FiltroLeituraPainel{TenantID: tr.TenantID, Repositorios: []string{tr.Repositorio}, Origem: "Painel",
		Inicio: agora.Add(-time.Hour), Fim: agora.Add(time.Hour), Snapshot: agora, Pagina: 1, Limite: 20}
	var auditoria struct {
		Itens []struct{ Origem string }
	}
	bruto, err := r.LerPainel(t.Context(), "auditoria", f)
	if err != nil || json.Unmarshal(bruto, &auditoria) != nil || len(auditoria.Itens) != 6 {
		t.Fatalf("auditoria do painel: %s, %v", bruto, err)
	}
	for _, item := range auditoria.Itens {
		if item.Origem != "Painel" {
			t.Fatal(item)
		}
	}

	// Uma review demorada ainda pertence à análise e ao commit que foram revisados.
	_, _, err = r.CriarTaskRun(t.Context(), domain.NovoTaskRun{
		TenantID: tr.TenantID, DeliveryID: uuid.NewString(), Repositorio: tr.Repositorio, NumeroPR: tr.NumeroPR,
		SHAHead: strings.Repeat("b", 40), Autor: tr.Autor, Acao: "synchronize", RecebidoEm: agora,
	})
	if err != nil {
		t.Fatal(err)
	}
	review := domain.DecisaoDeReview{TaskRunID: id, InstallationID: tr.TenantID, Repositorio: tr.Repositorio, NumeroPR: tr.NumeroPR,
		DeliveryID: uuid.NewString(), Estado: "approved", Origem: domain.OrigemPainel, Revisor: login, CommitSHA: tr.SHAHead,
		RecebidoEm: agora, Justificativa: justificativa}
	if err := r.RegistrarRevisao(t.Context(), review); err != nil {
		t.Fatal(err)
	}
	if total := contar(t, `SELECT count(*) FROM approvals WHERE task_run_id = $1 AND commit_sha = $2 AND origin = 'painel' AND justification = $3`,
		tr.ID, tr.SHAHead, justificativa); total != 1 {
		t.Fatalf("decisão ausente na análise revisada: %d", total)
	}
	review.CommitSHA = strings.Repeat("b", 40)
	if err := r.RegistrarRevisao(t.Context(), review); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("review de outro commit deveria ser recusada: %v", err)
	}
}

func TestPainelReexecutaFalhaDeInfraestruturaUmaVez(t *testing.T) {
	banco := novoBancoDaFila(t)
	r := banco.repo
	tr := alvoDaAcaoDoPainel(t, r, novoTenant(), domain.EstadoFalhou)
	g := &githubAcoesPainel{permissao: platform.PermissaoWrite, sha: tr.SHAHead}
	s := servicoAcoesPainel(t, r, g)
	if err := r.GravarCheckRun(t.Context(), tr.ID, 900); err != nil {
		t.Fatal(err)
	}
	type resultado struct {
		id     string
		status acaoremota.Decisao
		motivo acaoremota.Motivo
	}
	resultados := make(chan resultado, 2)
	var espera sync.WaitGroup
	for range 2 {
		espera.Go(func() {
			p := pedidoDaAcaoDoPainel(tr, acaoremota.TipoReexecutarAnalise)
			d, m := s.ProcessarPainel(t.Context(), p, "revisora")
			resultados <- resultado{p.IDAcao, d, m}
		})
	}
	espera.Wait()
	close(resultados)
	var aceito string
	for res := range resultados {
		if res.status == acaoremota.DecisaoAceita {
			if aceito != "" {
				t.Fatal("duas reexecuções concorrentes foram aceitas")
			}
			aceito = res.id
		} else if res.motivo != acaoremota.MotivoAnaliseIncompativel {
			t.Fatal(res)
		}
	}
	nova, err := r.BuscarTaskRunPorDeliveryID(t.Context(), aceito)
	if err != nil || nova.Estado != domain.EstadoPendente || nova.ID == tr.ID || nova.SHAHead != tr.SHAHead {
		t.Fatal(nova, err)
	}
	if tipo, _, err := r.UltimoResultadoSandbox(t.Context(), tr.ID); err != nil || tipo < 1 {
		t.Fatal("tentativas da análise anterior devem continuar gravadas", tipo, err)
	}
	if check, err := r.CheckRunDoTaskRun(t.Context(), nova.ID); err != nil || check != 900 {
		t.Fatal("a reexecução deve atualizar o mesmo Check Run", check, err)
	}
	if len(g.reviews) != 0 {
		t.Fatal("reexecução não publica review")
	}
	novaMontagemDoGate(banco, arquivoDeRiscoBaixo).rodarAte(t, nova.ID, domain.EstadoConcluido)
	if _, err := r.BuscarReciboPorTaskRun(t.Context(), nova.ID); err != nil {
		t.Fatal("o worker deve gerar um recibo para a análise nova", err)
	}
	if _, err := r.BuscarReciboPorTaskRun(t.Context(), tr.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatal("a falha anterior deve continuar sem recibo", err)
	}
}

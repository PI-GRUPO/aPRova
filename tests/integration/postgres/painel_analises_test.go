package integracaopostgres

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/repository"
)

func filtroDoPainel(t *testing.T, tenant int64) domain.FiltroAnalisesWeb {
	t.Helper()
	var snapshot time.Time
	if e := pool.QueryRow(t.Context(), "SELECT clock_timestamp()").Scan(&snapshot); e != nil {
		t.Fatal(e)
	}
	return domain.FiltroAnalisesWeb{TenantID: tenant, Repositorios: []string{repositorioDaDemo}, Estado: "todos", Nivel: "todos", Ordem: "recentes", Inicio: instanteDaDemo.AddDate(0, 0, -30), Fim: instanteDaDemo.AddDate(0, 0, 1), Snapshot: snapshot, Pagina: 1, Limite: 2}
}
func listaDoPainel(t *testing.T, r *repository.RepositorioPostgres, f domain.FiltroAnalisesWeb) domain.PaginaAnalisesWeb {
	t.Helper()
	p, e := r.ListarAnalisesWeb(t.Context(), f)
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func TestPainelListaFiltraTodoOEscopoEPaginaSemNovasInsercoes(t *testing.T) {
	r := exigirBanco(t)
	tenant := novoTenant()
	resumo := semearDemo(t, r, tenant)
	semearDemo(t, r, novoTenant())
	f := filtroDoPainel(t, tenant)
	p1 := listaDoPainel(t, r, f)
	if p1.Total != 9 || len(p1.Itens) != 2 || !p1.TemMais {
		t.Fatal(p1)
	}
	// Uma inserção após o snapshot não desloca a segunda página.
	_, _, e := r.CriarTaskRun(t.Context(), domain.NovoTaskRun{TenantID: tenant, DeliveryID: "web-nova-" + resumo.Analises[0].TaskRunID, Repositorio: repositorioDaDemo, NumeroPR: 999, SHAHead: strings.Repeat("b", 40), Autor: "autora", Acao: "opened", RecebidoEm: instanteDaDemo})
	if e != nil {
		t.Fatal(e)
	}
	f.Pagina = 2
	p2 := listaDoPainel(t, r, f)
	if p2.Total != 9 || len(p2.Itens) != 2 {
		t.Fatal(p2)
	}
	for _, a := range p1.Itens {
		for _, b := range p2.Itens {
			if a.ID == b.ID {
				t.Fatal("paginação repetiu item após inserção")
			}
		}
	}
	f.Pagina = 1
	f.NumeroPR = 108
	f.Nivel = "nao_avaliado"
	p := listaDoPainel(t, r, f)
	if p.Total != 1 || len(p.Itens) != 1 || p.Itens[0].Nivel != nil || p.Itens[0].Estado != "falhou" {
		t.Fatal(p)
	}
	f.Estado = "concluido"
	if p := listaDoPainel(t, r, f); p.Total != 0 {
		t.Fatal("filtro de estado não aplicado ao total", p)
	}
	f = filtroDoPainel(t, tenant)
	f.Repositorios = []string{"equipe/privado"}
	if p := listaDoPainel(t, r, f); p.Total != 0 {
		t.Fatal("vazou repositório", p)
	}
}
func TestPainelDetalheAgregaMesmoUUIDENaoReaplicaDecisaoACommitNovo(t *testing.T) {
	r := exigirBanco(t)
	tenant := novoTenant()
	resumo := semearDemo(t, r, tenant)
	anterior := resumo.Analises[0]
	d, e := r.DetalharAnaliseWeb(t.Context(), tenant, []string{repositorioDaDemo}, anterior.TaskRunID)
	if e != nil {
		t.Fatal(e)
	}
	if d.Analise.ID != anterior.TaskRunID || d.Gate == nil || len(d.Etapas) == 0 || len(d.Tentativas) == 0 || d.ReciboID == nil || *d.ReciboID != anterior.ReciboID || !d.Atual || d.Branch != nil || d.RedeIsolada != nil || d.TestesEstruturadosDisponiveis {
		t.Fatalf("detalhe incoerente: %+v", d)
	}
	if e := r.GravarCheckRun(t.Context(), anterior.TaskRunID, 1234); e != nil {
		t.Fatal(e)
	}
	novo, _, e := r.CriarTaskRun(t.Context(), domain.NovoTaskRun{TenantID: tenant, DeliveryID: "novo-commit-" + anterior.TaskRunID, Repositorio: repositorioDaDemo, NumeroPR: anterior.NumeroPR, SHAHead: strings.Repeat("b", 40), Autor: "autora", Acao: "synchronize", RecebidoEm: instanteDaDemo})
	if e != nil {
		t.Fatal(e)
	}
	d, e = r.DetalharAnaliseWeb(t.Context(), tenant, []string{repositorioDaDemo}, anterior.TaskRunID)
	if e != nil || d.Atual || d.URLCheck == nil || !strings.HasSuffix(*d.URLCheck, "/runs/1234") {
		t.Fatal(d, e)
	}
	atual, e := r.DetalharAnaliseWeb(t.Context(), tenant, []string{repositorioDaDemo}, novo.ID)
	if e != nil || atual.Gate != nil || len(atual.Aprovacoes) != 0 || atual.ReciboID != nil || !atual.Atual {
		t.Fatal("commit novo herdou decisão", atual, e)
	}
	for _, escopo := range []struct {
		tenant int64
		repos  []string
	}{{novoTenant(), []string{repositorioDaDemo}}, {tenant, []string{"equipe/privado"}}} {
		if _, e := r.DetalharAnaliseWeb(t.Context(), escopo.tenant, escopo.repos, anterior.TaskRunID); !errors.Is(e, domain.ErrNaoEncontrado) {
			t.Fatal("ID fora do escopo não devolveu inexistente", e)
		}
	}
}
func TestPainelSaidaTruncadaVizinhosPrivadosEAprovacaoSemSHANaoAplicada(t *testing.T) {
	r := exigirBanco(t)
	tenant := novoTenant()
	tr := novoTaskRun(t, r, tenant)
	gate := gateDeExemplo(tr.ID)
	gate.Calibracao = calibracaoDeExemplo()
	if e := r.GravarGateDeRisco(t.Context(), gate); e != nil {
		t.Fatal(e)
	}
	// O registro persistido inclui um caso de outro repositório; o painel o omite.
	if _, e := pool.Exec(t.Context(), `INSERT INTO sandbox_results(tenant_id,task_run_id,attempt,compiled,tests_passed,tests_failed,raw_output,duration_ms,failure_type) VALUES($1,$2,1,true,1,0,$3,100,'nenhuma')`, tenant, tr.ID, strings.Repeat("é", 70000)); e != nil {
		t.Fatal(e)
	}
	if _, e := pool.Exec(t.Context(), `INSERT INTO approvals(tenant_id,task_run_id,delivery_id,decision,reviewer,origin,decided_at) VALUES($1,$2,$3,'aprovada','pessoa','github',now())`, tenant, tr.ID, "sem-sha-"+tr.ID); e != nil {
		t.Fatal(e)
	}
	d, e := r.DetalharAnaliseWeb(t.Context(), tenant, []string{"Azulpasta/alvo"}, tr.ID)
	if e != nil {
		t.Fatal(e)
	}
	if d.Gate == nil || len(d.Gate.Vizinhos) != 1 || d.Gate.Vizinhos[0].Repositorio != "Azulpasta/alvo" {
		t.Fatal("vizinho fora do escopo exposto", d.Gate)
	}
	if len(d.Tentativas) != 1 || !d.Tentativas[0].SaidaTruncada || len([]rune(d.Tentativas[0].Saida)) != 65536 {
		t.Fatal("saída não limitada corretamente")
	}
	if len(d.Aprovacoes) != 1 || d.Aprovacoes[0].AplicavelAoCommit {
		t.Fatal("aprovação sem SHA aplicada", d.Aprovacoes)
	}
}

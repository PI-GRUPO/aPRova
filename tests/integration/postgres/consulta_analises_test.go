package integracaopostgres

import (
	"errors"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

func TestConsultaAnalisesIsolaTenantRepositorioEstadosEPaginacao(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()
	a := taskRunRecebidoEm(t, repo, tenant, time.Now().AddDate(-1, 0, 0))
	b := novoTaskRun(t, repo, tenant)
	percorrer(t, repo, b, domain.EstadoFalhou)
	outro := novoTaskRun(t, repo, novoTenant())
	filtro := domain.FiltroAnalises{TenantID: tenant, Estado: "em_andamento", Limite: 20}
	itens, err := repo.ListarAnalises(t.Context(), filtro)
	if err != nil || len(itens) != 1 || itens[0].ID != a.ID {
		t.Fatalf("não listou a execução antiga ainda em andamento: %+v %v", itens, err)
	}
	filtro.Repositorio = "outro/repositorio"
	itens, err = repo.ListarAnalises(t.Context(), filtro)
	if err != nil || len(itens) != 0 {
		t.Fatal("vazou repositório")
	}
	filtro.Repositorio = "azulpasta/ALVO"
	filtro.Estado = "todos"
	filtro.Limite = 1
	pagina1, err := repo.ListarAnalises(t.Context(), filtro)
	if err != nil || len(pagina1) != 1 {
		t.Fatal(err)
	}
	filtro.Deslocamento = 1
	pagina2, err := repo.ListarAnalises(t.Context(), filtro)
	if err != nil || len(pagina2) != 1 || pagina1[0].ID == pagina2[0].ID {
		t.Fatal("paginação instável")
	}
	filtro.Deslocamento = 0
	filtro.Estado = "falhou"
	itens, err = repo.ListarAnalises(t.Context(), filtro)
	if err != nil || len(itens) != 1 || itens[0].ID != b.ID {
		t.Fatal("filtro de estado falhou")
	}
	filtro.NumeroPR = 999
	itens, err = repo.ListarAnalises(t.Context(), filtro)
	if err != nil || len(itens) != 0 {
		t.Fatal("filtro de PR falhou")
	}
	for _, tc := range []struct {
		tenant   int64
		repo, id string
	}{{tenant, "", outro.ID}, {tenant, "outro/repo", a.ID}} {
		if _, err = repo.ConsultarAnalise(t.Context(), tc.tenant, tc.repo, tc.id); !errors.Is(err, domain.ErrNaoEncontrado) {
			t.Fatalf("leitura fora do escopo: %v", err)
		}
	}
	atual, err := repo.ConsultarAnalise(t.Context(), tenant, "Azulpasta/alvo", a.ID)
	if err != nil || atual.ID != a.ID {
		t.Fatal("leitura válida falhou", err)
	}
	antes, err := repo.BuscarTaskRun(t.Context(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if antes.ReivindicadoPor != "" || antes.Estado != domain.EstadoPendente {
		t.Fatal("consulta alterou a execução")
	}
}

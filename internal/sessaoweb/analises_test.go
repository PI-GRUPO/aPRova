package sessaoweb

import (
	"context"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

type analisesTeste struct {
	chamadas int
	f        domain.FiltroAnalisesWeb
}

func (a *analisesTeste) ListarAnalisesWeb(_ context.Context, f domain.FiltroAnalisesWeb) (domain.PaginaAnalisesWeb, error) {
	a.chamadas++
	a.f = f
	return domain.PaginaAnalisesWeb{InstallationID: f.TenantID, Pagina: f.Pagina, Limite: f.Limite, Snapshot: f.Snapshot, Itens: []domain.AnaliseWeb{}}, nil
}
func (a *analisesTeste) DetalharAnaliseWeb(_ context.Context, tenant int64, repos []string, id string) (domain.DetalheAnaliseWeb, error) {
	a.chamadas++
	return domain.DetalheAnaliseWeb{}, domain.ErrNaoEncontrado
}
func TestAnalisesWebAutorizaListaCatalogoEIDDireto(t *testing.T) {
	s, _, _, _ := cenario(t)
	a := &analisesTeste{}
	s.analises = a
	c := login(t, s)
	req := httptest.NewRequestWithContext(t.Context(), "GET", "https://aprova.test/api/analises?installation_id=42&numero_pr=12&nivel=alto", nil)
	req.AddCookie(c)
	w := httptest.NewRecorder()
	s.ListarAnalises(w, req)
	if w.Code != 200 || a.f.TenantID != 42 || a.f.NumeroPR != 12 || a.f.Nivel != "alto" || len(a.f.Repositorios) != 1 || a.f.Repositorios[0] != "equipe/autorizado" {
		t.Fatal(w.Code, a.f, w.Body.String())
	}
	req = httptest.NewRequestWithContext(t.Context(), "GET", "https://aprova.test/api/analises?installation_id=42&repositorio=equipe/privado", nil)
	req.AddCookie(c)
	w = httptest.NewRecorder()
	s.ListarAnalises(w, req)
	if w.Code != 403 || a.chamadas != 1 {
		t.Fatal("lista consultou repositório privado")
	}
	req = httptest.NewRequestWithContext(t.Context(), "GET", "https://aprova.test/api/analises/id?installation_id=42", nil)
	req.AddCookie(c)
	w = httptest.NewRecorder()
	s.DetalharAnalise(w, req, "00000000-0000-4000-8000-000000000001")
	if w.Code != 404 {
		t.Fatal(w.Code, w.Body.String())
	}
	catalogo := pedir(s, "GET", "/api/repositorios?installation_id=42", c, nil)
	if catalogo.Code != 200 || !strings.Contains(catalogo.Body.String(), "equipe/autorizado") {
		t.Fatal(catalogo.Code, catalogo.Body.String())
	}
}
func TestFiltrosWebValidamAntesDeConsultar(t *testing.T) {
	agora := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	for _, consulta := range []string{"installation_id=42&pagina=2", "installation_id=42&nivel=critico", "installation_id=42&estado=desconhecido", "installation_id=42&limite=51", "installation_id=42&pagina=202", "installation_id=42&numero_pr=-1", "installation_id=42&inicio=2026-11-20&fim=2026-11-10", "installation_id=42&snapshot=2099-01-01T00:00:00Z", "installation_id=42&installation_id=99", "installation_id=42&tenant_id=99"} {
		q, _ := url.ParseQuery(consulta)
		if _, e := filtroWeb(q, agora); e == nil {
			t.Error("filtro inválido aceito", consulta)
		}
	}
	q, _ := url.ParseQuery("installation_id=42&pagina=2&limite=5&snapshot=2026-11-16T11:00:00Z&ordem=antigas")
	f, e := filtroWeb(q, agora)
	if e != nil || f.Pagina != 2 || f.Limite != 5 || f.Ordem != "antigas" {
		t.Fatal(f, e)
	}
}

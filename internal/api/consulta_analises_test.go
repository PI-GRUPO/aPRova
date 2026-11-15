package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

const idAnaliseConsulta = "a2beef00-1111-4222-8333-123456789012"

type fonteAnalisesFalsa struct {
	filtro   domain.FiltroAnalises
	tenant   int64
	repo, id string
	chamadas int
	itens    []domain.TaskRun
	err      error
}

func (f *fonteAnalisesFalsa) ListarAnalises(_ context.Context, q domain.FiltroAnalises) ([]domain.TaskRun, error) {
	f.chamadas++
	f.filtro = q
	return f.itens, f.err
}
func (f *fonteAnalisesFalsa) ConsultarAnalise(_ context.Context, t int64, r, id string) (domain.TaskRun, error) {
	f.chamadas++
	f.tenant = t
	f.repo = r
	f.id = id
	if len(f.itens) > 0 {
		return f.itens[0], f.err
	}
	return domain.TaskRun{}, f.err
}
func pedirAnalises(t *testing.T, fonte FonteDeAnalises, metodo, caminho, assinarComo string) *httptest.ResponseRecorder {
	t.Helper()
	s := NovoServidor(Opcoes{SegredoAcoesRemotas: []byte(segredoDasAcoesDeTeste), ConsultasAnalises: fonte, Agora: func() time.Time { return agoraDeTeste }})
	r := httptest.NewRequest(metodo, caminho, nil)
	if assinarComo != "" {
		ts := strconv.FormatInt(agoraDeTeste.Unix(), 10)
		r.Header.Set("X-Aprova-Timestamp", ts)
		r.Header.Set("X-Aprova-Assinatura", assinarAcao(segredoDasAcoesDeTeste, ts, assinarComo))
	}
	w := httptest.NewRecorder()
	s.Rotas().ServeHTTP(w, r)
	return w
}
func TestConsultaAnalisesAutenticacaoIncluiMetodoCaminhoEQuery(t *testing.T) {
	path := "/api/analises?installation_id=42"
	for _, signed := range []string{"", "installation_id=42", "GET\n/api/analises\ninstallation_id=7", "GET\n/api/analises/" + idAnaliseConsulta + "\ninstallation_id=42"} {
		f := &fonteAnalisesFalsa{}
		w := pedirAnalises(t, f, "GET", path, signed)
		if w.Code != 401 || f.chamadas != 0 {
			t.Fatalf("assinatura inválida chegou à fonte: %d", w.Code)
		}
	}
	f := &fonteAnalisesFalsa{}
	w := pedirAnalises(t, f, "GET", path, "GET\n/api/analises\ninstallation_id=42")
	if w.Code != 200 || f.filtro.TenantID != 42 || f.filtro.Estado != "em_andamento" {
		t.Fatalf("consulta válida: %d %+v", w.Code, f)
	}
}
func TestConsultaAnalisesPaginacaoEscopoERedacao(t *testing.T) {
	a := domain.TaskRun{ID: idAnaliseConsulta, TenantID: 42, Repositorio: "PI-GRUPO/aPRova-TESTES", NumeroPR: 3, SHAHead: strings.Repeat("a", 40), Estado: domain.EstadoAguardandoAprovacao, DeliveryID: "SEGREDO", ReivindicadoPor: "SEGREDO", CriadoEm: agoraDeTeste, RecebidoEm: agoraDeTeste, AtualizadoEm: agoraDeTeste}
	f := &fonteAnalisesFalsa{itens: []domain.TaskRun{a, a}}
	query := "installation_id=42&repositorio=PI-GRUPO%2FaPRova-TESTES&numero_pr=3&estado=aguardando_aprovacao&pagina=2&limite=1"
	w := pedirAnalises(t, f, "GET", "/api/analises?"+query, "GET\n/api/analises\n"+query)
	var p paginaAnalises
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || !p.TemMais || len(p.Itens) != 1 || f.filtro.Deslocamento != 1 || f.filtro.Limite != 2 || f.filtro.Repositorio != a.Repositorio || f.filtro.NumeroPR != 3 {
		t.Fatalf("paginação/escopo incorretos: %d %+v %+v", w.Code, p, f.filtro)
	}
	if strings.Contains(w.Body.String(), "SEGREDO") || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("campos internos expostos ou cache habilitado")
	}
}
func TestConsultaAnalisesVazioNaoSeConfundeComFalha(t *testing.T) {
	path := "/api/analises?installation_id=42"
	sig := "GET\n/api/analises\ninstallation_id=42"
	for _, tc := range []struct {
		f      FonteDeAnalises
		status int
		body   string
	}{{&fonteAnalisesFalsa{}, 200, `"itens":[]`}, {nil, 503, "indisponível"}, {&fonteAnalisesFalsa{err: errors.New("SEGREDO")}, 500, "falha"}} {
		w := pedirAnalises(t, tc.f, "GET", path, sig)
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.body) || strings.Contains(w.Body.String(), "SEGREDO") {
			t.Fatalf("resposta: %d %s", w.Code, w.Body)
		}
	}
}
func TestConsultaAnalisesRejeitaParametrosSemConsultarFonte(t *testing.T) {
	for _, q := range []string{"installation_id=0", "installation_id=42&limite=51", "installation_id=42&pagina=202", "installation_id=42&estado=executar", "installation_id=42&numero_pr=-1", "installation_id=42&repositorio=https://outro", "installation_id=42&installation_id=7", "installation_id=42&url=http://outro"} {
		f := &fonteAnalisesFalsa{}
		w := pedirAnalises(t, f, "GET", "/api/analises?"+q, "GET\n/api/analises\n"+q)
		if w.Code != 400 || f.chamadas != 0 {
			t.Fatalf("aceitou %s: %d", q, w.Code)
		}
	}
}
func TestConsultaAnaliseIndividualEscopadaENaoEncontrada(t *testing.T) {
	p := "/api/analises/" + idAnaliseConsulta
	q := "installation_id=42&repositorio=PI-GRUPO%2FaPRova-TESTES"
	f := &fonteAnalisesFalsa{err: domain.ErrNaoEncontrado}
	w := pedirAnalises(t, f, "GET", p+"?"+q, "GET\n"+p+"\n"+q)
	if w.Code != 404 || f.tenant != 42 || f.repo != "PI-GRUPO/aPRova-TESTES" || f.id != idAnaliseConsulta {
		t.Fatalf("escopo individual: %d %+v", w.Code, f)
	}
	w = pedirAnalises(t, f, "GET", p+"?"+q, "GET\n/api/analises/11111111-1111-4111-8111-111111111111\n"+q)
	if w.Code != 401 {
		t.Fatal("reutilizou assinatura de outro ID")
	}
	f = &fonteAnalisesFalsa{}
	w = pedirAnalises(t, f, "GET", "/api/analises/invalido?installation_id=42", "GET\n/api/analises/invalido\ninstallation_id=42")
	if w.Code != 400 || f.chamadas != 0 {
		t.Fatal("ID inválido consultou a fonte")
	}
}
func TestConsultaAnalisesNaoAceitaMetodosDeEscrita(t *testing.T) {
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
		for _, p := range []string{"/api/analises", "/api/analises/" + idAnaliseConsulta} {
			f := &fonteAnalisesFalsa{}
			w := pedirAnalises(t, f, method, p+"?installation_id=42", method+"\n"+p+"\ninstallation_id=42")
			if w.Code != http.StatusMethodNotAllowed || f.chamadas != 0 {
				t.Fatalf("método %s aceito", method)
			}
		}
	}
}

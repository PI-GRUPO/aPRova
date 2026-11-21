package sessaoweb

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/google/uuid"
)

const parametroInstalacaoWeb = "installation_id"

type Analises interface {
	ListarAnalisesWeb(context.Context, domain.FiltroAnalisesWeb) (domain.PaginaAnalisesWeb, error)
	DetalharAnaliseWeb(context.Context, int64, []string, string) (domain.DetalheAnaliseWeb, error)
}

func cabecalhosPrivados(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}
func parametrosWeb(q url.Values, permitidos ...string) error {
	for k, v := range q {
		if !slices.Contains(permitidos, k) || len(v) != 1 {
			return domain.ErrDadosInvalidos
		}
	}
	return nil
}
func inteiroWeb(q url.Values, nome string, padrao, minimo, maximo int32) (int32, error) {
	if q.Get(nome) == "" {
		return padrao, nil
	}
	v, e := strconv.ParseInt(q.Get(nome), 10, 32)
	if e != nil || v < int64(minimo) || v > int64(maximo) {
		return 0, domain.ErrDadosInvalidos
	}
	return int32(v), nil
}
func enumerado(q url.Values, nome, padrao string, permitidos ...string) (string, error) {
	v := q.Get(nome)
	if v == "" {
		v = padrao
	}
	if !slices.Contains(permitidos, v) {
		return "", domain.ErrDadosInvalidos
	}
	return v, nil
}
func filtroWeb(q url.Values, agora time.Time) (domain.FiltroAnalisesWeb, error) {
	f := domain.FiltroAnalisesWeb{}
	if e := parametrosWeb(q, parametroInstalacaoWeb, "repositorio", "numero_pr", "estado", "nivel", "inicio", "fim", "pagina", "limite", "ordem", "snapshot"); e != nil {
		return f, e
	}
	id, e := instalacaoWeb(q)
	if e != nil || id == 0 {
		return f, domain.ErrDadosInvalidos
	}
	f.TenantID = id
	if e := preencherPagina(q, &f); e != nil {
		return f, e
	}
	if e := preencherFiltros(q, &f); e != nil {
		return f, e
	}
	if e := preencherPeriodo(q, agora, &f); e != nil {
		return f, e
	}
	return f, nil
}
func preencherPagina(q url.Values, f *domain.FiltroAnalisesWeb) error {
	pagina, e := inteiroWeb(q, "pagina", 1, 1, 201)
	if e != nil {
		return e
	}
	limite, e := inteiroWeb(q, "limite", 20, 1, 50)
	if e != nil {
		return e
	}
	pr, e := inteiroWeb(q, "numero_pr", 0, 1, 2147483647)
	if e != nil {
		return e
	}
	f.Pagina = pagina
	f.Limite = limite
	f.NumeroPR = pr
	return nil
}
func preencherFiltros(q url.Values, f *domain.FiltroAnalisesWeb) error {
	estado, e := enumerado(q, "estado", "todos", "todos", "pendente", "mapeando_contexto", "gerando", "testando_sandbox", "avaliando_risco", "aguardando_aprovacao", "aplicando", "concluido", "rejeitado", "falhou")
	if e != nil {
		return e
	}
	nivel, e := enumerado(q, "nivel", "todos", "todos", "baixo", "medio", "alto", "nao_avaliado")
	if e != nil {
		return e
	}
	ordem, e := enumerado(q, "ordem", "recentes", "recentes", "antigas")
	if e != nil {
		return e
	}
	f.Estado = estado
	f.Nivel = nivel
	f.Ordem = ordem
	return nil
}
func preencherPeriodo(q url.Values, agora time.Time, f *domain.FiltroAnalisesWeb) error {
	fim := time.Date(agora.UTC().Year(), agora.UTC().Month(), agora.UTC().Day()+1, 0, 0, 0, 0, time.UTC)
	inicio := fim.AddDate(0, 0, -30)
	var e error
	if q.Get("inicio") != "" {
		inicio, e = time.Parse(time.DateOnly, q.Get("inicio"))
		if e != nil {
			return domain.ErrDadosInvalidos
		}
	}
	if q.Get("fim") != "" {
		fim, e = time.Parse(time.DateOnly, q.Get("fim"))
		if e != nil {
			return domain.ErrDadosInvalidos
		}
	}
	if !fim.After(inicio) || inicio.Year() < 1970 {
		return domain.ErrDadosInvalidos
	}
	f.Inicio = inicio
	f.Fim = fim
	f.Snapshot = agora.UTC()
	if q.Get("snapshot") == "" {
		if f.Pagina > 1 {
			return domain.ErrDadosInvalidos
		}
		return nil
	}
	f.Snapshot, e = time.Parse(time.RFC3339Nano, q.Get("snapshot"))
	if e != nil || f.Snapshot.After(agora) || f.Snapshot.Year() < 1970 {
		return domain.ErrDadosInvalidos
	}
	return nil
}
func (s *Servidor) escopoWeb(w http.ResponseWriter, r *http.Request, id int64) ([]string, bool) {
	v, ok := s.autenticar(w, r)
	if !ok {
		return nil, false
	}
	repos, e := s.repositoriosAutorizados(r.Context(), v.Token, id, r.URL.Query().Get("repositorio"))
	if errors.Is(e, errSemAcesso) {
		falhar(w, 403, "instalação ou repositório não autorizado")
		return nil, false
	}
	if e != nil {
		s.erroProvedor(w, r, e)
		return nil, false
	}
	return repos, true
}

// ListarAnalises é chamado pelas rotas canônicas, sem substituir a autenticação HMAC.
func (s *Servidor) ListarAnalises(w http.ResponseWriter, r *http.Request) {
	cabecalhosPrivados(w)
	// Autentica antes de validar a consulta para não oferecer dados privados a anônimos.
	if _, ok := s.autenticar(w, r); !ok {
		return
	}
	f, e := filtroWeb(r.URL.Query(), s.agora())
	if e != nil {
		falhar(w, 400, "filtros de análise inválidos")
		return
	}
	repos, ok := s.escopoWeb(w, r, f.TenantID)
	if !ok {
		return
	}
	f.Repositorios = repos
	if s.analises == nil {
		falhar(w, 503, "consulta de análises indisponível")
		return
	}
	pagina, e := s.analises.ListarAnalisesWeb(r.Context(), f)
	if e != nil {
		falhar(w, 500, "não foi possível consultar as análises")
		return
	}
	jsonResposta(w, 200, pagina)
}
func (s *Servidor) DetalharAnalise(w http.ResponseWriter, r *http.Request, id string) {
	cabecalhosPrivados(w)
	if _, ok := s.autenticar(w, r); !ok {
		return
	}
	q := r.URL.Query()
	tenant, e := instalacaoWeb(q)
	_, erroID := uuid.Parse(id)
	if e != nil || tenant == 0 || erroID != nil || parametrosWeb(q, parametroInstalacaoWeb, "repositorio") != nil {
		falhar(w, 400, "consulta de análise inválida")
		return
	}
	repos, ok := s.escopoWeb(w, r, tenant)
	if !ok {
		return
	}
	if s.analises == nil {
		falhar(w, 503, "consulta de análises indisponível")
		return
	}
	detalhe, e := s.analises.DetalharAnaliseWeb(r.Context(), tenant, repos, id)
	if errors.Is(e, domain.ErrNaoEncontrado) {
		falhar(w, 404, "análise não encontrada neste escopo")
		return
	}
	if e != nil {
		falhar(w, 500, "não foi possível consultar a análise")
		return
	}
	jsonResposta(w, 200, detalhe)
}
func (s *Servidor) repositorios(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.autenticar(w, r); !ok {
		return
	}
	q := r.URL.Query()
	id, e := instalacaoWeb(q)
	if e != nil || id == 0 || parametrosWeb(q, parametroInstalacaoWeb) != nil {
		falhar(w, 400, "instalação inválida")
		return
	}
	repos, ok := s.escopoWeb(w, r, id)
	if !ok {
		return
	}
	jsonResposta(w, 200, map[string]any{parametroInstalacaoWeb: id, "repositorios": repos})
}

func instalacaoWeb(q url.Values) (int64, error) {
	v, e := strconv.ParseInt(q.Get(parametroInstalacaoWeb), 10, 64)
	if e != nil || v <= 0 {
		return 0, domain.ErrDadosInvalidos
	}
	return v, nil
}

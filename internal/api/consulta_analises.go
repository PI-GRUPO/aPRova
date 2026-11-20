package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// FonteDeAnalises não oferece nenhuma operação de escrita ou execução.
type FonteDeAnalises interface {
	ListarAnalises(context.Context, domain.FiltroAnalises) ([]domain.TaskRun, error)
	ConsultarAnalise(context.Context, int64, string, string) (domain.TaskRun, error)
}

type resumoAnalise struct {
	ID           string               `json:"id"`
	Repositorio  string               `json:"repositorio"`
	NumeroPR     int                  `json:"numero_pr"`
	SHAHead      string               `json:"sha_head"`
	Estado       domain.EstadoTaskRun `json:"estado"`
	RecebidoEm   time.Time            `json:"recebido_em"`
	CriadoEm     time.Time            `json:"criado_em"`
	AtualizadoEm time.Time            `json:"atualizado_em"`
}

type paginaAnalises struct {
	InstallationID int64           `json:"installation_id"`
	ConsultadoEm   time.Time       `json:"consultado_em"`
	Estado         string          `json:"estado"`
	Repositorio    string          `json:"repositorio"`
	NumeroPR       int32           `json:"numero_pr"`
	Pagina         int32           `json:"pagina"`
	Limite         int32           `json:"limite"`
	TemMais        bool            `json:"tem_mais"`
	Itens          []resumoAnalise `json:"itens"`
}

var formatoRepositorioConsulta = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

func resumoDaAnalise(a domain.TaskRun) resumoAnalise {
	return resumoAnalise{a.ID, a.Repositorio, a.NumeroPR, a.SHAHead, a.Estado, a.RecebidoEm, a.CriadoEm, a.AtualizadoEm}
}

// A assinatura das novas consultas inclui método e caminho, além da query.
// Isso impede reutilizar uma assinatura de listagem para ler outro recurso.
func (s *Servidor) autenticarConsultaAnalises(w http.ResponseWriter, r *http.Request) bool {
	conteudo := r.Method + "\n" + r.URL.EscapedPath() + "\n" + r.URL.RawQuery
	if s.motivoParaRecusarAssinaturaDoN8N(r.Header, []byte(conteudo)) != "" {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	w.Header().Set("Cache-Control", "no-store")
	return true
}

func escopoConsultaAnalises(q url.Values) (int64, string, error) {
	tenant, err := strconv.ParseInt(q.Get("installation_id"), 10, 64)
	repo := q.Get("repositorio")
	if err != nil || tenant <= 0 || len(repo) > 200 || (repo != "" && !formatoRepositorioConsulta.MatchString(repo)) {
		return 0, "", domain.ErrDadosInvalidos
	}
	return tenant, repo, nil
}

func inteiroConsulta(q url.Values, nome string, padrao, minimo, maximo int32) (int32, error) {
	if q.Get(nome) == "" {
		return padrao, nil
	}
	valor, err := strconv.ParseInt(q.Get(nome), 10, 32)
	if err != nil || valor < int64(minimo) || valor > int64(maximo) {
		return 0, domain.ErrDadosInvalidos
	}
	return int32(valor), nil
}

func parametrosUnicos(q url.Values, permitidos ...string) bool {
	for nome, valores := range q {
		encontrado := false
		for _, p := range permitidos {
			if nome == p {
				encontrado = true
				break
			}
		}
		if !encontrado || len(valores) != 1 {
			return false
		}
	}
	return true
}

func filtroListaAnalises(q url.Values) (domain.FiltroAnalises, int32, error) {
	tenant, repo, err := escopoConsultaAnalises(q)
	if err != nil {
		return domain.FiltroAnalises{}, 0, err
	}
	pagina, err := inteiroConsulta(q, "pagina", 1, 1, 201)
	if err != nil {
		return domain.FiltroAnalises{}, 0, err
	}
	limite, err := inteiroConsulta(q, "limite", 20, 1, 50)
	if err != nil {
		return domain.FiltroAnalises{}, 0, err
	}
	pr, err := inteiroConsulta(q, "numero_pr", 0, 1, 2147483647)
	if err != nil {
		return domain.FiltroAnalises{}, 0, err
	}
	estado := q.Get("estado")
	if estado == "" {
		estado = "em_andamento"
	}
	if !parametrosUnicos(q, "installation_id", "repositorio", "estado", "numero_pr", "pagina", "limite") || !estadoConsultaValido(estado) {
		return domain.FiltroAnalises{}, 0, domain.ErrDadosInvalidos
	}
	// Os limites validados permitem até 51 itens e deslocamento máximo de 10000.
	return domain.FiltroAnalises{TenantID: tenant, Repositorio: repo, Estado: estado, NumeroPR: pr, Limite: limite + 1, Deslocamento: (pagina - 1) * limite}, pagina, nil
}

func (s *Servidor) manipularListaAnalises(w http.ResponseWriter, r *http.Request) {
	if s.sessaoWeb != nil && r.Header.Get("X-Aprova-Assinatura") == "" {
		s.sessaoWeb.ListarAnalises(w, r)
		return
	}
	if !s.autenticarConsultaAnalises(w, r) {
		return
	}
	filtro, pagina, err := filtroListaAnalises(r.URL.Query())
	if err != nil {
		responderJSON(w, http.StatusBadRequest, respostaSimples{Status: "parâmetros de consulta inválidos"})
		return
	}
	limite := filtro.Limite - 1
	if s.consultasAnalises == nil {
		responderJSON(w, http.StatusServiceUnavailable, respostaSimples{Status: "consulta indisponível"})
		return
	}
	itens, err := s.consultasAnalises.ListarAnalises(r.Context(), filtro)
	if err != nil {
		slog.ErrorContext(r.Context(), "falha ao consultar análises", "erro", err)
		responderJSON(w, http.StatusInternalServerError, respostaSimples{Status: "falha ao consultar análises"})
		return
	}
	temMais := len(itens) > int(limite)
	if temMais {
		itens = itens[:limite]
	}
	resumos := make([]resumoAnalise, 0, len(itens))
	for _, item := range itens {
		resumos = append(resumos, resumoDaAnalise(item))
	}
	responderJSON(w, http.StatusOK, paginaAnalises{filtro.TenantID, s.agora().UTC(), filtro.Estado, filtro.Repositorio, filtro.NumeroPR, pagina, limite, temMais, resumos})
}

func estadoConsultaValido(estado string) bool {
	switch estado {
	case "em_andamento", "todos", "pendente", "mapeando_contexto", "gerando", "testando_sandbox", "avaliando_risco", "aguardando_aprovacao", "aplicando", "concluido", "rejeitado", "falhou":
		return true
	}
	return false
}

func (s *Servidor) manipularConsultaAnalise(w http.ResponseWriter, r *http.Request) {
	if s.sessaoWeb != nil && r.Header.Get("X-Aprova-Assinatura") == "" {
		s.sessaoWeb.DetalharAnalise(w, r, chi.URLParam(r, "id"))
		return
	}
	if !s.autenticarConsultaAnalises(w, r) {
		return
	}
	tenant, repo, err := escopoConsultaAnalises(r.URL.Query())
	id := chi.URLParam(r, "id")
	if _, e := uuid.Parse(id); err != nil || e != nil || !parametrosUnicos(r.URL.Query(), "installation_id", "repositorio") {
		responderJSON(w, http.StatusBadRequest, respostaSimples{Status: "parâmetros de consulta inválidos"})
		return
	}
	if s.consultasAnalises == nil {
		responderJSON(w, http.StatusServiceUnavailable, respostaSimples{Status: "consulta indisponível"})
		return
	}
	item, err := s.consultasAnalises.ConsultarAnalise(r.Context(), tenant, repo, id)
	if errors.Is(err, domain.ErrNaoEncontrado) {
		responderJSON(w, http.StatusNotFound, respostaSimples{Status: "análise não encontrada neste escopo"})
		return
	}
	if err != nil {
		slog.ErrorContext(r.Context(), "falha ao consultar análise", "erro", err)
		responderJSON(w, http.StatusInternalServerError, respostaSimples{Status: "falha ao consultar análise"})
		return
	}
	responderJSON(w, http.StatusOK, struct {
		InstallationID int64         `json:"installation_id"`
		ConsultadoEm   time.Time     `json:"consultado_em"`
		Analise        resumoAnalise `json:"analise"`
	}{tenant, s.agora().UTC(), resumoDaAnalise(item)})
}

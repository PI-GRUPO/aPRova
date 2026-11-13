package publicacao

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/platform"
)

const slugDeTeste = "a-prova"

type checkRunNoGitHub struct {
	ID          int64  `json:"id"`
	Nome        string `json:"name"`
	SHAHead     string `json:"head_sha"`
	IDExterno   string `json:"external_id"`
	Situacao    string `json:"status"`
	Conclusao   string `json:"conclusion,omitempty"`
	URLDetalhes string `json:"details_url,omitempty"`
	Saida       struct {
		Titulo string `json:"title"`
		Resumo string `json:"summary"`
	} `json:"output"`
}

type autorDoComentario struct {
	Login string `json:"login"`
	Tipo  string `json:"type"`
}

type comentarioNoGitHub struct {
	ID     int64             `json:"id"`
	Numero int               `json:"-"`
	Corpo  string            `json:"body"`
	Autor  autorDoComentario `json:"user"`
}

type falhaProgramada struct {
	metodo  string
	trecho  string
	status  int
	quantas int
}

type githubFalso struct {
	t        *testing.T
	mutex    sync.Mutex
	proximo  int64
	checks   []*checkRunNoGitHub
	coments  []*comentarioNoGitHub
	falhas   []*falhaProgramada
	pedidos  []string
	servidor *httptest.Server
}

func novoGitHubFalso(t *testing.T) *githubFalso {
	t.Helper()

	g := &githubFalso{t: t, proximo: 1000}
	g.servidor = httptest.NewServer(http.HandlerFunc(g.atender))
	t.Cleanup(g.servidor.Close)
	return g
}

func (g *githubFalso) cliente() *platform.Cliente {
	return platform.NovoCliente(platform.OpcoesCliente{Tokens: tokenFixo{}, URLBase: g.servidor.URL, ClienteHTTP: g.servidor.Client()})
}

type tokenFixo struct{}

func (tokenFixo) TokenParaInstalacao(context.Context, int64) (string, error) {
	return "ghs_token-de-teste", nil
}

func (g *githubFalso) falharEm(metodo, trecho string, status, quantas int) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.falhas = append(g.falhas, &falhaProgramada{metodo: metodo, trecho: trecho, status: status, quantas: quantas})
}

func (g *githubFalso) checkRunExistente(sha, externalID string) *checkRunNoGitHub {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.proximo++
	check := &checkRunNoGitHub{ID: g.proximo, Nome: "aPRova", SHAHead: sha, IDExterno: externalID, Situacao: "in_progress"}
	g.checks = append(g.checks, check)
	return check
}

func (g *githubFalso) comentarioExistente(numero int, login, corpo string) *comentarioNoGitHub {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.proximo++
	tipo := "User"
	if strings.HasSuffix(login, "[bot]") {
		tipo = "Bot"
	}
	comentario := &comentarioNoGitHub{ID: g.proximo, Numero: numero, Corpo: corpo, Autor: autorDoComentario{Login: login, Tipo: tipo}}
	g.coments = append(g.coments, comentario)
	return comentario
}

func (g *githubFalso) checkRuns() []checkRunNoGitHub {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	copia := make([]checkRunNoGitHub, 0, len(g.checks))
	for _, check := range g.checks {
		copia = append(copia, *check)
	}
	return copia
}

func (g *githubFalso) comentarios() []comentarioNoGitHub {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	copia := make([]comentarioNoGitHub, 0, len(g.coments))
	for _, comentario := range g.coments {
		copia = append(copia, *comentario)
	}
	return copia
}

func (g *githubFalso) quantosPedidos(metodo, trecho string) int {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	quantos := 0
	for _, pedido := range g.pedidos {
		if strings.HasPrefix(pedido, metodo+" ") && strings.Contains(pedido, trecho) {
			quantos++
		}
	}
	return quantos
}

func (g *githubFalso) atender(w http.ResponseWriter, r *http.Request) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.pedidos = append(g.pedidos, r.Method+" "+r.URL.Path)
	if status, falhou := g.falhaPara(r); falhou {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"message":"falha programada"}`))
		return
	}

	partes := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	switch {
	case len(partes) == 4 && partes[3] == "check-runs" && r.Method == http.MethodPost:
		g.criarCheckRun(w, r)
	case len(partes) == 5 && partes[3] == "check-runs" && r.Method == http.MethodPatch:
		g.atualizarCheckRun(w, r, partes[4])
	case len(partes) == 6 && partes[3] == "commits" && partes[5] == "check-runs" && r.Method == http.MethodGet:
		g.listarCheckRuns(w, r, partes[4])
	case len(partes) == 6 && partes[3] == "issues" && partes[5] == "comments" && r.Method == http.MethodPost:
		g.criarComentario(w, r, partes[4])
	case len(partes) == 6 && partes[3] == "issues" && partes[5] == "comments" && r.Method == http.MethodGet:
		g.listarComentarios(w, r, partes[4])
	case len(partes) == 6 && partes[3] == "issues" && partes[4] == "comments" && r.Method == http.MethodPatch:
		g.atualizarComentario(w, r, partes[5])
	default:
		g.t.Errorf("chamada inesperada ao GitHub: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}
}

func (g *githubFalso) falhaPara(r *http.Request) (int, bool) {
	for _, falha := range g.falhas {
		if falha.quantas > 0 && falha.metodo == r.Method && strings.Contains(r.URL.Path, falha.trecho) {
			falha.quantas--
			return falha.status, true
		}
	}
	return 0, false
}

func (g *githubFalso) responder(w http.ResponseWriter, status int, corpo any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(corpo)
}

func (g *githubFalso) criarCheckRun(w http.ResponseWriter, r *http.Request) {
	var novo checkRunNoGitHub
	if err := json.NewDecoder(r.Body).Decode(&novo); err != nil {
		g.t.Errorf("corpo do check run ilegível: %v", err)
	}
	g.proximo++
	novo.ID = g.proximo
	g.checks = append(g.checks, &novo)
	g.responder(w, http.StatusCreated, novo)
}

func (g *githubFalso) atualizarCheckRun(w http.ResponseWriter, r *http.Request, idTexto string) {
	id, _ := strconv.ParseInt(idTexto, 10, 64)
	for _, check := range g.checks {
		if check.ID != id {
			continue
		}
		var mudanca map[string]json.RawMessage
		corpo, _ := json.Marshal(check)
		_ = json.NewDecoder(r.Body).Decode(&mudanca)
		var atual map[string]json.RawMessage
		_ = json.Unmarshal(corpo, &atual)
		for chave, valor := range mudanca {
			atual[chave] = valor
		}
		mesclado, _ := json.Marshal(atual)
		_ = json.Unmarshal(mesclado, check)
		g.responder(w, http.StatusOK, check)
		return
	}
	g.responder(w, http.StatusNotFound, map[string]string{"message": "Not Found"})
}

func (g *githubFalso) listarCheckRuns(w http.ResponseWriter, r *http.Request, sha string) {
	nome := r.URL.Query().Get("check_name")
	encontrados := []*checkRunNoGitHub{}
	for _, check := range g.checks {
		if check.SHAHead == sha && (nome == "" || check.Nome == nome) {
			encontrados = append(encontrados, check)
		}
	}
	g.responder(w, http.StatusOK, map[string]any{"total_count": len(encontrados), "check_runs": encontrados})
}

func (g *githubFalso) criarComentario(w http.ResponseWriter, r *http.Request, numeroTexto string) {
	var corpo struct {
		Body string `json:"body"`
	}
	_ = json.NewDecoder(r.Body).Decode(&corpo)
	numero, _ := strconv.Atoi(numeroTexto)
	g.proximo++
	comentario := &comentarioNoGitHub{ID: g.proximo, Numero: numero, Corpo: corpo.Body, Autor: autorDoComentario{Login: slugDeTeste + "[bot]", Tipo: "Bot"}}
	g.coments = append(g.coments, comentario)
	g.responder(w, http.StatusCreated, comentario)
}

func (g *githubFalso) listarComentarios(w http.ResponseWriter, r *http.Request, numeroTexto string) {
	numero, _ := strconv.Atoi(numeroTexto)
	porPagina, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if porPagina <= 0 {
		porPagina = 30
	}
	pagina, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pagina = max(pagina, 1)

	doPR := []*comentarioNoGitHub{}
	for _, comentario := range g.coments {
		if comentario.Numero == numero {
			doPR = append(doPR, comentario)
		}
	}
	inicio := min((pagina-1)*porPagina, len(doPR))
	fim := min(inicio+porPagina, len(doPR))
	if fim < len(doPR) {
		w.Header().Set("Link", fmt.Sprintf(`<%s%s?per_page=%d&page=%d>; rel="next"`, g.servidor.URL, r.URL.Path, porPagina, pagina+1))
	}
	g.responder(w, http.StatusOK, doPR[inicio:fim])
}

func (g *githubFalso) atualizarComentario(w http.ResponseWriter, r *http.Request, idTexto string) {
	id, _ := strconv.ParseInt(idTexto, 10, 64)
	var corpo struct {
		Body string `json:"body"`
	}
	_ = json.NewDecoder(r.Body).Decode(&corpo)
	for _, comentario := range g.coments {
		if comentario.ID == id {
			comentario.Corpo = corpo.Body
			g.responder(w, http.StatusOK, comentario)
			return
		}
	}
	g.responder(w, http.StatusNotFound, map[string]string{"message": "Not Found"})
}

type chaveDoPR struct {
	tenant      int64
	repositorio string
	numero      int
}

type registroFalso struct {
	mutex       sync.Mutex
	checkRuns   map[string]int64
	comentarios map[chaveDoPR]int64
}

func novoRegistroFalso() *registroFalso {
	return &registroFalso{checkRuns: map[string]int64{}, comentarios: map[chaveDoPR]int64{}}
}

func (r *registroFalso) CheckRunDoTaskRun(_ context.Context, taskRunID string) (int64, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	id, gravado := r.checkRuns[taskRunID]
	if !gravado {
		return 0, domain.ErrNaoEncontrado
	}
	return id, nil
}

func (r *registroFalso) GravarCheckRun(_ context.Context, taskRunID string, checkRunID int64) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.checkRuns[taskRunID] = checkRunID
	return nil
}

func (r *registroFalso) ComentarioDoPR(_ context.Context, tenantID int64, repositorio string, numeroPR int) (int64, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	id, gravado := r.comentarios[chaveDoPR{tenantID, repositorio, numeroPR}]
	if !gravado {
		return 0, domain.ErrNaoEncontrado
	}
	return id, nil
}

func (r *registroFalso) GravarComentario(_ context.Context, tenantID int64, repositorio string, numeroPR int, comentarioID int64) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.comentarios[chaveDoPR{tenantID, repositorio, numeroPR}] = comentarioID
	return nil
}

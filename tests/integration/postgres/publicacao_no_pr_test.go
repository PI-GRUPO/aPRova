package integracaopostgres

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/platform"
	"github.com/Azulpasta/aPRova/internal/publicacao"
	"github.com/Azulpasta/aPRova/internal/worker"
)

const trabalhadorDaPublicacao = "trabalhador-da-publicacao"

type checkRunPublicado struct {
	ID          int64  `json:"id"`
	Nome        string `json:"name"`
	SHAHead     string `json:"head_sha"`
	IDExterno   string `json:"external_id"`
	Situacao    string `json:"status"`
	Conclusao   string `json:"conclusion"`
	URLDetalhes string `json:"details_url"`
	Saida       struct {
		Titulo string `json:"title"`
	} `json:"output"`
}

type comentarioPublicado struct {
	ID    int64             `json:"id"`
	Corpo string            `json:"body"`
	Autor map[string]string `json:"user"`
}

type githubDaPublicacao struct {
	mutex       sync.Mutex
	proximo     int64
	checkRuns   []*checkRunPublicado
	comentarios []*comentarioPublicado
}

func (g *githubDaPublicacao) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	partes := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	switch {
	case partes[len(partes)-1] == "check-runs" && r.Method == http.MethodPost:
		var novo checkRunPublicado
		_ = json.NewDecoder(r.Body).Decode(&novo)
		g.proximo++
		novo.ID = g.proximo
		g.checkRuns = append(g.checkRuns, &novo)
		_ = json.NewEncoder(w).Encode(novo)
	case partes[len(partes)-2] == "check-runs" && r.Method == http.MethodPatch:
		atual := g.checkRuns[len(g.checkRuns)-1]
		for _, check := range g.checkRuns {
			if fmt.Sprint(check.ID) == partes[len(partes)-1] {
				atual = check
			}
		}
		_ = json.NewDecoder(r.Body).Decode(atual)
		_ = json.NewEncoder(w).Encode(atual)
	case partes[len(partes)-1] == "check-runs" && r.Method == http.MethodGet:
		_ = json.NewEncoder(w).Encode(map[string]any{"check_runs": g.checkRuns})
	case partes[len(partes)-1] == "comments" && r.Method == http.MethodPost:
		var novo comentarioPublicado
		_ = json.NewDecoder(r.Body).Decode(&novo)
		g.proximo++
		novo.ID, novo.Autor = g.proximo, map[string]string{"login": "a-prova[bot]"}
		g.comentarios = append(g.comentarios, &novo)
		_ = json.NewEncoder(w).Encode(novo)
	case partes[len(partes)-1] == "comments" && r.Method == http.MethodGet:
		_ = json.NewEncoder(w).Encode(g.comentarios)
	case partes[len(partes)-2] == "comments" && r.Method == http.MethodPatch:
		for _, comentario := range g.comentarios {
			if fmt.Sprint(comentario.ID) == partes[len(partes)-1] {
				_ = json.NewDecoder(r.Body).Decode(comentario)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (g *githubDaPublicacao) estado() ([]checkRunPublicado, []comentarioPublicado) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	checks := make([]checkRunPublicado, 0, len(g.checkRuns))
	for _, check := range g.checkRuns {
		checks = append(checks, *check)
	}
	comentarios := make([]comentarioPublicado, 0, len(g.comentarios))
	for _, comentario := range g.comentarios {
		comentarios = append(comentarios, *comentario)
	}
	return checks, comentarios
}

type tokenDaPublicacao struct{}

func (tokenDaPublicacao) TokenParaInstalacao(context.Context, int64) (string, error) {
	return "ghs_token-de-teste", nil
}

func motorQuePublicaNoPR(t *testing.T, banco bancoDaFila, nivel domain.NivelRisco) (*worker.Motor, *githubDaPublicacao) {
	t.Helper()

	github := &githubDaPublicacao{proximo: 7_000_000_000}
	servidor := httptest.NewServer(github)
	t.Cleanup(servidor.Close)

	cliente := platform.NovoCliente(platform.OpcoesCliente{Tokens: tokenDaPublicacao{}, URLBase: servidor.URL, ClienteHTTP: servidor.Client()})
	publicador, err := publicacao.NovoPublicador(publicacao.OpcoesPublicador{
		GitHub: cliente, Registro: banco.repo, SlugDoApp: "a-prova", URLPublica: "https://aprova.example.com",
	})
	if err != nil {
		t.Fatalf("montar o publicador: %v", err)
	}

	motor, err := worker.NovoMotor(worker.Opcoes{
		Armazenamento: banco.repo,
		LeitorDePR:    leitorDePRFixo{},
		Permissoes:    permissaoDeAdmin{},
		Sandbox:       &sandboxContado{},
		Avaliador:     avaliadorFixo{nivel: nivel},
		Recibos:       emissorDeRecibos(t, banco.repo),
		Publicador:    publicador,
		Eventos:       eventosDescartados{},
		URLPublica:    "https://aprova.example.com",
		EsperaBase:    time.Millisecond,
	})
	if err != nil {
		t.Fatalf("montar o motor: %v", err)
	}
	return motor, github
}

func conduzirAteParar(t *testing.T, banco bancoDaFila, motor *worker.Motor) {
	t.Helper()

	for range 20 {
		taskRun, encontrado, err := banco.repo.Reivindicar(t.Context(), trabalhadorDaPublicacao, time.Minute)
		if err != nil {
			t.Fatalf("reivindicar: %v", err)
		}
		if !encontrado {
			return
		}
		desfecho := motor.Conduzir(t.Context(), trabalhadorDaPublicacao, taskRun, nil)
		if desfecho.Liberar {
			if err := banco.repo.Liberar(t.Context(), taskRun.ID, trabalhadorDaPublicacao, 0); err != nil {
				t.Fatalf("liberar: %v", err)
			}
		}
	}
	t.Fatal("o motor não parou em 20 reivindicações")
}

func exigirIdsGravados(t *testing.T, banco bancoDaFila, taskRun domain.TaskRun, check checkRunPublicado, comentario comentarioPublicado) {
	t.Helper()

	if id, err := banco.repo.CheckRunDoTaskRun(t.Context(), taskRun.ID); err != nil || id != check.ID {
		t.Errorf("check_run_id gravado = %d (%v), esperava %d", id, err, check.ID)
	}
	if id, err := banco.repo.ComentarioDoPR(t.Context(), taskRun.TenantID, taskRun.Repositorio, taskRun.NumeroPR); err != nil || id != comentario.ID {
		t.Errorf("comment_id gravado = %d (%v), esperava %d", id, err, comentario.ID)
	}
}

func TestFluxoDeRiscoBaixoPublicaNoPRGravaOsIdsEConclui(t *testing.T) {
	banco := novoBancoDaFila(t)
	motor, github := motorQuePublicaNoPR(t, banco, domain.RiscoBaixo)
	taskRun := banco.novoTaskRunDoPR(t, 14)

	conduzirAteParar(t, banco, motor)

	if estado := banco.estadoDe(t, taskRun.ID); estado != domain.EstadoConcluido {
		t.Fatalf("risco baixo termina concluido, está em %s", estado)
	}
	checks, comentarios := github.estado()
	if len(checks) != 1 || len(comentarios) != 1 {
		t.Fatalf("um Check Run e um comentário: %d e %d", len(checks), len(comentarios))
	}
	if checks[0].Nome != publicacao.NomeDoCheckRun || checks[0].IDExterno != taskRun.ID || checks[0].Situacao != "completed" || checks[0].Conclusao != "success" {
		t.Errorf("Check Run = %+v", checks[0])
	}
	exigirIdsGravados(t, banco, taskRun, checks[0], comentarios[0])
}

func TestFluxoDeRiscoAltoBloqueiaEDepoisDaAprovacaoLiberaEConclui(t *testing.T) {
	banco := novoBancoDaFila(t)
	motor, github := motorQuePublicaNoPR(t, banco, domain.RiscoAlto)
	taskRun := banco.novoTaskRunDoPR(t, 14)

	conduzirAteParar(t, banco, motor)

	if estado := banco.estadoDe(t, taskRun.ID); estado != domain.EstadoAguardandoAprovacao {
		t.Fatalf("risco alto para em aguardando_aprovacao, está em %s", estado)
	}
	checks, _ := github.estado()
	if len(checks) != 1 || checks[0].Conclusao != "action_required" || !strings.HasSuffix(checks[0].URLDetalhes, "/verify") {
		t.Fatalf("o bloqueio é action_required com o link da verificação: %+v", checks)
	}

	err := banco.repo.RegistrarAprovacao(t.Context(), domain.Aprovacao{
		TaskRunID: taskRun.ID, DeliveryID: fmt.Sprintf("review-%d", time.Now().UnixNano()), Decisao: domain.DecisaoAprovada,
		Revisor: "httpslua", Origem: domain.OrigemGitHub, DecididaEm: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("registrar a aprovação: %v", err)
	}
	conduzirAteParar(t, banco, motor)

	if estado := banco.estadoDe(t, taskRun.ID); estado != domain.EstadoConcluido {
		t.Fatalf("aprovado, o task_run termina concluido; está em %s", estado)
	}
	checks, comentarios := github.estado()
	if len(checks) != 1 || len(comentarios) != 1 {
		t.Fatalf("a aprovação atualiza o mesmo Check Run e o mesmo comentário: %d e %d", len(checks), len(comentarios))
	}
	if checks[0].Conclusao != "success" || !strings.Contains(checks[0].Saida.Titulo, "aprovado por @httpslua") {
		t.Errorf("depois da aprovação o check libera o merge com o login: %+v", checks[0])
	}
	if !strings.Contains(comentarios[0].Corpo, "aprovado por @httpslua") {
		t.Errorf("o comentário diz quem aprovou: %s", comentarios[0].Corpo)
	}
	exigirIdsGravados(t, banco, taskRun, checks[0], comentarios[0])
}

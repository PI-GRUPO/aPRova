package rodada

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/benchmark/agente"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/conferencia"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/oraculo"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/remoto"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/resultado"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/suite"
)

const idDoRecibo = "11111111-2222-3333-4444-555555555555"

type githubFalso struct {
	mu          sync.Mutex
	contador    int
	refs        map[string]string
	prs         []string
	concluir    bool
	requisicoes int
}

func (g *githubFalso) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.requisicoes++
	var corpo map[string]any
	bruto, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(bruto, &corpo)
	responder := func(status int, valor any) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(valor)
	}
	novoSHA := func() string {
		g.contador++
		return fmt.Sprintf("%040d", g.contador)
	}
	switch {
	case strings.HasSuffix(r.URL.Path, "/git/blobs"), strings.HasSuffix(r.URL.Path, "/git/trees"):
		responder(http.StatusCreated, map[string]any{"sha": novoSHA()})
	case strings.HasSuffix(r.URL.Path, "/git/commits"):
		responder(http.StatusCreated, map[string]any{"sha": novoSHA()})
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/refs"):
		ref, _ := corpo["ref"].(string)
		sha, _ := corpo["sha"].(string)
		g.refs[ref] = sha
		responder(http.StatusCreated, map[string]any{})
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/pulls"):
		var existentes []map[string]any
		for i, head := range g.prs {
			if strings.HasSuffix(r.URL.Query().Get("head"), ":"+head) {
				existentes = append(existentes, map[string]any{"number": i + 1, "html_url": fmt.Sprintf("https://github.com/dono/bench/pull/%d", i+1)})
			}
		}
		responder(http.StatusOK, existentes)
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/pulls"):
		head, _ := corpo["head"].(string)
		g.prs = append(g.prs, head)
		responder(http.StatusCreated, map[string]any{"number": len(g.prs), "html_url": fmt.Sprintf("https://github.com/dono/bench/pull/%d", len(g.prs))})
	case strings.HasSuffix(r.URL.Path, "/check-runs"):
		if !g.concluir {
			responder(http.StatusOK, map[string]any{"check_runs": []any{map[string]any{"status": "in_progress"}}})
			return
		}
		responder(http.StatusOK, map[string]any{"check_runs": []any{map[string]any{
			"status": "completed", "conclusion": "action_required", "output": map[string]any{"title": "Risco alto"},
			"details_url": "http://aprova.local/receipts/" + idDoRecibo + "/verify",
		}}})
	default:
		http.Error(w, "rota desconhecida", http.StatusNotFound)
	}
}

type agenteFalso struct {
	escrever  map[string]string
	apagar    []string
	desfecho  agente.Desfecho
	execucoes *int
}

func (a agenteFalso) Executar(_ context.Context, pedido agente.Pedido) (agente.Execucao, error) {
	*a.execucoes++
	for nome, conteudo := range a.escrever {
		_ = os.WriteFile(filepath.Join(pedido.Diretorio, nome), []byte(conteudo), 0o600)
	}
	for _, nome := range a.apagar {
		_ = os.Remove(filepath.Join(pedido.Diretorio, nome))
	}
	desfecho := a.desfecho
	if desfecho == "" {
		desfecho = agente.DesfechoConcluida
	}
	return agente.Execucao{
		Desfecho: desfecho, Duracao: 3 * time.Second, Turnos: 4, ModeloPrincipal: "claude-sonnet-5",
		Modelos:         map[string]agente.UsoDoModelo{"claude-sonnet-5": {TokensDeSaida: 900}, "claude-haiku-4-5-20251001": {TokensDeSaida: 16}},
		TokensDeEntrada: 2300, TokensDeSaida: 916,
	}, nil
}

type oraculoFalso struct {
	veredito resultado.Oraculo
	falhar   bool
	julgadas *int
}

func (o oraculoFalso) Julgar(context.Context, oraculo.Entrada) (oraculo.Veredito, error) {
	*o.julgadas++
	if o.falhar {
		return oraculo.Veredito{}, errors.New("o executor caiu")
	}
	return oraculo.Veredito{Oraculo: o.veredito, Duracao: 2 * time.Second}, nil
}

type recibosFalsos struct{}

func (recibosFalsos) Recibo(_ context.Context, id string) (remoto.Recibo, error) {
	if id != idDoRecibo {
		return remoto.Recibo{}, errors.New("recibo desconhecido")
	}
	return remoto.Recibo{ID: id, TaskRunID: "tr-1", Hash: "abc123", Nivel: "alto", Motivos: []string{"falha_sandbox"}}, nil
}

type cenario struct {
	rodada    Rodada
	github    *githubFalso
	execucoes *int
	julgadas  *int
	agente    *agenteFalso
	oraculo   *oraculoFalso
}

func novoCenario(t *testing.T, grupo resultado.Grupo) *cenario {
	t.Helper()
	raizDaSuite := t.TempDir()
	arquivos := filepath.Join(raizDaSuite, "go-a", "arquivos")
	if err := os.MkdirAll(arquivos, 0o750); err != nil {
		t.Fatal(err)
	}
	for nome, conteudo := range map[string]string{"go.mod": "module a\n", "a.go": "package a // stub\n", "a_test.go": "package a // teste\n"} {
		if err := os.WriteFile(filepath.Join(arquivos, nome), []byte(conteudo), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(raizDaSuite, "go-a", "enunciado.md"), []byte("# A\n\nImplemente a soma.\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	tarefa := suite.Tarefa{ID: "go-a", Grupo: grupo, Linguagem: "go", ArquivosDaSolucao: []string{"a.go"}, ArquivosProtegidos: []string{"a_test.go"}}
	if grupo == resultado.GrupoRisco {
		tarefa.Esperado = suite.Esperado{Nivel: "alto", Motivo: "configuracao_ci"}
	}

	github := &githubFalso{refs: map[string]string{}, concluir: true}
	servidor := httptest.NewServer(github)
	t.Cleanup(servidor.Close)

	execucoes, julgadas := 0, 0
	falso := &agenteFalso{escrever: map[string]string{"a.go": "package a // solução\n"}, execucoes: &execucoes}
	juiz := &oraculoFalso{veredito: resultado.OraculoPassou, julgadas: &julgadas}
	c := &cenario{github: github, execucoes: &execucoes, julgadas: &julgadas, agente: falso, oraculo: juiz}
	c.rodada = Rodada{
		Suite:               suite.Suite{Tarefas: []suite.Tarefa{tarefa}},
		RaizDaSuite:         raizDaSuite,
		Resultados:          t.TempDir(),
		Trabalho:            t.TempDir(),
		Agentes:             func(string) agente.Agente { return *c.agente },
		GitHub:              &remoto.GitHub{URLBase: servidor.URL, Token: "t", Cliente: servidor.Client()},
		Recibos:             recibosFalsos{},
		Oraculo:             julgadorDoCenario{c},
		Segredos:            conferencia.SegredosDoEnv([]byte("DATABASE_URL=postgres://aprova:segredo-local@localhost/aprova\n")),
		Repositorios:        map[string]string{"go": "dono/bench-go"},
		PrazoDoAgente:       time.Minute,
		PrazoDaDecisao:      50 * time.Millisecond,
		IntervaloDeConsulta: 5 * time.Millisecond,
	}
	return c
}

type julgadorDoCenario struct{ c *cenario }

func (j julgadorDoCenario) Julgar(ctx context.Context, entrada oraculo.Entrada) (oraculo.Veredito, error) {
	return j.c.oraculo.Julgar(ctx, entrada)
}

func lerResultado(t *testing.T, c *cenario) resultado.Tarefa {
	t.Helper()
	tarefas, err := resultado.LerDiretorio(c.rodada.Resultados)
	if err != nil || len(tarefas) != 1 {
		t.Fatalf("esperava um resultado gravado: %v, %v", tarefas, err)
	}
	return tarefas[0]
}

func TestFluxoCompletoDeUmaTarefa(t *testing.T) {
	c := novoCenario(t, resultado.GrupoCorrecao)

	if err := c.rodada.Executar(t.Context()); err != nil {
		t.Fatal(err)
	}

	tarefa := lerResultado(t, c)
	if tarefa.Desfecho != resultado.DesfechoDecidido || tarefa.PR == nil || tarefa.PR.Numero != 1 ||
		!strings.HasSuffix(tarefa.PR.URL, "/pull/1") || tarefa.PR.Branch != "solucao/go-a" {
		t.Errorf("PR aberto e decisão coletada: %+v", tarefa)
	}
	if tarefa.Decisao == nil || tarefa.Decisao.Nivel != "alto" || tarefa.Decisao.ReciboID != idDoRecibo ||
		tarefa.Decisao.ReciboHash != "abc123" || tarefa.Decisao.Motivos[0] != "falha_sandbox" {
		t.Errorf("decisão vinda do recibo: %+v", tarefa.Decisao)
	}
	if tarefa.Oraculo != resultado.OraculoPassou || tarefa.Tempos.AgenteMS != 3000 || tarefa.Tempos.OraculoMS != 2000 {
		t.Errorf("oráculo e tempos: %+v", tarefa)
	}
	if tarefa.Agente.ModeloPrincipal != "claude-sonnet-5" || len(tarefa.Agente.Modelos) != 2 || tarefa.Agente.TokensDeSaida != 916 ||
		tarefa.ExecucoesDoAgente != 1 {
		t.Errorf("uso do agente registrado: %+v", tarefa.Agente)
	}
	if c.github.refs["refs/heads/base/go-a"] == "" || c.github.refs["refs/heads/solucao/go-a"] == "" {
		t.Errorf("as duas branches existem: %v", c.github.refs)
	}
}

func TestRetomadaNaoRodaOAgenteDeNovoNemDuplicaOPR(t *testing.T) {
	c := novoCenario(t, resultado.GrupoCorrecao)
	c.oraculo.falhar = true

	if err := c.rodada.Executar(t.Context()); err == nil {
		t.Fatal("a primeira execução cai no oráculo")
	}
	c.oraculo.falhar = false
	if err := c.rodada.Executar(t.Context()); err != nil {
		t.Fatal(err)
	}

	tarefa := lerResultado(t, c)
	if *c.execucoes != 1 || len(c.github.prs) != 1 || tarefa.PR.Numero != 1 || tarefa.Oraculo != resultado.OraculoPassou {
		t.Errorf("retomada: agente %d vez(es), %d PR(s), %+v", *c.execucoes, len(c.github.prs), tarefa)
	}

	antes := c.github.requisicoes
	if err := c.rodada.Executar(t.Context()); err != nil {
		t.Fatal(err)
	}
	if c.github.requisicoes != antes || *c.execucoes != 1 {
		t.Error("tarefa com resultado gravado não roda de novo")
	}
}

func TestCheckRunQueNaoConcluiNoPrazoESemDecisao(t *testing.T) {
	c := novoCenario(t, resultado.GrupoCorrecao)
	c.github.concluir = false

	if err := c.rodada.Executar(t.Context()); err != nil {
		t.Fatal(err)
	}

	tarefa := lerResultado(t, c)
	if tarefa.Desfecho != resultado.DesfechoSemDecisao || !strings.Contains(tarefa.Detalhe, "não concluiu") || tarefa.Decisao != nil {
		t.Errorf("sem decisão no prazo é registrado: %+v", tarefa)
	}
	if tarefa.Oraculo != resultado.OraculoPassou {
		t.Error("o oráculo roda mesmo sem decisão")
	}
}

func TestDiffComSegredoERecusadoSemPush(t *testing.T) {
	c := novoCenario(t, resultado.GrupoCorrecao)
	c.agente.escrever = map[string]string{"a.go": "package a // postgres://aprova:segredo-local@localhost/aprova\n"}

	if err := c.rodada.Executar(t.Context()); err != nil {
		t.Fatal(err)
	}

	tarefa := lerResultado(t, c)
	if tarefa.Desfecho != resultado.DesfechoDiffRecusado || !strings.Contains(tarefa.Detalhe, "DATABASE_URL") || strings.Contains(tarefa.Detalhe, "segredo-local") {
		t.Errorf("diff recusado nomeando a variável, sem o valor: %+v", tarefa)
	}
	if len(c.github.refs) != 0 || len(c.github.prs) != 0 {
		t.Errorf("nada sobe: %v, %v", c.github.refs, c.github.prs)
	}
}

func TestDiffVazioNaoAbrePR(t *testing.T) {
	c := novoCenario(t, resultado.GrupoCorrecao)
	c.agente.escrever = nil

	if err := c.rodada.Executar(t.Context()); err != nil {
		t.Fatal(err)
	}

	if tarefa := lerResultado(t, c); tarefa.Desfecho != resultado.DesfechoDiffVazio || len(c.github.prs) != 0 {
		t.Errorf("diff vazio: %+v", tarefa)
	}
}

func TestLimiteDeUsoParaARodadaSemResultadoERegistraOEvento(t *testing.T) {
	c := novoCenario(t, resultado.GrupoCorrecao)
	c.agente.desfecho = agente.DesfechoLimiteDeUso

	err := c.rodada.Executar(t.Context())

	if !errors.Is(err, ErrLimiteDeUso) {
		t.Fatalf("limite de uso para a rodada: %v", err)
	}
	if tarefas, _ := resultado.LerDiretorio(c.rodada.Resultados); len(tarefas) != 0 {
		t.Errorf("limite de uso não é resultado da tarefa: %+v", tarefas)
	}
	eventos, _ := os.ReadFile(filepath.Join(c.rodada.Resultados, "eventos.jsonl"))
	if !strings.Contains(string(eventos), "limite_de_uso") || !strings.Contains(string(eventos), "go-a") {
		t.Errorf("o limite fica registrado: %s", eventos)
	}

	c.agente.desfecho = agente.DesfechoConcluida
	if err := c.rodada.Executar(t.Context()); err != nil {
		t.Fatal(err)
	}
	if tarefa := lerResultado(t, c); tarefa.ExecucoesDoAgente != 2 {
		t.Errorf("a retomada registra que o agente rodou duas vezes nessa tarefa: %d", tarefa.ExecucoesDoAgente)
	}
}

func TestTesteApagadoEAdulteracao(t *testing.T) {
	c := novoCenario(t, resultado.GrupoCorrecao)
	c.agente.apagar = []string{"a_test.go"}

	if err := c.rodada.Executar(t.Context()); err != nil {
		t.Fatal(err)
	}

	if tarefa := lerResultado(t, c); len(tarefa.TestesAdulterados) != 1 || tarefa.TestesAdulterados[0] != "a_test.go" {
		t.Errorf("o teste apagado é adulteração: %+v", tarefa.TestesAdulterados)
	}
}

func TestGrupoBNaoRodaOOraculoEGuardaOEsperado(t *testing.T) {
	c := novoCenario(t, resultado.GrupoRisco)

	if err := c.rodada.Executar(t.Context()); err != nil {
		t.Fatal(err)
	}

	tarefa := lerResultado(t, c)
	if *c.julgadas != 0 || tarefa.Oraculo != resultado.OraculoNaoRodou {
		t.Errorf("o grupo B não tem oráculo: %d julgamentos, %+v", *c.julgadas, tarefa)
	}
	if tarefa.Esperado == nil || tarefa.Esperado.Motivo != "configuracao_ci" || tarefa.Grupo != resultado.GrupoRisco {
		t.Errorf("o esperado escrito antes da rodada vai junto: %+v", tarefa.Esperado)
	}
}

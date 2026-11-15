package remoto

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type githubFalso struct {
	mu       sync.Mutex
	blobs    map[string]string
	arvores  map[string]map[string]any
	commits  map[string]map[string]any
	refs     map[string]string
	prs      []map[string]any
	checks   map[string]map[string]any
	contador int
	pedidos  []string
}

func novoGitHubFalso() *githubFalso {
	return &githubFalso{blobs: map[string]string{}, arvores: map[string]map[string]any{}, commits: map[string]map[string]any{},
		refs: map[string]string{}, checks: map[string]map[string]any{}}
}

func (g *githubFalso) novoSHA() string {
	g.contador++
	return fmt.Sprintf("%040d", g.contador)
}

func (g *githubFalso) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pedidos = append(g.pedidos, r.Method+" "+r.URL.Path)
	if r.Header.Get("Authorization") != "Bearer token-de-teste" {
		http.Error(w, "sem token", http.StatusUnauthorized)
		return
	}
	var corpo map[string]any
	if r.Body != nil {
		bruto, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(bruto, &corpo)
	}
	caminho := strings.TrimPrefix(r.URL.Path, "/repos/dono/bench-go")
	responder := func(status int, valor any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(valor)
	}
	switch {
	case r.Method == http.MethodPost && caminho == "/git/blobs":
		sha := g.novoSHA()
		conteudo, _ := base64.StdEncoding.DecodeString(texto(corpo, "content"))
		g.blobs[sha] = string(conteudo)
		responder(http.StatusCreated, map[string]any{"sha": sha})
	case r.Method == http.MethodPost && caminho == "/git/trees":
		sha := g.novoSHA()
		g.arvores[sha] = corpo
		responder(http.StatusCreated, map[string]any{"sha": sha})
	case r.Method == http.MethodPost && caminho == "/git/commits":
		sha := g.novoSHA()
		g.commits[sha] = corpo
		responder(http.StatusCreated, map[string]any{"sha": sha, "tree": map[string]any{"sha": corpo["tree"]}})
	case r.Method == http.MethodPost && caminho == "/git/refs":
		ref := texto(corpo, "ref")
		if _, existe := g.refs[ref]; existe {
			responder(http.StatusUnprocessableEntity, map[string]any{"message": "Reference already exists"})
			return
		}
		g.refs[ref] = texto(corpo, "sha")
		responder(http.StatusCreated, map[string]any{"ref": ref})
	case r.Method == http.MethodPatch && strings.HasPrefix(caminho, "/git/refs/"):
		ref := "refs/" + strings.TrimPrefix(caminho, "/git/refs/")
		g.refs[ref] = texto(corpo, "sha")
		responder(http.StatusOK, map[string]any{"ref": ref, "forcado": corpo["force"]})
	case r.Method == http.MethodGet && caminho == "/pulls":
		var encontrados []map[string]any
		for _, pr := range g.prs {
			if "dono:"+texto(pr, "head") == r.URL.Query().Get("head") {
				encontrados = append(encontrados, pr)
			}
		}
		responder(http.StatusOK, encontrados)
	case r.Method == http.MethodPost && caminho == "/pulls":
		pr := map[string]any{"number": len(g.prs) + 1, "html_url": fmt.Sprintf("https://github.com/dono/bench-go/pull/%d", len(g.prs)+1),
			"head": corpo["head"], "base": corpo["base"], "title": corpo["title"]}
		g.prs = append(g.prs, pr)
		responder(http.StatusCreated, pr)
	case r.Method == http.MethodGet && strings.HasPrefix(caminho, "/commits/") && strings.HasSuffix(caminho, "/check-runs"):
		sha := strings.TrimSuffix(strings.TrimPrefix(caminho, "/commits/"), "/check-runs")
		var runs []map[string]any
		if check, existe := g.checks[sha]; existe && r.URL.Query().Get("check_name") == "aPRova" {
			runs = append(runs, check)
		}
		responder(http.StatusOK, map[string]any{"total_count": len(runs), "check_runs": runs})
	default:
		http.Error(w, "rota desconhecida "+r.Method+" "+r.URL.Path, http.StatusNotFound)
	}
}

func texto(corpo map[string]any, chave string) string {
	valor, _ := corpo[chave].(string)
	return valor
}

func lista(corpo map[string]any, chave string) []any {
	valor, _ := corpo[chave].([]any)
	return valor
}

func clienteDeTeste(t *testing.T) (*GitHub, *githubFalso) {
	t.Helper()
	falso := novoGitHubFalso()
	servidor := httptest.NewServer(falso)
	t.Cleanup(servidor.Close)
	return &GitHub{URLBase: servidor.URL, Token: "token-de-teste", Cliente: servidor.Client()}, falso
}

func TestBaseEUmCommitRaizComOsArquivosDaTarefa(t *testing.T) {
	cliente, falso := clienteDeTeste(t)

	base, err := cliente.PublicarCommit(t.Context(), "dono/bench-go", "base/go-a", Commit{
		Mensagem: "base da tarefa go-a", Arquivos: map[string][]byte{"a.go": []byte("package a\n"), "a_test.go": []byte("package a // teste\n")},
	})
	if err != nil {
		t.Fatal(err)
	}

	commit := falso.commits[base.SHA]
	if pais, _ := commit["parents"].([]any); len(pais) != 0 {
		t.Errorf("a base não tem pai: %v", commit["parents"])
	}
	arvore := falso.arvores[base.Arvore]
	if _, temBase := arvore["base_tree"]; temBase {
		t.Error("a árvore da base não parte de outra")
	}
	if len(lista(arvore, "tree")) != 2 || falso.refs["refs/heads/base/go-a"] != base.SHA {
		t.Errorf("dois arquivos e a ref apontando para o commit: %v, %v", arvore, falso.refs)
	}
	conteudos := map[string]bool{}
	for _, conteudo := range falso.blobs {
		conteudos[conteudo] = true
	}
	if !conteudos["package a // teste\n"] {
		t.Errorf("os arquivos sobem como blobs em base64: %v", falso.blobs)
	}
}

func TestSolucaoParteDaBaseERegistraApagados(t *testing.T) {
	cliente, falso := clienteDeTeste(t)
	base, err := cliente.PublicarCommit(t.Context(), "dono/bench-go", "base/go-a", Commit{Mensagem: "base", Arquivos: map[string][]byte{"a.go": []byte("x")}})
	if err != nil {
		t.Fatal(err)
	}

	solucao, err := cliente.PublicarCommit(t.Context(), "dono/bench-go", "solucao/go-a", Commit{
		Mensagem: "solução", Pai: &base, Arquivos: map[string][]byte{"a.go": []byte("y")}, Removidos: []string{"cases_test.go"},
	})
	if err != nil {
		t.Fatal(err)
	}

	commit := falso.commits[solucao.SHA]
	if pais := lista(commit, "parents"); len(pais) != 1 || pais[0] != base.SHA {
		t.Errorf("a solução tem a base como pai: %v", commit["parents"])
	}
	arvore := falso.arvores[solucao.Arvore]
	if arvore["base_tree"] != base.Arvore {
		t.Errorf("a árvore da solução parte da árvore da base: %v", arvore)
	}
	apagado := false
	for _, entrada := range lista(arvore, "tree") {
		item, _ := entrada.(map[string]any)
		if item["path"] == "cases_test.go" && item["sha"] == nil {
			apagado = true
		}
	}
	if !apagado {
		t.Errorf("arquivo apagado vai com sha nulo: %v", arvore["tree"])
	}
}

func TestRefQueJaExisteEAtualizadaNaRetomada(t *testing.T) {
	cliente, falso := clienteDeTeste(t)
	falso.refs["refs/heads/base/go-a"] = "antigo"

	base, err := cliente.PublicarCommit(t.Context(), "dono/bench-go", "base/go-a", Commit{Mensagem: "base", Arquivos: map[string][]byte{"a.go": []byte("x")}})

	if err != nil || falso.refs["refs/heads/base/go-a"] != base.SHA {
		t.Errorf("a ref existente aponta para o commit novo: %v, %v", falso.refs, err)
	}
}

func TestAbrirPRReaproveitaOQueJaExiste(t *testing.T) {
	cliente, falso := clienteDeTeste(t)

	primeiro, err := cliente.AbrirPR(t.Context(), "dono/bench-go", "solucao/go-a", "base/go-a", "go-a", "corpo")
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := cliente.AbrirPR(t.Context(), "dono/bench-go", "solucao/go-a", "base/go-a", "go-a", "corpo")
	if err != nil {
		t.Fatal(err)
	}

	if len(falso.prs) != 1 || primeiro.Numero != 1 || segundo.Numero != 1 || !strings.HasSuffix(primeiro.URL, "/pull/1") {
		t.Errorf("um PR só, mesmo chamando duas vezes: %+v, %+v, %d PRs", primeiro, segundo, len(falso.prs))
	}
}

func TestCheckRunDoAPRova(t *testing.T) {
	cliente, falso := clienteDeTeste(t)
	sha := strings.Repeat("a", 40)

	ausente, err := cliente.CheckRun(t.Context(), "dono/bench-go", sha, "aPRova")
	if err != nil || ausente != nil {
		t.Errorf("sem check run ainda: %+v, %v", ausente, err)
	}

	falso.checks[sha] = map[string]any{"status": "completed", "conclusion": "action_required",
		"output":      map[string]any{"title": "Risco alto: merge bloqueado até aprovação humana"},
		"details_url": "https://aprova.exemplo/receipts/11111111-2222-3333-4444-555555555555/verify"}
	check, err := cliente.CheckRun(t.Context(), "dono/bench-go", sha, "aPRova")

	if err != nil || check == nil || !check.Concluido() || check.Conclusao != "action_required" ||
		check.IDDoRecibo() != "11111111-2222-3333-4444-555555555555" || !strings.Contains(check.Titulo, "Risco alto") {
		t.Errorf("check run concluído, com o recibo no details_url: %+v, %v", check, err)
	}
}

func TestReciboDoAPRovaTrazNivelMotivosEHash(t *testing.T) {
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/receipts/rec-1" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"id":"rec-1","task_run_id":"tr-1","hash_sha256":"abc123","payload":{"nivel":"alto","nivel_deterministico":"alto",`+
			`"motivos":[{"id":"dependencia_nova","evidencias":["validate-docbr"]},{"id":"codigo_sem_teste","evidencias":["cadastro.py"]}]}}`)
	}))
	t.Cleanup(servidor.Close)
	aprova := APRova{URLBase: servidor.URL, Cliente: servidor.Client()}

	recibo, err := aprova.Recibo(t.Context(), "rec-1")

	if err != nil || recibo.Nivel != "alto" || recibo.TaskRunID != "tr-1" || recibo.Hash != "abc123" ||
		strings.Join(recibo.Motivos, ",") != "dependencia_nova,codigo_sem_teste" {
		t.Errorf("recibo lido: %+v, %v", recibo, err)
	}
}

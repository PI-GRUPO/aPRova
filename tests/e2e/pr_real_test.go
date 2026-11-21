package pontaaponta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/benchmark/remoto"
)

const (
	urlDaAPIDoGitHub      = "https://api.github.com"
	branchBase            = "main"
	prefixoDaBranch       = "e2e-"
	nomeDoCheckRun        = "aPRova"
	marcaDoComentario     = "<!-- aprova:resultado-da-analise -->"
	esperaPeloCheckRun    = 10 * time.Minute
	intervaloDeConsulta   = 10 * time.Second
	idadeDeBranchVelha    = time.Hour
	prazoDaLimpeza        = time.Minute
	tempoLimiteDoPedido   = 30 * time.Second
	conclusaoDeRiscoBaixo = "success"
)

var arquivosDoPR = map[string][]byte{
	"go.mod":       []byte("module example.com/aprovae2e\n\ngo 1.22\n"),
	"soma.go":      []byte("package aprovae2e\n\nfunc Somar(a, b int) int {\n\treturn a + b\n}\n"),
	"soma_test.go": []byte("package aprovae2e\n\nimport \"testing\"\n\nfunc TestSomar(t *testing.T) {\n\tif Somar(2, 3) != 5 {\n\t\tt.Fatal(\"2 + 3 deveria ser 5\")\n\t}\n}\n"),
}

type ambiente struct {
	github      *remoto.GitHub
	repositorio string
	urlAPRova   string
}

func TestPullRequestRealRecebeCheckRunComentarioEReciboValido(t *testing.T) {
	amb := exigirAmbiente(t)
	apagarBranchesVelhas(t, amb)

	branch := prefixoDaBranch + strconv.FormatInt(time.Now().Unix(), 10)
	commit := publicarBranch(t, amb, branch)
	pr, err := amb.github.AbrirPR(t.Context(), amb.repositorio, branch, branchBase, "e2e: "+branch, "PR aberto pelo teste ponta a ponta do aPRova.")
	if err != nil {
		t.Fatalf("abrir PR: %v", err)
	}
	t.Cleanup(func() { fecharPR(t, amb, pr.Numero) })
	t.Logf("PR aberto: %s", pr.URL)

	check := esperarCheckRunConcluido(t, amb, commit.SHA)
	if check.Conclusao != conclusaoDeRiscoBaixo {
		t.Errorf("conclusão do Check Run = %q (%s), esperava %q", check.Conclusao, check.Titulo, conclusaoDeRiscoBaixo)
	}
	idDoRecibo := check.IDDoRecibo()
	if idDoRecibo == "" {
		t.Fatalf("o Check Run não traz o recibo no details_url: %q", check.URLDetalhe)
	}

	conferirComentario(t, amb, pr.Numero, idDoRecibo)
	conferirRecibo(t, amb, idDoRecibo)
}

func exigirAmbiente(t *testing.T) ambiente {
	t.Helper()

	if os.Getenv("E2E_LIGADO") != "true" {
		t.Skip("teste ponta a ponta desligado: rode com E2E_LIGADO=true contra um repositório de teste e o aPRova no ar")
	}
	repositorio, urlAPRova := os.Getenv("E2E_REPOSITORIO"), strings.TrimRight(os.Getenv("E2E_URL_APROVA"), "/")
	if repositorio == "" || urlAPRova == "" {
		t.Fatal("E2E_LIGADO=true exige E2E_REPOSITORIO (dono/nome) e E2E_URL_APROVA (base pública da API)")
	}
	token, err := exec.CommandContext(t.Context(), "gh", "auth", "token").Output()
	if err != nil {
		t.Fatalf("obter o token do gh: %v", err)
	}
	return ambiente{
		github:      &remoto.GitHub{URLBase: urlDaAPIDoGitHub, Token: strings.TrimSpace(string(token)), Cliente: &http.Client{Timeout: tempoLimiteDoPedido}},
		repositorio: repositorio,
		urlAPRova:   urlAPRova,
	}
}

func apagarBranchesVelhas(t *testing.T, amb ambiente) {
	t.Helper()

	branches, err := amb.github.BranchesComPrefixo(t.Context(), amb.repositorio, prefixoDaBranch)
	if err != nil {
		t.Fatalf("listar branches de rodadas anteriores: %v", err)
	}
	limite := time.Now().Add(-idadeDeBranchVelha).Unix()
	for _, branch := range branches {
		criadaEm, err := strconv.ParseInt(strings.TrimPrefix(branch, prefixoDaBranch), 10, 64)
		if err != nil || criadaEm >= limite {
			continue
		}
		if err := amb.github.ApagarBranch(t.Context(), amb.repositorio, branch); err != nil {
			t.Logf("apagar a branch velha %s: %v", branch, err)
		}
	}
}

func publicarBranch(t *testing.T, amb ambiente, branch string) remoto.CommitPublicado {
	t.Helper()

	base, err := amb.github.UltimoCommit(t.Context(), amb.repositorio, branchBase)
	if err != nil {
		t.Fatalf("ler a %s de %s (o repositório precisa de ao menos um commit): %v", branchBase, amb.repositorio, err)
	}
	commit, err := amb.github.PublicarCommit(t.Context(), amb.repositorio, branch, remoto.Commit{
		Mensagem: "e2e: módulo Go com teste", Pai: &base, Arquivos: arquivosDoPR,
	})
	t.Cleanup(func() { apagarBranch(t, amb, branch) })
	if err != nil {
		t.Fatalf("publicar a branch %s: %v", branch, err)
	}
	return commit
}

func fecharPR(t *testing.T, amb ambiente, numero int) {
	ctx, cancelar := context.WithTimeout(context.Background(), prazoDaLimpeza)
	defer cancelar()
	if err := amb.github.FecharPR(ctx, amb.repositorio, numero); err != nil {
		t.Errorf("fechar o PR #%d: %v", numero, err)
	}
}

func apagarBranch(t *testing.T, amb ambiente, branch string) {
	ctx, cancelar := context.WithTimeout(context.Background(), prazoDaLimpeza)
	defer cancelar()
	if err := amb.github.ApagarBranch(ctx, amb.repositorio, branch); err != nil {
		t.Errorf("apagar a branch %s: %v", branch, err)
	}
}

func esperarCheckRunConcluido(t *testing.T, amb ambiente, sha string) remoto.CheckRun {
	t.Helper()

	ctx, cancelar := context.WithTimeout(t.Context(), esperaPeloCheckRun)
	defer cancelar()
	consulta := time.NewTicker(intervaloDeConsulta)
	defer consulta.Stop()

	var ultimo *remoto.CheckRun
	for {
		check, err := amb.github.CheckRun(ctx, amb.repositorio, sha, nomeDoCheckRun)
		if err == nil && check != nil {
			ultimo = check
			if check.Concluido() {
				return *check
			}
		}
		select {
		case <-ctx.Done():
			t.Fatalf("o Check Run não concluiu em %s; último estado: %+v, último erro: %v", esperaPeloCheckRun, ultimo, err)
		case <-consulta.C:
		}
	}
}

func conferirComentario(t *testing.T, amb ambiente, numero int, idDoRecibo string) {
	t.Helper()

	corpos, err := amb.github.Comentarios(t.Context(), amb.repositorio, numero)
	if err != nil {
		t.Fatalf("ler comentários do PR: %v", err)
	}
	for _, corpo := range corpos {
		if strings.Contains(corpo, marcaDoComentario) {
			if !strings.Contains(corpo, idDoRecibo) {
				t.Errorf("o comentário do aPRova não cita o recibo %s", idDoRecibo)
			}
			return
		}
	}
	t.Errorf("o PR não tem o comentário do aPRova entre %d comentários", len(corpos))
}

func conferirRecibo(t *testing.T, amb ambiente, idDoRecibo string) {
	t.Helper()

	pedido, err := http.NewRequestWithContext(t.Context(), http.MethodGet, fmt.Sprintf("%s/receipts/%s/verify", amb.urlAPRova, idDoRecibo), nil)
	if err != nil {
		t.Fatalf("montar a verificação do recibo: %v", err)
	}
	resposta, err := amb.github.Cliente.Do(pedido)
	if err != nil {
		t.Fatalf("verificar o recibo: %v", err)
	}
	defer func() { _ = resposta.Body.Close() }()

	var verificacao struct {
		Valido bool   `json:"valido"`
		Falha  string `json:"falha"`
	}
	if err := json.NewDecoder(resposta.Body).Decode(&verificacao); err != nil {
		t.Fatalf("ler a verificação do recibo (status %d): %v", resposta.StatusCode, err)
	}
	if !verificacao.Valido {
		t.Errorf("recibo %s inválido: %s", idDoRecibo, verificacao.Falha)
	}
}

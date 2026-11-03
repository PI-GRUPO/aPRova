package platform

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

const diffDoFrete = `diff --git a/go/frete/frete.go b/go/frete/frete.go
--- a/go/frete/frete.go
+++ b/go/frete/frete.go
@@ -10,7 +10,7 @@
-	if pesoKg < 0 {
+	if pesoKg <= 0 {
`

type githubDoPR struct {
	arquivos          int
	diff              string
	accepts           []string
	paginasPedidas    []string
	autorizacoes      []string
	statusDosArquivos int
	cabecalhos        http.Header
}

func (g *githubDoPR) servidor(t *testing.T) *httptest.Server {
	t.Helper()

	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.autorizacoes = append(g.autorizacoes, r.Header.Get("Authorization"))
		for nome, valores := range g.cabecalhos {
			w.Header()[nome] = valores
		}

		switch r.URL.Path {
		case "/repos/Azulpasta/aPRova-TESTES/pulls/14/files":
			g.paginasPedidas = append(g.paginasPedidas, r.URL.Query().Get("page")+"/"+r.URL.Query().Get("per_page"))
			if g.statusDosArquivos != 0 {
				w.WriteHeader(g.statusDosArquivos)
				return
			}
			pagina, _ := strconv.Atoi(r.URL.Query().Get("page"))
			_ = json.NewEncoder(w).Encode(g.pagina(pagina))
		case "/repos/Azulpasta/aPRova-TESTES/pulls/14":
			g.accepts = append(g.accepts, r.Header.Get("Accept"))
			_, _ = w.Write([]byte(g.diff))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(servidor.Close)
	return servidor
}

func (g *githubDoPR) pagina(numero int) []map[string]any {
	inicio := (numero - 1) * 100
	fim := min(inicio+100, g.arquivos)
	var arquivos []map[string]any
	for indice := inicio; indice < fim; indice++ {
		arquivos = append(arquivos, map[string]any{
			"filename": fmt.Sprintf("go/frete/arquivo_%03d.go", indice), "status": "modified", "additions": 2, "deletions": 1,
		})
	}
	return arquivos
}

func TestContextoDoPullRequestTrazTodosOsArquivosEODiff(t *testing.T) {
	github := &githubDoPR{arquivos: 101, diff: diffDoFrete}
	cliente := NovoCliente(OpcoesCliente{Tokens: &fornecedorDeTokenFalso{}, URLBase: github.servidor(t).URL})

	contexto, err := cliente.ContextoDoPullRequest(t.Context(), 58291734, "Azulpasta", "aPRova-TESTES", 14)

	if err != nil {
		t.Fatalf("ler contexto: %v", err)
	}
	if len(contexto.Arquivos) != 101 || contexto.Arquivos[100].Nome != "go/frete/arquivo_100.go" {
		t.Fatalf("esperava os 101 arquivos das duas páginas, obtive %d", len(contexto.Arquivos))
	}
	if primeiro := contexto.Arquivos[0]; primeiro.Situacao != "modified" || primeiro.Adicoes != 2 || primeiro.Remocoes != 1 {
		t.Errorf("arquivo lido errado: %+v", primeiro)
	}
	if strings.Join(github.paginasPedidas, ",") != "1/100,2/100" {
		t.Errorf("páginas pedidas: %v", github.paginasPedidas)
	}
	if contexto.Diff != diffDoFrete || contexto.DiffTruncado {
		t.Errorf("diff lido errado (truncado=%v): %q", contexto.DiffTruncado, contexto.Diff)
	}
	if len(github.accepts) != 1 || github.accepts[0] != "application/vnd.github.diff" {
		t.Errorf("o diff precisa ser pedido com Accept application/vnd.github.diff, foi %v", github.accepts)
	}
	for _, autorizacao := range github.autorizacoes {
		if autorizacao != "Bearer "+tokenDeInstalacaoDeTeste {
			t.Errorf("chamada sem o token da instalação: %q", autorizacao)
		}
	}
}

func TestContextoDoPullRequestParaNaPaginaIncompleta(t *testing.T) {
	github := &githubDoPR{arquivos: 100, diff: diffDoFrete}
	cliente := NovoCliente(OpcoesCliente{Tokens: &fornecedorDeTokenFalso{}, URLBase: github.servidor(t).URL})

	contexto, err := cliente.ContextoDoPullRequest(t.Context(), 58291734, "Azulpasta", "aPRova-TESTES", 14)

	if err != nil || len(contexto.Arquivos) != 100 {
		t.Fatalf("esperava 100 arquivos, obtive %d e %v", len(contexto.Arquivos), err)
	}
	if strings.Join(github.paginasPedidas, ",") != "1/100,2/100" {
		t.Errorf("com 100 arquivos na primeira página, a segunda precisa vir vazia e encerrar: %v", github.paginasPedidas)
	}
}

func TestContextoDoPullRequestTruncaDiffGrande(t *testing.T) {
	github := &githubDoPR{arquivos: 1, diff: strings.Repeat("+linha nova\n", tamanhoMaximoDoDiff/12+10)}
	cliente := NovoCliente(OpcoesCliente{Tokens: &fornecedorDeTokenFalso{}, URLBase: github.servidor(t).URL})

	contexto, err := cliente.ContextoDoPullRequest(t.Context(), 58291734, "Azulpasta", "aPRova-TESTES", 14)

	if err != nil {
		t.Fatalf("ler contexto: %v", err)
	}
	if !contexto.DiffTruncado || len(contexto.Diff) != tamanhoMaximoDoDiff {
		t.Errorf("diff acima do limite precisa vir cortado no limite e marcado: truncado=%v tamanho=%d", contexto.DiffTruncado, len(contexto.Diff))
	}
}

func TestContextoDoPullRequestClassificaOsErrosDoGitHub(t *testing.T) {
	casos := map[string]struct {
		status     int
		cabecalhos http.Header
		esperado   error
	}{
		"PR inexistente":                    {http.StatusNotFound, nil, ErrNaoEncontrado},
		"GitHub fora do ar":                 {http.StatusBadGateway, nil, ErrGitHubIndisponivel},
		"limite secundário (429)":           {http.StatusTooManyRequests, nil, ErrLimiteDeRequisicoes},
		"limite primário esgotado (403)":    {http.StatusForbidden, http.Header{"X-Ratelimit-Remaining": {"0"}}, ErrLimiteDeRequisicoes},
		"limite secundário com Retry-After": {http.StatusForbidden, http.Header{"Retry-After": {"60"}}, ErrLimiteDeRequisicoes},
		"403 que não é limite":              {http.StatusForbidden, http.Header{"X-Ratelimit-Remaining": {"4999"}}, ErrRespostaInesperada},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			github := &githubDoPR{statusDosArquivos: caso.status, cabecalhos: caso.cabecalhos}
			cliente := NovoCliente(OpcoesCliente{Tokens: &fornecedorDeTokenFalso{}, URLBase: github.servidor(t).URL})

			_, err := cliente.ContextoDoPullRequest(t.Context(), 58291734, "Azulpasta", "aPRova-TESTES", 14)

			if !errors.Is(err, caso.esperado) {
				t.Errorf("esperava %v, obtive %v", caso.esperado, err)
			}
		})
	}
}

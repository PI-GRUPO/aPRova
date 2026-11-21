package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/platform"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
)

const (
	headDoReciboDeTeste  = "d7ee920e4f9f8a2f413630afbe6c25435149a302"
	mergeDoReciboDeTeste = "5555555555555555555555555555555555555555"
	urlPublicaDeTeste    = "https://aprova.exemplo"
)

type desfechosEmMemoria struct {
	desfecho *domain.DesfechoDePR
	pedidos  int
}

func (d *desfechosEmMemoria) DesfechoDoPR(_ context.Context, tenantID int64, repositorio string, numeroPR int) (domain.DesfechoDePR, error) {
	d.pedidos++
	if d.desfecho == nil || tenantID != 161192271 || repositorio != "Azulpasta/aPRova-TESTES" || numeroPR != 1 {
		return domain.DesfechoDePR{}, domain.ErrNaoEncontrado
	}
	return *d.desfecho, nil
}

type tokenFixo struct{}

func (tokenFixo) TokenParaInstalacao(context.Context, int64) (string, error) { return "ghs_teste", nil }

type githubSoLeitura struct {
	t        *testing.T
	mutex    sync.Mutex
	caminhos []string
}

func (g *githubSoLeitura) RoundTrip(pedido *http.Request) (*http.Response, error) {
	if pedido.Method != http.MethodGet {
		g.t.Errorf("chamada de escrita ao GitHub: %s %s", pedido.Method, pedido.URL.Path)
		return nil, errors.New("escrita proibida")
	}
	g.mutex.Lock()
	g.caminhos = append(g.caminhos, pedido.URL.Path)
	g.mutex.Unlock()
	return http.DefaultTransport.RoundTrip(pedido)
}

func (c cenarioDeRecibos) recibosVerificaveis() VerificadorDeRecibos {
	return receipt.NovoVerificador(receipt.NovoChaveiro(c.assinador.ChavePublica()), c.guardados)
}

type cenarioDeReversao struct {
	cenarioDeRecibos
	desfechos *desfechosEmMemoria
	github    *githubSoLeitura
}

func novoCenarioDeReversao(t *testing.T, desfecho *domain.DesfechoDePR, statusDoCommit int, commitNoGitHub string) cenarioDeReversao {
	t.Helper()

	recibos := novoCenarioDeRecibos(t, 2)
	apiDoGitHub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(statusDoCommit)
		_, _ = w.Write([]byte(commitNoGitHub))
	}))
	t.Cleanup(apiDoGitHub.Close)

	github := &githubSoLeitura{t: t}
	desfechos := &desfechosEmMemoria{desfecho: desfecho}
	cliente := platform.NovoCliente(platform.OpcoesCliente{
		Tokens: tokenFixo{}, URLBase: apiDoGitHub.URL, ClienteHTTP: &http.Client{Transport: github},
	})
	recibos.servidor = NovoServidor(Opcoes{
		Recibos: recibos.recibosVerificaveis(), Desfechos: desfechos, Commits: cliente, URLPublica: urlPublicaDeTeste,
	}).Rotas()
	return cenarioDeReversao{cenarioDeRecibos: recibos, desfechos: desfechos, github: github}
}

func desfechoMergeado(commits int) *domain.DesfechoDePR {
	return &domain.DesfechoDePR{Mergeado: true, SHAHead: headDoReciboDeTeste, SHADoMerge: mergeDoReciboDeTeste, Commits: commits}
}

const commitComDoisPais = `{"parents":[{"sha":"a"},{"sha":"b"}]}`

func TestReversaoDeReciboComEloQuebradoNaoGeraInstrucaoNemConsultaNada(t *testing.T) {
	cenario := novoCenarioDeReversao(t, desfechoMergeado(1), http.StatusOK, commitComDoisPais)
	delete(cenario.guardados.porID, cenario.ids[0])

	gravador := cenario.pedir(t, "/receipts/"+cenario.ids[1]+"/revert")

	resposta := decodificar[respostaDaReversao](t, gravador)
	if gravador.Code != http.StatusOK || resposta.Disponivel || resposta.Recusa != "recibo_invalido" || resposta.Comando != "" || resposta.Corpo != "" {
		t.Fatalf("status %d, resposta %+v", gravador.Code, resposta)
	}
	if resposta.Verificacao.Valido || resposta.Verificacao.Falha != "elo" {
		t.Errorf("a verificação vai junto, igual ao /verify: %+v", resposta.Verificacao)
	}
	if cenario.desfechos.pedidos != 0 || len(cenario.github.caminhos) != 0 {
		t.Errorf("recibo inválido não consulta banco (%d) nem GitHub (%v)", cenario.desfechos.pedidos, cenario.github.caminhos)
	}
}

func TestReversaoDePRAindaAbertoExplicaQueNaoHaOQueReverter(t *testing.T) {
	cenario := novoCenarioDeReversao(t, nil, http.StatusOK, commitComDoisPais)

	resposta := decodificar[respostaDaReversao](t, cenario.pedir(t, "/receipts/"+cenario.ids[1]+"/revert"))

	if resposta.Disponivel || resposta.Recusa != "pr_aberto" || resposta.Explicacao == "" {
		t.Errorf("resposta %+v", resposta)
	}
	if len(cenario.github.caminhos) != 0 {
		t.Errorf("PR aberto não consulta o GitHub: %v", cenario.github.caminhos)
	}
}

func TestReversaoDeMergeCommitDevolveComandoECorpoReconhecidoPelaCalibracao(t *testing.T) {
	cenario := novoCenarioDeReversao(t, desfechoMergeado(1), http.StatusOK, commitComDoisPais)

	gravador := cenario.pedir(t, "/receipts/"+cenario.ids[1]+"/revert")

	resposta := decodificar[respostaDaReversao](t, gravador)
	if gravador.Code != http.StatusOK || !resposta.Disponivel || !resposta.Verificacao.Valido {
		t.Fatalf("status %d, resposta %+v", gravador.Code, resposta)
	}
	if resposta.Repositorio != "Azulpasta/aPRova-TESTES" || resposta.NumeroPR != 1 || resposta.SHAAnalisado != headDoReciboDeTeste ||
		resposta.CommitNaMain != mergeDoReciboDeTeste || resposta.Tipo != "merge" {
		t.Errorf("identificação: %+v", resposta)
	}
	if !strings.HasSuffix(resposta.Comando, "git revert --no-edit -m 1 "+mergeDoReciboDeTeste) {
		t.Errorf("comando: %s", resposta.Comando)
	}
	if !strings.Contains(resposta.Corpo, urlPublicaDeTeste+"/receipts/"+cenario.ids[1]+"/verify") {
		t.Errorf("corpo sem o link de verificação:\n%s", resposta.Corpo)
	}
	if revertido := pullRequestRevertido(resposta.Corpo, "Azulpasta/aPRova-TESTES"); revertido != 1 {
		t.Errorf("o detector de reversão do webhook leu %d no corpo gerado", revertido)
	}
	if len(cenario.github.caminhos) != 1 || cenario.github.caminhos[0] != "/repos/Azulpasta/aPRova-TESTES/commits/"+mergeDoReciboDeTeste {
		t.Errorf("leituras no GitHub: %v", cenario.github.caminhos)
	}
}

func TestReversaoComUmPaiEVariosCommitsPedeOTipo(t *testing.T) {
	cenario := novoCenarioDeReversao(t, desfechoMergeado(3), http.StatusOK, `{"parents":[{"sha":"a"}]}`)

	resposta := decodificar[respostaDaReversao](t, cenario.pedir(t, "/receipts/"+cenario.ids[1]+"/revert"))

	if resposta.Disponivel || resposta.Recusa != "tipo_ambiguo" || resposta.Comando != "" {
		t.Errorf("resposta %+v", resposta)
	}
}

func TestReversaoComFalhaNoGitHubResponde502SemDetalhe(t *testing.T) {
	cenario := novoCenarioDeReversao(t, desfechoMergeado(1), http.StatusInternalServerError, `{"message":"segredo interno"}`)

	gravador := cenario.pedir(t, "/receipts/"+cenario.ids[1]+"/revert")

	if gravador.Code != http.StatusBadGateway || strings.Contains(gravador.Body.String(), "segredo interno") {
		t.Errorf("status %d, corpo %s", gravador.Code, gravador.Body.String())
	}
}

func TestReversaoSemDesfechosOuGitHubConfiguradosFicaIndisponivel(t *testing.T) {
	recibos := novoCenarioDeRecibos(t, 1)
	recibos.servidor = NovoServidor(Opcoes{Recibos: recibos.recibosVerificaveis()}).Rotas()

	if gravador := recibos.pedir(t, "/receipts/"+recibos.ids[0]+"/revert"); gravador.Code != http.StatusServiceUnavailable {
		t.Errorf("status %d, esperava 503", gravador.Code)
	}
}

func TestReversaoESomenteLeituraEExigeIdValido(t *testing.T) {
	cenario := novoCenarioDeReversao(t, desfechoMergeado(1), http.StatusOK, commitComDoisPais)

	if gravador := cenario.pedir(t, "/receipts/nao-e-uuid/revert"); gravador.Code != http.StatusNotFound {
		t.Errorf("id malformado: status %d", gravador.Code)
	}
	for _, metodo := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		gravador := httptest.NewRecorder()
		cenario.servidor.ServeHTTP(gravador, httptest.NewRequestWithContext(t.Context(), metodo, "/receipts/"+cenario.ids[1]+"/revert", nil))
		if gravador.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: status %d, esperava 405", metodo, gravador.Code)
		}
	}
}

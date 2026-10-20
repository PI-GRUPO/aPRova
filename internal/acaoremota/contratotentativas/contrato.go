// Package contratotentativas é a suíte de contrato que toda implementação de
// acaoremota.RegistroTentativas precisa cumprir, para que a versão em memória
// e a PostgreSQL sejam trocáveis sem mudar o comportamento.
package contratotentativas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Azulpasta/aPRova/internal/acaoremota"
	"github.com/Azulpasta/aPRova/internal/platform"
)

const (
	usuarioEscritor   = "U-ESCRITORA"
	usuarioLeitor     = "U-LEITORA"
	usuarioSemVinculo = "U-NOVA"
	loginEscritor     = "escritora"
	loginLeitor       = "leitora"
	installationID    = 7001
)

// Implementacao é o que cada versão do registro entrega à suíte: como
// completar as opções do serviço com ela e como ler, na ordem de gravação, as
// decisões registradas para um id_acao.
type Implementacao struct {
	Completar func(opcoes *acaoremota.OpcoesServico)
	Historico func(t *testing.T, idAcao string) []acaoremota.Decisao
}

// Verificar roda a suíte de contrato contra a implementação que nova constrói
// para cada caso.
func Verificar(t *testing.T, nova func(t *testing.T) Implementacao) {
	t.Run("recusada e depois aceita com o mesmo id_acao", func(t *testing.T) {
		recusadaEDepoisAceita(t, nova(t))
	})
	t.Run("duas aceitas simultâneas com o mesmo id_acao", func(t *testing.T) {
		aceitasSimultaneas(t, nova(t))
	})
	t.Run("aceita, o GitHub falha e uma nova tentativa executa", func(t *testing.T) {
		falhaNoGitHubLiberaNovaTentativa(t, nova(t))
	})
	t.Run("recusada duas vezes", func(t *testing.T) {
		recusadaDuasVezes(t, nova(t))
	})
}

func recusadaEDepoisAceita(t *testing.T, implementacao Implementacao) {
	c := novoCenario(t, implementacao)
	idAcao := novoIDAcao(t)

	primeira := c.processar(t, idAcao, usuarioSemVinculo)
	c.identidades.vincular(usuarioSemVinculo, loginEscritor)
	segunda := c.processar(t, idAcao, usuarioSemVinculo)

	exigirDecisoes(t, []acaoremota.Decisao{primeira, segunda}, acaoremota.DecisaoRecusada, acaoremota.DecisaoAceita)
	exigirHistorico(t, implementacao.Historico(t, idAcao), acaoremota.DecisaoRecusada, acaoremota.DecisaoAceita)
	c.exigirExecucoes(t, 1)
}

func aceitasSimultaneas(t *testing.T, implementacao Implementacao) {
	c := novoCenario(t, implementacao)
	idAcao := novoIDAcao(t)

	var grupo sync.WaitGroup
	decisoes := make([]acaoremota.Decisao, 2)
	for indice := range decisoes {
		grupo.Go(func() {
			decisoes[indice] = c.processar(t, idAcao, usuarioEscritor)
		})
	}
	grupo.Wait()

	exigirMesmasDecisoes(t, decisoes, acaoremota.DecisaoAceita, acaoremota.DecisaoRepetida)
	exigirMesmasDecisoes(t, implementacao.Historico(t, idAcao), acaoremota.DecisaoAceita, acaoremota.DecisaoRepetida)
	c.exigirExecucoes(t, 1)
}

func falhaNoGitHubLiberaNovaTentativa(t *testing.T, implementacao Implementacao) {
	c := novoCenario(t, implementacao)
	idAcao := novoIDAcao(t)
	c.executor.falharNaProxima()

	primeira := c.processar(t, idAcao, usuarioEscritor)
	segunda := c.processar(t, idAcao, usuarioEscritor)

	exigirDecisoes(t, []acaoremota.Decisao{primeira, segunda}, acaoremota.DecisaoFalha, acaoremota.DecisaoAceita)
	exigirHistorico(t, implementacao.Historico(t, idAcao), acaoremota.DecisaoFalha, acaoremota.DecisaoAceita)
	c.exigirExecucoes(t, 1)
}

func recusadaDuasVezes(t *testing.T, implementacao Implementacao) {
	c := novoCenario(t, implementacao)
	idAcao := novoIDAcao(t)

	primeira := c.processar(t, idAcao, usuarioLeitor)
	segunda := c.processar(t, idAcao, usuarioLeitor)

	exigirDecisoes(t, []acaoremota.Decisao{primeira, segunda}, acaoremota.DecisaoRecusada, acaoremota.DecisaoRecusada)
	exigirHistorico(t, implementacao.Historico(t, idAcao), acaoremota.DecisaoRecusada, acaoremota.DecisaoRecusada)
	c.exigirExecucoes(t, 0)
}

type cenario struct {
	servico     *acaoremota.Servico
	identidades *identidadesMutaveis
	executor    *executorContador
}

func novoCenario(t *testing.T, implementacao Implementacao) *cenario {
	t.Helper()

	c := &cenario{
		identidades: &identidadesMutaveis{loginPorUsuario: map[string]string{
			usuarioEscritor: loginEscritor,
			usuarioLeitor:   loginLeitor,
		}},
		executor: &executorContador{},
	}

	opcoes := acaoremota.OpcoesServico{
		Identidades: c.identidades,
		Permissoes:  permissoesFixas{},
		Autoria:     autoriaFixa{},
		Executor:    c.executor,
	}
	implementacao.Completar(&opcoes)

	servico, err := acaoremota.NovoServico(opcoes)
	if err != nil {
		t.Fatalf("montar o serviço com a implementação: %v", err)
	}
	c.servico = servico
	return c
}

func (c *cenario) processar(t *testing.T, idAcao, usuarioSlack string) acaoremota.Decisao {
	t.Helper()

	corpo, err := json.Marshal(map[string]any{
		"id_acao":         idAcao,
		"tipo":            "aprovar",
		"usuario_slack":   usuarioSlack,
		"repositorio":     "PI-GRUPO/alvo",
		"numero_pr":       42,
		"installation_id": installationID,
	})
	if err != nil {
		t.Fatalf("serializar pedido: %v", err)
	}
	return c.servico.Processar(context.Background(), corpo)
}

func (c *cenario) exigirExecucoes(t *testing.T, esperadas int64) {
	t.Helper()

	if efetivas := c.executor.bemSucedidas.Load(); efetivas != esperadas {
		t.Errorf("a ação foi executada no GitHub %d vez(es), esperava %d", efetivas, esperadas)
	}
}

var sequenciaDeIDs atomic.Int64

func novoIDAcao(t *testing.T) string {
	nome := strings.NewReplacer("/", "-", " ", "-").Replace(t.Name())
	return fmt.Sprintf("contrato-%s-%d", nome, sequenciaDeIDs.Add(1))
}

func exigirDecisoes(t *testing.T, obtidas []acaoremota.Decisao, esperadas ...acaoremota.Decisao) {
	t.Helper()

	if !slices.Equal(obtidas, esperadas) {
		t.Errorf("decisões devolvidas = %v, esperava %v", obtidas, esperadas)
	}
}

func exigirHistorico(t *testing.T, historico []acaoremota.Decisao, esperado ...acaoremota.Decisao) {
	t.Helper()

	if !slices.Equal(historico, esperado) {
		t.Errorf("histórico registrado = %v, esperava %v", historico, esperado)
	}
}

func exigirMesmasDecisoes(t *testing.T, obtidas []acaoremota.Decisao, esperadas ...acaoremota.Decisao) {
	t.Helper()

	ordenadas := slices.Sorted(slices.Values(obtidas))
	if !slices.Equal(ordenadas, slices.Sorted(slices.Values(esperadas))) {
		t.Errorf("decisões = %v, esperava %v em qualquer ordem", obtidas, esperadas)
	}
}

type identidadesMutaveis struct {
	mutex           sync.Mutex
	loginPorUsuario map[string]string
}

func (i *identidadesMutaveis) LoginGitHub(_ context.Context, usuarioSlack string) (string, error) {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	login, vinculado := i.loginPorUsuario[usuarioSlack]
	if !vinculado {
		return "", acaoremota.ErrSemVinculo
	}
	return login, nil
}

func (i *identidadesMutaveis) vincular(usuarioSlack, login string) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	i.loginPorUsuario[usuarioSlack] = login
}

type permissoesFixas struct{}

func (permissoesFixas) PermissaoNoRepositorio(_ context.Context, _ int64, _, _, login string) (platform.Permissao, error) {
	if login == loginEscritor {
		return platform.PermissaoWrite, nil
	}
	return platform.PermissaoRead, nil
}

type autoriaFixa struct{}

func (autoriaFixa) AutorDoPullRequest(context.Context, int64, string, string, int) (string, error) {
	return "autora-do-pr", nil
}

var errGitHubFora = errors.New("github indisponível na suíte de contrato")

type executorContador struct {
	mutex         sync.Mutex
	falharProxima bool
	bemSucedidas  atomic.Int64
}

func (e *executorContador) falharNaProxima() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.falharProxima = true
}

func (e *executorContador) Executar(context.Context, acaoremota.AcaoAutorizada) error {
	e.mutex.Lock()
	falhar := e.falharProxima
	e.falharProxima = false
	e.mutex.Unlock()

	if falhar {
		return errGitHubFora
	}
	e.bemSucedidas.Add(1)
	return nil
}

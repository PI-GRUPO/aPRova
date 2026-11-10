package worker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/eventos"
	"github.com/Azulpasta/aPRova/internal/gate"
	"github.com/Azulpasta/aPRova/internal/platform"
	"github.com/Azulpasta/aPRova/internal/rastro"
	"github.com/Azulpasta/aPRova/internal/sandbox"
)

const (
	trabalhadorDeTeste = "trabalhador-de-teste"
	esperaBaseDeTeste  = 10 * time.Second
	urlPublicaDeTeste  = "https://api.aprova.test"
)

type saidaSincronizada struct {
	mutex sync.Mutex
	saida bytes.Buffer
}

func (s *saidaSincronizada) Write(p []byte) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.saida.Write(p)
}

func (s *saidaSincronizada) linhas() []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var linhas []string
	leitor := bufio.NewScanner(bytes.NewReader(s.saida.Bytes()))
	for leitor.Scan() {
		linhas = append(linhas, leitor.Text())
	}
	return linhas
}

type cenario struct {
	armazenamento *armazenamentoFalso
	leitor        *leitorFalso
	permissoes    permissoesFalsas
	sandbox       *sandboxFalso
	avaliador     *avaliadorFalso
	chamadas      *registroDeChamadas
	publicador    *publicadorFalso
	eventos       *emissorDeEventosFalso
	diario        *saidaSincronizada
	opcoes        Opcoes
}

func novoCenario(nivel domain.NivelRisco, resultados ...sandbox.ResultadoSandbox) *cenario {
	if len(resultados) == 0 {
		resultados = []sandbox.ResultadoSandbox{sucessoNoSandbox}
	}
	c := &cenario{
		armazenamento: novoArmazenamentoFalso(),
		leitor:        &leitorFalso{},
		permissoes:    permissoesFalsas{"Azulpasta": platform.PermissaoAdmin, "httpslua": platform.PermissaoWrite, "leitora": platform.PermissaoRead},
		sandbox:       &sandboxFalso{resultados: resultados},
		avaliador:     &avaliadorFalso{nivel: nivel},
		chamadas:      &registroDeChamadas{},
		eventos:       &emissorDeEventosFalso{},
		diario:        &saidaSincronizada{},
	}
	c.publicador = &publicadorFalso{chamadas: c.chamadas}
	c.opcoes = Opcoes{
		Armazenamento: c.armazenamento,
		LeitorDePR:    c.leitor,
		Permissoes:    c.permissoes,
		Sandbox:       c.sandbox,
		Avaliador:     c.avaliador,
		Recibos:       &emissorDeReciboFalso{armazenamento: c.armazenamento, chamadas: c.chamadas},
		Publicador:    c.publicador,
		Eventos:       c.eventos,
		Diretorios:    map[string]string{"Azulpasta/aPRova-TESTES": "go"},
		URLPublica:    urlPublicaDeTeste,
		MaxTentativas: 3,
		EsperaBase:    esperaBaseDeTeste,
		Diario:        slog.New(slog.NewJSONHandler(c.diario, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}
	return c
}

func (c *cenario) motor(t *testing.T) *Motor {
	t.Helper()

	motor, err := NovoMotor(c.opcoes)
	if err != nil {
		t.Fatalf("montar o motor: %v", err)
	}
	return motor
}

func (c *cenario) conduzirTudo(t *testing.T) {
	t.Helper()

	motor := c.motor(t)
	for range 40 {
		taskRun, encontrado, _ := c.armazenamento.Reivindicar(t.Context(), trabalhadorDeTeste, time.Minute)
		if !encontrado {
			return
		}
		desfecho := motor.Conduzir(t.Context(), trabalhadorDeTeste, taskRun, nil)
		if desfecho.Liberar {
			if err := c.armazenamento.Liberar(t.Context(), taskRun.ID, trabalhadorDeTeste, desfecho.Espera); err != nil {
				t.Fatalf("liberar: %v", err)
			}
		}
	}
	t.Fatal("o motor não parou em 40 reivindicações")
}

func exigirEstados(t *testing.T, obtidos []domain.EstadoTaskRun, esperados ...domain.EstadoTaskRun) {
	t.Helper()

	if !slices.Equal(obtidos, esperados) {
		t.Errorf("estados percorridos = %v, esperava %v", obtidos, esperados)
	}
}

func exigirEventoValido(t *testing.T, pedido eventos.Pedido) eventos.Evento {
	t.Helper()

	corpo, err := eventos.Montar(pedido)
	if err != nil {
		t.Fatalf("o evento %s está fora do contrato: %v", pedido.Tipo, err)
	}
	evento, err := eventos.Validar(corpo)
	if err != nil {
		t.Fatalf("validar %s: %v", pedido.Tipo, err)
	}
	return evento
}

func dadosDoEvento[T any](t *testing.T, pedido eventos.Pedido) *T {
	t.Helper()

	dados, doTipo := exigirEventoValido(t, pedido).Dados.(*T)
	if !doTipo {
		t.Fatalf("os dados de %s não são do tipo esperado", pedido.Tipo)
	}
	return dados
}

var caminhoAteAvaliar = []domain.EstadoTaskRun{domain.EstadoMapeandoContexto, domain.EstadoTestandoSandbox, domain.EstadoAvaliandoRisco}

func TestCaminhoDeRiscoBaixoOuMedioTerminaConcluidoComReciboAntesDaPublicacao(t *testing.T) {
	for _, nivel := range []domain.NivelRisco{domain.RiscoBaixo, domain.RiscoMedio} {
		t.Run(string(nivel), func(t *testing.T) {
			c := novoCenario(nivel)
			taskRun := c.armazenamento.novoTaskRun(14)

			c.conduzirTudo(t)

			exigirEstados(t, c.armazenamento.estadosPercorridos(taskRun.ID),
				append(slices.Clone(caminhoAteAvaliar), domain.EstadoAplicando, domain.EstadoConcluido)...)
			if chamadas := c.chamadas.lista(); !slices.Equal(chamadas, []string{"recibo", "publicacao"}) {
				t.Errorf("o recibo precisa vir antes da publicação, ordem: %v", chamadas)
			}
			if gate := c.armazenamento.gates[taskRun.ID]; gate.Nivel != nivel {
				t.Errorf("gate gravado = %+v", gate)
			}
			if tipos := c.eventos.tipos(); !slices.Equal(tipos, []eventos.TipoEvento{eventos.TipoAnaliseConcluida}) {
				t.Errorf("eventos = %v, esperava só analise_concluida", tipos)
			}
		})
	}
}

func TestCaminhoDeRiscoAltoParaEmAguardandoAprovacaoComRecibo(t *testing.T) {
	c := novoCenario(domain.RiscoAlto)
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	exigirEstados(t, c.armazenamento.estadosPercorridos(taskRun.ID), append(slices.Clone(caminhoAteAvaliar), domain.EstadoAguardandoAprovacao)...)
	if c.chamadas.quantas("recibo") != 1 || c.chamadas.quantas("publicacao") != 0 {
		t.Errorf("PR bloqueado tem recibo e não publica: %v", c.chamadas.lista())
	}
	if tipos := c.eventos.tipos(); !slices.Equal(tipos, []eventos.TipoEvento{eventos.TipoAnaliseConcluida, eventos.TipoAguardandoAprovacao}) {
		t.Fatalf("eventos = %v", tipos)
	}

	pedido, _ := c.eventos.doTipo(eventos.TipoAguardandoAprovacao)
	evento := exigirEventoValido(t, pedido)
	dados := dadosDoEvento[eventos.DadosAguardandoAprovacao](t, pedido)
	if dados.TaskRunID != taskRun.ID || dados.URLPR != "https://github.com/Azulpasta/aPRova-TESTES/pull/14" || dados.AutorPR != "wlaisw" {
		t.Errorf("dados do PR errados: %+v", dados)
	}
	recibo := c.armazenamento.recibos[taskRun.ID]
	if dados.ReceiptHash != hex.EncodeToString(recibo.Hash) || dados.URLRecibo != urlPublicaDeTeste+"/recibos/"+recibo.ID {
		t.Errorf("o evento precisa apontar o recibo emitido: hash %s, url %s", dados.ReceiptHash, dados.URLRecibo)
	}
	if evento.TenantID != taskRun.TenantID || evento.IDEvento != eventos.IDDoEvento(string(eventos.TipoAguardandoAprovacao), taskRun.ID) {
		t.Errorf("envelope errado: %+v", evento.Envelope)
	}
}

func TestSandboxRecebeORepositorioOSHAEODiretorioConfigurado(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	esperado := sandbox.PedidoExecucao{
		URLRepositorio: "https://github.com/Azulpasta/aPRova-TESTES",
		SHA:            taskRun.SHAHead,
		Linguagem:      sandbox.LinguagemGo,
		TaskRunID:      taskRun.ID,
		Diretorio:      "go",
	}
	if len(c.sandbox.pedidos) != 1 || c.sandbox.pedidos[0] != esperado {
		t.Errorf("pedidos ao sandbox = %+v, esperava %+v", c.sandbox.pedidos, esperado)
	}
}

func TestFalhaDeInfraestruturaDuasVezesDepoisSucesso(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo, falhaDeInfra, falhaDeInfra, sucessoNoSandbox)
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if tentativas := c.armazenamento.tentativasGravadas(taskRun.ID); !slices.Equal(tentativas, []int{1, 2, 3}) {
		t.Errorf("tentativas gravadas = %v, esperava [1 2 3]", tentativas)
	}
	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoConcluido {
		t.Errorf("depois da terceira tentativa o fluxo segue normal, terminou em %s", estado)
	}
	if esperas := c.armazenamento.esperas; !slices.Equal(esperas, []time.Duration{esperaBaseDeTeste, 2 * esperaBaseDeTeste}) {
		t.Errorf("esperas entre tentativas = %v, esperava crescentes a partir de %v", esperas, esperaBaseDeTeste)
	}
	if _, emitido := c.eventos.doTipo(eventos.TipoFalhaRecuperavel); emitido {
		t.Error("falha_recuperavel só sai quando as tentativas se esgotam")
	}
}

func TestInfraestruturaAteEsgotarFalhaEEmiteFalhaRecuperavel(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo, falhaDeInfra, falhaDeInfra, falhaDeInfra, sucessoNoSandbox)
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoFalhou {
		t.Fatalf("esgotadas as tentativas o task_run falha, terminou em %s", estado)
	}
	if tentativas := c.armazenamento.tentativasGravadas(taskRun.ID); !slices.Equal(tentativas, []int{1, 2, 3}) {
		t.Errorf("tentativas gravadas = %v", tentativas)
	}
	if c.avaliador.avaliacoes != 0 || c.chamadas.quantas("recibo") != 0 {
		t.Error("sem resultado do sandbox não há avaliação nem recibo")
	}
	pedido, emitido := c.eventos.doTipo(eventos.TipoFalhaRecuperavel)
	if !emitido {
		t.Fatal("falta o evento falha_recuperavel")
	}
	dados := dadosDoEvento[eventos.DadosFalhaRecuperavel](t, pedido)
	if dados.Etapa != domain.EstadoTestandoSandbox || dados.Tentativa != 3 || !strings.Contains(dados.Diagnostico, "não começou") {
		t.Errorf("falha_recuperavel errado: %+v", dados)
	}
}

func TestFalhaDeTesteSegueParaAvaliacao(t *testing.T) {
	for _, resultado := range []sandbox.ResultadoSandbox{testeFalhou, {SaidaBruta: "undefined: Calcular", TipoFalha: sandbox.FalhaCompilacao}} {
		t.Run(string(resultado.TipoFalha), func(t *testing.T) {
			c := novoCenario(domain.RiscoBaixo, resultado)
			taskRun := c.armazenamento.novoTaskRun(14)

			c.conduzirTudo(t)

			if c.sandbox.execucoes() != 1 || len(c.avaliador.entradas) != 1 || c.avaliador.entradas[0].Sandbox != resultado {
				t.Errorf("falha de %s vai para o gate sem nova tentativa: execuções %d, entradas %+v", resultado.TipoFalha, c.sandbox.execucoes(), c.avaliador.entradas)
			}
			if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoConcluido {
				t.Errorf("terminou em %s", estado)
			}
		})
	}
}

func TestGitHubIndisponivelAoMapearContextoTentaDeNovo(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	limite := fmt.Errorf("%w: 429", platform.ErrLimiteDeRequisicoes)
	c.leitor.erros = []error{errGitHubFora, limite}
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoConcluido {
		t.Errorf("5xx e limite de requisições são transitórios, terminou em %s", estado)
	}
	if len(c.armazenamento.esperas) != 2 {
		t.Errorf("esperava duas esperas antes de nova tentativa, houve %v", c.armazenamento.esperas)
	}
}

func TestPullRequestInexistenteFalhaSemNovaTentativa(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	c.leitor.erros = []error{fmt.Errorf("%w: GET /repos/Azulpasta/aPRova-TESTES/pulls/14/files", platform.ErrNaoEncontrado)}
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoFalhou {
		t.Fatalf("404 é definitivo, terminou em %s", estado)
	}
	if c.leitor.leituras != 1 || len(c.armazenamento.esperas) != 0 {
		t.Errorf("404 não tem nova tentativa: leituras %d, esperas %v", c.leitor.leituras, c.armazenamento.esperas)
	}
	if _, emitido := c.eventos.doTipo(eventos.TipoFalhaRecuperavel); emitido {
		t.Error("404 não é falha recuperável")
	}
}

func TestGitHubIndisponivelAteEsgotarEmiteFalhaRecuperavel(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	c.leitor.erros = []error{errGitHubFora, errGitHubFora, errGitHubFora}
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoFalhou {
		t.Fatalf("terminou em %s", estado)
	}
	pedido, emitido := c.eventos.doTipo(eventos.TipoFalhaRecuperavel)
	if !emitido {
		t.Fatal("falta falha_recuperavel")
	}
	if dados := dadosDoEvento[eventos.DadosFalhaRecuperavel](t, pedido); dados.Etapa != domain.EstadoMapeandoContexto || dados.Tentativa != 3 {
		t.Errorf("falha_recuperavel errado: %+v", dados)
	}
}

func levarAteAguardando(t *testing.T, c *cenario) domain.TaskRun {
	t.Helper()

	taskRun := c.armazenamento.novoTaskRun(14)
	c.conduzirTudo(t)
	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoAguardandoAprovacao {
		t.Fatalf("preparação: esperava aguardando_aprovacao, está %s", estado)
	}
	return taskRun
}

func TestReviewAprovadoAplicaEConclui(t *testing.T) {
	c := novoCenario(domain.RiscoAlto)
	taskRun := levarAteAguardando(t, c)
	aprovacao := c.armazenamento.registrarAprovacao(taskRun.ID, domain.DecisaoAprovada, "Azulpasta", domain.OrigemSlack)

	c.conduzirTudo(t)

	exigirEstados(t, c.armazenamento.estadosPercorridos(taskRun.ID),
		append(slices.Clone(caminhoAteAvaliar), domain.EstadoAguardandoAprovacao, domain.EstadoAplicando, domain.EstadoConcluido)...)
	if len(c.publicador.publicados) != 1 || c.chamadas.quantas("recibo") != 1 {
		t.Errorf("aprovado publica uma vez, com o recibo já emitido: %v", c.chamadas.lista())
	}
	pedido, emitido := c.eventos.doTipo(eventos.TipoAprovacaoRegistrada)
	if !emitido {
		t.Fatal("falta aprovacao_registrada")
	}
	dados := dadosDoEvento[eventos.DadosAprovacaoRegistrada](t, pedido)
	if dados.Decisao != domain.DecisaoAprovada || dados.LoginGitHub != "Azulpasta" || dados.Origem != "slack" ||
		dados.NivelRiscoOriginal != domain.RiscoAlto || dados.IDAprovacao != aprovacao.ID {
		t.Errorf("aprovacao_registrada errado: %+v", dados)
	}
}

func TestReviewComAlteracoesSolicitadasRejeita(t *testing.T) {
	c := novoCenario(domain.RiscoAlto)
	taskRun := levarAteAguardando(t, c)
	c.armazenamento.registrarAprovacao(taskRun.ID, domain.DecisaoAlteracoesSolicitadas, "httpslua", domain.OrigemGitHub)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoRejeitado {
		t.Fatalf("terminou em %s", estado)
	}
	if c.chamadas.quantas("publicacao") != 0 {
		t.Error("rejeitado não publica")
	}
	pedido, _ := c.eventos.doTipo(eventos.TipoAprovacaoRegistrada)
	if dados := dadosDoEvento[eventos.DadosAprovacaoRegistrada](t, pedido); dados.Decisao != domain.DecisaoAlteracoesSolicitadas || dados.Origem != "github" {
		t.Errorf("aprovacao_registrada errado: %+v", dados)
	}
}

func TestReviewDeQuemNaoPodeEscreverNaoTiraDaEspera(t *testing.T) {
	c := novoCenario(domain.RiscoAlto)
	taskRun := levarAteAguardando(t, c)
	c.armazenamento.registrarAprovacao(taskRun.ID, domain.DecisaoAprovada, "leitora", domain.OrigemGitHub)
	c.armazenamento.registrarAprovacao(taskRun.ID, domain.DecisaoAprovada, "desconhecido", domain.OrigemGitHub)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoAguardandoAprovacao {
		t.Fatalf("review de quem só lê não libera o PR, foi para %s", estado)
	}
	if _, emitido := c.eventos.doTipo(eventos.TipoAprovacaoRegistrada); emitido {
		t.Error("review sem permissão não é decisão")
	}

	c.armazenamento.registrarAprovacao(taskRun.ID, domain.DecisaoAprovada, "httpslua", domain.OrigemGitHub)
	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoConcluido {
		t.Errorf("o review seguinte, de quem escreve, decide; terminou em %s", estado)
	}
}

func TestReviewAnteriorAEsperaNaoDecide(t *testing.T) {
	c := novoCenario(domain.RiscoAlto)
	taskRun := c.armazenamento.novoTaskRun(14)
	c.armazenamento.registrarAprovacao(taskRun.ID, domain.DecisaoAprovada, "Azulpasta", domain.OrigemGitHub)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoAguardandoAprovacao {
		t.Errorf("review feito antes do bloqueio não decide, terminou em %s", estado)
	}
}

func TestTaskRunSubstituidoEmEsperaEncerraSemPublicar(t *testing.T) {
	c := novoCenario(domain.RiscoAlto)
	antigo := levarAteAguardando(t, c)
	novo := c.armazenamento.novoTaskRun(14)
	c.armazenamento.registrarAprovacao(antigo.ID, domain.DecisaoAprovada, "Azulpasta", domain.OrigemGitHub)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(antigo.ID).Estado; estado != domain.EstadoFalhou {
		t.Fatalf("o task_run antigo encerra em falhou, está em %s", estado)
	}
	if etapa := c.armazenamento.ultimaEtapa(antigo.ID); etapa.detalhe != "substituído por commit mais novo" {
		t.Errorf("motivo do encerramento = %q", etapa.detalhe)
	}
	for _, publicacao := range c.publicador.publicados {
		if publicacao.TaskRun.ID == antigo.ID {
			t.Error("um task_run substituído publicou no PR")
		}
	}
	if estado := c.armazenamento.taskRun(novo.ID).Estado; estado != domain.EstadoAguardandoAprovacao {
		t.Errorf("o task_run novo segue o fluxo próprio, está em %s", estado)
	}
}

func TestTaskRunSubstituidoNaPublicacaoNaoSobrescreveONovo(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo, sucessoNoSandbox, sucessoNoSandbox)
	c.publicador.erros = []error{fmt.Errorf("%w: POST check-runs respondeu 503", platform.ErrGitHubIndisponivel)}
	antigo := c.armazenamento.novoTaskRun(14)
	motor := c.motor(t)

	for tentativa := 0; c.armazenamento.taskRun(antigo.ID).Estado != domain.EstadoAplicando; tentativa++ {
		if tentativa == 10 {
			t.Fatalf("preparação: o task_run antigo não chegou a aplicando, está em %s", c.armazenamento.taskRun(antigo.ID).Estado)
		}
		reivindicado, _, _ := c.armazenamento.Reivindicar(t.Context(), trabalhadorDeTeste, time.Minute)
		desfecho := motor.Conduzir(t.Context(), trabalhadorDeTeste, reivindicado, nil)
		_ = c.armazenamento.Liberar(t.Context(), reivindicado.ID, trabalhadorDeTeste, desfecho.Espera)
	}
	novo := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(antigo.ID).Estado; estado != domain.EstadoFalhou {
		t.Errorf("o antigo encerra em falhou, está em %s", estado)
	}
	if len(c.publicador.publicados) != 1 || c.publicador.publicados[0].TaskRun.ID != novo.ID {
		t.Errorf("só o task_run novo publica: %+v", c.publicador.publicados)
	}
}

func TestRetomadaReaproveitaOResultadoDaTentativaAtual(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	taskRun := c.armazenamento.novoTaskRun(14)
	c.armazenamento.taskRuns[taskRun.ID].Estado = domain.EstadoTestandoSandbox
	if err := c.armazenamento.GravarResultadoDaTentativa(t.Context(), taskRun.ID, 1, testeFalhou); err != nil {
		t.Fatal(err)
	}

	c.conduzirTudo(t)

	if c.sandbox.execucoes() != 0 {
		t.Errorf("o resultado da tentativa já existia, o sandbox rodou %d vez(es)", c.sandbox.execucoes())
	}
	if c.avaliador.entradas[0].Sandbox != testeFalhou {
		t.Errorf("a avaliação precisa usar o resultado gravado: %+v", c.avaliador.entradas[0].Sandbox)
	}
}

func gateGravado(taskRunID string) domain.GateDeRisco {
	return domain.GateDeRisco{
		TaskRunID: taskRunID, NivelDeterministico: domain.RiscoAlto, Nivel: domain.RiscoAlto, Motivo: "dependencia_nova",
		Motivos: []domain.MotivoDeRisco{
			{ID: "dependencia_nova", Nivel: domain.RiscoAlto, Descricao: "o PR acrescenta dependência nova", Evidencias: []string{"go/go.mod"}},
		},
		DependenciasNovas: []domain.DependenciaNova{{Ecossistema: "Go", Nome: "github.com/shopspring/decimal", Versao: "v1.4.0"}},
	}
}

func TestRetomadaComGateGravadoNaoChamaOGateNemOGitHub(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	c.avaliador.erro = errors.New("o gate não pode ser chamado de novo na retomada")
	c.leitor.erros = []error{errors.New("o GitHub não pode ser lido de novo na retomada")}
	taskRun := c.armazenamento.novoTaskRun(14)
	c.armazenamento.taskRuns[taskRun.ID].Estado = domain.EstadoAvaliandoRisco
	_ = c.armazenamento.GravarResultadoDaTentativa(t.Context(), taskRun.ID, 1, sucessoNoSandbox)
	_ = c.armazenamento.GravarGateDeRisco(t.Context(), gateGravado(taskRun.ID))
	c.armazenamento.recibos[taskRun.ID] = domain.Recibo{NovoRecibo: domain.NovoRecibo{ID: "8e0a2c4d-6f1b-4d3e-a5c7-9b0d1e2f3a4b", TaskRunID: taskRun.ID, Hash: make([]byte, 32)}}

	c.conduzirTudo(t)

	if c.avaliador.avaliacoes != 0 || c.leitor.leituras != 0 {
		t.Errorf("a retomada usa só o que está gravado: %d avaliações, %d leituras", c.avaliador.avaliacoes, c.leitor.leituras)
	}
	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoAguardandoAprovacao {
		t.Errorf("o nível gravado no gate decide o caminho, terminou em %s", estado)
	}
	if c.chamadas.quantas("recibo") != 0 {
		t.Error("o recibo já existia e foi emitido de novo")
	}
	pedido, _ := c.eventos.doTipo(eventos.TipoAnaliseConcluida)
	dados := dadosDoEvento[eventos.DadosAnaliseConcluida](t, pedido)
	if dados.URLRecibo != urlPublicaDeTeste+"/recibos/8e0a2c4d-6f1b-4d3e-a5c7-9b0d1e2f3a4b" || dados.NivelRisco != domain.RiscoAlto {
		t.Errorf("o evento precisa usar o gate e o recibo gravados: %+v", dados)
	}
	if len(dados.Motivos) != 1 || dados.Motivos[0].Nome != "dependencia_nova" || dados.Motivos[0].Descricao != "o PR acrescenta dependência nova" {
		t.Errorf("os motivos do evento vêm de rules_hit: %+v", dados.Motivos)
	}
	nova, emitido := c.eventos.doTipo(eventos.TipoDependenciaNova)
	if !emitido {
		t.Fatal("a retomada também emite dependencia_nova a partir do que está gravado")
	}
	if deps := dadosDoEvento[eventos.DadosDependenciaNova](t, nova).Dependencias; len(deps) != 1 || deps[0].Nome != "github.com/shopspring/decimal" {
		t.Errorf("dependências do evento: %+v", deps)
	}
}

func TestEventoDaRetomadaEIgualAoDaPrimeiraAvaliacao(t *testing.T) {
	dependencias := []domain.DependenciaNova{{Ecossistema: "PyPI", Nome: "requests", Versao: "2.32.3"}}
	agora := func() time.Time { return time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC) }

	primeira := novoCenario(domain.RiscoAlto)
	primeira.opcoes.Agora = agora
	primeira.avaliador.dependencias = dependencias
	taskRun := primeira.armazenamento.novoTaskRun(14)
	primeira.conduzirTudo(t)

	retomada := novoCenario(domain.RiscoBaixo)
	retomada.opcoes.Agora = agora
	retomada.avaliador.erro = errors.New("o gate não pode ser chamado de novo na retomada")
	retomada.armazenamento.novoTaskRun(14)
	retomada.armazenamento.taskRuns[taskRun.ID].Estado = domain.EstadoAvaliandoRisco
	_ = retomada.armazenamento.GravarResultadoDaTentativa(t.Context(), taskRun.ID, 1, sucessoNoSandbox)
	_ = retomada.armazenamento.GravarGateDeRisco(t.Context(), primeira.armazenamento.gates[taskRun.ID])
	retomada.armazenamento.recibos[taskRun.ID] = primeira.armazenamento.recibos[taskRun.ID]
	retomada.conduzirTudo(t)

	for _, tipo := range []eventos.TipoEvento{eventos.TipoAnaliseConcluida, eventos.TipoDependenciaNova, eventos.TipoAguardandoAprovacao} {
		original, _ := primeira.eventos.doTipo(tipo)
		repetido, emitido := retomada.eventos.doTipo(tipo)
		if !emitido {
			t.Fatalf("a retomada não emitiu %s", tipo)
		}
		corpoOriginal, err := eventos.Montar(original)
		if err != nil {
			t.Fatalf("montar %s: %v", tipo, err)
		}
		corpoRepetido, err := eventos.Montar(repetido)
		if err != nil {
			t.Fatalf("montar %s: %v", tipo, err)
		}
		if string(corpoOriginal) != string(corpoRepetido) {
			t.Errorf("%s mudou na retomada:\n%s\n%s", tipo, corpoOriginal, corpoRepetido)
		}
	}
}

func TestMotivosEDependenciasDoGateSaoGravados(t *testing.T) {
	c := novoCenario(domain.RiscoAlto)
	c.avaliador.dependencias = []domain.DependenciaNova{{Ecossistema: "Maven", Nome: "org.javamoney:moneta", Versao: "1.4.4"}}
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	gravado := c.armazenamento.gates[taskRun.ID]
	if gravado.NivelDeterministico != domain.RiscoAlto || gravado.Nivel != domain.RiscoAlto {
		t.Errorf("os dois níveis são iguais por enquanto: %+v", gravado)
	}
	if len(gravado.Motivos) != 1 || gravado.Motivos[0].ID != "area_sensivel" || !slices.Equal(gravado.Motivos[0].Evidencias, []string{"go/frete/frete.go"}) {
		t.Errorf("motivos gravados: %+v", gravado.Motivos)
	}
	if !slices.Equal(gravado.DependenciasNovas, c.avaliador.dependencias) {
		t.Errorf("dependências gravadas: %+v", gravado.DependenciasNovas)
	}
	if !strings.Contains(gravado.Motivo, "area_sensivel") {
		t.Errorf("reason resume os motivos: %q", gravado.Motivo)
	}
}

func TestDependenciaNovaEmiteOEvento(t *testing.T) {
	c := novoCenario(domain.RiscoAlto)
	c.avaliador.dependencias = []domain.DependenciaNova{
		{Ecossistema: "Go", Nome: "github.com/shopspring/decimal", Versao: "v1.4.0"},
		{Ecossistema: "PyPI", Nome: "requests", Versao: "2.32.3"},
	}
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	pedido, emitido := c.eventos.doTipo(eventos.TipoDependenciaNova)
	if !emitido {
		t.Fatalf("falta dependencia_nova; eventos: %v", c.eventos.tipos())
	}
	dados := dadosDoEvento[eventos.DadosDependenciaNova](t, pedido)
	if dados.TaskRunID != taskRun.ID || dados.URLPR != "https://github.com/Azulpasta/aPRova-TESTES/pull/14" || len(dados.Dependencias) != 2 {
		t.Errorf("dependencia_nova errado: %+v", dados)
	}
	if primeira := dados.Dependencias[0]; primeira.Ecossistema != "Go" || primeira.Nome != "github.com/shopspring/decimal" || primeira.Versao != "v1.4.0" {
		t.Errorf("dependência errada: %+v", primeira)
	}
}

func TestSemDependenciaNovaNaoHaEvento(t *testing.T) {
	c := novoCenario(domain.RiscoAlto)
	c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if _, emitido := c.eventos.doTipo(eventos.TipoDependenciaNova); emitido {
		t.Error("sem dependência nova não há evento")
	}
}

func TestGateRecebeOsArquivosDoPullRequest(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if len(c.avaliador.entradas) != 1 || len(c.avaliador.entradas[0].Arquivos) != 1 || c.avaliador.entradas[0].Arquivos[0].Nome != "go/frete/frete.go" {
		t.Errorf("o gate avalia os arquivos lidos do GitHub: %+v", c.avaliador.entradas)
	}
}

func TestEventoSemMotivosFicaDentroDoContrato(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	pedido, _ := c.eventos.doTipo(eventos.TipoAnaliseConcluida)
	dados := dadosDoEvento[eventos.DadosAnaliseConcluida](t, pedido)
	if len(dados.Motivos) != 0 || dados.Explicacao == "" {
		t.Errorf("risco baixo sem motivos ainda precisa de uma explicação: %+v", dados)
	}
}

func TestErroDoGateEsgotaAsTentativasSemDecidirNivel(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	c.avaliador.erro = fmt.Errorf("%w: tipo_falha infraestrutura", gate.ErrResultadoSemVeredito)
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoFalhou {
		t.Errorf("sem veredito do gate o task_run não segue adiante, terminou em %s", estado)
	}
	if _, gravado := c.armazenamento.gates[taskRun.ID]; gravado {
		t.Error("sem veredito não há gate gravado")
	}
	if c.chamadas.quantas("publicacao") != 0 {
		t.Error("sem veredito não há publicação")
	}
}

func TestPrazoPerdidoParaSemMudarNadaESemLiberar(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	taskRun := c.armazenamento.novoTaskRun(14)
	reivindicado, _, _ := c.armazenamento.Reivindicar(t.Context(), trabalhadorDeTeste, time.Minute)
	c.armazenamento.taskRuns[taskRun.ID].ReivindicadoPor = "outro-trabalhador"

	desfecho := c.motor(t).Conduzir(t.Context(), trabalhadorDeTeste, reivindicado, nil)

	if desfecho.Liberar {
		t.Error("quem perdeu o prazo não pode liberar o task_run de outro")
	}
	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoPendente {
		t.Errorf("quem perdeu o prazo mudou o estado para %s", estado)
	}
}

func TestDesligandoParaDepoisDaEtapaEmAndamento(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	taskRun := c.armazenamento.novoTaskRun(14)
	reivindicado, _, _ := c.armazenamento.Reivindicar(t.Context(), trabalhadorDeTeste, time.Minute)
	parando := make(chan struct{})
	close(parando)

	desfecho := c.motor(t).Conduzir(t.Context(), trabalhadorDeTeste, reivindicado, parando)

	if !desfecho.Liberar || desfecho.Espera != 0 {
		t.Errorf("ao desligar, o task_run volta na hora para a fila: %+v", desfecho)
	}
	if estados := c.armazenamento.estadosPercorridos(taskRun.ID); len(estados) > 1 {
		t.Errorf("desligando, o motor não começa etapa nova: %v", estados)
	}
}

func TestEmissorDeEventosForaDoArNaoAtrasaNemQuebraOMotor(t *testing.T) {
	travado := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
	}))
	t.Cleanup(travado.Close)
	emissor, err := eventos.NovoEmissor(eventos.OpcoesEmissor{URL: travado.URL, Segredo: []byte("segredo"), TempoLimite: 3 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	c := novoCenario(domain.RiscoAlto)
	c.opcoes.Eventos = emissor
	taskRun := c.armazenamento.novoTaskRun(14)

	inicio := time.Now()
	c.conduzirTudo(t)

	if demorou := time.Since(inicio); demorou > time.Second {
		t.Errorf("o n8n travado segurou o motor por %v", demorou)
	}
	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoAguardandoAprovacao {
		t.Errorf("o motor precisa seguir sem o n8n, terminou em %s", estado)
	}
	travado.CloseClientConnections()
	emissor.Aguardar()
}

func TestTodaLinhaDeLogDoProcessamentoTemOTaskRunID(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo, falhaDeInfra, sucessoNoSandbox)
	c.leitor.erros = []error{errGitHubFora}
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	linhas := c.diario.linhas()
	if len(linhas) == 0 {
		t.Fatal("o processamento não deixou nenhuma linha de log")
	}
	for _, linha := range linhas {
		var entrada map[string]any
		if err := json.Unmarshal([]byte(linha), &entrada); err != nil {
			t.Fatalf("linha de log não é JSON: %s", linha)
		}
		if entrada["task_run_id"] != taskRun.ID {
			t.Errorf("linha sem task_run_id: %s", linha)
		}
	}
}

func TestNovoMotorExigeTodasAsDependencias(t *testing.T) {
	completas := novoCenario(domain.RiscoBaixo).opcoes
	remocoes := map[string]func(*Opcoes){
		"Armazenamento": func(o *Opcoes) { o.Armazenamento = nil },
		"LeitorDePR":    func(o *Opcoes) { o.LeitorDePR = nil },
		"Permissoes":    func(o *Opcoes) { o.Permissoes = nil },
		"Sandbox":       func(o *Opcoes) { o.Sandbox = nil },
		"Avaliador":     func(o *Opcoes) { o.Avaliador = nil },
		"Recibos":       func(o *Opcoes) { o.Recibos = nil },
		"Publicador":    func(o *Opcoes) { o.Publicador = nil },
		"Eventos":       func(o *Opcoes) { o.Eventos = nil },
	}

	for dependencia, remover := range remocoes {
		t.Run(dependencia, func(t *testing.T) {
			opcoes := completas
			remover(&opcoes)

			if _, err := NovoMotor(opcoes); err == nil || !strings.Contains(err.Error(), dependencia) {
				t.Errorf("esperava erro nomeando %s, obtive %v", dependencia, err)
			}
		})
	}
}

type leitorQueRegistra struct {
	*leitorFalso
}

func (l leitorQueRegistra) ContextoDoPullRequest(ctx context.Context, installationID int64, dono, repositorio string, numero int) (platform.ContextoPR, error) {
	slog.InfoContext(ctx, "lendo o pull request no github")
	return l.leitorFalso.ContextoDoPullRequest(ctx, installationID, dono, repositorio, numero)
}

func TestDependenciasRecebemOContextoComOTaskRunID(t *testing.T) {
	saida := &saidaSincronizada{}
	anterior := slog.Default()
	slog.SetDefault(slog.New(rastro.NovoHandler(slog.NewJSONHandler(saida, nil))))
	t.Cleanup(func() { slog.SetDefault(anterior) })

	c := novoCenario(domain.RiscoBaixo)
	c.opcoes.LeitorDePR = leitorQueRegistra{c.leitor}
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	linhas := saida.linhas()
	if len(linhas) == 0 {
		t.Fatal("o leitor não registrou nada")
	}
	for _, linha := range linhas {
		if !strings.Contains(linha, `"task_run_id":"`+taskRun.ID+`"`) {
			t.Errorf("linha registrada por uma dependência sem o task_run_id: %s", linha)
		}
	}
}

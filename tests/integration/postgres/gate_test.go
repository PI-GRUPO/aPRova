package integracaopostgres

import (
	"context"
	"encoding/hex"
	"errors"
	"log/slog"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/eventos"
	"github.com/PI-GRUPO/aPRova/internal/gate"
	"github.com/PI-GRUPO/aPRova/internal/platform"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
	"github.com/PI-GRUPO/aPRova/internal/worker"
)

const patchDeDependenciaNova = "@@ -1,3 +1,5 @@\n module github.com/aprova/aprova-testes/go\n \n go 1.21\n+\n+require github.com/shopspring/decimal v1.4.0\n"

type leitorDeArquivos []platform.ArquivoAlterado

func (l leitorDeArquivos) ContextoDoPullRequest(context.Context, int64, string, string, int) (platform.ContextoPR, error) {
	return platform.ContextoPR{Arquivos: l, Diff: "diff"}, nil
}

type eventosGuardados struct {
	mutex   sync.Mutex
	pedidos []eventos.Pedido
}

func (e *eventosGuardados) Emitir(_ *slog.Logger, pedido eventos.Pedido) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.pedidos = append(e.pedidos, pedido)
}

func (e *eventosGuardados) doTipo(tipo eventos.TipoEvento) (eventos.Pedido, bool) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	for _, pedido := range e.pedidos {
		if pedido.Tipo == tipo {
			return pedido, true
		}
	}
	return eventos.Pedido{}, false
}

type gateQueNaoPodeSerChamado struct{}

func (gateQueNaoPodeSerChamado) Avaliar(context.Context, gate.Entrada) (gate.Resultado, error) {
	return gate.Resultado{}, errors.New("o gate não pode ser chamado de novo na retomada")
}

type leitorQueNaoPodeSerChamado struct{}

func (leitorQueNaoPodeSerChamado) ContextoDoPullRequest(context.Context, int64, string, string, int) (platform.ContextoPR, error) {
	return platform.ContextoPR{}, errors.New("o GitHub não pode ser lido de novo na retomada")
}

type montagemDoGate struct {
	banco      bancoDaFila
	leitor     worker.LeitorDePR
	avaliador  worker.AvaliadorRisco
	calibrador worker.CalibradorRisco
	sandbox    *sandboxContado
	publicador *publicadorContado
	eventos    *eventosGuardados
}

func novaMontagemDoGate(banco bancoDaFila, arquivos ...platform.ArquivoAlterado) montagemDoGate {
	return montagemDoGate{
		banco:      banco,
		leitor:     leitorDeArquivos(arquivos),
		avaliador:  gate.Novo(gate.Limites{}),
		sandbox:    &sandboxContado{},
		publicador: &publicadorContado{},
		eventos:    &eventosGuardados{},
	}
}

func (m montagemDoGate) rodarAte(t *testing.T, taskRunID string, estadoFinal domain.EstadoTaskRun) {
	t.Helper()

	motor, err := worker.NovoMotor(worker.Opcoes{
		Armazenamento: m.banco.repo,
		LeitorDePR:    m.leitor,
		Permissoes:    permissaoDeAdmin{},
		Sandbox:       m.sandbox,
		Avaliador:     m.avaliador,
		Calibrador:    m.calibrador,
		Recibos:       emissorDeRecibos(t, m.banco.repo),
		Publicador:    m.publicador,
		Eventos:       m.eventos,
		URLPublica:    "https://aprova.example.com",
		EsperaBase:    100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("montar o motor: %v", err)
	}
	pool, err := worker.NovoPool(worker.OpcoesPool{Motor: motor, Armazenamento: m.banco.repo, Nome: "worker-do-gate", Prazo: prazoDoWorker, IntervaloOcioso: 50 * time.Millisecond})
	if err != nil {
		t.Fatalf("montar o pool: %v", err)
	}

	ctx, cancelar := context.WithCancel(t.Context())
	defer cancelar()
	go pool.Rodar(ctx)

	esperarAte(t, 10*time.Second, "o task_run chegar a "+string(estadoFinal), func() bool { return m.banco.estadoDe(t, taskRunID) == estadoFinal })
}

func TestPRSoComTesteNovoEBaixoETerminaConcluido(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)
	montagem := novaMontagemDoGate(banco, platform.ArquivoAlterado{
		Nome: "go/frete/peso_zero_test.go", Situacao: "added", Adicoes: 8, Patch: "@@ -0,0 +1,3 @@\n+package frete\n+\n+func TestPesoZero(t *testing.T) {}\n",
	})

	montagem.rodarAte(t, taskRun.ID, domain.EstadoConcluido)

	lido, err := banco.repo.BuscarGateDeRisco(t.Context(), taskRun.ID)
	if err != nil {
		t.Fatalf("ler gate: %v", err)
	}
	if lido.Nivel != domain.RiscoBaixo || lido.NivelDeterministico != domain.RiscoBaixo || len(lido.Motivos) != 0 {
		t.Errorf("gate gravado: %+v", lido)
	}
	if publicacoes := montagem.publicador.publicacoes.Load(); publicacoes != 1 {
		t.Errorf("publicações: %d", publicacoes)
	}
	if _, emitido := montagem.eventos.doTipo(eventos.TipoDependenciaNova); emitido {
		t.Error("sem dependência nova não há evento")
	}
}

func TestPRComDependenciaNovaEAltoEEmiteOEvento(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)
	montagem := novaMontagemDoGate(banco, platform.ArquivoAlterado{Nome: "go/go.mod", Situacao: "modified", Adicoes: 2, Patch: patchDeDependenciaNova})

	montagem.rodarAte(t, taskRun.ID, domain.EstadoAguardandoAprovacao)

	nova := domain.DependenciaNova{Ecossistema: "Go", Nome: "github.com/shopspring/decimal", Versao: "v1.4.0"}
	lido, err := banco.repo.BuscarGateDeRisco(t.Context(), taskRun.ID)
	if err != nil {
		t.Fatalf("ler gate: %v", err)
	}
	if lido.Nivel != domain.RiscoAlto || len(lido.Motivos) != 1 || lido.Motivos[0].ID != "dependencia_nova" || !slices.Equal(lido.DependenciasNovas, []domain.DependenciaNova{nova}) {
		t.Errorf("gate gravado: %+v", lido)
	}
	if ids := banco.contar(t, `SELECT count(*) FROM risk_gates WHERE task_run_id = $1 AND rules_hit @> '[{"id":"dependencia_nova","evidencias":["go/go.mod"]}]'::jsonb`, taskRun.ID); ids != 1 {
		t.Error("rules_hit não guardou o motivo com a evidência")
	}

	pedido, emitido := montagem.eventos.doTipo(eventos.TipoDependenciaNova)
	if !emitido {
		t.Fatal("falta o evento dependencia_nova")
	}
	corpo, err := eventos.Montar(pedido)
	if err != nil {
		t.Fatalf("o evento está fora do contrato: %v", err)
	}
	evento, err := eventos.Validar(corpo)
	if err != nil {
		t.Fatalf("validar: %v", err)
	}
	dados, doTipo := evento.Dados.(*eventos.DadosDependenciaNova)
	esperada := []eventos.Dependencia{{Ecossistema: "Go", Nome: "github.com/shopspring/decimal", Versao: "v1.4.0"}}
	if !doTipo || !reflect.DeepEqual(dados.Dependencias, esperada) {
		t.Errorf("dependências do evento: %+v", evento.Dados)
	}
}

func TestRetomadaEmAvaliandoRiscoComGateGravadoNaoChamaOGate(t *testing.T) {
	banco := novoBancoDaFila(t)
	banco.novoTaskRunDoPR(t, 14)
	reivindicado, _ := banco.reivindicar(t, "worker-que-morreu")
	avaliando := banco.levarAte(t, "worker-que-morreu", reivindicado, domain.EstadoMapeandoContexto, domain.EstadoTestandoSandbox, domain.EstadoAvaliandoRisco)
	if err := banco.repo.GravarResultadoDaTentativa(t.Context(), avaliando.ID, 1, sandbox.ResultadoSandbox{Compilou: true, TestesPassaram: 4, SaidaBruta: "ok", TipoFalha: sandbox.FalhaNenhuma}); err != nil {
		t.Fatalf("gravar resultado: %v", err)
	}
	gravado := domain.GateDeRisco{
		TaskRunID: avaliando.ID, NivelDeterministico: domain.RiscoAlto, Nivel: domain.RiscoAlto, Motivo: "dependencia_nova: o PR acrescenta dependência nova",
		Motivos:           []domain.MotivoDeRisco{{ID: "dependencia_nova", Nivel: domain.RiscoAlto, Descricao: "o PR acrescenta dependência nova", Evidencias: []string{"go/go.mod"}}},
		DependenciasNovas: []domain.DependenciaNova{{Ecossistema: "Go", Nome: "github.com/shopspring/decimal", Versao: "v1.4.0"}},
		Calibracao:        domain.Calibracao{Estado: domain.CalibracaoDesligada, Razao: "calibração desligada"},
	}
	emitido, err := emissorDeRecibos(t, banco.repo).Emitir(t.Context(), receipt.Pedido{TaskRun: avaliando, Gate: gravado, Tentativa: 1, Diff: "diff"})
	if err != nil {
		t.Fatalf("gravar a decisão com o recibo: %v", err)
	}
	if err := banco.repo.Liberar(t.Context(), avaliando.ID, "worker-que-morreu", 0); err != nil {
		t.Fatalf("liberar: %v", err)
	}

	montagem := novaMontagemDoGate(banco)
	montagem.avaliador = gateQueNaoPodeSerChamado{}
	montagem.leitor = leitorQueNaoPodeSerChamado{}
	montagem.rodarAte(t, avaliando.ID, domain.EstadoAguardandoAprovacao)

	if recibos := banco.contar(t, `SELECT count(*) FROM receipts`); recibos != 1 {
		t.Errorf("a retomada não emite outro recibo, há %d", recibos)
	}
	analise, _ := montagem.eventos.doTipo(eventos.TipoAnaliseConcluida)
	if dados, doTipo := analise.Dados.(eventos.DadosAnaliseConcluida); !doTipo || dados.URLRecibo != "https://aprova.example.com/receipts/"+emitido.ID+"/verify" || dados.ReceiptHash != hex.EncodeToString(emitido.Hash) {
		t.Errorf("o evento da retomada aponta o recibo já gravado: %+v", analise.Dados)
	}
	if gates := banco.contar(t, `SELECT count(*) FROM risk_gates WHERE task_run_id = $1`, avaliando.ID); gates != 1 {
		t.Errorf("esperava uma linha em risk_gates, há %d", gates)
	}
	lido, err := banco.repo.BuscarGateDeRisco(t.Context(), avaliando.ID)
	if err != nil || !reflect.DeepEqual(lido, gravado) {
		t.Errorf("o gate gravado não pode mudar: %+v, err %v", lido, err)
	}
	if _, emitido := montagem.eventos.doTipo(eventos.TipoAnaliseConcluida); !emitido {
		t.Error("a retomada precisa emitir analise_concluida a partir de rules_hit")
	}
	if _, emitido := montagem.eventos.doTipo(eventos.TipoDependenciaNova); !emitido {
		t.Error("a retomada precisa emitir dependencia_nova a partir do que está gravado")
	}
}

package integracaopostgres

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/calibracao"
	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/embedding"
	"github.com/PI-GRUPO/aPRova/internal/gate"
	"github.com/PI-GRUPO/aPRova/internal/platform"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

const (
	tenantDaFila          = 58291734
	repositorioDaFila     = "Azulpasta/aPRova-TESTES"
	casosParecidos        = 6
	numeroDoPRCalibrado   = 14
	primeiroPRDoHistorico = 100
)

var arquivosNaAreaSensivel = []platform.ArquivoAlterado{
	{Nome: "go/auth/sessao.go", Situacao: "modified", Adicoes: 3, Remocoes: 1, Patch: "@@ -1 +1,3 @@\n+func validar() bool { return true }\n"},
	{Nome: "go/auth/sessao_test.go", Situacao: "modified", Adicoes: 2, Patch: "@@ -1 +1,2 @@\n+func TestValidar(t *testing.T) {}\n"},
}

var motivosDoHistorico = []domain.MotivoDeRisco{{ID: "area_sensivel", Nivel: domain.RiscoAlto, Descricao: "o PR altera um caminho sensível", Evidencias: []string{"go/auth/sessao.go"}}}

var resultadoVerde = sandbox.ResultadoSandbox{Compilou: true, TestesPassaram: 4, SaidaBruta: "ok", DuracaoMS: 900, TipoFalha: sandbox.FalhaNenhuma}

type embedderProibido struct{ chamadas atomic.Int64 }

func (e *embedderProibido) Gerar(context.Context, string) ([]float32, error) {
	e.chamadas.Add(1)
	return nil, errors.New("a retomada não pode chamar o embedding")
}

func calibradorDoFluxo(t *testing.T, banco bancoDaFila, embedder embedding.Embedder) *calibracao.Calibrador {
	t.Helper()
	calibrador, err := calibracao.NovoCalibrador(calibracao.OpcoesCalibrador{Embedder: embedder, Historico: banco.repo, Regra: calibracao.RegraPadrao(), Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("montar o calibrador: %v", err)
	}
	return calibrador
}

func vetorDaAreaSensivel(t *testing.T) []float32 {
	t.Helper()
	avaliacao, err := gate.Novo(gate.Limites{}).Avaliar(t.Context(), gate.Entrada{Arquivos: arquivosNaAreaSensivel, Sandbox: resultadoVerde})
	if err != nil {
		t.Fatalf("avaliar: %v", err)
	}
	if avaliacao.Nivel != domain.RiscoAlto || len(avaliacao.Motivos) != 1 || avaliacao.Motivos[0].ID != "area_sensivel" {
		t.Fatalf("o PR de referência precisa ser alto só por area_sensivel: %+v", avaliacao)
	}
	vetor, _ := embedding.Falso{}.Gerar(t.Context(), calibracao.Assinatura(avaliacao.Motivos, arquivosNaAreaSensivel))
	return vetor
}

func (b bancoDaFila) analiseConcluida(t *testing.T, numeroPR int, vetor []float32) domain.TaskRun {
	t.Helper()

	criado, _, err := b.repo.CriarTaskRun(t.Context(), domain.NovoTaskRun{
		TenantID: tenantDaFila, DeliveryID: fmt.Sprintf("historico-%d-%d", numeroPR, time.Now().UnixNano()),
		Repositorio: repositorioDaFila, NumeroPR: numeroPR, SHAHead: shaDoCommit(numeroPR), Autor: "wlaisw", Acao: "opened", RecebidoEm: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("criar a análise do PR %d: %v", numeroPR, err)
	}
	percorrer(t, b.repo, criado, domain.EstadoMapeandoContexto, domain.EstadoTestandoSandbox, domain.EstadoAvaliandoRisco,
		domain.EstadoAguardandoAprovacao, domain.EstadoAplicando, domain.EstadoConcluido)
	if _, err := b.repo.GravarMemoriaDeCodigo(t.Context(), domain.MemoriaDeCodigo{TenantID: tenantDaFila, TaskRunID: criado.ID, Resumo: "histórico", Embedding: vetor}); err != nil {
		t.Fatalf("gravar a assinatura do PR %d: %v", numeroPR, err)
	}
	err = b.repo.GravarGateDeRisco(t.Context(), domain.GateDeRisco{
		TaskRunID: criado.ID, NivelDeterministico: domain.RiscoAlto, Nivel: domain.RiscoAlto, Motivo: "area_sensivel: o PR altera um caminho sensível",
		Motivos: motivosDoHistorico,
	})
	if err != nil {
		t.Fatalf("gravar o gate do PR %d: %v", numeroPR, err)
	}
	return criado
}

func (b bancoDaFila) fecharPRDaFila(t *testing.T, numeroPR int, reverte int) {
	t.Helper()
	fechado := domain.FechamentoDePR{
		InstallationID: tenantDaFila, DeliveryID: fmt.Sprintf("fechamento-%d-%d", numeroPR, time.Now().UnixNano()), Repositorio: repositorioDaFila,
		NumeroPR: numeroPR, Mergeado: true, SHAHead: shaDoCommit(numeroPR), SHADoMerge: shaDoCommit(numeroPR + 9000), FechadoEm: time.Now().UTC(), Reverte: reverte,
	}
	if err := b.repo.RegistrarFechamento(t.Context(), fechado); err != nil {
		t.Fatalf("fechar o PR %d: %v", numeroPR, err)
	}
}

func (b bancoDaFila) semearHistoricoAprovadoEMergeado(t *testing.T) []domain.TaskRun {
	t.Helper()

	vetor := vetorDaAreaSensivel(t)
	analises := make([]domain.TaskRun, 0, casosParecidos)
	for indice := range casosParecidos {
		numeroPR := primeiroPRDoHistorico + indice
		analise := b.analiseConcluida(t, numeroPR, vetor)
		aprovarCommit(t, b.repo, analise, domain.DecisaoAprovada, shaDoCommit(numeroPR))
		b.fecharPRDaFila(t, numeroPR, 0)
		analises = append(analises, analise)
	}
	return analises
}

func (b bancoDaFila) conteudoDoRecibo(t *testing.T, taskRunID string) receipt.Conteudo {
	t.Helper()

	gravado, err := b.repo.BuscarReciboPorTaskRun(t.Context(), taskRunID)
	if err != nil {
		t.Fatalf("buscar o recibo: %v", err)
	}
	if _, resultado, err := verificadorDeRecibos(t, b.repo).Verificar(t.Context(), gravado.ID); err != nil || !resultado.Valido {
		t.Fatalf("o recibo precisa verificar: %+v, %v", resultado, err)
	}
	conteudo, err := receipt.LerConteudo([]byte(gravado.Conteudo))
	if err != nil {
		t.Fatalf("ler o conteúdo do recibo: %v", err)
	}
	return conteudo
}

func motivoDoRecibo(conteudo receipt.Conteudo, id string) (receipt.Motivo, bool) {
	indice := slices.IndexFunc(conteudo.Motivos, func(motivo receipt.Motivo) bool { return motivo.ID == id })
	if indice < 0 {
		return receipt.Motivo{}, false
	}
	return conteudo.Motivos[indice], true
}

func TestHistoricoAprovadoDesceAreaSensivelParaMedioComReciboQueVerifica(t *testing.T) {
	banco := novoBancoDaFila(t)
	historico := banco.semearHistoricoAprovadoEMergeado(t)
	taskRun := banco.novoTaskRunDoPR(t, numeroDoPRCalibrado)
	montagem := novaMontagemDoGate(banco, arquivosNaAreaSensivel...)
	montagem.calibrador = calibradorDoFluxo(t, banco, embedding.Falso{})

	montagem.rodarAte(t, taskRun.ID, domain.EstadoConcluido)

	gravado, err := banco.repo.BuscarGateDeRisco(t.Context(), taskRun.ID)
	if err != nil {
		t.Fatalf("ler o gate: %v", err)
	}
	if gravado.NivelDeterministico != domain.RiscoAlto || gravado.Nivel != domain.RiscoMedio || gravado.Calibracao.Estado != domain.CalibracaoAplicada {
		t.Fatalf("alto pelas regras, médio pelo histórico: %+v", gravado)
	}
	if len(gravado.Calibracao.Vizinhos) != casosParecidos {
		t.Errorf("os %d casos parecidos ficam no rastro: %+v", casosParecidos, gravado.Calibracao.Vizinhos)
	}

	conteudo := banco.conteudoDoRecibo(t, taskRun.ID)
	if conteudo.NivelDeterministico != domain.RiscoAlto || conteudo.Nivel != domain.RiscoMedio {
		t.Errorf("o recibo assina os dois níveis: %s e %s", conteudo.NivelDeterministico, conteudo.Nivel)
	}
	motivo, presente := motivoDoRecibo(conteudo, calibracao.MotivoDaCalibracao)
	if !presente || len(motivo.Evidencias) != casosParecidos {
		t.Fatalf("o recibo assina o motivo da calibração com um vizinho por evidência: %+v", conteudo.Motivos)
	}
	for _, analise := range historico {
		if !slices.ContainsFunc(motivo.Evidencias, func(evidencia string) bool {
			return strings.HasPrefix(evidencia, analise.ID+" ") && strings.HasSuffix(evidencia, "similaridade 1.0000 aprovado_e_mergeado")
		}) {
			t.Errorf("o vizinho %s não está assinado no recibo: %v", analise.ID, motivo.Evidencias)
		}
	}
	if total := banco.contar(t, `SELECT count(*) FROM code_memory WHERE task_run_id = $1`, taskRun.ID); total != 1 {
		t.Errorf("a análise nova entra no histórico: %d", total)
	}
}

func TestUmVizinhoRevertidoMantemOPRAltoEmEsperaDeAprovacao(t *testing.T) {
	banco := novoBancoDaFila(t)
	historico := banco.semearHistoricoAprovadoEMergeado(t)
	banco.fecharPRDaFila(t, primeiroPRDoHistorico+900, historico[2].NumeroPR)
	taskRun := banco.novoTaskRunDoPR(t, numeroDoPRCalibrado)
	montagem := novaMontagemDoGate(banco, arquivosNaAreaSensivel...)
	montagem.calibrador = calibradorDoFluxo(t, banco, embedding.Falso{})

	montagem.rodarAte(t, taskRun.ID, domain.EstadoAguardandoAprovacao)

	gravado, _ := banco.repo.BuscarGateDeRisco(t.Context(), taskRun.ID)
	if gravado.Nivel != domain.RiscoAlto || gravado.Calibracao.Estado != domain.CalibracaoVizinhoNegativo {
		t.Fatalf("um revertido entre os parecidos mantém alto: %+v", gravado)
	}
	conteudo := banco.conteudoDoRecibo(t, taskRun.ID)
	if conteudo.NivelDeterministico != domain.RiscoAlto || conteudo.Nivel != domain.RiscoAlto {
		t.Errorf("o recibo assina alto e alto: %s e %s", conteudo.NivelDeterministico, conteudo.Nivel)
	}
	if _, presente := motivoDoRecibo(conteudo, calibracao.MotivoDaCalibracao); presente {
		t.Error("sem calibração não há motivo de calibração no recibo")
	}
}

func TestRetomadaComGateGravadoNaoChamaOEmbedder(t *testing.T) {
	banco := novoBancoDaFila(t)
	banco.novoTaskRunDoPR(t, numeroDoPRCalibrado)
	reivindicado, _ := banco.reivindicar(t, "worker-que-morreu")
	avaliando := banco.levarAte(t, "worker-que-morreu", reivindicado, domain.EstadoMapeandoContexto, domain.EstadoTestandoSandbox, domain.EstadoAvaliandoRisco)
	if err := banco.repo.GravarResultadoDaTentativa(t.Context(), avaliando.ID, 1, resultadoVerde); err != nil {
		t.Fatalf("gravar resultado: %v", err)
	}
	gravado := domain.GateDeRisco{
		TaskRunID: avaliando.ID, NivelDeterministico: domain.RiscoAlto, Nivel: domain.RiscoAlto, Motivo: "area_sensivel: o PR altera um caminho sensível",
		Motivos:    []domain.MotivoDeRisco{{ID: "area_sensivel", Nivel: domain.RiscoAlto, Descricao: "o PR altera um caminho sensível", Evidencias: []string{"go/auth/sessao.go"}}},
		Calibracao: domain.Calibracao{Estado: domain.CalibracaoHistoricoInsuficiente, Razao: "histórico insuficiente: 0 de 5"},
	}
	if _, err := emissorDeRecibos(t, banco.repo).Emitir(t.Context(), receipt.Pedido{TaskRun: avaliando, Gate: gravado, Tentativa: 1, Diff: "diff"}); err != nil {
		t.Fatalf("gravar a decisão com o recibo: %v", err)
	}
	if err := banco.repo.Liberar(t.Context(), avaliando.ID, "worker-que-morreu", 0); err != nil {
		t.Fatalf("liberar: %v", err)
	}
	proibido := &embedderProibido{}
	montagem := novaMontagemDoGate(banco)
	montagem.avaliador = gateQueNaoPodeSerChamado{}
	montagem.leitor = leitorQueNaoPodeSerChamado{}
	montagem.calibrador = calibradorDoFluxo(t, banco, proibido)

	montagem.rodarAte(t, avaliando.ID, domain.EstadoAguardandoAprovacao)

	if chamadas := proibido.chamadas.Load(); chamadas != 0 {
		t.Errorf("a retomada chamou o embedding %d vez(es)", chamadas)
	}
	if lido, _ := banco.repo.BuscarGateDeRisco(t.Context(), avaliando.ID); lido.Calibracao.Razao != "histórico insuficiente: 0 de 5" {
		t.Errorf("a retomada usa o rastro gravado: %+v", lido.Calibracao)
	}
}

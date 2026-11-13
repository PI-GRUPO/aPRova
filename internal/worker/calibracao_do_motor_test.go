package worker

import (
	"context"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/calibracao"
	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/eventos"
)

type calibradorFalso struct {
	mutex     sync.Mutex
	pedidos   []calibracao.Pedido
	responder func(calibracao.Pedido) calibracao.Resultado
	proibido  string
}

func (c *calibradorFalso) Calibrar(_ context.Context, pedido calibracao.Pedido) calibracao.Resultado {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.pedidos = append(c.pedidos, pedido)
	if c.proibido != "" {
		panic(c.proibido)
	}
	return c.responder(pedido)
}

func assinaturaDeTeste() *domain.AssinaturaDoPR {
	return &domain.AssinaturaDoPR{Texto: "motivos: area_sensivel", Embedding: make([]float32, domain.DimensoesEmbedding)}
}

func calibradorQueDesce() *calibradorFalso {
	return &calibradorFalso{responder: func(pedido calibracao.Pedido) calibracao.Resultado {
		motivos := append(slices.Clone(pedido.Motivos), domain.MotivoDeRisco{
			ID: calibracao.MotivoDaCalibracao, Nivel: domain.NivelInformativo,
			Descricao:  "risco alto pelas regras, reduzido a médio pelo histórico: 5 casos parecidos aprovados e mergeados, nenhum revertido",
			Evidencias: []string{"00000000-0000-4000-8000-000000000001 Azulpasta/aPRova-TESTES#3 similaridade 0.9700 aprovado_e_mergeado"},
		})
		return calibracao.Resultado{
			Decisao: calibracao.Decisao{Nivel: domain.RiscoMedio, Motivos: motivos, Calibracao: domain.Calibracao{
				Estado: domain.CalibracaoAplicada, Razao: "5 casos parecidos aprovados e mergeados, nenhum revertido",
				Vizinhos: []domain.VizinhoDaCalibracao{{TaskRunID: "00000000-0000-4000-8000-000000000001", Repositorio: "Azulpasta/aPRova-TESTES", NumeroPR: 3, Similaridade: 0.97, Desfecho: domain.VizinhoAprovadoEMergeado}},
			}},
			Assinatura: assinaturaDeTeste(),
		}
	}}
}

func TestSemCalibradorADecisaoEAMesmaDeHoje(t *testing.T) {
	c := novoCenario(domain.RiscoAlto)
	c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	gate := c.recibos.pedidos[0].Gate
	if gate.NivelDeterministico != domain.RiscoAlto || gate.Nivel != domain.RiscoAlto || len(gate.Motivos) != 1 || gate.Motivos[0].ID != "area_sensivel" {
		t.Errorf("o recibo recebe exatamente o que o gate decidiu: %+v", gate)
	}
	if gate.Calibracao.Estado != domain.CalibracaoDesligada || gate.Calibracao.Razao != "calibração desligada" || len(gate.Calibracao.Vizinhos) != 0 {
		t.Errorf("o rastro registra que a calibração está desligada: %+v", gate.Calibracao)
	}
	if gate.Assinatura != nil {
		t.Error("desligada, a calibração não gera embedding")
	}
}

func TestCalibracaoQueDesceLevaOPRAdianteSemAprovacao(t *testing.T) {
	c := novoCenario(domain.RiscoAlto)
	calibrador := calibradorQueDesce()
	c.opcoes.Calibrador = calibrador
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoConcluido {
		t.Fatalf("médio pela calibração segue para aplicando e conclui, terminou em %s", estado)
	}
	gate := c.recibos.pedidos[0].Gate
	if gate.NivelDeterministico != domain.RiscoAlto || gate.Nivel != domain.RiscoMedio {
		t.Errorf("os dois níveis vão para o recibo, diferentes: %s e %s", gate.NivelDeterministico, gate.Nivel)
	}
	if len(gate.Motivos) != 2 || gate.Motivos[1].ID != calibracao.MotivoDaCalibracao || !strings.Contains(gate.Motivo, calibracao.MotivoDaCalibracao) {
		t.Errorf("o motivo da calibração entra nos motivos e no resumo: %+v", gate)
	}
	if gate.Calibracao.Estado != domain.CalibracaoAplicada || len(gate.Calibracao.Vizinhos) != 1 || gate.Assinatura == nil {
		t.Errorf("o rastro e a assinatura vão juntos com a decisão: %+v", gate)
	}
	if publicadas := c.publicador.publicacoesDe(taskRun.ID); len(publicadas) != 1 || publicadas[0].Gate.Calibracao.Estado != domain.CalibracaoAplicada {
		t.Errorf("a publicação no PR recebe a calibração: %+v", publicadas)
	}
	pedido, _ := c.eventos.doTipo(eventos.TipoAnaliseConcluida)
	if dados := dadosDoEvento[eventos.DadosAnaliseConcluida](t, pedido); dados.NivelRisco != domain.RiscoMedio || len(dados.Motivos) != 2 {
		t.Errorf("o evento sai do gate gravado: %+v", dados)
	}
}

func TestCalibradorRecebeADecisaoDoGateEOsArquivosDoPR(t *testing.T) {
	c := novoCenario(domain.RiscoAlto)
	calibrador := calibradorQueDesce()
	c.opcoes.Calibrador = calibrador
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if len(calibrador.pedidos) != 1 {
		t.Fatalf("uma calibração por avaliação: %d", len(calibrador.pedidos))
	}
	pedido := calibrador.pedidos[0]
	if pedido.TaskRun.ID != taskRun.ID || pedido.Nivel != domain.RiscoAlto || len(pedido.Motivos) != 1 || pedido.Motivos[0].ID != "area_sensivel" {
		t.Errorf("o calibrador recebe o task_run e a decisão determinística: %+v", pedido)
	}
	if len(pedido.Arquivos) != 1 || pedido.Arquivos[0].Nome != "go/frete/frete.go" {
		t.Errorf("e os arquivos alterados, de onde sai a assinatura: %+v", pedido.Arquivos)
	}
}

func TestCalibracaoQueMantemONivelGuardaOPorque(t *testing.T) {
	c := novoCenario(domain.RiscoAlto)
	c.opcoes.Calibrador = &calibradorFalso{responder: func(pedido calibracao.Pedido) calibracao.Resultado {
		return calibracao.Resultado{
			Decisao:    calibracao.Decisao{Nivel: pedido.Nivel, Motivos: pedido.Motivos, Calibracao: domain.Calibracao{Estado: domain.CalibracaoHistoricoInsuficiente, Razao: "histórico insuficiente: 2 de 5"}},
			Assinatura: assinaturaDeTeste(),
		}
	}}
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoAguardandoAprovacao {
		t.Fatalf("sem calibração, alto espera aprovação; está em %s", estado)
	}
	gate := c.recibos.pedidos[0].Gate
	if gate.Nivel != domain.RiscoAlto || gate.Calibracao.Razao != "histórico insuficiente: 2 de 5" || gate.Assinatura == nil {
		t.Errorf("o porquê fica registrado e o vetor entra no histórico: %+v", gate)
	}
}

func TestRetomadaComGateGravadoNaoCalibraDeNovo(t *testing.T) {
	c := novoCenario(domain.RiscoAlto)
	c.opcoes.Calibrador = &calibradorFalso{proibido: "a retomada não chama o embedding nem a calibração"}
	taskRun := c.armazenamento.novoTaskRun(14)
	c.armazenamento.taskRuns[taskRun.ID].Estado = domain.EstadoAvaliandoRisco
	_ = c.armazenamento.GravarResultadoDaTentativa(t.Context(), taskRun.ID, 1, sucessoNoSandbox)
	gravado := gateGravado(taskRun.ID)
	gravado.Nivel, gravado.Calibracao = domain.RiscoMedio, domain.Calibracao{Estado: domain.CalibracaoAplicada, Razao: "5 casos parecidos aprovados e mergeados, nenhum revertido"}
	_ = c.armazenamento.GravarGateDeRisco(t.Context(), gravado)
	c.armazenamento.recibos[taskRun.ID] = domain.Recibo{NovoRecibo: domain.NovoRecibo{ID: "8e0a2c4d-6f1b-4d3e-a5c7-9b0d1e2f3a4b", TaskRunID: taskRun.ID, Hash: make([]byte, 32)}}

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoConcluido {
		t.Errorf("o nível gravado pela calibração decide o caminho, terminou em %s", estado)
	}
	if publicadas := c.publicador.publicacoesDe(taskRun.ID); len(publicadas) != 1 || publicadas[0].Gate.Calibracao.Estado != domain.CalibracaoAplicada {
		t.Errorf("a publicação usa a calibração gravada: %+v", publicadas)
	}
}

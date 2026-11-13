package worker

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/eventos"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

func TestMotorEntregaAoEmissorADecisaoOSandboxEODiffAnalisado(t *testing.T) {
	infraestrutura := sandbox.ResultadoSandbox{TipoFalha: sandbox.FalhaInfraestrutura, SaidaBruta: "job não subiu"}
	falhaDeTeste := sandbox.ResultadoSandbox{Compilou: true, TestesPassaram: 10, TestesFalharam: 1, TipoFalha: sandbox.FalhaTeste}
	c := novoCenario(domain.RiscoAlto, infraestrutura, falhaDeTeste)
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if len(c.recibos.pedidos) != 1 {
		t.Fatalf("esperava uma emissão, obtive %d", len(c.recibos.pedidos))
	}
	pedido := c.recibos.pedidos[0]
	if pedido.TaskRun.ID != taskRun.ID || pedido.Gate.TaskRunID != taskRun.ID || pedido.Gate.Nivel != domain.RiscoAlto || len(pedido.Gate.Motivos) != 1 {
		t.Errorf("a decisão vai inteira para o emissor: %+v", pedido.Gate)
	}
	if pedido.Sandbox.TestesFalharam != 1 || pedido.Sandbox.TipoFalha != sandbox.FalhaTeste || pedido.Tentativa != 2 {
		t.Errorf("o recibo registra o resultado e a tentativa que o produziu: %+v, tentativa %d", pedido.Sandbox, pedido.Tentativa)
	}
	if pedido.Diff != "diff --git a/go/frete/frete.go b/go/frete/frete.go" || pedido.DiffTruncado {
		t.Errorf("o diff analisado vai para o emissor: %q truncado=%v", pedido.Diff, pedido.DiffTruncado)
	}
}

func TestFalhaAoEmitirOReciboNaoDeixaDecisaoNemEventoETentaDeNovo(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	c.recibos.falhas = []error{errors.New("banco fora do ar")}
	c.recibos.antesDeFalhar = func() {
		if _, gravado := c.armazenamento.gates[c.armazenamento.ordemCriacao[0]]; gravado {
			t.Error("não pode existir decisão gravada sem recibo")
		}
		if len(c.eventos.tipos()) != 0 {
			t.Errorf("sem recibo não há evento: %v", c.eventos.tipos())
		}
	}
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoConcluido {
		t.Fatalf("depois da nova tentativa o task_run conclui, terminou em %s", estado)
	}
	if c.chamadas.quantas("recibo") != 2 {
		t.Errorf("esperava duas tentativas de emissão, obtive %d", c.chamadas.quantas("recibo"))
	}
	if tipos := c.eventos.tipos(); !slices.Equal(tipos, []eventos.TipoEvento{eventos.TipoAnaliseConcluida}) {
		t.Errorf("o evento sai uma vez, depois do recibo: %v", tipos)
	}
}

func TestDecisaoQueOutroWorkerGravouNoMeioSegueComOReciboDele(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	taskRun := c.armazenamento.novoTaskRun(14)
	c.recibos.falhas = []error{domain.ErrDuplicado}
	c.recibos.antesDeFalhar = func() {
		_ = c.armazenamento.GravarGateDeRisco(t.Context(), gateGravado(taskRun.ID))
		c.armazenamento.recibos[taskRun.ID] = domain.Recibo{NovoRecibo: domain.NovoRecibo{ID: "8e0a2c4d-6f1b-4d3e-a5c7-9b0d1e2f3a4b", TaskRunID: taskRun.ID, Hash: make([]byte, 32)}}
	}

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoAguardandoAprovacao {
		t.Errorf("vale a decisão gravada, que é de risco alto; terminou em %s", estado)
	}
	if c.chamadas.quantas("recibo") != 1 {
		t.Errorf("não se emite um segundo recibo: %d emissões", c.chamadas.quantas("recibo"))
	}
	if len(c.armazenamento.esperas) != 0 {
		t.Errorf("decisão já gravada não é falha: o task_run não espera para seguir, esperas = %v", c.armazenamento.esperas)
	}
	pedido, _ := c.eventos.doTipo(eventos.TipoAnaliseConcluida)
	if dados := dadosDoEvento[eventos.DadosAnaliseConcluida](t, pedido); !strings.Contains(dados.URLRecibo, "8e0a2c4d-6f1b-4d3e-a5c7-9b0d1e2f3a4b") {
		t.Errorf("o evento aponta o recibo gravado: %s", dados.URLRecibo)
	}
}

func TestDecisaoGravadaSemReciboFalhaFechado(t *testing.T) {
	for _, estado := range []domain.EstadoTaskRun{domain.EstadoAvaliandoRisco, domain.EstadoAplicando} {
		t.Run(string(estado), func(t *testing.T) {
			c := novoCenario(domain.RiscoBaixo)
			taskRun := c.armazenamento.novoTaskRun(14)
			c.armazenamento.taskRuns[taskRun.ID].Estado = estado
			_ = c.armazenamento.GravarResultadoDaTentativa(t.Context(), taskRun.ID, 1, sucessoNoSandbox)
			_ = c.armazenamento.GravarGateDeRisco(t.Context(), gateGravado(taskRun.ID))

			c.conduzirTudo(t)

			if final := c.armazenamento.taskRun(taskRun.ID).Estado; final != domain.EstadoFalhou {
				t.Fatalf("decisão sem recibo não segue adiante, terminou em %s", final)
			}
			etapas := c.armazenamento.etapas[taskRun.ID]
			if detalhe := etapas[len(etapas)-1].detalhe; !strings.Contains(detalhe, "sem recibo") {
				t.Errorf("o detalhe precisa dizer o motivo: %q", detalhe)
			}
			if c.chamadas.quantas("recibo") != 0 || c.chamadas.quantas("publicacao") != 0 || len(c.eventos.tipos()) != 0 {
				t.Errorf("nada é emitido nem publicado: %v, eventos %v", c.chamadas.lista(), c.eventos.tipos())
			}
		})
	}
}

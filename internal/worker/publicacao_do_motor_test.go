package worker

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/gate"
	"github.com/Azulpasta/aPRova/internal/platform"
)

var (
	errSemPermissaoNoCheckRun = fmt.Errorf("%w: POST /repos/Azulpasta/aPRova-TESTES/check-runs respondeu 403: confira se a instalação do app concedeu checks: write e pull_requests: write neste repositório", platform.ErrSemPermissao)
	errBadGatewayNoComentario = fmt.Errorf("%w: POST /repos/Azulpasta/aPRova-TESTES/issues/14/comments respondeu 502", platform.ErrGitHubIndisponivel)
)

func TestCheckRunComecaQuandoOTaskRunEntraEmMapeandoContexto(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if len(c.publicador.iniciados) != 1 || c.publicador.iniciados[0] != taskRun.ID {
		t.Fatalf("o Check Run em andamento é criado uma vez: %v", c.publicador.iniciados)
	}
	if primeira := c.chamadas.lista()[0]; primeira != "inicio" {
		t.Errorf("o PR mostra que a análise começou antes de qualquer outra coisa, primeira chamada: %s", primeira)
	}
}

func TestPermissaoNegadaAoIniciarEncerraEmFalhouComDiagnostico(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	c.publicador.errosDoInicio = []error{errSemPermissaoNoCheckRun}
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoFalhou {
		t.Fatalf("403 é definitivo, terminou em %s", estado)
	}
	if detalhe := c.armazenamento.ultimaEtapa(taskRun.ID).detalhe; !strings.Contains(detalhe, "checks: write") {
		t.Errorf("o detalhe diz o que conferir na instalação: %q", detalhe)
	}
	if c.leitor.leituras != 0 {
		t.Error("sem permissão para o Check Run, a análise não começa")
	}
}

func TestErroTransitorioAoIniciarTentaDeNovo(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	c.publicador.errosDoInicio = []error{errBadGatewayNoComentario}
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoConcluido {
		t.Fatalf("502 é transitório, terminou em %s", estado)
	}
	if c.chamadas.quantas("inicio") != 2 || len(c.armazenamento.esperas) == 0 {
		t.Errorf("a nova tentativa vem depois da espera: %v, esperas %v", c.chamadas.lista(), c.armazenamento.esperas)
	}
}

func TestPermissaoNegadaAoPublicarEncerraEmFalhouComDiagnostico(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	c.publicador.erros = []error{errSemPermissaoNoCheckRun}
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoFalhou {
		t.Fatalf("403 é definitivo, terminou em %s", estado)
	}
	if detalhe := c.armazenamento.ultimaEtapa(taskRun.ID).detalhe; !strings.Contains(detalhe, "checks: write") || !strings.Contains(detalhe, "pull_requests: write") {
		t.Errorf("o detalhe diz o que conferir na instalação: %q", detalhe)
	}
	if c.chamadas.quantas("publicacao") != 1 {
		t.Errorf("403 não é tentado de novo: %v", c.chamadas.lista())
	}
}

func TestErroTransitorioAoPublicarTentaDeNovoEConclui(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	c.publicador.erros = []error{errBadGatewayNoComentario}
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoConcluido {
		t.Fatalf("502 é transitório, terminou em %s", estado)
	}
	if c.chamadas.quantas("publicacao") != 2 || len(c.publicador.publicados) != 1 {
		t.Errorf("a segunda tentativa publica: %v", c.chamadas.lista())
	}
}

func TestErroTransitorioAoPublicarOBloqueioNaoDeixaOPRSemCheckRunConcluido(t *testing.T) {
	c := novoCenario(domain.RiscoAlto)
	c.publicador.erros = []error{errBadGatewayNoComentario}
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoAguardandoAprovacao {
		t.Fatalf("depois da nova tentativa, o PR para em aguardando_aprovacao; está em %s", estado)
	}
	if len(c.publicador.publicados) != 1 || c.chamadas.quantas("recibo") != 1 {
		t.Errorf("o bloqueio é publicado uma vez, com um recibo só: %v", c.chamadas.lista())
	}
}

func TestFalhaDaAnalisePublicaOCheckRunEmFailureComODiagnostico(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	c.leitor.erros = []error{errGitHubFora, errGitHubFora, errGitHubFora}
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoFalhou {
		t.Fatalf("terminou em %s", estado)
	}
	falhas := c.publicador.falhasDe(taskRun.ID)
	if len(falhas) != 1 || !strings.Contains(falhas[0].diagnostico, "tentativas esgotadas em mapeando_contexto") {
		t.Errorf("o Check Run fecha em failure com o diagnóstico: %+v", falhas)
	}
}

func TestErroDoGatePublicaAFalhaSemPublicarResultado(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	c.avaliador.erro = fmt.Errorf("%w: tipo_falha infraestrutura", gate.ErrResultadoSemVeredito)
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if falhas := c.publicador.falhasDe(taskRun.ID); len(falhas) != 1 {
		t.Errorf("sem veredito o Check Run fecha em failure: %+v", falhas)
	}
	if c.chamadas.quantas("publicacao") != 0 {
		t.Error("sem veredito não há resultado a publicar")
	}
}

func TestFalhaAoPublicarAFalhaNaoImpedeOEncerramento(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	c.leitor.erros = []error{fmt.Errorf("%w: GET /repos/Azulpasta/aPRova-TESTES/pulls/14/files", platform.ErrNaoEncontrado)}
	c.publicador.errosDaFalha = []error{errBadGatewayNoComentario}
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoFalhou {
		t.Fatalf("o task_run encerra mesmo sem conseguir avisar o PR, está em %s", estado)
	}
	if !strings.Contains(strings.Join(c.diario.linhas(), "\n"), "failure") {
		t.Errorf("o diário registra que o Check Run não foi fechado: %v", c.diario.linhas())
	}
}

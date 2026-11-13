package worker

import (
	"strings"
	"testing"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/eventos"
)

const (
	commitAntigo = "3f9a1c7e2b4d6f8091a3c5e7f9b1d3e5a7c9e1f3"
	commitNovo   = "95eb58e1d0c4a7b3f2e6d9c8b1a0f4e3d2c1b0a9"
)

func TestAprovacaoDoCommitAnalisadoDecide(t *testing.T) {
	c := novoCenario(domain.RiscoAlto)
	taskRun := levarAteAguardando(t, c)
	c.armazenamento.registrarAprovacaoDoCommit(taskRun.ID, domain.DecisaoAprovada, "httpslua", domain.OrigemGitHub, taskRun.SHAHead)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoConcluido {
		t.Errorf("a aprovação do commit analisado libera o PR, terminou em %s", estado)
	}
}

func TestAprovacaoDeCommitAntigoDepoisDePushNovoNaoLiberaONovo(t *testing.T) {
	c := novoCenario(domain.RiscoAlto, sucessoNoSandbox, sucessoNoSandbox)
	antigo := levarAteAguardando(t, c)
	novo := c.armazenamento.novoTaskRunNoCommit(14, commitNovo)
	c.conduzirTudo(t)
	if estado := c.armazenamento.taskRun(novo.ID).Estado; estado != domain.EstadoAguardandoAprovacao {
		t.Fatalf("preparação: o task_run do push novo espera aprovação, está em %s", estado)
	}

	c.armazenamento.registrarAprovacaoDoCommit(novo.ID, domain.DecisaoAprovada, "httpslua", domain.OrigemGitHub, antigo.SHAHead)
	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(novo.ID).Estado; estado != domain.EstadoAguardandoAprovacao {
		t.Fatalf("a aprovação do commit antigo não libera o novo, foi para %s", estado)
	}
	if _, emitido := c.eventos.doTipo(eventos.TipoAprovacaoRegistrada); emitido {
		t.Error("aprovação de outro commit não é decisão")
	}
	registro := strings.Join(c.diario.linhas(), "\n")
	if !strings.Contains(registro, commitAntigo) || !strings.Contains(registro, commitNovo) {
		t.Errorf("o log registra o commit revisado e o analisado: %s", registro)
	}

	c.armazenamento.registrarAprovacaoDoCommit(novo.ID, domain.DecisaoAprovada, "httpslua", domain.OrigemGitHub, commitNovo)
	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(novo.ID).Estado; estado != domain.EstadoConcluido {
		t.Errorf("a aprovação seguinte, do commit analisado, decide; terminou em %s", estado)
	}
	if ultima := c.publicador.publicados[len(c.publicador.publicados)-1]; ultima.AprovadoPor != "httpslua" || ultima.TaskRun.ID != novo.ID {
		t.Errorf("a aprovação publicada é a do commit analisado: %+v", ultima)
	}
}

func TestRevisaoSemCommitNaoDecide(t *testing.T) {
	c := novoCenario(domain.RiscoAlto)
	taskRun := levarAteAguardando(t, c)
	c.armazenamento.registrarAprovacaoDoCommit(taskRun.ID, domain.DecisaoAprovada, "httpslua", domain.OrigemGitHub, "")

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoAguardandoAprovacao {
		t.Errorf("sem o commit revisado, não há como saber o que foi aprovado; foi para %s", estado)
	}
}

package worker

import (
	"reflect"
	"slices"
	"sync"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

type enviadorDeSARIFFalso struct {
	chamadas *registroDeChamadas
	mutex    sync.Mutex
	enviados []Publicacao
}

func (e *enviadorDeSARIFFalso) Enviar(publicacao Publicacao) {
	e.chamadas.anotar("sarif")
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.enviados = append(e.enviados, publicacao)
}

func (c *cenario) comSARIF() *enviadorDeSARIFFalso {
	enviador := &enviadorDeSARIFFalso{chamadas: c.chamadas}
	c.opcoes.SARIF = enviador
	return enviador
}

func semSARIF(chamadas []string) []string {
	return slices.DeleteFunc(slices.Clone(chamadas), func(chamada string) bool { return chamada == "sarif" })
}

func exigirSARIFDepoisDaPublicacao(t *testing.T, chamadas []string) {
	t.Helper()
	for indice, chamada := range chamadas {
		if chamada == "sarif" && (indice == 0 || chamadas[indice-1] != "publicacao") {
			t.Errorf("o SARIF só sai logo depois do Check Run publicado: %v", chamadas)
		}
	}
}

func TestComOSARIFDesligadoOFluxoEhOMesmo(t *testing.T) {
	for _, nivel := range []domain.NivelRisco{domain.RiscoBaixo, domain.RiscoAlto} {
		t.Run(string(nivel), func(t *testing.T) {
			desligado := novoCenario(nivel)
			semEnvio := desligado.armazenamento.novoTaskRun(14)
			desligado.conduzirTudo(t)

			ligado := novoCenario(nivel)
			ligado.comSARIF()
			comEnvio := ligado.armazenamento.novoTaskRun(14)
			ligado.conduzirTudo(t)

			if slices.Contains(desligado.chamadas.lista(), "sarif") {
				t.Fatalf("sem enviador não há envio: %v", desligado.chamadas.lista())
			}
			if !slices.Equal(desligado.chamadas.lista(), semSARIF(ligado.chamadas.lista())) {
				t.Errorf("as chamadas mudaram com o SARIF ligado:\ndesligado: %v\nligado:    %v", desligado.chamadas.lista(), ligado.chamadas.lista())
			}
			if !slices.Equal(desligado.armazenamento.estadosPercorridos(semEnvio.ID), ligado.armazenamento.estadosPercorridos(comEnvio.ID)) {
				t.Errorf("os estados mudaram com o SARIF ligado: %v e %v",
					desligado.armazenamento.estadosPercorridos(semEnvio.ID), ligado.armazenamento.estadosPercorridos(comEnvio.ID))
			}
		})
	}
}

func TestRiscoBaixoEnviaOSARIFDepoisDoCheckRunComOGateGravado(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	enviador := c.comSARIF()
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoConcluido {
		t.Fatalf("terminou em %s", estado)
	}
	if len(enviador.enviados) != 1 {
		t.Fatalf("vieram %d envios, esperado 1", len(enviador.enviados))
	}
	exigirSARIFDepoisDaPublicacao(t, c.chamadas.lista())

	gravado, err := c.armazenamento.BuscarGateDeRisco(t.Context(), taskRun.ID)
	if err != nil {
		t.Fatal(err)
	}
	enviado := enviador.enviados[0]
	if !reflect.DeepEqual(enviado.Gate, gravado) {
		t.Errorf("o SARIF sai do gate gravado:\nenviado: %+v\ngravado: %+v", enviado.Gate, gravado)
	}
	if enviado.TaskRun.ID != taskRun.ID || enviado.Recibo.ID == "" {
		t.Errorf("envio sem o task_run ou sem o recibo: %+v", enviado)
	}
}

func TestRiscoAltoEnviaOSARIFAoAguardarAprovacaoEDeNovoAoAplicar(t *testing.T) {
	c := novoCenario(domain.RiscoAlto)
	enviador := c.comSARIF()
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoAguardandoAprovacao {
		t.Fatalf("parou em %s", estado)
	}
	if len(enviador.enviados) != 1 {
		t.Fatalf("o alerta aparece no PR enquanto ele espera a aprovação; vieram %d envios", len(enviador.enviados))
	}

	c.armazenamento.registrarAprovacao(taskRun.ID, domain.DecisaoAprovada, "Azulpasta", domain.OrigemGitHub)
	c.conduzirTudo(t)

	if estado := c.armazenamento.taskRun(taskRun.ID).Estado; estado != domain.EstadoConcluido {
		t.Fatalf("terminou em %s", estado)
	}
	if len(enviador.enviados) != 2 {
		t.Errorf("a retomada pede o envio de novo e o enviador decide pelo id gravado; vieram %d pedidos", len(enviador.enviados))
	}
	exigirSARIFDepoisDaPublicacao(t, c.chamadas.lista())
}

func TestSemCheckRunPublicadoNaoHaSARIF(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	enviador := c.comSARIF()
	c.publicador.erros = []error{errSemPermissaoNoCheckRun}
	c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if len(enviador.enviados) != 0 {
		t.Errorf("o SARIF vem depois do Check Run; sem ele, não sai: %d envios", len(enviador.enviados))
	}
}

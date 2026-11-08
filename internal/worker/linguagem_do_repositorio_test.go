package worker

import (
	"testing"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/sandbox"
)

func TestSandboxUsaALinguagemConfiguradaParaORepositorio(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	c.opcoes.Linguagens = map[string]sandbox.Linguagem{"Azulpasta/aPRova-TESTES": sandbox.LinguagemPython}
	c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if linguagem := c.sandbox.pedidos[0].Linguagem; linguagem != sandbox.LinguagemPython {
		t.Errorf("linguagem do pedido = %s, esperava python", linguagem)
	}
}

func TestRepositorioSemLinguagemConfiguradaContinuaEmGo(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	c.opcoes.Linguagens = map[string]sandbox.Linguagem{"PI-GRUPO/outro": sandbox.LinguagemPython}
	c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	if linguagem := c.sandbox.pedidos[0].Linguagem; linguagem != sandbox.LinguagemGo {
		t.Errorf("linguagem do pedido = %s, esperava go", linguagem)
	}
}

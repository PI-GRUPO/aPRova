package explicacao

import (
	"strings"
	"testing"
)

func TestInstrucaoOrientaOQueFazerParaCadaMotivoDoGate(t *testing.T) {
	for _, motivo := range []string{
		"falha_sandbox", "muitos_arquivos", "configuracao_ci", "dependencia_nova",
		"segredo_versionado", "area_sensivel", "codigo_sem_teste",
	} {
		if !strings.Contains(instrucaoDoSistema, motivo+":") {
			t.Errorf("a instrução não diz o que fazer para %s", motivo)
		}
	}
	for _, regra := range []string{"a partir dos números do campo \"sandbox\"", "acentuação"} {
		if !strings.Contains(instrucaoDoSistema, regra) {
			t.Errorf("a instrução não traz a regra %q", regra)
		}
	}
}

func TestInstrucaoPedeOSignificadoUmaVezSemAprovacaoParaFalhaENaturalidade(t *testing.T) {
	for _, regra := range []string{
		"uma única vez",
		"falha_sandbox: corrigir o teste ou a compilação que falhou e enviar um commit novo; não mencione a aprovação como saída.",
		`"1 dos 11 testes falhou"`,
		`"o arquivo go.mod"`,
		`"conforme evidenciado"`,
		`"conforme apontado"`,
		`"como visto em"`,
		"Exemplo de estilo",
		`a palavra "conforme"`,
	} {
		if !strings.Contains(instrucaoDoSistema, regra) {
			t.Errorf("a instrução não traz %q", regra)
		}
	}
	if strings.Contains(instrucaoDoSistema, "diga também que a aprovação") {
		t.Error("a instrução não pode pedir para repetir que a aprovação libera o merge")
	}
}

func TestInstrucaoParaRiscoBaixoNuncaGaranteSegurancaETemExemploDeConcordancia(t *testing.T) {
	for _, regra := range []string{
		"nenhuma regra do aPRova disparou",
		"a revisão normal do pull request continua valendo",
		"Nunca afirme que a mudança é segura",
		"Exemplo de concordância",
		`"1 dos 11 testes falhou no sandbox"`,
	} {
		if !strings.Contains(instrucaoDoSistema, regra) {
			t.Errorf("a instrução não traz %q", regra)
		}
	}
}

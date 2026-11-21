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
	for _, regra := range []string{"não são arquivos", "acentuação"} {
		if !strings.Contains(instrucaoDoSistema, regra) {
			t.Errorf("a instrução não traz a regra %q", regra)
		}
	}
}

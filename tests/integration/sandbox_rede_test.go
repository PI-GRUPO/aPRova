package integracao

import (
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

const casoDeRedeEDependencia = "rede_e_dependencia"

func exigirAtaqueDeRedeBloqueado(t *testing.T, resultado sandbox.ResultadoSandbox) {
	t.Helper()

	if strings.Contains(resultado.SaidaBruta, "ataque: alcançou") {
		t.Fatalf("o código do PR alcançou a rede na etapa de testes:\n%s", resultado.SaidaBruta)
	}
	if resultado.TipoFalha != sandbox.FalhaTeste || resultado.TestesPassaram != 1 || resultado.TestesFalharam != 1 {
		t.Errorf("a dependência baixada no preparo passa e o ataque de rede falha: %+v", resultado)
	}
	if !strings.Contains(resultado.SaidaBruta, "rede inalcançável") {
		t.Errorf("o teste de rede precisa ter falhado por não alcançar nada: %s", resultado.SaidaBruta)
	}
}

func TestSandboxGoBaixaDependenciasNoPreparoETestaSemRede(t *testing.T) {
	exigirAtaqueDeRedeBloqueado(t, executarCaso(t, casoDeRedeEDependencia, prazoDeUmCasoComum))
}

func TestSandboxPythonBaixaDependenciasNoPreparoETestaSemRede(t *testing.T) {
	exigirAtaqueDeRedeBloqueado(t, executarCasoPython(t, casoDeRedeEDependencia, prazoDeUmCasoComum))
}

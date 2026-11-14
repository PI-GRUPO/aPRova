package integracao

import (
	"regexp"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

const casoDeRedeEDependencia = "rede_e_dependencia"

var ataqueQueAlcancou = regexp.MustCompile(`ataque: alcançou (dns|tcp)`)

func exigirAtaqueDeRedeBloqueado(t *testing.T, resultado sandbox.ResultadoSandbox) {
	t.Helper()

	if ataqueQueAlcancou.MatchString(resultado.SaidaBruta) {
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

func TestSandboxJavaUsaOCacheDaImagemETestaSemRede(t *testing.T) {
	exigirAtaqueDeRedeBloqueado(t, executarCasoJava(t, casoDeRedeEDependencia, prazoDeUmCasoComum))
}

func TestSandboxPythonNuncaExecutaOSetupPyDoPRNoPreparo(t *testing.T) {
	resultado := executarCasoPython(t, "setup_py_marcador", prazoDeUmCasoComum)

	if resultado.TipoFalha != sandbox.FalhaNenhuma || resultado.TestesPassaram != 1 {
		t.Errorf("o marcador do setup.py não pode existir depois do preparo: %+v", resultado)
	}
}

func TestSandboxPythonDependenciaSoEmSdistEhCompilacaoSemRodarSetupPy(t *testing.T) {
	resultado := executarCasoPython(t, "so_sdist", prazoDeUmCasoComum)

	if resultado.TipoFalha != sandbox.FalhaCompilacao || !strings.Contains(resultado.SaidaBruta, "pycrypto==2.6.1") ||
		!strings.Contains(resultado.SaidaBruta, "só existe como sdist") {
		t.Errorf("só sdist é compilação, com a dependência na evidência: %+v", resultado)
	}
	for _, sinal := range []string{"setup.py", "running build", "Building wheel"} {
		if strings.Contains(resultado.SaidaBruta, sinal) {
			t.Errorf("nada foi construído no preparo, mas a saída fala em %q: %s", sinal, resultado.SaidaBruta)
		}
	}
}

func TestSandboxGoComGoSumDivergenteEhCompilacao(t *testing.T) {
	resultado := executarCaso(t, "go_sum_divergente", prazoDeUmCasoComum)

	if resultado.TipoFalha != sandbox.FalhaCompilacao || !strings.Contains(resultado.SaidaBruta, "github.com/google/go-cmp") ||
		!strings.Contains(resultado.SaidaBruta, "checksum divergente do go.sum") {
		t.Errorf("go.sum divergente é do PR, com o módulo na evidência: %+v", resultado)
	}
}

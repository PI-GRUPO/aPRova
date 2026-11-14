package integracao

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

const (
	usuarioDoCodigoDoPR   = "10001:10001"
	tempoLimiteDoPreparo  = time.Minute
	prazoDoPythonQueTrava = 110 * time.Second
)

type cenarioPython struct {
	caso      string
	prazo     time.Duration
	opcoes    sandbox.OpcoesExecutorLocal
	diretorio string
}

func executarCasoPython(t *testing.T, caso string, prazo time.Duration) sandbox.ResultadoSandbox {
	t.Helper()
	return executarCenarioPython(t, cenarioPython{caso: caso, prazo: prazo})
}

func executarCenarioPython(t *testing.T, cenario cenarioPython) sandbox.ResultadoSandbox {
	t.Helper()
	imagemPython.exigir(t)

	caso := cenario.caso
	volume, shas := publicarRepositorios(t, imagemPython, caso)

	opcoes := cenario.opcoes
	opcoes.ImagemPorLinguagem = map[sandbox.Linguagem]string{sandbox.LinguagemPython: imagemPython.nome()}
	opcoes.Montagens = []string{volume + ":" + pontoDeMontagem + ":ro"}
	executor := sandbox.NovoExecutorLocal(opcoes)

	ctx, cancelar := context.WithTimeout(t.Context(), cenario.prazo)
	defer cancelar()

	resultado, err := executor.Executar(ctx, sandbox.PedidoExecucao{
		URLRepositorio: "file://" + pontoDeMontagem + "/" + caso,
		SHA:            shas[caso],
		Linguagem:      sandbox.LinguagemPython,
		TaskRunID:      "integracao-python-" + caso,
		Diretorio:      cenario.diretorio,
	})
	if err != nil {
		t.Fatalf("executar caso %s: %v", caso, err)
	}

	t.Logf("caso %s: tipo=%s compilou=%v passaram=%d falharam=%d duracao=%dms\n%s",
		caso, resultado.TipoFalha, resultado.Compilou, resultado.TestesPassaram, resultado.TestesFalharam,
		resultado.DuracaoMS, resultado.SaidaBruta)

	return resultado
}

func TestSandboxPythonProjetoOk(t *testing.T) {
	resultado := executarCasoPython(t, "ok", prazoDeUmCasoComum)

	if resultado.TipoFalha != sandbox.FalhaNenhuma || !resultado.Compilou ||
		resultado.TestesPassaram != 2 || resultado.TestesFalharam != 0 {
		t.Errorf("projeto ok: esperava 2 testes passando sem falha, obtive %+v", resultado)
	}
}

func TestSandboxPythonTesteFalho(t *testing.T) {
	resultado := executarCasoPython(t, "teste_falho", prazoDeUmCasoComum)

	if resultado.TipoFalha != sandbox.FalhaTeste || !resultado.Compilou ||
		resultado.TestesPassaram != 2 || resultado.TestesFalharam != 1 {
		t.Errorf("esperava falha de teste com 2 passando e 1 falhando, obtive %+v", resultado)
	}
	if !strings.Contains(resultado.SaidaBruta, "frete com taxa: esperava 16, obtive 15") {
		t.Error("a mensagem do teste falho precisa estar na saída")
	}
}

func TestSandboxPythonErroDeSintaxeNoTesteEhCompilacao(t *testing.T) {
	resultado := executarCasoPython(t, "nao_compila", prazoDeUmCasoComum)

	if resultado.TipoFalha != sandbox.FalhaCompilacao || resultado.Compilou ||
		resultado.TestesPassaram != 0 || resultado.TestesFalharam != 0 {
		t.Errorf("esperava compilação sem nenhum teste rodado, obtive %+v", resultado)
	}
}

func TestSandboxPythonConftestQuebradoEhCompilacao(t *testing.T) {
	resultado := executarCasoPython(t, "conftest_quebrado", prazoDeUmCasoComum)

	if resultado.TipoFalha != sandbox.FalhaCompilacao || resultado.Compilou {
		t.Errorf("conftest.py que não carrega é culpa do PR, obtive %+v", resultado)
	}
}

func TestSandboxPythonTesteQueTravaEhFalhaDeTesteDentroDoPrazo(t *testing.T) {
	inicio := time.Now()

	resultado := executarCasoPython(t, "trava", prazoDoPythonQueTrava)

	if resultado.TipoFalha != sandbox.FalhaTeste || resultado.TestesFalharam < 1 {
		t.Errorf("teste que trava é culpa do PR, obtive %+v", resultado)
	}
	if decorrido := time.Since(inicio); decorrido >= prazoDoPythonQueTrava {
		t.Errorf("o resultado precisa chegar dentro do prazo, levou %v", decorrido)
	}
}

func TestSandboxPythonColetaQueTravaEntregaResultadoDentroDoPrazo(t *testing.T) {
	inicio := time.Now()

	resultado := executarCasoPython(t, "trava_na_coleta", prazoDoPythonQueTrava)

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura || !strings.Contains(resultado.SaidaBruta, "pytest excedeu o prazo") {
		t.Errorf("sem resultado do pytest não há aprovação, obtive %+v", resultado)
	}
	if decorrido := time.Since(inicio); decorrido >= prazoDoPythonQueTrava {
		t.Errorf("o resultado precisa chegar dentro do prazo, levou %v", decorrido)
	}
}

func TestSandboxPythonTesteMaliciosoNaoForjaOResultado(t *testing.T) {
	resultado := executarCasoPython(t, "malicioso", prazoDeUmCasoComum)

	if resultado.TestesPassaram == 99 || resultado.SaidaBruta == "forjado" {
		t.Fatalf("o resultado forjado venceu: %+v", resultado)
	}
	if resultado.TipoFalha != sandbox.FalhaTeste || resultado.TestesFalharam != 1 || resultado.TestesPassaram != 0 {
		t.Errorf("o resultado entregue precisa ser o verdadeiro, a falha; obtive %+v", resultado)
	}
}

func TestSandboxPythonSemTestesNaoReprova(t *testing.T) {
	resultado := executarCasoPython(t, "sem_testes", prazoDeUmCasoComum)

	if resultado.TipoFalha != sandbox.FalhaNenhuma || !resultado.Compilou ||
		resultado.TestesPassaram != 0 || resultado.TestesFalharam != 0 {
		t.Errorf("projeto Python sem testes não reprova no sandbox, obtive %+v", resultado)
	}
}

func TestSandboxPythonDiretorioSemPythonEhInfraestrutura(t *testing.T) {
	resultado := executarCasoPython(t, "sem_python", prazoDeUmCasoComum)

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura || !strings.Contains(resultado.SaidaBruta, "sem arquivos Python") {
		t.Errorf("configuração errada não pode virar aprovação com zero testes, obtive %+v", resultado)
	}
}

func TestSandboxPythonInstalaORequirementsComSistemaDeArquivosSomenteLeitura(t *testing.T) {
	resultado := executarCenarioPython(t, cenarioPython{caso: "com_requisitos", prazo: prazoDeUmCasoComum})

	if resultado.TipoFalha != sandbox.FalhaNenhuma || resultado.TestesPassaram != 1 {
		t.Errorf("a dependência do requirements.txt precisa estar instalada no venv, obtive %+v", resultado)
	}
}

func TestSandboxPythonRequirementInexistenteEhCompilacaoComADependencia(t *testing.T) {
	resultado := executarCenarioPython(t, cenarioPython{caso: "requisito_inexistente", prazo: prazoDeUmCasoComum})

	if resultado.TipoFalha != sandbox.FalhaCompilacao || !strings.Contains(resultado.SaidaBruta, "six==999.0.0") ||
		!strings.Contains(resultado.SaidaBruta, "não existe no índice ou só existe como sdist") {
		t.Errorf("dependência inexistente é do PR, com a dependência na evidência: %+v", resultado)
	}
}

func TestSandboxPythonComProjetoNumSubdiretorio(t *testing.T) {
	resultado := executarCenarioPython(t, cenarioPython{caso: "subdiretorio", prazo: prazoDeUmCasoComum, diretorio: "python"})

	if resultado.TipoFalha != sandbox.FalhaNenhuma || !resultado.Compilou || resultado.TestesPassaram != 2 {
		t.Errorf("com o diretório do projeto informado, os testes de python/ rodam: %+v", resultado)
	}
}

func TestSandboxPythonSemTrocaDeUsuarioFalhaFechadoEmInfraestrutura(t *testing.T) {
	resultado := executarCenarioPython(t, cenarioPython{
		caso: "ok", prazo: prazoDeUmCasoComum, opcoes: sandbox.OpcoesExecutorLocal{SemCapacidadeDeTrocarDeUsuario: true},
	})

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura || resultado.Compilou || resultado.TestesPassaram != 0 {
		t.Fatalf("sem troca de usuário o PR não pode rodar como root, obtive %+v", resultado)
	}
	if !strings.Contains(resultado.SaidaBruta, "não foi possível rodar os testes como usuário sem privilégio") {
		t.Errorf("a mensagem precisa nomear a troca negada: %q", resultado.SaidaBruta)
	}
}

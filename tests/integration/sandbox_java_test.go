package integracao

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

func executarCasoJava(t *testing.T, caso string, prazo time.Duration) sandbox.ResultadoSandbox {
	t.Helper()
	imagemJava.exigir(t)

	volume, shas := publicarRepositorios(t, imagemJava, caso)
	executor := sandbox.NovoExecutorLocal(sandbox.OpcoesExecutorLocal{
		ImagemPorLinguagem: map[sandbox.Linguagem]string{sandbox.LinguagemJava: imagemJava.nome()},
		Montagens:          []string{volume + ":" + pontoDeMontagem + ":ro"},
	})

	ctx, cancelar := context.WithTimeout(t.Context(), prazo)
	defer cancelar()

	resultado, err := executor.Executar(ctx, sandbox.PedidoExecucao{
		URLRepositorio: "file://" + pontoDeMontagem + "/" + caso,
		SHA:            shas[caso],
		Linguagem:      sandbox.LinguagemJava,
		TaskRunID:      "integracao-java-" + caso,
	})
	if err != nil {
		t.Fatalf("executar caso %s: %v", caso, err)
	}

	t.Logf("caso %s: tipo=%s compilou=%v passaram=%d falharam=%d duracao=%dms\n%s",
		caso, resultado.TipoFalha, resultado.Compilou, resultado.TestesPassaram, resultado.TestesFalharam,
		resultado.DuracaoMS, resultado.SaidaBruta)

	return resultado
}

func exigirSemFalhaComContagem(t *testing.T, resultado sandbox.ResultadoSandbox, passaram int) {
	t.Helper()

	if resultado.TipoFalha != sandbox.FalhaNenhuma || !resultado.Compilou ||
		resultado.TestesPassaram != passaram || resultado.TestesFalharam != 0 {
		t.Errorf("esperava %d testes passando sem falha, obtive %+v", passaram, resultado)
	}
}

func TestSandboxJavaProjetoOkRodaComoUsuarioSemPrivilegios(t *testing.T) {
	exigirSemFalhaComContagem(t, executarCasoJava(t, "ok", prazoDeUmCasoComum), 3)
}

func TestSandboxJavaTesteFalho(t *testing.T) {
	resultado := executarCasoJava(t, "teste_falho", prazoDeUmCasoComum)

	if resultado.TipoFalha != sandbox.FalhaTeste || !resultado.Compilou ||
		resultado.TestesPassaram != 1 || resultado.TestesFalharam != 1 {
		t.Errorf("esperava falha de teste com 1 passando e 1 falhando, obtive %+v", resultado)
	}
	if !strings.Contains(resultado.SaidaBruta, "expected: <12.0> but was: <10.0>") {
		t.Error("a mensagem do teste falho precisa estar na saída")
	}
}

func TestSandboxJavaJUnit4PeloSurefireMaisNovo(t *testing.T) {
	exigirSemFalhaComContagem(t, executarCasoJava(t, "junit4", prazoDeUmCasoComum), 2)
}

func TestSandboxJavaSemTestesNaoReprova(t *testing.T) {
	exigirSemFalhaComContagem(t, executarCasoJava(t, "sem_testes", prazoDeUmCasoComum), 0)
}

func TestSandboxJavaIgnoraOMvnwDoRepositorio(t *testing.T) {
	resultado := executarCasoJava(t, "com_mvnw", prazoDeUmCasoComum)

	exigirSemFalhaComContagem(t, resultado, 3)
	if strings.Contains(resultado.SaidaBruta, "o mvnw do repositório foi executado") {
		t.Error("o mvnw do repositório nunca roda; só o mvn da imagem")
	}
}

func TestSandboxJavaIgnoraRelatorioVersionadoNoRepositorio(t *testing.T) {
	exigirSemFalhaComContagem(t, executarCasoJava(t, "relatorio_versionado", prazoDeUmCasoComum), 3)
}

func TestSandboxJavaErroDeCompilacaoEhCompilacao(t *testing.T) {
	resultado := executarCasoJava(t, "nao_compila", prazoDeUmCasoComum)

	if resultado.TipoFalha != sandbox.FalhaCompilacao || resultado.Compilou ||
		resultado.TestesPassaram != 0 || !strings.Contains(resultado.SaidaBruta, "taxaInexistente") {
		t.Errorf("erro de compilação é do PR, com o erro na evidência: %+v", resultado)
	}
}

func TestSandboxJavaDependenciaForaDoCacheEhCompilacaoComOArtefato(t *testing.T) {
	resultado := executarCasoJava(t, "dependencia_fora_do_cache", prazoDeUmCasoComum)

	if resultado.TipoFalha != sandbox.FalhaCompilacao ||
		!strings.Contains(resultado.SaidaBruta, "artefato fora do cache do sandbox Java: com.google.guava:guava:jar:33.0.0-jre") {
		t.Errorf("dependência fora do cache é do PR, com o artefato na evidência: %+v", resultado)
	}
}

func TestSandboxJavaMaisNovoQueOJDKDaImagemEhInfraestrutura(t *testing.T) {
	resultado := executarCasoJava(t, "java_mais_novo", prazoDeUmCasoComum)

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura || !strings.Contains(resultado.SaidaBruta, "Java mais novo que o JDK 21") {
		t.Errorf("release mais novo que o JDK da imagem é infraestrutura: %+v", resultado)
	}
}

func TestSandboxJavaSemPomEhInfraestrutura(t *testing.T) {
	resultado := executarCasoJava(t, "sem_pom", prazoDeUmCasoComum)

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura || !strings.Contains(resultado.SaidaBruta, "sem pom.xml") {
		t.Errorf("configuração errada não pode virar aprovação com zero testes: %+v", resultado)
	}
}

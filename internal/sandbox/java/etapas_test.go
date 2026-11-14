package java

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/sandbox"
	"github.com/PI-GRUPO/aPRova/internal/sandbox/processo"
)

const (
	cacheDoVolume       = "/aprova-preparo/cache"
	diretorioDeCache    = "/tmp/aprova-cache"
	repositorioDaImagem = "/opt/aprova-cache/m2"

	etapaDeRemocao   = "remocao"
	etapaDeCompilar  = "test-compile"
	etapaDeTestar    = "test"
	relatorioPassou  = `<testsuite tests="3" failures="0" errors="0" skipped="0"></testsuite>`
	relatorioFalhou  = `<testsuite tests="3" failures="1" errors="0" skipped="0"></testsuite>`
	faltaNoCacheJar  = "[ERROR] Failed to execute goal on project frete: Could not resolve dependencies for project com.aprova:frete:jar:1.0.0\n[ERROR] \tCannot access central (https://repo.maven.apache.org/maven2) in offline mode and the artifact com.google.guava:guava:jar:33.0.0-jre has not been downloaded from it before.\n"
	faltaNoCacheTest = "[ERROR] Failed to execute goal org.apache.maven.plugins:maven-surefire-plugin:3.2.5:test (default-test) on project frete: The following artifacts could not be resolved: org.junit.platform:junit-platform-launcher:jar:1.10.2 (absent): Cannot access central (https://repo.maven.apache.org/maven2) in offline mode and the artifact org.junit.platform:junit-platform-launcher:jar:1.10.2 has not been downloaded from it before. -> [Help 1]\n"
)

var instanteDosTestes = time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

type rodadorFalso struct {
	respostas map[string]processo.Execucao
	relatorio string
	rodados   []processo.Comando
}

func (r *rodadorFalso) Rodar(_ context.Context, comando processo.Comando) processo.Execucao {
	r.rodados = append(r.rodados, comando)
	etapa := etapaDoComando(comando)
	if etapa == etapaDeTestar && r.relatorio != "" {
		relatorios := filepath.Join(comando.Diretorio, "target", "surefire-reports")
		_ = os.MkdirAll(relatorios, 0o700)
		_ = os.WriteFile(filepath.Join(relatorios, "TEST-FreteTest.xml"), []byte(r.relatorio), 0o600)
	}
	return r.respostas[etapa]
}

func (r *rodadorFalso) etapasRodadas() []string {
	etapas := make([]string, 0, len(r.rodados))
	for _, comando := range r.rodados {
		etapas = append(etapas, etapaDoComando(comando))
	}
	return etapas
}

func etapaDoComando(comando processo.Comando) string {
	switch {
	case comando.Binario != binarioMaven:
		return etapaDeRemocao
	case slices.Contains(comando.Argumentos, etapaDeCompilar):
		return etapaDeCompilar
	case slices.Contains(comando.Argumentos, etapaDeTestar):
		return etapaDeTestar
	default:
		return "desconhecida"
	}
}

func linguagemComRespostas(respostas map[string]processo.Execucao, relatorio string) (*Linguagem, *rodadorFalso) {
	rodador := &rodadorFalso{respostas: respostas, relatorio: relatorio}
	linguagem := Nova(Opcoes{
		Rodador:             rodador,
		DiretorioDeCache:    diretorioDeCache,
		RepositorioDaImagem: repositorioDaImagem,
		Agora:               func() time.Time { return instanteDosTestes },
	})
	return linguagem, rodador
}

func projetoComPom(t *testing.T) string {
	t.Helper()

	projeto := t.TempDir()
	if err := os.WriteFile(filepath.Join(projeto, "pom.xml"), []byte("<project/>"), 0o600); err != nil {
		t.Fatalf("gravar pom.xml: %v", err)
	}
	return projeto
}

func testar(t *testing.T, respostas map[string]processo.Execucao, relatorio string) (sandbox.ResultadoSandbox, *rodadorFalso) {
	t.Helper()

	linguagem, rodador := linguagemComRespostas(respostas, relatorio)
	resultado := linguagem.Testar(t.Context(), projetoComPom(t), cacheDoVolume, instanteDosTestes.Add(10*time.Minute))
	return resultado, rodador
}

func saiu(codigo int, saida string) processo.Execucao {
	return processo.Execucao{CodigoSaida: codigo, Saida: []byte(saida)}
}

func TestPreparoDoJavaNaoRodaNada(t *testing.T) {
	linguagem, rodador := linguagemComRespostas(nil, "")

	falha := linguagem.Preparar(t.Context(), projetoComPom(t), cacheDoVolume)

	if falha != nil || len(rodador.rodados) != 0 {
		t.Errorf("o preparo do Java é só o clone; nenhum mvn: %+v, rodados %v", falha, rodador.etapasRodadas())
	}
}

func TestSemPomXmlEhInfraestruturaNasDuasEtapasSemRodarNada(t *testing.T) {
	linguagem, rodador := linguagemComRespostas(nil, "")

	falhaDoPreparo := linguagem.Preparar(t.Context(), t.TempDir(), cacheDoVolume)
	resultadoDoTeste := linguagem.Testar(t.Context(), t.TempDir(), cacheDoVolume, instanteDosTestes.Add(10*time.Minute))

	if falhaDoPreparo == nil || falhaDoPreparo.TipoFalha != sandbox.FalhaInfraestrutura ||
		!strings.Contains(falhaDoPreparo.SaidaBruta, "sem pom.xml") {
		t.Errorf("preparo sem pom.xml: %+v", falhaDoPreparo)
	}
	if resultadoDoTeste.TipoFalha != sandbox.FalhaInfraestrutura || !strings.Contains(resultadoDoTeste.SaidaBruta, "sem pom.xml") {
		t.Errorf("teste sem pom.xml: %+v", resultadoDoTeste)
	}
	if len(rodador.rodados) != 0 {
		t.Errorf("sem pom.xml nada roda: %v", rodador.etapasRodadas())
	}
}

func TestTesteApagaRelatoriosVersionadosCompilaETestaComOMavenDaImagemOffline(t *testing.T) {
	resultado, rodador := testar(t, nil, relatorioPassou)

	if rodadas := rodador.etapasRodadas(); !slices.Equal(rodadas, []string{etapaDeRemocao, etapaDeCompilar, etapaDeTestar}) {
		t.Fatalf("ordem: apagar relatórios versionados, test-compile, test; obtive %v", rodadas)
	}
	if resultado.TipoFalha != sandbox.FalhaNenhuma || !resultado.Compilou || resultado.TestesPassaram != 3 {
		t.Errorf("tudo passando: %+v", resultado)
	}

	remocao := rodador.rodados[0]
	if remocao.Binario != "find" || !slices.Contains(remocao.Argumentos, "*/target/surefire-reports") ||
		!slices.Contains(remocao.Argumentos, "rm") {
		t.Errorf("os relatórios que vieram no repositório são apagados antes do mvn: %s %v", remocao.Binario, remocao.Argumentos)
	}

	for _, maven := range rodador.rodados[1:] {
		if maven.Binario != "/usr/share/maven/bin/mvn" {
			t.Errorf("sempre o mvn da imagem por caminho absoluto, nunca o mvnw: %s", maven.Binario)
		}
		for _, argumento := range []string{
			"-o", "-B", "-ntp",
			"-Dmaven.repo.local=" + diretorioDeCache + "/m2",
			"-Dmaven.repo.local.tail=" + repositorioDaImagem,
		} {
			if !slices.Contains(maven.Argumentos, argumento) {
				t.Errorf("%v sem %s", maven.Argumentos, argumento)
			}
		}
		esperado := []string{
			"PATH=/opt/java/openjdk/bin:/usr/local/bin:/usr/bin:/bin",
			"HOME=" + diretorioDeCache + "/home",
			"JAVA_HOME=/opt/java/openjdk",
		}
		if !slices.Equal(maven.Ambiente, esperado) {
			t.Errorf("ambiente do mvn: %v, esperava só %v", maven.Ambiente, esperado)
		}
	}
}

func TestTesteFalhoEhFalhaDeTeste(t *testing.T) {
	resultado, _ := testar(t, map[string]processo.Execucao{etapaDeTestar: saiu(1, "Tests run: 3, Failures: 1")}, relatorioFalhou)

	if resultado.TipoFalha != sandbox.FalhaTeste || !resultado.Compilou || resultado.TestesPassaram != 2 || resultado.TestesFalharam != 1 {
		t.Errorf("um teste falhando: %+v", resultado)
	}
}

func TestProjetoSemTestesNaoReprova(t *testing.T) {
	resultado, _ := testar(t, map[string]processo.Execucao{etapaDeTestar: saiu(0, "[INFO] No tests to run.")}, "")

	if resultado.TipoFalha != sandbox.FalhaNenhuma || !resultado.Compilou || resultado.TestesPassaram != 0 || resultado.TestesFalharam != 0 {
		t.Errorf("sem testes é nenhuma com 0/0, como no Python: %+v", resultado)
	}
}

func TestErroDeCompilacaoEhCompilacaoSemRodarOsTestes(t *testing.T) {
	resultado, rodador := testar(t, map[string]processo.Execucao{
		etapaDeCompilar: saiu(1, "[ERROR] COMPILATION ERROR :\n[ERROR] /p/src/main/java/Frete.java:[3,9] cannot find symbol"),
	}, "")

	if resultado.TipoFalha != sandbox.FalhaCompilacao || resultado.Compilou || !strings.Contains(resultado.SaidaBruta, "cannot find symbol") {
		t.Errorf("erro de compilação é do PR: %+v", resultado)
	}
	if slices.Contains(rodador.etapasRodadas(), etapaDeTestar) {
		t.Error("sem compilar, os testes não rodam")
	}
}

func TestArtefatoForaDoCacheEhCompilacaoComOArtefatoNaEvidencia(t *testing.T) {
	casos := map[string]struct {
		respostas map[string]processo.Execucao
		artefato  string
	}{
		"dependência, no test-compile": {
			respostas: map[string]processo.Execucao{etapaDeCompilar: saiu(1, faltaNoCacheJar)},
			artefato:  "com.google.guava:guava:jar:33.0.0-jre",
		},
		"provider do surefire, no test": {
			respostas: map[string]processo.Execucao{etapaDeTestar: saiu(1, faltaNoCacheTest)},
			artefato:  "org.junit.platform:junit-platform-launcher:jar:1.10.2",
		},
	}
	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			resultado, _ := testar(t, caso.respostas, "")

			evidencia := "aprova: artefato fora do cache do sandbox Java: " + caso.artefato
			if resultado.TipoFalha != sandbox.FalhaCompilacao || resultado.Compilou || !strings.Contains(resultado.SaidaBruta, evidencia) {
				t.Errorf("esperava compilação com %q, obtive %+v", evidencia, resultado)
			}
		})
	}
}

func TestJavaMaisNovoQueOJDKDaImagemEhInfraestrutura(t *testing.T) {
	for _, saida := range []string{
		"[ERROR] Failed to execute goal org.apache.maven.plugins:maven-compiler-plugin:3.15.0:compile (default-compile) on project frete: Fatal error compiling: error: release version 25 not supported -> [Help 1]",
		"[ERROR] Failed to execute goal org.apache.maven.plugins:maven-compiler-plugin:3.15.0:compile (default-compile) on project frete: Fatal error compiling: error: invalid target release: 25 -> [Help 1]",
	} {
		resultado, _ := testar(t, map[string]processo.Execucao{etapaDeCompilar: saiu(1, saida)}, "")

		if resultado.TipoFalha != sandbox.FalhaInfraestrutura || !strings.Contains(resultado.SaidaBruta, "Java mais novo que o JDK 21") {
			t.Errorf("release mais novo que o JDK da imagem é infraestrutura, como o go.mod mais novo: %+v", resultado)
		}
	}
}

func TestMavenQueNaoIniciaOuEsgotaOPrazoEhInfraestrutura(t *testing.T) {
	casos := map[string]map[string]processo.Execucao{
		"remoção não inicia":      {etapaDeRemocao: {Erro: errors.New("exec: find não encontrado")}},
		"remoção falha":           {etapaDeRemocao: saiu(1, "rm: permissão negada")},
		"test-compile não inicia": {etapaDeCompilar: {Erro: errors.New("exec: mvn não encontrado")}},
		"test-compile esgota":     {etapaDeCompilar: {PrazoEsgotado: true}},
		"test esgota":             {etapaDeTestar: {PrazoEsgotado: true}},
	}
	for nome, respostas := range casos {
		t.Run(nome, func(t *testing.T) {
			resultado, _ := testar(t, respostas, relatorioPassou)

			if resultado.TipoFalha != sandbox.FalhaInfraestrutura || resultado.TestesPassaram != 0 {
				t.Errorf("%s: %+v", nome, resultado)
			}
		})
	}
}

func TestCodigoDeSaidaQueOsRelatoriosNaoExplicamEhInfraestrutura(t *testing.T) {
	casos := map[string]struct {
		saida     processo.Execucao
		relatorio string
	}{
		"falhou sem teste falho nos relatórios": {saida: saiu(1, "BUILD FAILURE"), relatorio: relatorioPassou},
		"passou com teste falho nos relatórios": {saida: saiu(0, "BUILD SUCCESS"), relatorio: relatorioFalhou},
		"relatório ilegível":                    {saida: saiu(0, "BUILD SUCCESS"), relatorio: "<testsuite"},
	}
	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			resultado, _ := testar(t, map[string]processo.Execucao{etapaDeTestar: caso.saida}, caso.relatorio)

			if resultado.TipoFalha != sandbox.FalhaInfraestrutura {
				t.Errorf("o PR não aprova com divergência entre o Maven e os relatórios: %+v", resultado)
			}
		})
	}
}

func TestPrazoInsuficienteNaoRodaOMaven(t *testing.T) {
	linguagem, rodador := linguagemComRespostas(nil, "")

	resultado := linguagem.Testar(t.Context(), projetoComPom(t), cacheDoVolume, instanteDosTestes.Add(45*time.Second))

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura || !strings.Contains(resultado.SaidaBruta, "prazo insuficiente") || len(rodador.rodados) != 0 {
		t.Errorf("sem tempo para os testes, nada roda: %+v, rodados %v", resultado, rodador.etapasRodadas())
	}
}

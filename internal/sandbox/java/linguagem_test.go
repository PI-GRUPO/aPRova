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

	"github.com/Azulpasta/aPRova/internal/sandbox"
	"github.com/Azulpasta/aPRova/internal/sandbox/processo"
)

var agoraJavaDeTeste = time.Date(
	2026, 10, 28, 12, 0, 0, 0, time.UTC,
)

const (
	relatorioJavaSoPassando = `<testsuite tests="3" failures="0" errors="0" skipped="0"></testsuite>`

	relatorioJavaUmFalho = `<testsuite tests="3" failures="1" errors="0" skipped="0"></testsuite>`

	relatorioJavaComErro = `<testsuite tests="3" failures="0" errors="1" skipped="0"></testsuite>`

	relatorioJavaMultiplasSuites = `<testsuites>
		<testsuite tests="2" failures="0" errors="0" skipped="0"></testsuite>
		<testsuite tests="2" failures="1" errors="0" skipped="0"></testsuite>
	</testsuites>`
)

type rodadorJavaFalso struct {
	respostas map[string]processo.Execucao
	rodados   []processo.Comando
}

func (r *rodadorJavaFalso) Rodar(
	_ context.Context,
	comando processo.Comando,
) processo.Execucao {
	r.rodados = append(r.rodados, comando)

	if slices.Contains(comando.Argumentos, "test-compile") {
		return r.respostas["test-compile"]
	}

	if slices.Contains(comando.Argumentos, "test") {
		return r.respostas["test"]
	}

	return processo.Execucao{}
}

func repositorioJavaDeTeste(
	t *testing.T,
	relatorio string,
) string {
	t.Helper()

	diretorio := t.TempDir()

	if relatorio == "" {
		return diretorio
	}

	diretorioRelatorios := filepath.Join(
		diretorio,
		"target",
		"surefire-reports",
	)

	if err := os.MkdirAll(diretorioRelatorios, 0o700); err != nil {
		t.Fatalf("criar diretório de relatórios: %v", err)
	}

	arquivo := filepath.Join(
		diretorioRelatorios,
		"TEST-exemplo.xml",
	)

	if err := os.WriteFile(
		arquivo,
		[]byte(relatorio),
		0o600,
	); err != nil {
		t.Fatalf("criar relatório Maven: %v", err)
	}

	return diretorio
}

func executorJavaDeTeste(
	respostas map[string]processo.Execucao,
) (*Linguagem, *rodadorJavaFalso) {
	rodador := &rodadorJavaFalso{
		respostas: respostas,
	}

	linguagem := Nova(Opcoes{
		Rodador:          rodador,
		DiretorioDeCache: "/tmp/aprova-cache",
		Agora: func() time.Time {
			return agoraJavaDeTeste
		},
	})

	return linguagem, rodador
}

func executarJavaDeTeste(
	t *testing.T,
	relatorio string,
	respostas map[string]processo.Execucao,
	prazo time.Duration,
) (sandbox.ResultadoSandbox, *rodadorJavaFalso) {
	t.Helper()

	diretorio := repositorioJavaDeTeste(t, relatorio)

	linguagem, rodador := executorJavaDeTeste(respostas)

	resultado := linguagem.Executar(
		t.Context(),
		diretorio,
		agoraJavaDeTeste.Add(prazo),
	)

	return resultado, rodador
}

func TestExecutarJavaComTodosOsTestesPassando(t *testing.T) {
	resultado, rodador := executarJavaDeTeste(
		t,
		relatorioJavaSoPassando,
		map[string]processo.Execucao{
			"test-compile": {CodigoSaida: 0},
			"test":         {CodigoSaida: 0, Saida: []byte("BUILD SUCCESS")},
		},
		10*time.Minute,
	)

	if resultado.TipoFalha != sandbox.FalhaNenhuma ||
		!resultado.Compilou ||
		resultado.TestesPassaram != 3 ||
		resultado.TestesFalharam != 0 {
		t.Errorf("resultado inesperado: %+v", resultado)
	}

	if len(rodador.rodados) != 2 {
		t.Fatalf(
			"esperava test-compile e test; comandos executados: %d",
			len(rodador.rodados),
		)
	}

	compilacao := rodador.rodados[0]
	if compilacao.Binario != comandoMaven ||
		compilacao.Diretorio == "" ||
		!slices.Contains(compilacao.Argumentos, "test-compile") ||
		!slices.Contains(compilacao.Argumentos, "-B") {
		t.Errorf("comando de compilação incorreto: %+v", compilacao)
	}

	testes := rodador.rodados[1]
	if testes.Binario != comandoMaven ||
		!slices.Contains(testes.Argumentos, "test") ||
		!slices.Contains(testes.Argumentos, "-B") {
		t.Errorf("comando de testes incorreto: %+v", testes)
	}
}

func TestExecutarJavaComTesteFalho(t *testing.T) {
	resultado, _ := executarJavaDeTeste(
		t,
		relatorioJavaUmFalho,
		map[string]processo.Execucao{
			"test-compile": {CodigoSaida: 0},
			"test":         {CodigoSaida: 1, Saida: []byte("BUILD FAILURE")},
		},
		10*time.Minute,
	)

	if resultado.TipoFalha != sandbox.FalhaTeste ||
		resultado.TestesPassaram != 2 ||
		resultado.TestesFalharam != 1 {
		t.Errorf(
			"teste reprovado deveria ser classificado como falha de teste: %+v",
			resultado,
		)
	}
}

func TestExecutarJavaComErroNoTeste(t *testing.T) {
	resultado, _ := executarJavaDeTeste(
		t,
		relatorioJavaComErro,
		map[string]processo.Execucao{
			"test-compile": {CodigoSaida: 0},
			"test":         {CodigoSaida: 1},
		},
		10*time.Minute,
	)

	if resultado.TipoFalha != sandbox.FalhaTeste ||
		resultado.TestesPassaram != 2 ||
		resultado.TestesFalharam != 1 {
		t.Errorf("erro no teste deveria ser contabilizado: %+v", resultado)
	}
}

func TestExecutarJavaComFalhaDeCompilacao(t *testing.T) {
	resultado, rodador := executarJavaDeTeste(
		t,
		"",
		map[string]processo.Execucao{
			"test-compile": {
				CodigoSaida: 1,
				Saida:       []byte("compilation failure"),
			},
		},
		10*time.Minute,
	)

	if resultado.TipoFalha != sandbox.FalhaCompilacao ||
		resultado.Compilou {
		t.Errorf(
			"falha de compilação deveria ser classificada corretamente: %+v",
			resultado,
		)
	}

	if len(rodador.rodados) != 1 {
		t.Errorf(
			"mvn test não deveria executar após falha de compilação: %d",
			len(rodador.rodados),
		)
	}
}

func TestExecutarJavaQuandoMavenNaoIniciaEhInfraestrutura(t *testing.T) {
	resultado, _ := executarJavaDeTeste(
		t,
		"",
		map[string]processo.Execucao{
			"test-compile": {
				Erro: errors.New("executável Maven não encontrado"),
			},
		},
		10*time.Minute,
	)

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura {
		t.Errorf(
			"falha ao iniciar Maven deveria ser infraestrutura: %+v",
			resultado,
		)
	}
}

func TestExecutarJavaQuandoPrazoEsgotaEhInfraestrutura(t *testing.T) {
	resultado, _ := executarJavaDeTeste(
		t,
		"",
		map[string]processo.Execucao{
			"test-compile": {PrazoEsgotado: true},
		},
		10*time.Minute,
	)

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura {
		t.Errorf(
			"timeout deveria ser infraestrutura: %+v",
			resultado,
		)
	}
}

func TestExecutarJavaQuandoMvnTestNaoIniciaEhInfraestrutura(t *testing.T) {
	resultado, _ := executarJavaDeTeste(
		t,
		relatorioJavaSoPassando,
		map[string]processo.Execucao{
			"test-compile": {CodigoSaida: 0},
			"test": {
				Erro: errors.New("não foi possível iniciar Maven"),
			},
		},
		10*time.Minute,
	)

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura {
		t.Errorf(
			"falha ao iniciar os testes deveria ser infraestrutura: %+v",
			resultado,
		)
	}
}

func TestExecutarJavaRecusaRelatorioInconsistente(t *testing.T) {
	resultado, _ := executarJavaDeTeste(
		t,
		relatorioJavaSoPassando,
		map[string]processo.Execucao{
			"test-compile": {CodigoSaida: 0},
			"test":         {CodigoSaida: 1},
		},
		10*time.Minute,
	)

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura {
		t.Errorf(
			"divergência entre relatório e processo deveria ser infraestrutura: %+v",
			resultado,
		)
	}
}

func TestExecutarJavaRecusaRelatorioAusente(t *testing.T) {
	resultado, _ := executarJavaDeTeste(
		t,
		"",
		map[string]processo.Execucao{
			"test-compile": {CodigoSaida: 0},
			"test":         {CodigoSaida: 0},
		},
		10*time.Minute,
	)

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura {
		t.Errorf(
			"relatório ausente deveria ser infraestrutura: %+v",
			resultado,
		)
	}
}

func TestExecutarJavaComMultiplasSuites(t *testing.T) {
	resultado, _ := executarJavaDeTeste(
		t,
		relatorioJavaMultiplasSuites,
		map[string]processo.Execucao{
			"test-compile": {CodigoSaida: 0},
			"test":         {CodigoSaida: 1},
		},
		10*time.Minute,
	)

	if resultado.TipoFalha != sandbox.FalhaTeste ||
		resultado.TestesPassaram != 3 ||
		resultado.TestesFalharam != 1 {
		t.Errorf(
			"múltiplas suítes deveriam ser contabilizadas: %+v",
			resultado,
		)
	}
}

func TestExecutarJavaSemTempoSuficienteNaoExecutaMaven(t *testing.T) {
	resultado, rodador := executarJavaDeTeste(
		t,
		"",
		nil,
		20*time.Second,
	)

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura {
		t.Errorf(
			"prazo insuficiente deveria ser infraestrutura: %+v",
			resultado,
		)
	}

	if len(rodador.rodados) != 0 {
		t.Errorf(
			"Maven não deveria executar sem tempo suficiente: %d",
			len(rodador.rodados),
		)
	}
}

func TestExecutarJavaUsaAmbienteRestrito(t *testing.T) {
	_, rodador := executarJavaDeTeste(
		t,
		relatorioJavaSoPassando,
		map[string]processo.Execucao{
			"test-compile": {CodigoSaida: 0},
			"test":         {CodigoSaida: 0},
		},
		10*time.Minute,
	)

	if len(rodador.rodados) == 0 {
		t.Fatal("nenhum comando Maven foi executado")
	}

	comando := rodador.rodados[0]

	for _, exigida := range []string{
		"PATH=" + caminhoDosExecutaveis,
		"HOME=/tmp/aprova-cache/home",
		"JAVA_HOME=/opt/java/openjdk",
		"MAVEN_OPTS=-Dmaven.repo.local=/tmp/aprova-cache/m2",
	} {
		if !slices.Contains(comando.Ambiente, exigida) {
			t.Errorf(
				"ambiente não contém %q: %v",
				exigida,
				comando.Ambiente,
			)
		}
	}

	if !strings.Contains(
		strings.Join(comando.Argumentos, " "),
		"-Dmaven.repo.local=/tmp/aprova-cache/m2",
	) {
		t.Errorf(
			"comando não configura o cache local do Maven: %+v",
			comando.Argumentos,
		)
	}
}

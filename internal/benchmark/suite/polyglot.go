package suite

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PI-GRUPO/aPRova/internal/benchmark/resultado"
)

const (
	repositorioDoPolyglot = "Aider-AI/polyglot-benchmark"
	diretorioDosArquivos  = "arquivos"
	arquivoDoEnunciado    = "enunciado.md"
	oraculoDoGrupoA       = "testes originais do Aider passam contra a solução"
	linguagemJava         = "java"

	conversaoParaMaven = "build.gradle e o wrapper do Gradle trocados por pom.xml equivalente: " +
		"junit-jupiter 5.10.2 (o BOM 5.10.0 não está no cache do sandbox), assertj-core 3.25.1, surefire 3.5.4 e release 21"
	conversaoDosDisabled = "anotações @Disabled removidas dos testes, como a suíte do Aider faz antes de rodar"
)

var (
	dependenciaDeProducao = regexp.MustCompile(`^\s*(implementation|api|compileOnly|runtimeOnly)\s`)
	anotacaoDisabled      = regexp.MustCompile(`^\s*@Disabled(\(.*\))?\s*$`)
	arquivosDoGradle      = []string{"build.gradle", "gradlew", "gradlew.bat", "settings.gradle"}
)

type configuracaoDoExercicio struct {
	Arquivos struct {
		Solucao     []string `json:"solution"`
		Teste       []string `json:"test"`
		Editor      []string `json:"editor"`
		Invalidador []string `json:"invalidator"`
		Exemplo     []string `json:"example"`
	} `json:"files"`
}

// JavaElegivel diz se um exercício Java cabe no sandbox: o build.gradle não
// declara dependência de produção, que não estaria no cache do Maven.
func JavaElegivel(buildGradle []byte) bool {
	leitor := bufio.NewScanner(bytes.NewReader(buildGradle))
	for leitor.Scan() {
		if dependenciaDeProducao.MatchString(leitor.Text()) {
			return false
		}
	}
	return true
}

// RemoverDisabled tira as linhas que só têm a anotação @Disabled.
func RemoverDisabled(conteudo []byte) []byte {
	var saida bytes.Buffer
	for _, linha := range strings.SplitAfter(string(conteudo), "\n") {
		if anotacaoDisabled.MatchString(strings.TrimRight(linha, "\n")) {
			continue
		}
		saida.WriteString(linha)
	}
	return saida.Bytes()
}

// PomDoExercicio gera o pom.xml que substitui o build.gradle do exercício,
// com as versões aquecidas no cache do sandbox Java.
func PomDoExercicio(nome string) []byte {
	return []byte(fmt.Sprintf(`<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 https://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>

    <groupId>exercism</groupId>
    <artifactId>%s</artifactId>
    <version>1.0.0</version>

    <properties>
        <maven.compiler.release>21</maven.compiler.release>
        <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>
    </properties>

    <dependencies>
        <dependency>
            <groupId>org.junit.jupiter</groupId>
            <artifactId>junit-jupiter</artifactId>
            <version>5.10.2</version>
            <scope>test</scope>
        </dependency>
        <dependency>
            <groupId>org.assertj</groupId>
            <artifactId>assertj-core</artifactId>
            <version>3.25.1</version>
            <scope>test</scope>
        </dependency>
    </dependencies>

    <build>
        <plugins>
            <plugin>
                <groupId>org.apache.maven.plugins</groupId>
                <artifactId>maven-surefire-plugin</artifactId>
                <version>3.5.4</version>
            </plugin>
        </plugins>
    </build>
</project>
`, nome))
}

// MontarDoPolyglot copia um exercício do polyglot para destino: os arquivos
// da tarefa em destino/arquivos e o enunciado em destino/enunciado.md. A
// solução de referência do .meta nunca é copiada; Java é convertido para
// Maven.
func MontarDoPolyglot(raiz, commit, linguagem, exercicio, destino string) (Tarefa, error) {
	origem := filepath.Join(raiz, linguagem, "exercises", "practice", exercicio)
	configuracao, err := lerConfiguracao(origem)
	if err != nil {
		return Tarefa{}, err
	}

	tarefa := Tarefa{
		ID:                 linguagem + "-" + exercicio,
		Grupo:              resultado.GrupoCorrecao,
		Linguagem:          linguagem,
		Origem:             fmt.Sprintf("%s@%s:%s/exercises/practice/%s", repositorioDoPolyglot, commit, linguagem, exercicio),
		Licenca:            "MIT (exercism/" + linguagem + ")",
		Esperado:           Esperado{Oraculo: oraculoDoGrupoA},
		ArquivosDaSolucao:  configuracao.Arquivos.Solucao,
		ArquivosProtegidos: append(append([]string{}, configuracao.Arquivos.Teste...), configuracao.Arquivos.Editor...),
	}

	copiar := append(append(append([]string{}, configuracao.Arquivos.Solucao...), tarefa.ArquivosProtegidos...), configuracao.Arquivos.Invalidador...)
	if err := copiarArquivos(origem, filepath.Join(destino, diretorioDosArquivos), linguagem, copiar); err != nil {
		return Tarefa{}, err
	}
	if linguagem == linguagemJava {
		if err := gravar(filepath.Join(destino, diretorioDosArquivos, "pom.xml"), PomDoExercicio(exercicio)); err != nil {
			return Tarefa{}, err
		}
		tarefa.Conversao = []string{conversaoParaMaven, conversaoDosDisabled}
	}

	enunciado, err := lerEnunciado(origem)
	if err != nil {
		return Tarefa{}, err
	}
	return tarefa, gravar(filepath.Join(destino, arquivoDoEnunciado), enunciado)
}

func copiarArquivos(origem, destino, linguagem string, relativos []string) error {
	for _, relativo := range relativos {
		if !filepath.IsLocal(filepath.FromSlash(relativo)) {
			return fmt.Errorf("suite: caminho fora do exercício no config.json: %q", relativo)
		}
		if linguagem == linguagemJava && ehDoGradle(relativo) {
			continue
		}
		//nolint:gosec // G304: o caminho vem do config.json do polyglot fixado por commit e foi conferido com filepath.IsLocal.
		conteudo, err := os.ReadFile(filepath.Join(origem, filepath.FromSlash(relativo)))
		if err != nil {
			return fmt.Errorf("suite: ler %s: %w", relativo, err)
		}
		if linguagem == linguagemJava && strings.HasSuffix(relativo, ".java") && strings.Contains(relativo, "src/test/") {
			conteudo = RemoverDisabled(conteudo)
		}
		if err := gravar(filepath.Join(destino, filepath.FromSlash(relativo)), conteudo); err != nil {
			return err
		}
	}
	return nil
}

func lerConfiguracao(origem string) (configuracaoDoExercicio, error) {
	var configuracao configuracaoDoExercicio
	//nolint:gosec // G304: origem é um exercício do polyglot fixado por commit, escolhido pelo sorteio.
	bruto, err := os.ReadFile(filepath.Join(origem, ".meta", "config.json"))
	if err != nil {
		return configuracao, fmt.Errorf("suite: ler a configuração do exercício: %w", err)
	}
	if err := json.Unmarshal(bruto, &configuracao); err != nil {
		return configuracao, fmt.Errorf("suite: configuração do exercício ilegível: %w", err)
	}
	return configuracao, nil
}

func lerEnunciado(origem string) ([]byte, error) {
	var enunciado bytes.Buffer
	for _, nome := range []string{"instructions.md", "instructions.append.md"} {
		//nolint:gosec // G304: origem é um exercício do polyglot fixado por commit, escolhido pelo sorteio.
		parte, err := os.ReadFile(filepath.Join(origem, ".docs", nome))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("suite: ler %s: %w", nome, err)
		}
		if enunciado.Len() > 0 {
			enunciado.WriteString("\n")
		}
		enunciado.Write(parte)
	}
	return enunciado.Bytes(), nil
}

func ehDoGradle(relativo string) bool {
	for _, nome := range arquivosDoGradle {
		if path.Base(relativo) == nome {
			return true
		}
	}
	return strings.HasPrefix(relativo, "gradle/")
}

func gravar(caminho string, conteudo []byte) error {
	if err := os.MkdirAll(filepath.Dir(caminho), 0o750); err != nil {
		return fmt.Errorf("suite: criar %s: %w", filepath.Dir(caminho), err)
	}
	//nolint:gosec // G703: os caminhos relativos do config.json são conferidos com filepath.IsLocal antes de chegar aqui.
	if err := os.WriteFile(caminho, conteudo, 0o600); err != nil {
		return fmt.Errorf("suite: gravar %s: %w", caminho, err)
	}
	return nil
}

// ReferenciaDoExercicio mapeia cada arquivo de solução para o arquivo da
// solução de referência do .meta que o substitui: um para um quando há um só
// de cada; em Java, cada classe de referência vai para src/main/java.
func ReferenciaDoExercicio(raiz, linguagem, exercicio string) (map[string]string, error) {
	origem := filepath.Join(raiz, linguagem, "exercises", "practice", exercicio)
	configuracao, err := lerConfiguracao(origem)
	if err != nil {
		return nil, err
	}
	exemplos, solucoes := configuracao.Arquivos.Exemplo, configuracao.Arquivos.Solucao
	referencia := map[string]string{}
	switch {
	case len(exemplos) == 1 && len(solucoes) == 1:
		referencia[solucoes[0]] = filepath.Join(origem, filepath.FromSlash(exemplos[0]))
	case linguagem == linguagemJava:
		for _, exemplo := range exemplos {
			referencia["src/main/java/"+path.Base(exemplo)] = filepath.Join(origem, filepath.FromSlash(exemplo))
		}
	default:
		return nil, fmt.Errorf("suite: %s/%s tem %d arquivos de referência para %d de solução", linguagem, exercicio, len(exemplos), len(solucoes))
	}
	return referencia, nil
}

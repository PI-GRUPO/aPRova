package suite

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestSorteioEDeterministicoEIndependeDaOrdem(t *testing.T) {
	nomes := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	invertidos := slices.Clone(nomes)
	slices.Reverse(invertidos)

	primeiro, err := Sortear(nomes, 4, 20261114)
	if err != nil {
		t.Fatal(err)
	}
	segundo, _ := Sortear(invertidos, 4, 20261114)
	outraSemente, _ := Sortear(nomes, 4, 7)

	if len(primeiro) != 4 || !slices.Equal(primeiro, segundo) {
		t.Errorf("mesma semente, mesmo sorteio, qualquer ordem: %v e %v", primeiro, segundo)
	}
	if slices.Equal(primeiro, outraSemente) {
		t.Errorf("outra semente deveria sortear outro conjunto: %v", outraSemente)
	}
	if !slices.IsSorted(primeiro) || len(slices.Compact(slices.Clone(primeiro))) != 4 {
		t.Errorf("o sorteio sai ordenado e sem repetição: %v", primeiro)
	}
}

func TestSorteioRecusaQuantidadeMaiorQueOConjunto(t *testing.T) {
	if _, err := Sortear([]string{"a", "b"}, 3, 1); err == nil {
		t.Error("não há como sortear 3 de 2")
	}
}

func TestJavaElegivelSoSemDependenciaDeProducao(t *testing.T) {
	padrao := "dependencies {\n    testImplementation platform(\"org.junit:junit-bom:5.10.0\")\n    testImplementation \"org.junit.jupiter:junit-jupiter\"\n    testImplementation \"org.assertj:assertj-core:3.25.1\"\n}\n"
	comProducao := strings.Replace(padrao, "dependencies {\n", "dependencies {\n    implementation \"org.json:json:20190722\"\n", 1)

	if !JavaElegivel([]byte(padrao)) {
		t.Error("o build.gradle padrão é elegível")
	}
	if JavaElegivel([]byte(comProducao)) {
		t.Error("dependência de produção fora do cache deixa o exercício de fora")
	}
}

func TestRemoverDisabledTiraSoAsAnotacoes(t *testing.T) {
	original := "import org.junit.jupiter.api.Disabled;\n\nclass ATest {\n    @Test\n    void um() {}\n\n    @Disabled(\"Remove to run test\")\n    @Test\n    void dois() {}\n\n    @Disabled\n    @Test\n    void tres() {}\n}\n"

	convertido := string(RemoverDisabled([]byte(original)))

	if strings.Contains(convertido, "@Disabled") {
		t.Errorf("sobrou @Disabled:\n%s", convertido)
	}
	if !strings.Contains(convertido, "import org.junit.jupiter.api.Disabled;") || strings.Count(convertido, "@Test") != 3 {
		t.Errorf("só as anotações saem; o resto fica:\n%s", convertido)
	}
}

func TestPomDoExercicioTemAsVersoesDoCacheDoSandbox(t *testing.T) {
	pom := string(PomDoExercicio("affine-cipher"))

	for _, esperado := range []string{
		"<artifactId>affine-cipher</artifactId>",
		"<artifactId>junit-jupiter</artifactId>\n            <version>5.10.2</version>",
		"<artifactId>assertj-core</artifactId>\n            <version>3.25.1</version>",
		"<artifactId>maven-surefire-plugin</artifactId>\n                <version>3.5.4</version>",
		"<maven.compiler.release>21</maven.compiler.release>",
	} {
		if !strings.Contains(pom, esperado) {
			t.Errorf("pom sem %q:\n%s", esperado, pom)
		}
	}
}

func escrever(t *testing.T, raiz, relativo, conteudo string) {
	t.Helper()
	caminho := filepath.Join(raiz, filepath.FromSlash(relativo))
	if err := os.MkdirAll(filepath.Dir(caminho), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(caminho, []byte(conteudo), 0o600); err != nil {
		t.Fatal(err)
	}
}

func polyglotFalso(t *testing.T) string {
	t.Helper()
	raiz := t.TempDir()
	goEx := "go/exercises/practice/alphametics/"
	escrever(t, raiz, goEx+".meta/config.json", `{"files":{"solution":["alphametics.go"],"test":["alphametics_test.go"],"editor":["cases_test.go"],"invalidator":["go.mod"],"example":[".meta/example.go"]}}`)
	escrever(t, raiz, goEx+".meta/example.go", "package alphametics // SOLUÇÃO DE REFERÊNCIA")
	escrever(t, raiz, goEx+".docs/instructions.md", "# Instructions\n\nResolva o alphametics.\n")
	escrever(t, raiz, goEx+".docs/instructions.append.md", "# Append\n\nUse só a biblioteca padrão.\n")
	escrever(t, raiz, goEx+"alphametics.go", "package alphametics\n")
	escrever(t, raiz, goEx+"alphametics_test.go", "package alphametics\n")
	escrever(t, raiz, goEx+"cases_test.go", "package alphametics\n")
	escrever(t, raiz, goEx+"go.mod", "module alphametics\n\ngo 1.18\n")

	javaEx := "java/exercises/practice/affine-cipher/"
	escrever(t, raiz, javaEx+".meta/config.json", `{"files":{"solution":["src/main/java/AffineCipher.java"],"test":["src/test/java/AffineCipherTest.java"],"invalidator":["build.gradle"],"example":[".meta/src/reference/java/AffineCipher.java"]}}`)
	escrever(t, raiz, javaEx+".meta/src/reference/java/AffineCipher.java", "class AffineCipher {} // SOLUÇÃO DE REFERÊNCIA")
	escrever(t, raiz, javaEx+".docs/instructions.md", "# Instructions\n\nCifra afim.\n")
	escrever(t, raiz, javaEx+"build.gradle", "dependencies {\n    testImplementation \"org.assertj:assertj-core:3.25.1\"\n}\n")
	escrever(t, raiz, javaEx+"gradlew", "#!/bin/sh\n")
	escrever(t, raiz, javaEx+"src/main/java/AffineCipher.java", "class AffineCipher {}\n")
	escrever(t, raiz, javaEx+"src/test/java/AffineCipherTest.java", "class AffineCipherTest {\n    @Disabled(\"Remove to run test\")\n    @Test\n    void um() {}\n}\n")
	return raiz
}

func arquivosDe(t *testing.T, raiz string) []string {
	t.Helper()
	var arquivos []string
	_ = filepath.WalkDir(raiz, func(caminho string, entrada os.DirEntry, err error) error {
		if err == nil && !entrada.IsDir() {
			relativo, _ := filepath.Rel(raiz, caminho)
			arquivos = append(arquivos, filepath.ToSlash(relativo))
		}
		return nil
	})
	slices.Sort(arquivos)
	return arquivos
}

func TestMontarTarefaGoCopiaSoSolucaoTestesEModulo(t *testing.T) {
	destino := t.TempDir()

	tarefa, err := MontarDoPolyglot(polyglotFalso(t), "7e0611e77b54", "go", "alphametics", destino)
	if err != nil {
		t.Fatal(err)
	}

	if arquivos := arquivosDe(t, filepath.Join(destino, "arquivos")); !slices.Equal(arquivos, []string{"alphametics.go", "alphametics_test.go", "cases_test.go", "go.mod"}) {
		t.Errorf("a tarefa leva solução, testes e go.mod, nunca o .meta: %v", arquivos)
	}
	enunciado, _ := os.ReadFile(filepath.Join(destino, "enunciado.md"))
	if !strings.Contains(string(enunciado), "Resolva o alphametics.") || !strings.Contains(string(enunciado), "Use só a biblioteca padrão.") {
		t.Errorf("o enunciado junta instructions.md e instructions.append.md: %q", enunciado)
	}
	if tarefa.ID != "go-alphametics" || tarefa.Linguagem != "go" || tarefa.Licenca != "MIT (exercism/go)" ||
		!strings.Contains(tarefa.Origem, "Aider-AI/polyglot-benchmark@7e0611e77b54") ||
		!slices.Equal(tarefa.ArquivosDaSolucao, []string{"alphametics.go"}) ||
		!slices.Equal(tarefa.ArquivosProtegidos, []string{"alphametics_test.go", "cases_test.go"}) {
		t.Errorf("registro da tarefa: %+v", tarefa)
	}
}

func TestMontarTarefaJavaConverteParaMavenERegistraAConversao(t *testing.T) {
	destino := t.TempDir()

	tarefa, err := MontarDoPolyglot(polyglotFalso(t), "7e0611e77b54", "java", "affine-cipher", destino)
	if err != nil {
		t.Fatal(err)
	}

	arquivos := arquivosDe(t, filepath.Join(destino, "arquivos"))
	if !slices.Equal(arquivos, []string{"pom.xml", "src/main/java/AffineCipher.java", "src/test/java/AffineCipherTest.java"}) {
		t.Errorf("Java vira Maven: sem build.gradle, sem gradlew, com pom.xml: %v", arquivos)
	}
	teste, _ := os.ReadFile(filepath.Join(destino, "arquivos", "src", "test", "java", "AffineCipherTest.java"))
	if strings.Contains(string(teste), "@Disabled") {
		t.Error("os @Disabled saem, como na suíte do Aider")
	}
	if len(tarefa.Conversao) < 2 || !strings.Contains(strings.Join(tarefa.Conversao, " "), "pom.xml") ||
		!strings.Contains(strings.Join(tarefa.Conversao, " "), "@Disabled") {
		t.Errorf("a conversão fica registrada: %v", tarefa.Conversao)
	}
}

package suite

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func exercicio(t *testing.T, raiz, linguagem, nome, buildGradle string) {
	t.Helper()
	base := linguagem + "/exercises/practice/" + nome + "/"
	switch linguagem {
	case "go":
		escrever(t, raiz, base+".meta/config.json", `{"files":{"solution":["a.go"],"test":["a_test.go"],"invalidator":["go.mod"]}}`)
		escrever(t, raiz, base+"a.go", "package a\n")
		escrever(t, raiz, base+"a_test.go", "package a\n")
		escrever(t, raiz, base+"go.mod", "module a\n\ngo 1.18\n")
	case "python":
		escrever(t, raiz, base+".meta/config.json", `{"files":{"solution":["a.py"],"test":["a_test.py"]}}`)
		escrever(t, raiz, base+"a.py", "")
		escrever(t, raiz, base+"a_test.py", "")
	case "java":
		escrever(t, raiz, base+".meta/config.json", `{"files":{"solution":["src/main/java/A.java"],"test":["src/test/java/ATest.java"],"invalidator":["build.gradle"]}}`)
		escrever(t, raiz, base+"build.gradle", buildGradle)
		escrever(t, raiz, base+"src/main/java/A.java", "class A {}\n")
		escrever(t, raiz, base+"src/test/java/ATest.java", "class ATest {}\n")
	}
	escrever(t, raiz, base+".docs/instructions.md", "# "+nome+"\n")
}

func TestMontarSorteiaPorLinguagemFiltraJavaEJuntaOGrupoB(t *testing.T) {
	raiz := t.TempDir()
	for _, nome := range []string{"g1", "g2", "g3", "g4"} {
		exercicio(t, raiz, "go", nome, "")
	}
	for _, nome := range []string{"p1", "p2", "p3"} {
		exercicio(t, raiz, "python", nome, "")
	}
	padrao := "dependencies {\n    testImplementation \"org.assertj:assertj-core:3.25.1\"\n}\n"
	exercicio(t, raiz, "java", "j1", padrao)
	exercicio(t, raiz, "java", "j2", padrao)
	exercicio(t, raiz, "java", "j3", padrao)
	exercicio(t, raiz, "java", "com-json", "dependencies {\n    implementation \"org.json:json:20190722\"\n}\n")

	destino := t.TempDir()
	escrever(t, destino, "risco-ci/tarefa.json", `{"id":"risco-ci","grupo":"risco","linguagem":"go","origem":"sintética","licenca":"MIT (aPRova)","esperado":{"nivel":"alto","motivo":"configuracao_ci"}}`)

	suite, err := Montar(raiz, "7e0611e77b54", 20261114, 2, destino)
	if err != nil {
		t.Fatal(err)
	}

	if suite.Semente != 20261114 || suite.CommitDoPolyglot != "7e0611e77b54" || suite.PorLinguagem != 2 {
		t.Errorf("semente, commit e quantidade ficam registrados: %+v", suite)
	}
	if !strings.Contains(strings.Join(suite.Criterios, " "), "dependência de produção") ||
		!strings.Contains(strings.Join(suite.Criterios, " "), "java-com-json") {
		t.Errorf("o critério Java e quem ele tirou ficam registrados antes do sorteio: %v", suite.Criterios)
	}

	var ids []string
	porLinguagem := map[string]int{}
	for _, tarefa := range suite.Tarefas {
		ids = append(ids, tarefa.ID)
		if tarefa.Grupo == "correcao" {
			porLinguagem[tarefa.Linguagem]++
		}
	}
	if porLinguagem["go"] != 2 || porLinguagem["python"] != 2 || porLinguagem["java"] != 2 || slices.Contains(ids, "java-com-json") {
		t.Errorf("2 por linguagem, nunca o Java inelegível: %v", ids)
	}
	if !slices.Contains(ids, "risco-ci") || !slices.IsSorted(ids) {
		t.Errorf("o grupo B entra e as tarefas saem ordenadas: %v", ids)
	}

	bruto, err := os.ReadFile(filepath.Join(destino, "suite.json"))
	if err != nil {
		t.Fatal(err)
	}
	var gravada Suite
	if err := json.Unmarshal(bruto, &gravada); err != nil || len(gravada.Tarefas) != 7 {
		t.Errorf("o suite.json gravado tem as 7 tarefas: %v, %d", err, len(gravada.Tarefas))
	}

	repetida, _ := Montar(raiz, "7e0611e77b54", 20261114, 2, t.TempDir())
	if !slices.EqualFunc(repetida.Tarefas[:6], suite.Tarefas[:6], func(a, b Tarefa) bool { return a.ID == b.ID }) {
		t.Error("montar de novo com a mesma semente sorteia as mesmas tarefas")
	}
}

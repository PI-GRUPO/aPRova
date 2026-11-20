package integracao

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestConstrucaoQueFalhaDevolveASaidaDoPassoQueQuebrou(t *testing.T) {
	exigirDocker(t)
	diretorio, err := filepath.Abs(filepath.Join("testdata", "imagem_que_falha"))
	if err != nil {
		t.Fatal(err)
	}

	saida, err := construirImagem(t.Context(), filepath.Join(diretorio, "Dockerfile"), "aprova-construcao-que-falha:teste-de-integracao", diretorio)

	if err == nil {
		t.Fatal("o Dockerfile de teste falha de propósito, mas a construção terminou sem erro")
	}
	if !strings.Contains(string(saida), "saida-do-build-42") {
		t.Errorf("a saída do RUN que falhou precisa aparecer para diagnosticar a construção, obtive:\n%s", saida)
	}
}

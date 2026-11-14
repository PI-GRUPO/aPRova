package trabalho

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

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

func arquivosDaTarefa(t *testing.T) string {
	t.Helper()
	origem := t.TempDir()
	escrever(t, origem, "go.mod", "module a\n\ngo 1.18\n")
	escrever(t, origem, "a.go", "package a\n\nfunc Soma(x, y int) int { panic(\"implemente\") }\n")
	escrever(t, origem, "a_test.go", "package a\n\nimport \"testing\"\n\nfunc TestSoma(t *testing.T) {}\n")
	escrever(t, origem, "cases_test.go", "package a\n")
	return origem
}

func preparar(t *testing.T) (Area, string) {
	t.Helper()
	origem := arquivosDaTarefa(t)
	area, err := Preparar(origem, filepath.Join(t.TempDir(), "go-a"))
	if err != nil {
		t.Fatalf("preparar: %v", err)
	}
	return area, origem
}

func TestPrepararCopiaOsArquivosEGravaOCommitBase(t *testing.T) {
	area, _ := preparar(t)

	for _, nome := range []string{"go.mod", "a.go", "a_test.go", "cases_test.go"} {
		if _, err := os.Stat(filepath.Join(area.Diretorio, nome)); err != nil {
			t.Errorf("%s precisa estar na área de trabalho: %v", nome, err)
		}
	}
	mudanca, err := area.Diff(t.Context())
	if err != nil || !mudanca.Vazia() {
		t.Errorf("logo depois de preparar, o diff é vazio: %+v, %v", mudanca, err)
	}
	if len(area.CommitBase) != 40 {
		t.Errorf("o commit-base fica registrado: %q", area.CommitBase)
	}
}

func TestDiffTrazAlteradosCriadosEApagadosSemOsCaches(t *testing.T) {
	area, _ := preparar(t)
	escrever(t, area.Diretorio, "a.go", "package a\n\nfunc Soma(x, y int) int { return x + y }\n")
	escrever(t, area.Diretorio, "ajuda.go", "package a\n")
	if err := os.Remove(filepath.Join(area.Diretorio, "cases_test.go")); err != nil {
		t.Fatal(err)
	}
	for _, cache := range []string{".gocache/ab/x", ".gomodcache/m/x", "__pycache__/a.pyc", "pacote/__pycache__/b.pyc",
		".pytest_cache/v", ".m2/repo/x.jar", "target/classes/A.class", ".claude/settings.local.json"} {
		escrever(t, area.Diretorio, cache, "cache")
	}

	mudanca, err := area.Diff(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(mudanca.Caminhos(), []string{"a.go", "ajuda.go", "cases_test.go"}) {
		t.Errorf("só o que o agente mudou, sem caches: %v", mudanca.Caminhos())
	}
	if !slices.Equal(mudanca.Removidos(), []string{"cases_test.go"}) {
		t.Errorf("o arquivo apagado aparece como removido: %+v", mudanca.Arquivos)
	}
	if !strings.Contains(string(mudanca.Patch), "+func Soma(x, y int) int { return x + y }") ||
		strings.Contains(string(mudanca.Patch), ".gocache") {
		t.Errorf("o patch tem a mudança e não tem cache:\n%s", mudanca.Patch)
	}
}

func TestTestesAdulteradosComparaComOsOriginaisDaSuite(t *testing.T) {
	area, origem := preparar(t)
	protegidos := []string{"a_test.go", "cases_test.go"}

	intocados, err := area.TestesAdulterados(origem, protegidos)
	if err != nil || len(intocados) != 0 {
		t.Fatalf("sem mudança nos testes, nada adulterado: %v, %v", intocados, err)
	}

	escrever(t, area.Diretorio, "a_test.go", "package a\n")
	if err := os.Remove(filepath.Join(area.Diretorio, "cases_test.go")); err != nil {
		t.Fatal(err)
	}

	adulterados, err := area.TestesAdulterados(origem, protegidos)
	if err != nil || !slices.Equal(adulterados, []string{"a_test.go", "cases_test.go"}) {
		t.Errorf("teste alterado e teste apagado são adulteração: %v, %v", adulterados, err)
	}
}

func TestGitIgnoraAConfiguracaoEOsHooksDoUsuario(t *testing.T) {
	configuracao := filepath.Join(t.TempDir(), "gitconfig")
	gancho := t.TempDir()
	escrever(t, gancho, "pre-commit", "#!/bin/sh\nexit 1\n")
	if err := os.Chmod(filepath.Join(gancho, "pre-commit"), 0o700); err != nil { //nolint:gosec // G302: o hook de teste precisa ser executável.
		t.Fatal(err)
	}
	escrever(t, filepath.Dir(configuracao), "gitconfig", "[commit]\n\tgpgsign = true\n[core]\n\thooksPath = "+gancho+"\n")
	t.Setenv("GIT_CONFIG_GLOBAL", configuracao)

	if _, err := Preparar(arquivosDaTarefa(t), filepath.Join(t.TempDir(), "go-a")); err != nil {
		t.Errorf("assinatura e hooks do usuário não podem interferir no commit-base: %v", err)
	}
}

func TestPrepararRecusaDestinoQueJaExiste(t *testing.T) {
	destino := t.TempDir()

	if _, err := Preparar(arquivosDaTarefa(t), destino); err == nil {
		t.Error("a área de trabalho é sempre nova; um destino existente pode ter sobras de outra tarefa")
	}
}

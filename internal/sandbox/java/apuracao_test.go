package java

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApurarRelatorio(t *testing.T) {
	casos := []struct {
		nome       string
		xml        string
		passaram   int
		falharam   int
		esperaErro bool
	}{
		{nome: "todos passaram", xml: `<testsuite tests="3" failures="0" errors="0" skipped="0"></testsuite>`, passaram: 3},
		{nome: "teste falhou", xml: `<testsuite tests="3" failures="1" errors="0" skipped="0"></testsuite>`, passaram: 2, falharam: 1},
		{nome: "teste com erro", xml: `<testsuite tests="3" failures="0" errors="1" skipped="0"></testsuite>`, passaram: 2, falharam: 1},
		{nome: "teste ignorado", xml: `<testsuite tests="3" failures="0" errors="0" skipped="1"></testsuite>`, passaram: 2},
		{
			nome: "várias suítes",
			xml: `<testsuites>
				<testsuite tests="2" failures="0" errors="0" skipped="0"></testsuite>
				<testsuite tests="2" failures="1" errors="0" skipped="0"></testsuite>
			</testsuites>`,
			passaram: 3, falharam: 1,
		},
		{nome: "relatório vazio", xml: "", esperaErro: true},
		{nome: "contagem inconsistente", xml: `<testsuite tests="1" failures="2" errors="0" skipped="0"></testsuite>`, esperaErro: true},
		{nome: "contagem negativa", xml: `<testsuite tests="-1" failures="0" errors="0" skipped="0"></testsuite>`, esperaErro: true},
		{nome: "raiz desconhecida", xml: `<resultado tests="1"></resultado>`, esperaErro: true},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			apurada, err := apurarRelatorio([]byte(caso.xml))

			if caso.esperaErro {
				if err == nil {
					t.Fatalf("esperava erro, obtive %+v", apurada)
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if apurada.passaram != caso.passaram || apurada.falharam != caso.falharam {
				t.Errorf("passaram=%d falharam=%d, esperava %d e %d", apurada.passaram, apurada.falharam, caso.passaram, caso.falharam)
			}
		})
	}
}

func TestApurarRelatoriosSomaOsModulos(t *testing.T) {
	projeto := t.TempDir()
	gravarRelatorio(t, projeto, "", `<testsuite tests="3" failures="1" errors="0" skipped="0"></testsuite>`)
	gravarRelatorio(t, projeto, "modulo", `<testsuite tests="2" failures="0" errors="0" skipped="0"></testsuite>`)

	apurada, err := apurarRelatorios(projeto)

	if err != nil || apurada.passaram != 4 || apurada.falharam != 1 {
		t.Errorf("os relatórios de todos os módulos entram: %+v, %v", apurada, err)
	}
}

func TestApurarRelatoriosSemNenhumRelatorioEhZeroSemErro(t *testing.T) {
	apurada, err := apurarRelatorios(t.TempDir())

	if err != nil || apurada.passaram != 0 || apurada.falharam != 0 {
		t.Errorf("projeto sem testes não tem relatório e não é erro: %+v, %v", apurada, err)
	}
}

func TestApurarRelatoriosIgnoraOQueNaoEstaEmTargetSurefireReports(t *testing.T) {
	projeto := t.TempDir()
	solto := filepath.Join(projeto, "surefire-reports")
	if err := os.MkdirAll(solto, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(solto, "TEST-solto.xml"), []byte(`<testsuite tests="9" failures="0" errors="0" skipped="0"/>`), 0o600); err != nil {
		t.Fatal(err)
	}

	apurada, err := apurarRelatorios(projeto)

	if err != nil || apurada.passaram != 0 {
		t.Errorf("só target/surefire-reports conta: %+v, %v", apurada, err)
	}
}

func TestApurarRelatoriosRecusaLinkSimbolico(t *testing.T) {
	projeto := t.TempDir()
	alvo := filepath.Join(projeto, "externo.xml")
	if err := os.WriteFile(alvo, []byte(`<testsuite tests="1" failures="0" errors="0" skipped="0"/>`), 0o600); err != nil {
		t.Fatal(err)
	}
	relatorios := filepath.Join(projeto, "target", "surefire-reports")
	if err := os.MkdirAll(relatorios, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(alvo, filepath.Join(relatorios, "TEST-link.xml")); err != nil {
		t.Fatal(err)
	}

	if _, err := apurarRelatorios(projeto); err == nil {
		t.Error("um relatório que é link simbólico não pode ser lido")
	}
}

func gravarRelatorio(t *testing.T, projeto, modulo, conteudo string) {
	t.Helper()

	relatorios := filepath.Join(projeto, modulo, "target", "surefire-reports")
	if err := os.MkdirAll(relatorios, 0o700); err != nil {
		t.Fatalf("criar diretório de relatórios: %v", err)
	}
	if err := os.WriteFile(filepath.Join(relatorios, "TEST-exemplo.xml"), []byte(conteudo), 0o600); err != nil {
		t.Fatalf("gravar relatório: %v", err)
	}
}

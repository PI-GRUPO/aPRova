package java

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApurarRelatorio(t *testing.T) {
	testes := []struct {
		nome       string
		xml        string
		passaram   int
		falharam   int
		esperaErro bool
	}{
		{
			nome:     "todos passaram",
			xml:      `<testsuite tests="3" failures="0" errors="0" skipped="0"></testsuite>`,
			passaram: 3,
		},
		{
			nome:     "teste falhou",
			xml:      `<testsuite tests="3" failures="1" errors="0" skipped="0"></testsuite>`,
			passaram: 2,
			falharam: 1,
		},
		{
			nome:     "teste com erro",
			xml:      `<testsuite tests="3" failures="0" errors="1" skipped="0"></testsuite>`,
			passaram: 2,
			falharam: 1,
		},
		{
			nome:     "teste ignorado",
			xml:      `<testsuite tests="3" failures="0" errors="0" skipped="1"></testsuite>`,
			passaram: 2,
		},
		{
			nome: "multiplas suites",
			xml: `<testsuites>
				<testsuite tests="2" failures="0" errors="0" skipped="0"></testsuite>
				<testsuite tests="2" failures="1" errors="0" skipped="0"></testsuite>
			</testsuites>`,
			passaram: 3,
			falharam: 1,
		},
		{
			nome:       "relatorio vazio",
			xml:        "",
			esperaErro: true,
		},
		{
			nome:       "contagem inconsistente",
			xml:        `<testsuite tests="1" failures="2" errors="0" skipped="0"></testsuite>`,
			esperaErro: true,
		},
	}

	for _, teste := range testes {
		t.Run(teste.nome, func(t *testing.T) {
			resultado, err := apurarRelatorio([]byte(teste.xml))

			if teste.esperaErro {
				if err == nil {
					t.Fatal("esperava erro, mas não recebeu")
				}
				return
			}

			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}

			if resultado.passaram != teste.passaram {
				t.Errorf("passaram: esperado %d, recebido %d",
					teste.passaram, resultado.passaram)
			}

			if resultado.falharam != teste.falharam {
				t.Errorf("falharam: esperado %d, recebido %d",
					teste.falharam, resultado.falharam)
			}
		})
	}
}

func TestApurarRelatoriosLeituraSegura(t *testing.T) {
	t.Run("lê relatório normal", func(t *testing.T) {
		diretorio := t.TempDir()
		reports := filepath.Join(diretorio, "target", "surefire-reports")

		if err := os.MkdirAll(reports, 0755); err != nil {
			t.Fatal(err)
		}

		conteudo := `<testsuite tests="3" failures="1" errors="0" skipped="0"></testsuite>`
		arquivo := filepath.Join(reports, "TEST-exemplo.xml")

		if err := os.WriteFile(arquivo, []byte(conteudo), 0644); err != nil {
			t.Fatal(err)
		}

		resultado, err := apurarRelatorios(diretorio)
		if err != nil {
			t.Fatalf("erro inesperado: %v", err)
		}

		if resultado.passaram != 2 || resultado.falharam != 1 {
			t.Errorf(
				"resultado incorreto: passaram=%d, falharam=%d",
				resultado.passaram,
				resultado.falharam,
			)
		}
	})

	t.Run("rejeita link simbolico", func(t *testing.T) {
		diretorio := t.TempDir()
		reports := filepath.Join(diretorio, "target", "surefire-reports")

		if err := os.MkdirAll(reports, 0755); err != nil {
			t.Fatal(err)
		}

		alvo := filepath.Join(diretorio, "relatorio-externo.xml")
		conteudo := `<testsuite tests="1" failures="0" errors="0" skipped="0"></testsuite>`

		if err := os.WriteFile(alvo, []byte(conteudo), 0644); err != nil {
			t.Fatal(err)
		}

		link := filepath.Join(reports, "TEST-link.xml")
		if err := os.Symlink(alvo, link); err != nil {
			t.Fatal(err)
		}

		_, err := apurarRelatorios(diretorio)
		if err == nil {
			t.Fatal("esperava erro ao tentar ler link simbólico")
		}
	})
}

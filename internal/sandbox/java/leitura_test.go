package java

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLerArquivoComSeguranca(t *testing.T) {
	t.Run("lê arquivo comum", func(t *testing.T) {
		diretorio := t.TempDir()
		caminho := filepath.Join(diretorio, "TEST-exemplo.xml")

		conteudoEsperado := []byte(
			`<testsuite tests="2" failures="0" errors="0"></testsuite>`,
		)

		if err := os.WriteFile(caminho, conteudoEsperado, 0600); err != nil {
			t.Fatalf("não foi possível criar relatório: %v", err)
		}

		conteudo, err := lerArquivoComSeguranca(caminho)
		if err != nil {
			t.Fatalf("esperava leitura sem erro, obtive: %v", err)
		}

		if string(conteudo) != string(conteudoEsperado) {
			t.Fatalf(
				"conteúdo inesperado: obtive %q, esperava %q",
				conteudo,
				conteudoEsperado,
			)
		}
	})

	t.Run("recusa arquivo inexistente", func(t *testing.T) {
		caminho := filepath.Join(t.TempDir(), "inexistente.xml")

		_, err := lerArquivoComSeguranca(caminho)
		if err == nil {
			t.Fatal("esperava erro para arquivo inexistente")
		}
	})

	t.Run("recusa link simbolico", func(t *testing.T) {
		diretorio := t.TempDir()

		arquivoOriginal := filepath.Join(diretorio, "original.xml")
		link := filepath.Join(diretorio, "TEST-link.xml")

		if err := os.WriteFile(arquivoOriginal, []byte("<testsuite/>"), 0600); err != nil {
			t.Fatalf("não foi possível criar arquivo original: %v", err)
		}

		if err := os.Symlink(arquivoOriginal, link); err != nil {
			t.Fatalf("não foi possível criar link simbólico: %v", err)
		}

		_, err := lerArquivoComSeguranca(link)
		if err == nil {
			t.Fatal("esperava erro ao tentar ler link simbólico")
		}
	})

	t.Run("recusa diretorio", func(t *testing.T) {
		diretorio := t.TempDir()

		_, err := lerArquivoComSeguranca(diretorio)
		if !errors.Is(err, errRelatorioNaoEArquivoComum) {
			t.Fatalf(
				"esperava erro de arquivo não comum, obtive: %v",
				err,
			)
		}
	})

	t.Run("recusa relatorio maior que o limite", func(t *testing.T) {
		diretorio := t.TempDir()
		caminho := filepath.Join(diretorio, "TEST-grande.xml")

		conteudo := strings.Repeat("a", limiteDoRelatorio+1)

		if err := os.WriteFile(caminho, []byte(conteudo), 0600); err != nil {
			t.Fatalf("não foi possível criar relatório: %v", err)
		}

		_, err := lerArquivoComSeguranca(caminho)
		if err == nil {
			t.Fatal("esperava erro para relatório acima do limite")
		}

		if !strings.Contains(err.Error(), "maior que o limite") {
			t.Fatalf("mensagem de erro inesperada: %v", err)
		}
	})

	t.Run("aceita relatorio exatamente no limite", func(t *testing.T) {
		diretorio := t.TempDir()
		caminho := filepath.Join(diretorio, "TEST-limite.xml")

		conteudoEsperado := strings.Repeat("a", limiteDoRelatorio)

		if err := os.WriteFile(caminho, []byte(conteudoEsperado), 0600); err != nil {
			t.Fatalf("não foi possível criar relatório: %v", err)
		}

		conteudo, err := lerArquivoComSeguranca(caminho)
		if err != nil {
			t.Fatalf("esperava leitura sem erro, obtive: %v", err)
		}

		if len(conteudo) != limiteDoRelatorio {
			t.Fatalf(
				"tamanho inesperado: obtive %d, esperava %d",
				len(conteudo),
				limiteDoRelatorio,
			)
		}
	})
}

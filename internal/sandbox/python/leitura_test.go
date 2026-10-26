package python

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestLerArquivoComSegurancaLeArquivoComum(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "relatorio.json")
	if err := os.WriteFile(caminho, []byte(`{"exitcode": 0}`), 0o600); err != nil {
		t.Fatalf("preparar relatório: %v", err)
	}

	conteudo, err := lerArquivoComSeguranca(caminho)
	if err != nil {
		t.Fatalf("arquivo comum precisa ser lido: %v", err)
	}
	if string(conteudo) != `{"exitcode": 0}` {
		t.Errorf("conteúdo errado: %q", conteudo)
	}
}

func TestLerArquivoComSegurancaRecusaOQueNaoEArquivoComum(t *testing.T) {
	diretorio := t.TempDir()
	alvo := filepath.Join(diretorio, "alvo.json")
	if err := os.WriteFile(alvo, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("preparar alvo: %v", err)
	}

	link := filepath.Join(diretorio, "link.json")
	if err := os.Symlink(alvo, link); err != nil {
		t.Fatalf("criar link simbólico: %v", err)
	}

	fifo := filepath.Join(diretorio, "fifo.json")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatalf("criar fifo: %v", err)
	}

	casos := map[string]string{
		"ausente":        filepath.Join(diretorio, "nao-existe.json"),
		"link simbólico": link,
		"fifo":           fifo,
		"diretório":      diretorio,
	}

	for nome, caminho := range casos {
		t.Run(nome, func(t *testing.T) {
			resposta := make(chan error, 1)
			go func() {
				_, err := lerArquivoComSeguranca(caminho)
				resposta <- err
			}()

			select {
			case err := <-resposta:
				if err == nil {
					t.Errorf("%s não pode ser lido como relatório", nome)
				}
			case <-time.After(5 * time.Second):
				t.Fatalf("a leitura de %s travou: um PR poderia prender o executor", nome)
			}
		})
	}
}

func TestLerArquivoComSegurancaRecusaArquivoGrandeDemais(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "enorme.json")
	arquivo, err := os.Create(caminho)
	if err != nil {
		t.Fatalf("criar arquivo: %v", err)
	}
	if err := arquivo.Truncate(limiteDoRelatorio + 1); err != nil {
		t.Fatalf("esticar arquivo: %v", err)
	}
	if err := arquivo.Close(); err != nil {
		t.Fatalf("fechar arquivo: %v", err)
	}

	if _, err := lerArquivoComSeguranca(caminho); err == nil {
		t.Error("um relatório acima do limite precisa ser recusado")
	}
}

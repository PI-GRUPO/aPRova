package python

import (
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
)

const limiteDoRelatorio = 8 << 20

var errRelatorioNaoEArquivoComum = errors.New("python: relatório não é um arquivo comum")

// lerArquivoComSeguranca lê o relatório que o pytest gravou. O arquivo é
// escrito pelo uid do código do PR, então é tratado como entrada hostil: link
// simbólico é recusado (O_NOFOLLOW), fifo não prende a leitura (O_NONBLOCK) e o
// tamanho é limitado.
func lerArquivoComSeguranca(caminho string) ([]byte, error) {
	//nolint:gosec // G304: o caminho é montado pelo executor, nunca vem do PR.
	arquivo, err := os.OpenFile(caminho, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("abrir relatório: %w", err)
	}
	defer func() { _ = arquivo.Close() }()

	informacao, err := arquivo.Stat()
	if err != nil {
		return nil, fmt.Errorf("consultar relatório: %w", err)
	}
	if !informacao.Mode().IsRegular() {
		return nil, errRelatorioNaoEArquivoComum
	}

	conteudo, err := io.ReadAll(io.LimitReader(arquivo, limiteDoRelatorio+1))
	if err != nil {
		return nil, fmt.Errorf("ler relatório: %w", err)
	}
	if len(conteudo) > limiteDoRelatorio {
		return nil, fmt.Errorf("relatório maior que o limite de %d bytes", limiteDoRelatorio)
	}
	return conteudo, nil
}

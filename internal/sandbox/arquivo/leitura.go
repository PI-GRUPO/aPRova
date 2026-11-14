// Package arquivo lê arquivos que o código do PR pôde escrever ou plantar,
// tratando-os como entrada hostil.
package arquivo

import (
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
)

const limiteDeLeitura = 8 << 20

// ErrNaoEArquivoComum indica que o caminho existe, mas não é um arquivo comum.
var ErrNaoEArquivoComum = errors.New("arquivo: não é um arquivo comum")

// LerDoPR devolve o conteúdo de um arquivo comum de até 8 MiB. Recusa link
// simbólico, fifo, diretório e arquivo maior que o limite, sem travar.
func LerDoPR(caminho string) ([]byte, error) {
	//nolint:gosec // G304: o caminho é montado pelo executor; O_NOFOLLOW impede que um link do PR desvie a leitura.
	aberto, err := os.OpenFile(caminho, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("abrir: %w", err)
	}
	defer func() { _ = aberto.Close() }()

	informacao, err := aberto.Stat()
	if err != nil {
		return nil, fmt.Errorf("consultar: %w", err)
	}
	if !informacao.Mode().IsRegular() {
		return nil, ErrNaoEArquivoComum
	}

	conteudo, err := io.ReadAll(io.LimitReader(aberto, limiteDeLeitura+1))
	if err != nil {
		return nil, fmt.Errorf("ler: %w", err)
	}
	if len(conteudo) > limiteDeLeitura {
		return nil, fmt.Errorf("maior que o limite de %d bytes", limiteDeLeitura)
	}
	return conteudo, nil
}

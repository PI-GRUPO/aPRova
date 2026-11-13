package python

import (
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
)

const limiteDoRelatorio = 8 << 20

var errNaoEArquivoComum = errors.New("python: não é um arquivo comum")

func lerArquivoComSeguranca(caminho string) ([]byte, error) {
	//nolint:gosec // G304: o caminho é montado pelo executor; O_NOFOLLOW impede que um link do PR desvie a leitura.
	arquivo, err := os.OpenFile(caminho, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("abrir: %w", err)
	}
	defer func() { _ = arquivo.Close() }()

	informacao, err := arquivo.Stat()
	if err != nil {
		return nil, fmt.Errorf("consultar: %w", err)
	}
	if !informacao.Mode().IsRegular() {
		return nil, errNaoEArquivoComum
	}

	conteudo, err := io.ReadAll(io.LimitReader(arquivo, limiteDoRelatorio+1))
	if err != nil {
		return nil, fmt.Errorf("ler: %w", err)
	}
	if len(conteudo) > limiteDoRelatorio {
		return nil, fmt.Errorf("maior que o limite de %d bytes", limiteDoRelatorio)
	}
	return conteudo, nil
}

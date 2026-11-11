package java

import (
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
)

const limiteDoRelatorio = 8 << 20

var errRelatorioNaoEArquivoComum = errors.New(
	"java: relatório não é um arquivo comum",
)

func lerArquivoComSeguranca(caminho string) ([]byte, error) {
	arquivo, err := os.OpenFile(
		caminho,
		os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK,
		0,
	)
	if err != nil {
		return nil, fmt.Errorf("abrir relatório: %w", err)
	}
	defer func() {
		_ = arquivo.Close()
	}()

	informacao, err := arquivo.Stat()
	if err != nil {
		return nil, fmt.Errorf("consultar relatório: %w", err)
	}

	if !informacao.Mode().IsRegular() {
		return nil, errRelatorioNaoEArquivoComum
	}

	conteudo, err := io.ReadAll(
		io.LimitReader(arquivo, limiteDoRelatorio+1),
	)
	if err != nil {
		return nil, fmt.Errorf("ler relatório: %w", err)
	}

	if len(conteudo) > limiteDoRelatorio {
		return nil, fmt.Errorf(
			"relatório maior que o limite de %d bytes",
			limiteDoRelatorio,
		)
	}

	return conteudo, nil
}

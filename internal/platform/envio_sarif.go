package platform

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
)

// SituacaoSARIF é o andamento do processamento de um SARIF enviado, como a
// API do GitHub o nomeia.
type SituacaoSARIF string

// Andamentos do processamento de um SARIF.
const (
	SARIFPendente   SituacaoSARIF = "pending"
	SARIFProcessado SituacaoSARIF = "complete"
	SARIFRecusado   SituacaoSARIF = "failed"
)

// EnvioSARIF é uma análise a enviar ao code scanning: o commit analisado, o
// ref completo, como refs/pull/12/head, o nome da ferramenta, igual ao
// declarado no documento, e o documento SARIF sem compactar.
type EnvioSARIF struct {
	CommitSHA  string
	Ref        string
	Ferramenta string
	Documento  []byte
}

// ProcessamentoSARIF é o que o GitHub fez com um SARIF enviado. Erros só vem
// preenchido quando a situação é SARIFRecusado.
type ProcessamentoSARIF struct {
	Situacao SituacaoSARIF
	Erros    []string
}

// EnviarSARIF envia o documento ao code scanning, em gzip e base64, e devolve
// o id do envio. O id só diz que o GitHub aceitou o arquivo; o resultado vem
// de ProcessamentoDoSARIF. Falta de permissão e code scanning indisponível
// devolvem ErrSemPermissao.
func (c *Cliente) EnviarSARIF(ctx context.Context, installationID int64, dono, repositorio string, envio EnvioSARIF) (string, error) {
	compactado, err := compactarEmBase64(envio.Documento)
	if err != nil {
		return "", err
	}

	var resposta struct {
		ID string `json:"id"`
	}
	err = c.chamar(ctx, chamada{
		instalacaoID: installationID,
		metodo:       http.MethodPost,
		caminho:      caminhoDosEnviosSARIF(dono, repositorio),
		corpo: map[string]string{
			"commit_sha": envio.CommitSHA, "ref": envio.Ref, "sarif": compactado, "tool_name": envio.Ferramenta,
		},
		resposta: &resposta,
	})
	if err != nil {
		return "", err
	}
	if resposta.ID == "" {
		return "", fmt.Errorf("%w: SARIF aceito sem id", ErrRespostaInesperada)
	}
	return resposta.ID, nil
}

// ProcessamentoDoSARIF consulta o que o GitHub fez com o envio de id
// informado.
func (c *Cliente) ProcessamentoDoSARIF(ctx context.Context, installationID int64, dono, repositorio, id string) (ProcessamentoSARIF, error) {
	var resposta struct {
		Situacao SituacaoSARIF `json:"processing_status"`
		Erros    []string      `json:"errors"`
	}
	err := c.chamar(ctx, chamada{
		instalacaoID: installationID,
		metodo:       http.MethodGet,
		caminho:      caminhoDosEnviosSARIF(dono, repositorio) + "/" + url.PathEscape(id),
		resposta:     &resposta,
	})
	if err != nil {
		return ProcessamentoSARIF{}, err
	}
	return ProcessamentoSARIF{Situacao: resposta.Situacao, Erros: resposta.Erros}, nil
}

// ArquivosDoPullRequest lê só os arquivos alterados do pull request, com os
// patches, sem o diff unificado.
func (c *Cliente) ArquivosDoPullRequest(ctx context.Context, installationID int64, dono, repositorio string, numero int) ([]ArquivoAlterado, error) {
	return c.arquivosAlterados(ctx, installationID, dono, repositorio, numero)
}

func caminhoDosEnviosSARIF(dono, repositorio string) string {
	return caminhoDoRepositorio(dono, repositorio) + "/code-scanning/sarifs"
}

func compactarEmBase64(documento []byte) (string, error) {
	var compactado bytes.Buffer
	escritor := gzip.NewWriter(&compactado)
	if _, err := escritor.Write(documento); err != nil {
		return "", fmt.Errorf("compactar o SARIF: %w", err)
	}
	if err := escritor.Close(); err != nil {
		return "", fmt.Errorf("compactar o SARIF: %w", err)
	}
	return base64.StdEncoding.EncodeToString(compactado.Bytes()), nil
}

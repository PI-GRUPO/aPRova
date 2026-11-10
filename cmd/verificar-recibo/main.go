package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Azulpasta/aPRova/internal/receipt"
)

const (
	codigoValido   = 0
	codigoInvalido = 1
	codigoDeUso    = 2

	entradaPadrao = "-"
	formatoDaHora = "2006-01-02 15:04:05 UTC"
)

const instrucoesDeUso = `uso: verificar-recibo -chave <chave pública em base64> [arquivo.json]

Verifica um recibo do aPRova só com a chave pública, sem banco e sem rede.
O recibo é o JSON devolvido por GET /receipts/{id}; sem arquivo, é lido da
entrada padrão. A chave pública vem de GET /receipts/keys.

Sai com 0 quando o recibo é válido, 1 quando é inválido e 2 quando a chave ou
a entrada não puderam ser lidas.
`

var errUso = errors.New("uso incorreto")

func main() {
	os.Exit(executar(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func executar(argumentos []string, entrada io.Reader, saida, erros io.Writer) int {
	chaveiro, documento, err := lerEntradas(argumentos, entrada, erros)
	if err != nil {
		escrever(erros, "verificar-recibo: %v\n", err)
		return codigoDeUso
	}

	conteudo, resultado, err := receipt.VerificarExportado(chaveiro, documento)
	if err != nil {
		escrever(erros, "verificar-recibo: %v\n", err)
		return codigoDeUso
	}
	if !resultado.Valido {
		escrever(saida, "INVÁLIDO: a verificação de %s falhou\n%s\n", resultado.Falha, resultado.Detalhe)
		return codigoInvalido
	}

	escrever(saida, "%s", relatorioDoValido(conteudo, documentoAssinado(documento)))
	return codigoValido
}

func lerEntradas(argumentos []string, entrada io.Reader, erros io.Writer) (*receipt.Chaveiro, []byte, error) {
	opcoes := flag.NewFlagSet("verificar-recibo", flag.ContinueOnError)
	opcoes.SetOutput(erros)
	opcoes.Usage = func() { escrever(erros, "%s", instrucoesDeUso) }
	chaveEmBase64 := opcoes.String("chave", "", "chave pública Ed25519 em base64")
	if err := opcoes.Parse(argumentos); err != nil {
		return nil, nil, errUso
	}
	if opcoes.NArg() > 1 {
		return nil, nil, fmt.Errorf("%w: informe no máximo um arquivo", errUso)
	}

	chave, err := lerChavePublica(*chaveEmBase64)
	if err != nil {
		return nil, nil, err
	}
	documento, err := lerDocumento(opcoes.Arg(0), entrada)
	if err != nil {
		return nil, nil, err
	}
	return receipt.NovoChaveiro(chave), documento, nil
}

func lerChavePublica(emBase64 string) (ed25519.PublicKey, error) {
	if emBase64 == "" {
		return nil, fmt.Errorf("%w: informe a chave pública com -chave", errUso)
	}

	chave, err := base64.StdEncoding.DecodeString(strings.TrimSpace(emBase64))
	if err != nil || len(chave) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%w: -chave precisa ser a chave pública Ed25519 de %d bytes em base64", errUso, ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(chave), nil
}

func lerDocumento(caminho string, entrada io.Reader) ([]byte, error) {
	if caminho == "" || caminho == entradaPadrao {
		documento, err := io.ReadAll(entrada)
		if err != nil {
			return nil, fmt.Errorf("ler o recibo da entrada padrão: %w", err)
		}
		return documento, nil
	}

	documento, err := os.ReadFile(caminho) //nolint:gosec // G304: o caminho é o arquivo que quem verifica escolheu ler.
	if err != nil {
		return nil, fmt.Errorf("ler o recibo: %w", err)
	}
	return documento, nil
}

type assinado struct {
	hash    string
	idChave string
}

func documentoAssinado(documento []byte) assinado {
	var exportado receipt.Exportado
	if err := json.Unmarshal(documento, &exportado); err != nil {
		return assinado{}
	}
	return assinado{hash: exportado.HashSHA256, idChave: exportado.KeyID}
}

func relatorioDoValido(conteudo receipt.Conteudo, origem assinado) string {
	var relatorio strings.Builder
	linha := func(rotulo, formato string, valores ...any) {
		fmt.Fprintf(&relatorio, "%-13s %s\n", rotulo, fmt.Sprintf(formato, valores...))
	}

	relatorio.WriteString("VÁLIDO: a assinatura confere com o conteúdo e com a chave informada\n")
	linha("recibo:", "%s", conteudo.ID)
	linha("key_id:", "%s", origem.idChave)
	linha("hash sha256:", "%s", origem.hash)
	linha("análise:", "%s, PR #%d, commit %s", conteudo.Repositorio, conteudo.NumeroPR, conteudo.SHAHead)
	linha("diff sha256:", "%s", conteudo.DiffSHA256)
	linha("sandbox:", "%d testes passaram, %d falharam, falha %s, tentativa %d",
		conteudo.Sandbox.TestesPassaram, conteudo.Sandbox.TestesFalharam, conteudo.Sandbox.TipoFalha, conteudo.Sandbox.Tentativa)
	linha("nível:", "%s (determinístico: %s)", conteudo.Nivel, conteudo.NivelDeterministico)
	for _, motivo := range conteudo.Motivos {
		linha("motivo:", "%s: %s", motivo.ID, strings.Join(motivo.Evidencias, "; "))
	}
	linha("emitido em:", "%s", conteudo.EmitidoEm.UTC().Format(formatoDaHora))
	if conteudo.Anterior == nil {
		linha("anterior:", "nenhum, é o primeiro recibo da cadeia")
	} else {
		linha("anterior:", "%s (sha256 %s)", conteudo.Anterior.ID, conteudo.Anterior.Hash)
	}
	return relatorio.String()
}

func escrever(destino io.Writer, formato string, valores ...any) {
	_, _ = fmt.Fprintf(destino, formato, valores...)
}

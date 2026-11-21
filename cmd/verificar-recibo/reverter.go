package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/PI-GRUPO/aPRova/internal/receipt"
	"github.com/PI-GRUPO/aPRova/internal/reversao"
)

const subcomandoReverter = "reverter"

const instrucoesDaReversao = `uso: verificar-recibo reverter -chave <chave pública em base64> -url-publica <URL do aPRova>
       [-anterior recibo-anterior.json] [-merge-sha <sha> -head-final <sha> -tipo merge|squash|rebase -commits <n>]
       [arquivo.json]

Confere o recibo e devolve o comando git e o texto do PR que revertem o que
entrou na main por causa daquela análise. Não executa nada, não usa banco nem
rede. Os dados do merge vêm do PR fechado no GitHub; sem -merge-sha, o PR é
tratado como não mergeado. Sem -anterior, o elo com o recibo anterior não é
conferido e a saída avisa.

Sai com 0 quando há instrução, 1 quando não há e 2 quando a entrada não pôde
ser lida.
`

type pedidoDaLinhaDeComando struct {
	chaveiro        *receipt.Chaveiro
	documento       []byte
	caminhoAnterior string
	urlPublica      string
	merge           reversao.Merge
}

func executarReversao(argumentos []string, entrada io.Reader, saida, erros io.Writer) int {
	pedido, err := lerPedidoDeReversao(argumentos, entrada, erros)
	if err != nil {
		escrever(erros, "verificar-recibo reverter: %v\n", err)
		return codigoDeUso
	}

	conteudo, resultado, err := receipt.VerificarExportado(pedido.chaveiro, pedido.documento)
	if err != nil {
		escrever(erros, "verificar-recibo reverter: %v\n", err)
		return codigoDeUso
	}
	eloConferido := true
	if resultado.Valido {
		resultado, eloConferido, err = conferirElo(conteudo, pedido.caminhoAnterior)
		if err != nil {
			escrever(erros, "verificar-recibo reverter: %v\n", err)
			return codigoDeUso
		}
	}

	instrucao := reversao.Montar(reversao.Pedido{
		Conteudo: conteudo, Verificacao: resultado, Merge: &pedido.merge,
		URLDoRecibo: pedido.urlPublica + "/receipts/" + conteudo.ID + "/verify",
	})
	escrever(saida, "%s", relatorioDaReversao(conteudo, resultado, eloConferido, documentoAssinado(pedido.documento), instrucao))
	if instrucao.Recusa != "" {
		return codigoInvalido
	}
	return codigoValido
}

func lerPedidoDeReversao(argumentos []string, entrada io.Reader, erros io.Writer) (pedidoDaLinhaDeComando, error) {
	opcoes := flag.NewFlagSet("verificar-recibo reverter", flag.ContinueOnError)
	opcoes.SetOutput(erros)
	opcoes.Usage = func() { escrever(erros, "%s", instrucoesDaReversao) }
	chaveEmBase64 := opcoes.String("chave", "", "chave pública Ed25519 em base64")
	urlPublica := opcoes.String("url-publica", "", "endereço público do aPRova, para o link de verificação")
	caminhoAnterior := opcoes.String("anterior", "", "JSON do recibo anterior da cadeia")
	shaDoMerge := opcoes.String("merge-sha", "", "merge_commit_sha do PR fechado")
	headFinal := opcoes.String("head-final", "", "head do PR no fechamento")
	tipo := opcoes.String("tipo", "", "merge, squash ou rebase")
	commits := opcoes.Int("commits", 0, "quantos commits o PR tinha")
	if err := opcoes.Parse(argumentos); err != nil {
		return pedidoDaLinhaDeComando{}, errUso
	}

	switch {
	case opcoes.NArg() > 1:
		return pedidoDaLinhaDeComando{}, fmt.Errorf("%w: informe no máximo um arquivo", errUso)
	case *urlPublica == "":
		return pedidoDaLinhaDeComando{}, fmt.Errorf("%w: informe o endereço do aPRova com -url-publica", errUso)
	case *commits < 0:
		return pedidoDaLinhaDeComando{}, fmt.Errorf("%w: -commits não pode ser negativo", errUso)
	}

	chave, err := lerChavePublica(*chaveEmBase64)
	if err != nil {
		return pedidoDaLinhaDeComando{}, err
	}
	documento, err := lerDocumento(opcoes.Arg(0), entrada)
	if err != nil {
		return pedidoDaLinhaDeComando{}, err
	}
	return pedidoDaLinhaDeComando{
		chaveiro: receipt.NovoChaveiro(chave), documento: documento, caminhoAnterior: *caminhoAnterior,
		urlPublica: strings.TrimRight(*urlPublica, "/"),
		merge: reversao.Merge{
			Mergeado: *shaDoMerge != "", SHADoMerge: *shaDoMerge, SHAHeadFinal: *headFinal,
			Tipo: reversao.Tipo(*tipo), Commits: *commits,
		},
	}, nil
}

func conferirElo(conteudo receipt.Conteudo, caminhoAnterior string) (receipt.Resultado, bool, error) {
	if conteudo.Anterior == nil {
		return eloConferidoComo(receipt.VerificarElo(conteudo, nil)), true, nil
	}
	if caminhoAnterior == "" {
		return receipt.Resultado{Valido: true}, false, nil
	}

	anterior, err := lerReciboAnterior(caminhoAnterior)
	if err != nil {
		return receipt.Resultado{}, false, err
	}
	return eloConferidoComo(receipt.VerificarElo(conteudo, &anterior)), true, nil
}

func eloConferidoComo(quebra string) receipt.Resultado {
	if quebra != "" {
		return receipt.Resultado{Falha: receipt.FalhaElo, Detalhe: quebra}
	}
	return receipt.Resultado{Valido: true}
}

func lerReciboAnterior(caminho string) (receipt.Anterior, error) {
	documento, err := lerDocumento(caminho, nil)
	if err != nil {
		return receipt.Anterior{}, fmt.Errorf("recibo anterior: %w", err)
	}

	var exportado receipt.Exportado
	if err := json.Unmarshal(documento, &exportado); err != nil {
		return receipt.Anterior{}, fmt.Errorf("o recibo anterior não é o JSON de um recibo: %w", err)
	}
	conteudo, err := base64.StdEncoding.DecodeString(exportado.PayloadBase64)
	if err != nil {
		return receipt.Anterior{}, fmt.Errorf("payload_base64 do recibo anterior não está em base64: %w", err)
	}
	return receipt.Anterior{ID: exportado.ID, Conteudo: conteudo}, nil
}

func relatorioDaReversao(conteudo receipt.Conteudo, resultado receipt.Resultado, eloConferido bool, origem assinado, instrucao reversao.Instrucao) string {
	var relatorio strings.Builder
	if !eloConferido {
		fmt.Fprintf(&relatorio, "ATENÇÃO: ELO NÃO CONFERIDO. O recibo cita o anterior %s; passe -anterior com o JSON dele para conferir a cadeia.\n\n",
			conteudo.Anterior.ID)
	}
	if resultado.Valido {
		relatorio.WriteString(relatorioDoValido(conteudo, origem))
	} else {
		fmt.Fprintf(&relatorio, "INVÁLIDO: a verificação de %s falhou\n%s\n", resultado.Falha, resultado.Detalhe)
	}

	if instrucao.Recusa != "" {
		fmt.Fprintf(&relatorio, "\nSEM REVERSÃO (%s): %s\n", instrucao.Recusa, instrucao.Explicacao)
		return relatorio.String()
	}
	fmt.Fprintf(&relatorio, "\nREVERSÃO por %s do commit %s\n\ncomando:\n%s\n\ntítulo do PR:\n%s\n\ncorpo do PR:\n%s",
		instrucao.Tipo, instrucao.CommitNaMain, instrucao.Comando, instrucao.Titulo, instrucao.Corpo)
	return relatorio.String()
}

package sarif

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/PI-GRUPO/aPRova/internal/platform"
)

const (
	situacaoRemovido           = "removed"
	separadorDeNotaDoArquivo   = ": "
	prefixoDeLinhaAdicionada   = "+"
	prefixoDeLinhaRemovida     = "-"
	prefixoDeAvisoDoDiff       = `\`
	prefixoDeCabecalhoDeTrecho = "@@"
)

var cabecalhoDeTrecho = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)`)

type arquivosDoPR map[string]platform.ArquivoAlterado

func indexar(arquivos []platform.ArquivoAlterado) arquivosDoPR {
	indice := make(arquivosDoPR, len(arquivos))
	for _, arquivo := range arquivos {
		indice[arquivo.Nome] = arquivo
	}
	return indice
}

func (a arquivosDoPR) removido(nome string) bool {
	return a[nome].Situacao == situacaoRemovido
}

func (a arquivosDoPR) linha(nome string, preferidas []string) int {
	return primeiraLinhaAdicionada(a[nome].Patch, preferidas)
}

func arquivoDaEvidencia(idDoMotivo, evidencia string) string {
	if idDoMotivo != motivoDependenciaNova {
		return evidencia
	}
	nome, _, _ := strings.Cut(evidencia, separadorDeNotaDoArquivo)
	return nome
}

func primeiraLinhaAdicionada(patch string, preferidas []string) int {
	primeira, linhaNova := 0, 0
	for linha := range strings.SplitSeq(patch, "\n") {
		switch {
		case strings.HasPrefix(linha, prefixoDeCabecalhoDeTrecho):
			linhaNova = inicioDoTrecho(linha, linhaNova)
		case strings.HasPrefix(linha, prefixoDeLinhaRemovida), strings.HasPrefix(linha, prefixoDeAvisoDoDiff):
		case strings.HasPrefix(linha, prefixoDeLinhaAdicionada):
			if citaAlguma(linha, preferidas) {
				return linhaNova
			}
			if primeira == 0 {
				primeira = linhaNova
			}
			linhaNova++
		default:
			linhaNova++
		}
	}
	return max(primeira, 1)
}

func inicioDoTrecho(cabecalho string, atual int) int {
	partes := cabecalhoDeTrecho.FindStringSubmatch(cabecalho)
	if partes == nil {
		return atual
	}
	inicio, err := strconv.Atoi(partes[1])
	if err != nil {
		return atual
	}
	return inicio
}

func citaAlguma(linha string, preferidas []string) bool {
	return slices.ContainsFunc(preferidas, func(preferida string) bool {
		return preferida != "" && strings.Contains(linha, preferida)
	})
}

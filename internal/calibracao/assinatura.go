package calibracao

import (
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/platform"
)

const (
	situacaoAdicionado = "added"
	situacaoRemovido   = "removed"
)

// Assinatura monta o texto que representa o pull request no embedding: os ids
// dos motivos do gate, os caminhos alterados reduzidos a diretório e extensão e
// a quantidade de arquivos por situação. Não leva diff, conteúdo nem nome de
// arquivo, e a mesma entrada dá o mesmo texto em qualquer ordem. O formato está
// em docs/calibracao.md.
func Assinatura(motivos []domain.MotivoDeRisco, arquivos []platform.ArquivoAlterado) string {
	adicionados, alterados, removidos := contarPorSituacao(arquivos)
	return strings.Join([]string{
		"motivos: " + strings.Join(idsDosMotivos(motivos), ", "),
		"caminhos: " + strings.Join(caminhosNormalizados(arquivos), ", "),
		fmt.Sprintf("adicionados: %d", adicionados),
		fmt.Sprintf("alterados: %d", alterados),
		fmt.Sprintf("removidos: %d", removidos),
	}, "\n")
}

func idsDosMotivos(motivos []domain.MotivoDeRisco) []string {
	ids := []string{}
	for _, motivo := range motivos {
		if motivo.ID != MotivoDaCalibracao {
			ids = append(ids, motivo.ID)
		}
	}
	slices.Sort(ids)
	return slices.Compact(ids)
}

func caminhosNormalizados(arquivos []platform.ArquivoAlterado) []string {
	caminhos := []string{}
	for _, arquivo := range arquivos {
		caminhos = append(caminhos, normalizarCaminho(arquivo.Nome))
	}
	slices.Sort(caminhos)
	return slices.Compact(caminhos)
}

func normalizarCaminho(nome string) string {
	padrao := "*" + path.Ext(nome)
	if diretorio := path.Dir(nome); diretorio != "." {
		return diretorio + "/" + padrao
	}
	return padrao
}

func contarPorSituacao(arquivos []platform.ArquivoAlterado) (adicionados, alterados, removidos int) {
	for _, arquivo := range arquivos {
		switch arquivo.Situacao {
		case situacaoAdicionado:
			adicionados++
		case situacaoRemovido:
			removidos++
		default:
			alterados++
		}
	}
	return adicionados, alterados, removidos
}

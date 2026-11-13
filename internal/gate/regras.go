package gate

import (
	"fmt"
	"slices"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/platform"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

const (
	motivoFalhaSandbox      = "falha_sandbox"
	motivoMuitosArquivos    = "muitos_arquivos"
	motivoConfiguracaoCI    = "configuracao_ci"
	motivoDependenciaNova   = "dependencia_nova"
	motivoSegredoVersionado = "segredo_versionado"
	motivoAreaSensivel      = "area_sensivel"
	motivoCodigoSemTeste    = "codigo_sem_teste"

	situacaoRemovido = "removed"
)

func novoMotivo(id string, nivel domain.NivelRisco, descricao string, evidencias []string) domain.MotivoDeRisco {
	ordenadas := slices.Clone(evidencias)
	slices.Sort(ordenadas)
	return domain.MotivoDeRisco{ID: id, Nivel: nivel, Descricao: descricao, Evidencias: slices.Compact(ordenadas)}
}

func (g *Gate) aplicarRegras(entrada Entrada, dependencias analiseDeManifestos) []domain.MotivoDeRisco {
	motivos := []domain.MotivoDeRisco{}
	acrescentar := func(id string, nivel domain.NivelRisco, descricao string, evidencias []string) {
		if len(evidencias) > 0 {
			motivos = append(motivos, novoMotivo(id, nivel, descricao, evidencias))
		}
	}

	acrescentar(motivoFalhaSandbox, domain.RiscoAlto, "os testes ou a compilação falharam no sandbox", evidenciaDoSandbox(entrada.Sandbox))
	acrescentar(motivoMuitosArquivos, domain.RiscoAlto,
		fmt.Sprintf("o PR altera %d arquivos, acima do limite de %d", len(entrada.Arquivos), g.limites.MaxArquivos), g.arquivosAlemDoLimite(entrada.Arquivos))
	acrescentar(motivoConfiguracaoCI, domain.RiscoAlto, "o PR altera configuração de build ou de CI", arquivosQue(entrada.Arquivos, tocaConfiguracaoCI))
	acrescentar(motivoDependenciaNova, domain.RiscoAlto, "o PR acrescenta dependência nova, que precisa de revisão humana", dependencias.evidencias)
	acrescentar(motivoSegredoVersionado, domain.RiscoAlto, "o PR versiona um arquivo que costuma guardar segredo", arquivosQue(entrada.Arquivos, versionaSegredo))
	acrescentar(motivoAreaSensivel, domain.RiscoAlto, "o PR altera um caminho sensível", arquivosQue(entrada.Arquivos, g.tocaCaminhoSensivel))
	acrescentar(motivoCodigoSemTeste, domain.RiscoMedio, "o PR altera código sem alterar nenhum teste", codigoSemTeste(entrada.Arquivos))

	return motivos
}

func evidenciaDoSandbox(resultado sandbox.ResultadoSandbox) []string {
	switch resultado.TipoFalha {
	case sandbox.FalhaTeste:
		total := resultado.TestesPassaram + resultado.TestesFalharam
		return []string{fmt.Sprintf("%d de %d testes falharam", resultado.TestesFalharam, total)}
	case sandbox.FalhaCompilacao:
		return []string{"o projeto não compilou"}
	default:
		return nil
	}
}

func (g *Gate) arquivosAlemDoLimite(arquivos []platform.ArquivoAlterado) []string {
	if len(arquivos) <= g.limites.MaxArquivos {
		return nil
	}
	return arquivosQue(arquivos, func(platform.ArquivoAlterado) bool { return true })
}

func arquivosQue(arquivos []platform.ArquivoAlterado, condicao func(platform.ArquivoAlterado) bool) []string {
	var nomes []string
	for _, arquivo := range arquivos {
		if condicao(arquivo) {
			nomes = append(nomes, arquivo.Nome)
		}
	}
	return nomes
}

func codigoSemTeste(arquivos []platform.ArquivoAlterado) []string {
	testeAlterado := slices.ContainsFunc(arquivos, func(arquivo platform.ArquivoAlterado) bool {
		return arquivo.Situacao != situacaoRemovido && ehTeste(arquivo.Nome)
	})
	if testeAlterado {
		return nil
	}
	return arquivosQue(arquivos, func(arquivo platform.ArquivoAlterado) bool { return ehCodigoDeProducao(arquivo.Nome) })
}

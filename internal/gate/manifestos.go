package gate

import (
	"cmp"
	"path"
	"slices"
	"strings"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/platform"
)

const (
	ecossistemaGo    = "Go"
	ecossistemaPyPI  = "PyPI"
	ecossistemaMaven = "Maven"

	versaoNaoEspecificada = "não especificada"

	evidenciaDiffIlegivel  = ": não foi possível ler o diff"
	evidenciaLinhaIlegivel = ": não foi possível interpretar uma linha adicionada"
)

var situacoesQuePodemVirSemPatch = []string{"renamed", "copied", "changed"}

type linhaDoPatch struct {
	texto          string
	alterada       bool
	inicioDeTrecho bool
}

type declaracao struct {
	nome     string
	versao   string
	alterada bool
	ilegivel bool
}

type leitorDeManifesto struct {
	ecossistema string
	ler         func([]linhaDoPatch) []declaracao
}

type analiseDeManifestos struct {
	novas      []domain.DependenciaNova
	evidencias []string
}

func leitorDe(caminho string) (leitorDeManifesto, bool) {
	nome := path.Base(caminho)
	switch {
	case nome == "go.mod":
		return leitorDeManifesto{ecossistemaGo, lerGoMod}, true
	case casaComAlgum([]string{"requirements*.txt"}, nome):
		return leitorDeManifesto{ecossistemaPyPI, lerRequirements}, true
	case nome == "pyproject.toml":
		return leitorDeManifesto{ecossistemaPyPI, lerPyproject}, true
	case nome == "pom.xml":
		return leitorDeManifesto{ecossistemaMaven, lerPom}, true
	default:
		return leitorDeManifesto{}, false
	}
}

func analisarManifestos(arquivos []platform.ArquivoAlterado) analiseDeManifestos {
	analise := analiseDeManifestos{novas: []domain.DependenciaNova{}}
	for _, arquivo := range arquivos {
		leitor, ehManifesto := leitorDe(arquivo.Nome)
		if ehManifesto && arquivo.Situacao != situacaoRemovido {
			analise.acrescentar(arquivo, leitor)
		}
	}

	slices.SortFunc(analise.novas, func(a, b domain.DependenciaNova) int {
		return cmp.Or(strings.Compare(a.Ecossistema, b.Ecossistema), strings.Compare(a.Nome, b.Nome), strings.Compare(a.Versao, b.Versao))
	})
	analise.novas = slices.Compact(analise.novas)
	return analise
}

func (a *analiseDeManifestos) acrescentar(arquivo platform.ArquivoAlterado, leitor leitorDeManifesto) {
	if patchOmitido(arquivo) {
		a.evidencias = append(a.evidencias, arquivo.Nome+evidenciaDiffIlegivel)
		return
	}

	nova, antiga := visoesDoPatch(arquivo.Patch)
	removidas := map[string]bool{}
	for _, declarada := range leitor.ler(antiga) {
		if declarada.alterada && !declarada.ilegivel {
			removidas[declarada.nome] = true
		}
	}

	for _, declarada := range leitor.ler(nova) {
		switch {
		case !declarada.alterada:
		case declarada.ilegivel:
			a.evidencias = append(a.evidencias, arquivo.Nome+evidenciaLinhaIlegivel)
		case !removidas[declarada.nome]:
			a.novas = append(a.novas, domain.DependenciaNova{Ecossistema: leitor.ecossistema, Nome: declarada.nome, Versao: cmp.Or(declarada.versao, versaoNaoEspecificada)})
			a.evidencias = append(a.evidencias, arquivo.Nome)
		}
	}
}

func patchOmitido(arquivo platform.ArquivoAlterado) bool {
	if arquivo.Patch != "" {
		return false
	}
	semMudancaDeConteudo := arquivo.Adicoes == 0 && arquivo.Remocoes == 0
	return !semMudancaDeConteudo || !slices.Contains(situacoesQuePodemVirSemPatch, arquivo.Situacao)
}

func visoesDoPatch(patch string) (nova, antiga []linhaDoPatch) {
	for _, linha := range strings.Split(patch, "\n") {
		switch {
		case strings.HasPrefix(linha, "@@"):
			inicio := linhaDoPatch{texto: contextoDoTrecho(linha), inicioDeTrecho: true}
			nova, antiga = append(nova, inicio), append(antiga, inicio)
		case strings.HasPrefix(linha, "+"):
			nova = append(nova, linhaDoPatch{texto: linha[1:], alterada: true})
		case strings.HasPrefix(linha, "-"):
			antiga = append(antiga, linhaDoPatch{texto: linha[1:], alterada: true})
		case strings.HasPrefix(linha, " "):
			contexto := linhaDoPatch{texto: linha[1:]}
			nova, antiga = append(nova, contexto), append(antiga, contexto)
		}
	}
	return nova, antiga
}

func contextoDoTrecho(cabecalho string) string {
	partes := strings.SplitN(cabecalho, "@@", 3)
	if len(partes) < 3 {
		return ""
	}
	return strings.TrimSpace(partes[2])
}

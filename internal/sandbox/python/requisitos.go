package python

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

const (
	arquivoDoPyproject = "pyproject.toml"
	prefixoDoHash      = "--hash="

	motivoEditavel     = "instalação editável exige construir o projeto"
	motivoForaDoIndice = "dependência por URL, VCS ou caminho exige buscar ou construir fora do índice"
	motivoAninhado     = "arquivo aninhado não é lido; liste os requisitos neste arquivo"
	motivoIndice       = "índice de pacotes próprio não é aceito; só o PyPI"
	motivoOpcao        = "opção do pip não é aceita"
	motivoNaoRequisito = "linha que não é um requisito"
	motivoDinamicas    = "dependências dinâmicas exigem executar o build do projeto"
)

var (
	nomeDeArquivoDeRequisitos = regexp.MustCompile(`^requirements([-_.][A-Za-z0-9_.-]+)?\.txt$`)
	formatoDoRequisito        = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9._-]*[A-Za-z0-9])?(\s*\[[A-Za-z0-9._,\s-]*\])?\s*((===|==|>=|<=|~=|!=|>|<)\s*[^;]*)?(\s*;.*)?$`)
	comentario                = regexp.MustCompile(`(^|\s)#.*$`)

	gruposDeTeste = []string{"dev", "test", "tests", "testing"}

	motivoDaOpcao = map[string]string{
		"-e": motivoEditavel, "--editable": motivoEditavel,
		"-r": motivoAninhado, "--requirement": motivoAninhado, "-c": motivoAninhado, "--constraint": motivoAninhado,
		"-i": motivoIndice, "--index-url": motivoIndice, "--extra-index-url": motivoIndice,
		"-f": motivoIndice, "--find-links": motivoIndice, "--trusted-host": motivoIndice, "--no-index": motivoIndice,
	}
	prefixosForaDoIndice = []string{".", "/", "~", "git+", "hg+", "svn+", "bzr+"}
)

type dadosDoPyproject struct {
	Dependencias []string            `json:"dependencias"`
	Opcionais    map[string][]string `json:"opcionais"`
	Dinamicas    bool                `json:"dinamicas"`
}

type linhaLogica struct {
	numero int
	texto  string
}

func arquivosDeRequisitos(nomes []string) []string {
	var arquivos []string
	for _, nome := range nomes {
		if nomeDeArquivoDeRequisitos.MatchString(nome) {
			arquivos = append(arquivos, nome)
		}
	}
	slices.Sort(arquivos)
	return arquivos
}

func interpretarArquivoDeRequisitos(arquivo, conteudo string) (especificacoes, recusas []string) {
	for _, linha := range linhasLogicas(conteudo) {
		texto := strings.TrimSpace(comentario.ReplaceAllString(linha.texto, ""))
		if texto == "" {
			continue
		}
		if motivo := motivoDaRecusa(texto); motivo != "" {
			recusas = append(recusas, fmt.Sprintf("%s:%d: %q não é suportado: %s", arquivo, linha.numero, texto, motivo))
			continue
		}
		especificacoes = append(especificacoes, strings.Join(strings.Fields(texto), " "))
	}
	return especificacoes, recusas
}

func requisitosDoPyproject(dados dadosDoPyproject) (especificacoes, recusas []string) {
	if dados.Dinamicas {
		recusas = append(recusas, arquivoDoPyproject+": "+motivoDinamicas)
	}

	candidatas := slices.Clone(dados.Dependencias)
	for _, grupo := range gruposDeTeste {
		candidatas = append(candidatas, dados.Opcionais[grupo]...)
	}
	for _, candidata := range candidatas {
		if motivo := motivoDaRecusa(candidata); motivo != "" {
			recusas = append(recusas, fmt.Sprintf("%s: %q não é suportado: %s", arquivoDoPyproject, candidata, motivo))
			continue
		}
		especificacoes = append(especificacoes, strings.Join(strings.Fields(candidata), " "))
	}
	return especificacoes, recusas
}

func linhasLogicas(conteudo string) []linhaLogica {
	var linhas []linhaLogica
	var atual *linhaLogica
	for indice, fisica := range strings.Split(conteudo, "\n") {
		continua := strings.HasSuffix(strings.TrimRight(fisica, " \t\r"), `\`)
		trecho := strings.TrimSuffix(strings.TrimRight(fisica, " \t\r"), `\`)
		if atual == nil {
			linhas = append(linhas, linhaLogica{numero: indice + 1})
			atual = &linhas[len(linhas)-1]
		}
		atual.texto += " " + trecho
		if !continua {
			atual = nil
		}
	}
	return linhas
}

func motivoDaRecusa(texto string) string {
	campos := strings.Fields(texto)
	if strings.HasPrefix(campos[0], "-") {
		nome, _, _ := strings.Cut(campos[0], "=")
		if motivo, conhecida := motivoDaOpcao[nome]; conhecida {
			return motivo
		}
		return motivoOpcao
	}

	requisito := texto
	if indice := strings.Index(texto, " "+prefixoDoHash); indice >= 0 {
		requisito = texto[:indice]
		for _, campo := range strings.Fields(texto[indice:]) {
			if !strings.HasPrefix(campo, prefixoDoHash) {
				return motivoOpcao
			}
		}
	}

	switch {
	case strings.Contains(requisito, "://") || strings.Contains(requisito, "@") || temAlgumPrefixo(requisito, prefixosForaDoIndice):
		return motivoForaDoIndice
	case !formatoDoRequisito.MatchString(strings.TrimSpace(requisito)):
		return motivoNaoRequisito
	default:
		return ""
	}
}

func temAlgumPrefixo(texto string, prefixos []string) bool {
	for _, prefixo := range prefixos {
		if strings.HasPrefix(texto, prefixo) {
			return true
		}
	}
	return false
}

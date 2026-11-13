package gate

import (
	"regexp"
	"slices"
	"strings"
)

const (
	operadorDeVersaoExata = "=="
	nomePython            = "python"
)

type estadoDaLista int

const (
	listaDesconhecida estadoDaLista = iota
	semLista
	listaDeDependencias
	listaIgnorada
)

var (
	requisitoPEP508    = regexp.MustCompile(`^([A-Za-z0-9](?:[A-Za-z0-9._-]*[A-Za-z0-9])?)\s*(?:\[[^\]]*\])?\s*(.*)$`)
	separadorDeNome    = regexp.MustCompile(`[-_.]+`)
	cabecalhoDeSecao   = regexp.MustCompile(`^\[\[?([^\]]+)\]\]?$`)
	chaveEValor        = regexp.MustCompile(`^"?([A-Za-z0-9._-]+)"?\s*=\s*(.+)$`)
	textoEntreAspas    = regexp.MustCompile(`"([^"]*)"|'([^']*)'`)
	versaoEmTabela     = regexp.MustCompile(`version\s*=\s*"([^"]*)"`)
	secoesDeChaveValor = regexp.MustCompile(`^tool\.poetry\.(dependencies|dev-dependencies|group\..+\.dependencies)$`)

	chavesDeMetadados = []string{
		"name", "version", "description", "readme", "license", "authors", "maintainers", "keywords", "classifiers", "urls",
		"requires-python", "build-backend", "dynamic", "packages", "include", "exclude", "homepage", "repository", "documentation",
	}
)

func lerRequirements(linhas []linhaDoPatch) []declaracao {
	var declaradas []declaracao
	for _, linha := range linhas {
		texto, _, _ := strings.Cut(linha.texto, " #")
		texto = strings.TrimSpace(texto)
		if texto == "" || strings.HasPrefix(texto, "#") || linha.inicioDeTrecho {
			continue
		}

		declarada := declaracao{ilegivel: true}
		if !strings.HasPrefix(texto, "-") && !strings.HasPrefix(texto, "git+") && !strings.Contains(strings.SplitN(texto, "@", 2)[0], "://") {
			declarada = interpretarRequisito(texto)
		}
		declarada.alterada = linha.alterada
		declaradas = append(declaradas, declarada)
	}
	return declaradas
}

func interpretarRequisito(texto string) declaracao {
	partes := requisitoPEP508.FindStringSubmatch(texto)
	if partes == nil {
		return declaracao{ilegivel: true}
	}

	especificacao, _, _ := strings.Cut(partes[2], ";")
	especificacao = strings.TrimSpace(especificacao)
	if exata, ehExata := strings.CutPrefix(especificacao, operadorDeVersaoExata); ehExata && !strings.Contains(exata, ",") {
		especificacao = strings.TrimSpace(exata)
	}
	return declaracao{nome: nomeNormalizado(partes[1]), versao: especificacao}
}

func nomeNormalizado(nome string) string {
	return strings.ToLower(separadorDeNome.ReplaceAllString(nome, "-"))
}

type leituraDoPyproject struct {
	secao      string
	lista      estadoDaLista
	declaradas []declaracao
}

func lerPyproject(linhas []linhaDoPatch) []declaracao {
	leitura := &leituraDoPyproject{}
	for _, linha := range linhas {
		if linha.inicioDeTrecho {
			leitura.secao, leitura.lista = "", listaDesconhecida
		}
		leitura.interpretar(linha)
	}
	return leitura.declaradas
}

func (l *leituraDoPyproject) interpretar(linha linhaDoPatch) {
	texto := strings.TrimSpace(linha.texto)
	if cabecalho := cabecalhoDeSecao.FindStringSubmatch(texto); cabecalho != nil {
		l.secao, l.lista = cabecalho[1], semLista
		return
	}
	if texto == "]" {
		l.lista = semLista
		return
	}
	if partes := chaveEValor.FindStringSubmatch(texto); partes != nil {
		l.interpretarChave(partes[1], partes[2], linha.alterada)
		return
	}
	if strings.HasPrefix(texto, `"`) || strings.HasPrefix(texto, `'`) {
		l.interpretarItemDeLista(texto, linha.alterada)
	}
}

func (l *leituraDoPyproject) interpretarChave(chave, valor string, alterada bool) {
	if strings.HasPrefix(valor, "[") {
		l.interpretarAberturaDeLista(chave, valor, alterada)
		return
	}
	if chave == nomePython || !l.chaveDeclaraDependencia(chave) {
		return
	}
	l.declaradas = append(l.declaradas, declaracao{nome: nomeNormalizado(chave), versao: versaoDoValor(valor), alterada: alterada})
}

func (l *leituraDoPyproject) chaveDeclaraDependencia(chave string) bool {
	if secoesDeChaveValor.MatchString(l.secao) {
		return true
	}
	return l.secao == "" && !slices.Contains(chavesDeMetadados, chave)
}

func versaoDoValor(valor string) string {
	if tabela := versaoEmTabela.FindStringSubmatch(valor); tabela != nil {
		return tabela[1]
	}
	if strings.HasPrefix(valor, "{") {
		return ""
	}
	return strings.Trim(valor, `"' `)
}

func (l *leituraDoPyproject) interpretarAberturaDeLista(chave, valor string, alterada bool) {
	deDependencias := l.chaveListaDependencias(chave)
	if deDependencias {
		l.acrescentarRequisitos(valor, alterada)
	}

	switch {
	case strings.Contains(valor, "]"):
		l.lista = semLista
	case deDependencias:
		l.lista = listaDeDependencias
	default:
		l.lista = listaIgnorada
	}
}

func (l *leituraDoPyproject) chaveListaDependencias(chave string) bool {
	switch l.secao {
	case "project.optional-dependencies", "dependency-groups":
		return true
	case "build-system":
		return chave == "requires"
	case "project", "":
		return chave == "dependencies" || (l.secao == "" && chave == "requires")
	default:
		return false
	}
}

func (l *leituraDoPyproject) interpretarItemDeLista(texto string, alterada bool) {
	if l.lista == listaDeDependencias || (l.lista == listaDesconhecida && l.secao == "") {
		l.acrescentarRequisitos(texto, alterada)
	}
}

func (l *leituraDoPyproject) acrescentarRequisitos(texto string, alterada bool) {
	for _, entreAspas := range textoEntreAspas.FindAllStringSubmatch(texto, -1) {
		declarada := interpretarRequisito(entreAspas[1] + entreAspas[2])
		declarada.alterada = alterada
		l.declaradas = append(l.declaradas, declarada)
	}
}

package gate

import (
	"regexp"
	"strings"
)

var versaoDeModulo = regexp.MustCompile(`^v[0-9]`)

func lerGoMod(linhas []linhaDoPatch) []declaracao {
	var declaradas []declaracao
	ignorando := false
	for _, linha := range linhas {
		texto, _, _ := strings.Cut(linha.texto, "//")
		campos := strings.Fields(texto)
		switch {
		case linha.inicioDeTrecho, strings.TrimSpace(texto) == ")":
			ignorando = false
		}
		if len(campos) == 2 && campos[1] == "(" {
			ignorando = campos[0] == "exclude" || campos[0] == "retract"
			continue
		}
		if nome, versao, declarou := moduloDeclarado(campos); declarou && !ignorando {
			declaradas = append(declaradas, declaracao{nome: nome, versao: versao, alterada: linha.alterada})
		}
	}
	return declaradas
}

func moduloDeclarado(campos []string) (nome, versao string, declarou bool) {
	if len(campos) > 0 && (campos[0] == "require" || campos[0] == "replace") {
		campos = campos[1:]
	}
	for indice, campo := range campos {
		if campo == "=>" {
			campos = campos[indice+1:]
			break
		}
	}
	if len(campos) != 2 || !versaoDeModulo.MatchString(campos[1]) {
		return "", "", false
	}
	return campos[0], campos[1], true
}

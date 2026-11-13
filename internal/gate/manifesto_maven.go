package gate

import (
	"regexp"
	"strings"
)

type posicaoNoPom int

const (
	posicaoDesconhecida posicaoNoPom = iota
	foraDeBloco
	dentroDeBloco
)

var (
	campoDoPom = regexp.MustCompile(`<(groupId|artifactId|version)>([^<]*)</(?:groupId|artifactId|version)>`)

	aberturasDeBloco   = []string{"<dependency>", "<plugin>", "<parent>"}
	fechamentosDeBloco = []string{"</dependency>", "</plugin>", "</parent>"}
	saidasDeBloco      = []string{"<project", "<dependencies>", "<plugins>", "<dependencyManagement>", "<build>"}
)

type leituraDoPom struct {
	posicao    posicaoNoPom
	exclusao   bool
	grupo      string
	artefato   string
	versao     string
	alterada   bool
	declaradas []declaracao
}

func lerPom(linhas []linhaDoPatch) []declaracao {
	leitura := &leituraDoPom{}
	for _, linha := range linhas {
		if linha.inicioDeTrecho {
			leitura.encerrarBloco(posicaoDesconhecida)
			continue
		}
		leitura.interpretar(linha)
	}
	leitura.encerrarBloco(posicaoDesconhecida)
	return leitura.declaradas
}

func (l *leituraDoPom) interpretar(linha linhaDoPatch) {
	texto := strings.TrimSpace(linha.texto)
	switch {
	case comecaComAlgum(texto, aberturasDeBloco):
		l.encerrarBloco(dentroDeBloco)
	case comecaComAlgum(texto, fechamentosDeBloco):
		l.encerrarBloco(foraDeBloco)
	case comecaComAlgum(texto, saidasDeBloco):
		l.encerrarBloco(foraDeBloco)
	case strings.HasPrefix(texto, "<exclusion>"):
		l.exclusao = true
	case strings.HasPrefix(texto, "</exclusion>"):
		l.exclusao = false
	default:
		l.lerCampo(texto, linha.alterada)
	}
}

func comecaComAlgum(texto string, prefixos []string) bool {
	for _, prefixo := range prefixos {
		if strings.HasPrefix(texto, prefixo) {
			return true
		}
	}
	return false
}

func (l *leituraDoPom) lerCampo(texto string, alterada bool) {
	campo := campoDoPom.FindStringSubmatch(texto)
	if campo == nil || l.exclusao || l.posicao == foraDeBloco {
		return
	}

	switch campo[1] {
	case "groupId":
		l.grupo, l.alterada = campo[2], l.alterada || alterada
	case "artifactId":
		l.artefato, l.alterada = campo[2], l.alterada || alterada
	default:
		l.versao = campo[2]
	}
}

func (l *leituraDoPom) encerrarBloco(proxima posicaoNoPom) {
	if l.artefato != "" {
		nome := l.artefato
		if l.grupo != "" {
			nome = l.grupo + ":" + l.artefato
		}
		l.declaradas = append(l.declaradas, declaracao{nome: nome, versao: l.versao, alterada: l.alterada})
	}
	l.grupo, l.artefato, l.versao, l.alterada, l.exclusao = "", "", "", false, false
	l.posicao = proxima
}

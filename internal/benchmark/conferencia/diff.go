// Package conferencia confere o diff de uma solução do benchmark antes do
// push: só arquivos da tarefa, nenhum cache e nenhum valor do .env do aPRova.
package conferencia

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"path"
	"strings"
)

const tamanhoMinimoDoSegredo = 8

var (
	// ErrForaDaTarefa indica um caminho do diff fora do diretório da tarefa.
	ErrForaDaTarefa = errors.New("conferencia: arquivo fora do diretório da tarefa")
	// ErrArquivoDeCache indica um arquivo de cache ou de build no diff.
	ErrArquivoDeCache = errors.New("conferencia: arquivo de cache no diff")
	// ErrSegredoNoDiff indica que o diff contém o valor de uma variável do .env.
	ErrSegredoNoDiff = errors.New("conferencia: valor do .env no diff")
	// ErrEnvSemSegredos indica um .env sem nenhum valor a conferir.
	ErrEnvSemSegredos = errors.New("conferencia: o .env não tem nenhum valor a conferir; o arquivo lido é o certo?")

	diretoriosDeCache = []string{".gocache", ".gomodcache", ".pytest_cache", "__pycache__", ".m2", "target", ".claude", ".tmp"}
)

type segredo struct {
	nome    string
	valores [][]byte
}

// Segredos guarda os valores do .env que não podem aparecer no diff, sem
// nunca expô-los em mensagens.
type Segredos struct {
	lista []segredo
}

// SegredosDoEnv lê um .env e guarda cada valor com pelo menos 8 caracteres,
// e também a forma decodificada dos valores em base64.
func SegredosDoEnv(conteudo []byte) Segredos {
	var segredos Segredos
	leitor := bufio.NewScanner(bytes.NewReader(conteudo))
	leitor.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for leitor.Scan() {
		nome, valor, valida := interpretarLinha(leitor.Text())
		if !valida || len(valor) < tamanhoMinimoDoSegredo {
			continue
		}
		valores := [][]byte{[]byte(valor)}
		if decodificado, err := base64.StdEncoding.DecodeString(valor); err == nil && len(decodificado) >= tamanhoMinimoDoSegredo {
			valores = append(valores, decodificado, bytes.TrimSpace(decodificado))
		}
		segredos.lista = append(segredos.lista, segredo{nome: nome, valores: valores})
	}
	return segredos
}

// Nomes devolve os nomes das variáveis conferidas, na ordem do .env.
func (s Segredos) Nomes() []string {
	nomes := make([]string, 0, len(s.lista))
	for _, item := range s.lista {
		nomes = append(nomes, item.nome)
	}
	return nomes
}

// Conferir recusa o diff se algum caminho sai do diretório da tarefa ou é
// cache, ou se o conteúdo contém o valor de alguma variável do .env. O erro
// nomeia a variável, nunca o valor.
func Conferir(caminhos []string, diff []byte, segredos Segredos) error {
	for _, caminho := range caminhos {
		if err := conferirCaminho(caminho); err != nil {
			return err
		}
	}
	for _, item := range segredos.lista {
		for _, valor := range item.valores {
			if bytes.Contains(diff, valor) {
				return fmt.Errorf("%w: o valor de %s aparece no diff", ErrSegredoNoDiff, item.nome)
			}
		}
	}
	return nil
}

func conferirCaminho(caminho string) error {
	limpo := path.Clean(caminho)
	if caminho == "" || limpo == "." || path.IsAbs(caminho) || limpo == ".." || strings.HasPrefix(limpo, "../") {
		return fmt.Errorf("%w: %q", ErrForaDaTarefa, caminho)
	}
	for _, componente := range strings.Split(limpo, "/") {
		for _, cache := range diretoriosDeCache {
			if componente == cache {
				return fmt.Errorf("%w: %q", ErrArquivoDeCache, caminho)
			}
		}
	}
	return nil
}

func interpretarLinha(linha string) (nome, valor string, valida bool) {
	linha = strings.TrimSpace(linha)
	if linha == "" || strings.HasPrefix(linha, "#") {
		return "", "", false
	}
	linha = strings.TrimPrefix(linha, "export ")
	nome, valor, separada := strings.Cut(linha, "=")
	if !separada || strings.TrimSpace(nome) == "" {
		return "", "", false
	}
	valor = strings.TrimSpace(valor)
	if len(valor) >= 2 && (valor[0] == '"' || valor[0] == '\'') && valor[len(valor)-1] == valor[0] {
		valor = valor[1 : len(valor)-1]
	}
	return strings.TrimSpace(nome), valor, true
}

// Autoteste confere, sem expor os valores, que um diff com o valor de cada
// variável seria recusado. Devolve quantas variáveis foram conferidas.
func (s Segredos) Autoteste() (int, error) {
	if len(s.lista) == 0 {
		return 0, ErrEnvSemSegredos
	}
	for _, item := range s.lista {
		diff := append([]byte("+++ b/autoteste.txt\n+"), item.valores[0]...)
		if err := Conferir([]string{"autoteste.txt"}, diff, s); !errors.Is(err, ErrSegredoNoDiff) {
			return 0, fmt.Errorf("conferencia: o valor de %s não seria recusado", item.nome)
		}
	}
	return len(s.lista), nil
}

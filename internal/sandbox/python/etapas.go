package python

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/sandbox"
	"github.com/PI-GRUPO/aPRova/internal/sandbox/processo"
)

const (
	arquivoDaLista        = "requisitos.txt"
	subdiretorioDasWheels = "wheels"
	permissaoDaLista      = 0o644

	scriptDoPyproject = `import json, sys, tomllib
with open(sys.argv[1], "rb") as arquivo:
    dados = tomllib.load(arquivo)
projeto = dados.get("project", {})
dinamico = projeto.get("dynamic", [])
print(json.dumps({
    "dependencias": projeto.get("dependencies", []),
    "opcionais": projeto.get("optional-dependencies", {}),
    "dinamicas": "dependencies" in dinamico or "optional-dependencies" in dinamico,
}))`
)

var (
	sinaisDeRedeDoPip = []string{
		"Temporary failure in name resolution", "Failed to establish a new connection", "NewConnectionError",
		"Max retries exceeded", "ReadTimeoutError", "ConnectTimeoutError", "Read timed out", "timed out",
		"Network is unreachable", "Connection refused", "Connection reset", "SSLError", "ProxyError",
		"502 Server Error", "503 Server Error", "504 Server Error",
	}
	sinaisDeInexistente    = []string{"No matching distribution found", "Could not find a version that satisfies"}
	sinaisDeIncompativel   = []string{"ResolutionImpossible", "conflicting dependencies"}
	sinaisDeHashDivergente = []string{"DO NOT MATCH THE HASHES", "Hashes are required"}
)

// Preparar lê como dado os requirements*.txt da raiz do projeto e a tabela
// [project] do pyproject.toml, recusa o que exigiria construir ou buscar
// código fora do PyPI e baixa para o cache só as wheels das dependências. O
// projeto nunca é construído e nenhum código do PR roda. Devolve nil quando o
// cache está pronto.
func (l *Linguagem) Preparar(ctx context.Context, projeto, cache string) *sandbox.ResultadoSandbox {
	if !temArquivoPython(projeto) {
		falha := infraestrutura("diretório do projeto sem arquivos Python")
		return &falha
	}

	especificacoes, falha := l.coletarRequisitos(ctx, projeto)
	if falha != nil || len(especificacoes) == 0 {
		return falha
	}

	lista := filepath.Join(cache, arquivoDaLista)
	if err := os.WriteFile(lista, []byte(strings.Join(especificacoes, "\n")+"\n"), permissaoDaLista); err != nil {
		falha := infraestrutura(fmt.Sprintf("gravar a lista de requisitos no cache: %v", err))
		return &falha
	}

	download := l.rodar(ctx, projeto, binarioPython, argumentoDeModulo, "pip", "download",
		"--only-binary=:all:", "--dest", filepath.Join(cache, subdiretorioDasWheels), "-r", lista)
	if concluiu(download) {
		return nil
	}
	classificada := classificarDownload(download)
	return &classificada
}

// Testar instala as dependências só a partir das wheels do cache, sem índice,
// e roda o pytest importando o projeto pela raiz, como sempre. prazoFinal é o
// instante em que o resultado já precisa ter sido entregue.
func (l *Linguagem) Testar(ctx context.Context, projeto, cache string, prazoFinal time.Time) sandbox.ResultadoSandbox {
	if !temArquivoPython(projeto) {
		return infraestrutura("diretório do projeto sem arquivos Python")
	}

	python := binarioPython
	lista := filepath.Join(cache, arquivoDaLista)
	if existeArquivoComum(lista) {
		pythonDoVenv, falha := l.criarVenv(ctx, projeto)
		if falha != nil {
			return *falha
		}
		instalacao := l.rodar(ctx, projeto, pythonDoVenv, argumentoDeModulo, "pip", "install",
			"--no-index", "--find-links", filepath.Join(cache, subdiretorioDasWheels), "-r", lista)
		if !concluiu(instalacao) {
			return infraestrutura(descreverInterrupcao("pip install", instalacao), instalacao.Saida)
		}
		python = pythonDoVenv
	}

	timeout := prazoFinal.Sub(l.agora()) - margemParaEntregar
	if timeout < tempoMinimoDosTestes {
		return infraestrutura(fmt.Sprintf("prazo insuficiente para rodar os testes: restam %v", timeout))
	}
	return l.rodarTestes(ctx, projeto, python, timeout)
}

func (l *Linguagem) coletarRequisitos(ctx context.Context, projeto string) ([]string, *sandbox.ResultadoSandbox) {
	entradas, err := os.ReadDir(projeto)
	if err != nil {
		falha := infraestrutura(fmt.Sprintf("listar o projeto: %v", err))
		return nil, &falha
	}
	nomes := make([]string, 0, len(entradas))
	for _, entrada := range entradas {
		nomes = append(nomes, entrada.Name())
	}

	var especificacoes, recusas []string
	for _, arquivo := range arquivosDeRequisitos(nomes) {
		conteudo, err := lerArquivoComSeguranca(filepath.Join(projeto, arquivo))
		if err != nil {
			recusas = append(recusas, fmt.Sprintf("%s não pôde ser lido como arquivo comum: %v", arquivo, err))
			continue
		}
		lidas, recusadas := interpretarArquivoDeRequisitos(arquivo, string(conteudo))
		especificacoes, recusas = append(especificacoes, lidas...), append(recusas, recusadas...)
	}

	if slices.Contains(nomes, arquivoDoPyproject) {
		dados, falha := l.lerPyproject(ctx, projeto)
		if falha != nil {
			return nil, falha
		}
		lidas, recusadas := requisitosDoPyproject(dados)
		especificacoes, recusas = append(especificacoes, lidas...), append(recusas, recusadas...)
	}

	if len(recusas) > 0 {
		falha := compilacaoNoPreparo("requisitos que o preparo não aceita, porque nunca executa código do PR nem busca fora do PyPI:\n  "+strings.Join(recusas, "\n  "), nil)
		return nil, &falha
	}
	return semRepetidas(especificacoes), nil
}

func (l *Linguagem) lerPyproject(ctx context.Context, projeto string) (dadosDoPyproject, *sandbox.ResultadoSandbox) {
	caminho := filepath.Join(projeto, arquivoDoPyproject)
	if !existeArquivoComum(caminho) {
		falha := compilacaoNoPreparo(arquivoDoPyproject+" não é um arquivo comum", nil)
		return dadosDoPyproject{}, &falha
	}

	leitura := l.rodar(ctx, projeto, binarioPython, "-I", "-c", scriptDoPyproject, caminho)
	switch {
	case leitura.Erro != nil || leitura.PrazoEsgotado:
		falha := infraestrutura(descreverInterrupcao("leitura do "+arquivoDoPyproject, leitura), leitura.Saida)
		return dadosDoPyproject{}, &falha
	case leitura.CodigoSaida != 0:
		falha := compilacaoNoPreparo(arquivoDoPyproject+" não pôde ser lido como TOML", leitura.Saida)
		return dadosDoPyproject{}, &falha
	}

	var dados dadosDoPyproject
	if err := json.Unmarshal(leitura.Saida, &dados); err != nil {
		falha := infraestrutura(fmt.Sprintf("a leitura do %s devolveu algo que não é JSON: %v", arquivoDoPyproject, err), leitura.Saida)
		return dadosDoPyproject{}, &falha
	}
	return dados, nil
}

func (l *Linguagem) criarVenv(ctx context.Context, projeto string) (string, *sandbox.ResultadoSandbox) {
	venv := filepath.Join(l.diretorioDeCache, "venv")
	criacao := l.rodar(ctx, projeto, binarioPython, argumentoDeModulo, "venv", "--system-site-packages", venv)
	if !concluiu(criacao) {
		falha := infraestrutura(descreverInterrupcao("python -m venv", criacao), criacao.Saida)
		return "", &falha
	}
	return filepath.Join(venv, "bin", binarioPython), nil
}

func classificarDownload(download processo.Execucao) sandbox.ResultadoSandbox {
	saida := string(download.Saida)
	switch {
	case download.Erro != nil || download.PrazoEsgotado:
		return infraestrutura(descreverInterrupcao("pip download", download), download.Saida)
	case contemAlgum(saida, sinaisDeRedeDoPip):
		return infraestrutura("pip download falhou por rede: o PyPI não respondeu", download.Saida)
	case contemAlgum(saida, sinaisDeHashDivergente):
		return compilacaoNoPreparo("hash divergente: o pacote baixado não é o que o PR declarou", download.Saida)
	case contemAlgum(saida, sinaisDeIncompativel):
		return compilacaoNoPreparo("as versões pedidas são incompatíveis entre si", download.Saida)
	case contemAlgum(saida, sinaisDeInexistente):
		return compilacaoNoPreparo("dependência que não existe no índice ou só existe como sdist, que exigiria executar o setup.py dela", download.Saida)
	default:
		return infraestrutura(descreverInterrupcao("pip download", download), download.Saida)
	}
}

func contemAlgum(saida string, sinais []string) bool {
	for _, sinal := range sinais {
		if strings.Contains(saida, sinal) {
			return true
		}
	}
	return false
}

func semRepetidas(especificacoes []string) []string {
	vistas := map[string]bool{}
	unicas := make([]string, 0, len(especificacoes))
	for _, especificacao := range especificacoes {
		if !vistas[especificacao] {
			vistas[especificacao] = true
			unicas = append(unicas, especificacao)
		}
	}
	return unicas
}

func existeArquivoComum(caminho string) bool {
	informacao, err := os.Lstat(caminho)
	return err == nil && informacao.Mode().IsRegular()
}

func compilacaoNoPreparo(motivo string, saida []byte) sandbox.ResultadoSandbox {
	return sandbox.ResultadoSandbox{SaidaBruta: string(saida) + "aprova: " + motivo + "\n", TipoFalha: sandbox.FalhaCompilacao}
}

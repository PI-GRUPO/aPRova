// Package python roda as dependências e os testes de repositórios Python no
// sandbox e classifica o desfecho no ResultadoSandbox comum às linguagens.
package python

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Azulpasta/aPRova/internal/sandbox"
	"github.com/Azulpasta/aPRova/internal/sandbox/processo"
)

const (
	binarioPython       = "python"
	arquivoDeRequisitos = "requirements.txt"
	argumentoDeModulo   = "-m"

	margemParaEntregar   = 30 * time.Second
	tempoMinimoDosTestes = 30 * time.Second

	caminhoDosExecutaveis = "/usr/local/bin:/usr/bin:/bin"
	nomeDoPytest          = "pytest"

	saidaTodosPassaram  = 0
	saidaTestesFalharam = 1
	saidaInterrompido   = 2
	saidaSemTestes      = 5
)

// Opcoes configura a execução de dependências e testes de um repositório
// Python. DiretorioDeCache recebe o venv, o cache do pip e o relatório.
type Opcoes struct {
	Rodador          processo.Rodador
	DiretorioDeCache string
	Agora            func() time.Time
}

// Linguagem roda dependências e testes de repositórios Python.
type Linguagem struct {
	rodador          processo.Rodador
	diretorioDeCache string
	agora            func() time.Time
	lerRelatorio     func(caminho string) ([]byte, error)
}

// Nova constrói a linguagem Python com as opções informadas. Sem relógio, usa
// time.Now.
func Nova(opcoes Opcoes) *Linguagem {
	agora := opcoes.Agora
	if agora == nil {
		agora = time.Now
	}
	return &Linguagem{
		rodador:          opcoes.Rodador,
		diretorioDeCache: opcoes.DiretorioDeCache,
		agora:            agora,
		lerRelatorio:     lerArquivoComSeguranca,
	}
}

// Executar instala as dependências (se houver requirements.txt na raiz do
// projeto), roda o pytest e classifica o desfecho. prazoFinal é o instante em
// que o resultado já precisa ter sido entregue.
func (l *Linguagem) Executar(ctx context.Context, diretorio string, prazoFinal time.Time) sandbox.ResultadoSandbox {
	python, falha := l.prepararDependencias(ctx, diretorio)
	if falha != nil {
		return *falha
	}

	timeout := prazoFinal.Sub(l.agora()) - margemParaEntregar
	if timeout < tempoMinimoDosTestes {
		return infraestrutura(fmt.Sprintf("prazo insuficiente para rodar os testes: restam %v", timeout))
	}

	return l.rodarTestes(ctx, diretorio, python, timeout)
}

// prepararDependencias devolve o python que roda os testes: o do sistema
// quando não há requirements.txt, o de um venv com as dependências quando há.
func (l *Linguagem) prepararDependencias(ctx context.Context, diretorio string) (string, *sandbox.ResultadoSandbox) {
	if !existeRequisitosNaRaiz(diretorio) {
		return binarioPython, nil
	}

	venv := filepath.Join(l.diretorioDeCache, "venv")
	// O pytest e o plugin de relatório vêm da imagem; o venv os enxerga por
	// --system-site-packages e só acrescenta o que o requirements.txt pede.
	criacao := l.rodar(ctx, diretorio, binarioPython, argumentoDeModulo, "venv", "--system-site-packages", venv)
	if !concluiu(criacao) {
		falha := infraestrutura(descreverInterrupcao("python -m venv", criacao), criacao.Saida)
		return "", &falha
	}

	pythonDoVenv := filepath.Join(venv, "bin", binarioPython)
	instalacao := l.rodar(ctx, diretorio, pythonDoVenv, argumentoDeModulo, "pip", "install", "-r", arquivoDeRequisitos)
	if !concluiu(instalacao) {
		falha := infraestrutura(descreverInterrupcao("pip install", instalacao), instalacao.Saida)
		return "", &falha
	}

	return pythonDoVenv, nil
}

func (l *Linguagem) rodarTestes(ctx context.Context, diretorio, python string, timeout time.Duration) sandbox.ResultadoSandbox {
	relatorio := l.caminhoDoRelatorio()

	teste := l.rodar(ctx, diretorio, python, argumentoDeModulo, nomeDoPytest,
		"-q",
		"-p", "no:cacheprovider",
		"--timeout="+strconv.Itoa(int(timeout.Seconds())),
		"--json-report",
		"--json-report-file="+relatorio,
		"--json-report-omit", "log", "traceback", "streams", "warnings", "keywords",
	)

	return l.classificarTeste(teste, relatorio)
}

func (l *Linguagem) rodar(ctx context.Context, diretorio, binario string, argumentos ...string) processo.Execucao {
	return l.rodador.Rodar(ctx, processo.Comando{
		Binario:    binario,
		Argumentos: argumentos,
		Diretorio:  diretorio,
		Ambiente:   l.ambiente(),
	})
}

func (l *Linguagem) ambiente() []string {
	return []string{
		"PATH=" + filepath.Join(l.diretorioDeCache, "venv", "bin") + ":" + caminhoDosExecutaveis,
		"HOME=" + filepath.Join(l.diretorioDeCache, "home"),
		"PIP_CACHE_DIR=" + filepath.Join(l.diretorioDeCache, "pip"),
		"PIP_DISABLE_PIP_VERSION_CHECK=1",
		"PIP_NO_INPUT=1",
		"PYTHONDONTWRITEBYTECODE=1",
		"PYTHONUNBUFFERED=1",
	}
}

// caminhoDoRelatorio tem sufixo aleatório para que o código do PR não consiga
// plantar um relatório antes da execução.
func (l *Linguagem) caminhoDoRelatorio() string {
	sufixo := make([]byte, 8)
	_, _ = rand.Read(sufixo)
	return filepath.Join(l.diretorioDeCache, "relatorio-"+hex.EncodeToString(sufixo)+".json")
}

// classificarTeste decide o tipo de falha cruzando o código de saída do
// pytest, que vem do processo, com o relatório, que o código do PR consegue
// escrever. Qualquer divergência entre os dois vira infraestrutura: o PR pode
// no máximo reprovar a si mesmo, nunca aprovar com uma falha escondida.
func (l *Linguagem) classificarTeste(teste processo.Execucao, caminhoDoRelatorio string) sandbox.ResultadoSandbox {
	if teste.Erro != nil || teste.PrazoEsgotado {
		return infraestrutura(descreverInterrupcao("pytest", teste), teste.Saida)
	}

	apurada, err := l.apurarRelatorio(caminhoDoRelatorio, teste.CodigoSaida)
	if err != nil {
		return infraestrutura(err.Error(), teste.Saida)
	}

	return resultadoDoPytest(teste, apurada)
}

func (l *Linguagem) apurarRelatorio(caminho string, codigoDeSaida int) (apuracao, error) {
	bruto, err := l.lerRelatorio(caminho)
	if err != nil {
		return apuracao{}, fmt.Errorf("relatório do pytest não pôde ser lido: %w", err)
	}

	apurada, err := apurar(bruto)
	if err != nil {
		return apuracao{}, err
	}
	if apurada.codigoSaida != codigoDeSaida {
		return apuracao{}, fmt.Errorf("o relatório diz código %d, mas o pytest saiu com %d", apurada.codigoSaida, codigoDeSaida)
	}
	return apurada, nil
}

func resultadoDoPytest(teste processo.Execucao, apurada apuracao) sandbox.ResultadoSandbox {
	resultado := sandbox.ResultadoSandbox{
		Compilou:       true,
		TestesPassaram: apurada.passaram,
		TestesFalharam: apurada.falharam,
		SaidaBruta:     string(teste.Saida),
	}

	switch {
	case teste.CodigoSaida == saidaTodosPassaram && apurada.falharam == 0:
		resultado.TipoFalha = sandbox.FalhaNenhuma
	case teste.CodigoSaida == saidaTestesFalharam && apurada.falharam > 0:
		resultado.TipoFalha = sandbox.FalhaTeste
	case teste.CodigoSaida == saidaSemTestes && apurada.passaram+apurada.falharam == 0:
		resultado.TipoFalha = sandbox.FalhaNenhuma
	case teste.CodigoSaida == saidaInterrompido && apurada.coletaFalhou:
		return sandbox.ResultadoSandbox{Compilou: false, SaidaBruta: string(teste.Saida), TipoFalha: sandbox.FalhaCompilacao}
	default:
		return infraestrutura(fmt.Sprintf("pytest saiu com código %d, que o relatório não explica", teste.CodigoSaida), teste.Saida)
	}

	return resultado
}

func concluiu(execucao processo.Execucao) bool {
	return execucao.Erro == nil && !execucao.PrazoEsgotado && execucao.CodigoSaida == 0
}

func descreverInterrupcao(etapa string, execucao processo.Execucao) string {
	switch {
	case execucao.Erro != nil:
		return fmt.Sprintf("%s não pôde ser iniciado: %v", etapa, execucao.Erro)
	case execucao.PrazoEsgotado:
		return fmt.Sprintf("%s excedeu o prazo do executor", etapa)
	default:
		return fmt.Sprintf("%s falhou com código %d", etapa, execucao.CodigoSaida)
	}
}

func existeRequisitosNaRaiz(diretorio string) bool {
	informacao, err := os.Stat(filepath.Join(diretorio, arquivoDeRequisitos))
	return err == nil && informacao.Mode().IsRegular()
}

func infraestrutura(motivo string, saidas ...[]byte) sandbox.ResultadoSandbox {
	var saida strings.Builder
	for _, trecho := range saidas {
		saida.Write(trecho)
	}
	saida.WriteString("aprova: " + motivo + "\n")

	return sandbox.ResultadoSandbox{SaidaBruta: saida.String(), TipoFalha: sandbox.FalhaInfraestrutura}
}

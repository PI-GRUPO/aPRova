package agente

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"
)

const (
	esperaPelosDescritores = 5 * time.Second
	limiteDoDetalhe        = 500

	configuracaoDoSandbox = `{"sandbox":{"enabled":true,"allowUnsandboxedCommands":false,` +
		`"filesystem":{"denyRead":["~/"]}},"permissions":{"deny":["Read(~/**)","Edit(~/**)","Write(~/**)"]}}`
)

var (
	ferramentasSemExecucao = "Read,Edit,Write,Glob,Grep"
	ferramentasPermitidas  = []string{"Bash(go test:*)", "Bash(go build:*)", "Bash(pytest:*)", "Bash(mvn:*)"}
	chavesProibidas        = []string{"ANTHROPIC_API_KEY", "GEMINI_API_KEY", "GOOGLE_API_KEY"}
	sinalDeLimiteDeUso     = regexp.MustCompile(`(?i)usage limit|hit your limit|limit reached|rate limit`)
)

// Claude roda o claude CLI em modo headless, com o sandbox do Claude Code
// ligado, edição só no cwd e uma lista fechada de comandos de teste. Binario é
// o caminho absoluto do executável; Ambiente é todo o ambiente do processo,
// do qual as chaves de API são sempre retiradas. SemExecucao deixa o agente
// só com as ferramentas de ler e editar, sem nenhum comando.
type Claude struct {
	Binario     string
	Modelo      string
	Ambiente    []string
	SemExecucao bool
}

type saidaDoClaude struct {
	Subtipo      string                `json:"subtype"`
	ComErro      bool                  `json:"is_error"`
	Resultado    string                `json:"result"`
	Turnos       int                   `json:"num_turns"`
	UsoPorModelo map[string]usoNaSaida `json:"modelUsage"`
}

type usoNaSaida struct {
	Entrada     int64 `json:"inputTokens"`
	Saida       int64 `json:"outputTokens"`
	CacheLido   int64 `json:"cacheReadInputTokens"`
	CacheCriado int64 `json:"cacheCreationInputTokens"`
}

// Executar roda o agente no diretório do pedido até o prazo, matando o grupo
// de processos inteiro se o prazo acabar. O desfecho vem do JSON que o claude
// imprime, nunca do código de saída.
func (c *Claude) Executar(ctx context.Context, pedido Pedido) (Execucao, error) {
	if pedido.Diretorio == "" || pedido.Prazo <= 0 {
		return Execucao{}, errors.New("agente: o pedido precisa do diretório da tarefa e de um prazo")
	}
	ctx, cancelar := context.WithTimeout(ctx, pedido.Prazo)
	defer cancelar()

	comando := exec.CommandContext(ctx, c.Binario, c.argumentos(pedido.Instrucao)...) //nolint:gosec // G204: o binário é o claude configurado; os argumentos são montados aqui.
	comando.Dir = pedido.Diretorio
	comando.Env = ambienteSemChaves(c.Ambiente)
	comando.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	comando.Cancel = func() error { return syscall.Kill(-comando.Process.Pid, syscall.SIGKILL) }
	comando.WaitDelay = esperaPelosDescritores

	var saida, erros bytes.Buffer
	comando.Stdout, comando.Stderr = &saida, &erros

	inicio := time.Now()
	errDaExecucao := comando.Run()
	execucao := Execucao{Saida: saida.Bytes(), Erros: erros.Bytes(), Duracao: time.Since(inicio), CodigoDeSaida: comando.ProcessState.ExitCode()}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		if comando.Process != nil {
			_ = syscall.Kill(-comando.Process.Pid, syscall.SIGKILL)
		}
		execucao.Desfecho = DesfechoPrazoEsgotado
		execucao.Detalhe = fmt.Sprintf("o agente passou do prazo de %v", pedido.Prazo)
		return execucao, nil
	}
	var saidaComErro *exec.ExitError
	if errDaExecucao != nil && !errors.As(errDaExecucao, &saidaComErro) {
		return execucao, fmt.Errorf("agente: o claude não pôde ser executado: %w", errDaExecucao)
	}
	classificar(&execucao)
	return execucao, nil
}

func (c *Claude) argumentos(instrucao string) []string {
	argumentos := []string{
		"-p", instrucao,
		"--output-format", "json",
		"--model", c.Modelo,
		"--permission-mode", "acceptEdits",
		"--setting-sources", "",
		"--strict-mcp-config",
		"--settings", configuracaoDoSandbox,
	}
	if c.SemExecucao {
		return append(argumentos, "--tools", ferramentasSemExecucao)
	}
	return append(append(argumentos, "--allowedTools"), ferramentasPermitidas...)
}

func classificar(execucao *Execucao) {
	var lida saidaDoClaude
	if err := json.Unmarshal(bytes.TrimSpace(execucao.Saida), &lida); err != nil {
		if sinalDeLimiteDeUso.Match(execucao.Erros) {
			execucao.Desfecho = DesfechoLimiteDeUso
			execucao.Detalhe = resumir(string(execucao.Erros))
			return
		}
		execucao.Desfecho = DesfechoSaidaIlegivel
		execucao.Detalhe = fmt.Sprintf("saída %d, sem o JSON do claude: %s", execucao.CodigoDeSaida, resumir(string(execucao.Erros)))
		return
	}

	execucao.Resposta = lida.Resultado
	execucao.Turnos = lida.Turnos
	registrarUso(execucao, lida.UsoPorModelo)

	switch {
	case lida.ComErro && sinalDeLimiteDeUso.MatchString(lida.Resultado):
		execucao.Desfecho = DesfechoLimiteDeUso
		execucao.Detalhe = resumir(lida.Resultado)
	case lida.ComErro || lida.Subtipo != "success":
		execucao.Desfecho = DesfechoErroReportado
		execucao.Detalhe = resumir(lida.Subtipo + ": " + lida.Resultado)
	default:
		execucao.Desfecho = DesfechoConcluida
	}
}

func registrarUso(execucao *Execucao, usoPorModelo map[string]usoNaSaida) {
	execucao.Modelos = make(map[string]UsoDoModelo, len(usoPorModelo))
	var maiorSaida int64 = -1
	for modelo, uso := range usoPorModelo {
		execucao.Modelos[modelo] = UsoDoModelo{
			TokensDeEntrada: uso.Entrada, TokensDeSaida: uso.Saida,
			TokensDeCacheLidos: uso.CacheLido, TokensDeCacheCriados: uso.CacheCriado,
		}
		execucao.TokensDeEntrada += uso.Entrada
		execucao.TokensDeSaida += uso.Saida
		if uso.Saida > maiorSaida || (uso.Saida == maiorSaida && modelo < execucao.ModeloPrincipal) {
			maiorSaida = uso.Saida
			execucao.ModeloPrincipal = modelo
		}
	}
}

func ambienteSemChaves(ambiente []string) []string {
	limpo := make([]string, 0, len(ambiente))
	for _, variavel := range ambiente {
		nome, _, _ := strings.Cut(variavel, "=")
		proibida := false
		for _, chave := range chavesProibidas {
			proibida = proibida || nome == chave
		}
		if !proibida {
			limpo = append(limpo, variavel)
		}
	}
	return limpo
}

func resumir(texto string) string {
	texto = strings.TrimSpace(texto)
	if len(texto) > limiteDoDetalhe {
		return texto[:limiteDoDetalhe] + "…"
	}
	return texto
}

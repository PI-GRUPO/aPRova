package agente

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

const scriptDoAgenteFalso = `#!/bin/sh
printf '%s\n' "$@" > "$FALSO_REGISTRO/argumentos"
env > "$FALSO_REGISTRO/ambiente"
pwd > "$FALSO_REGISTRO/diretorio"
case "$FALSO_MODO" in
sucesso)
  echo 'pronto' > solucao.txt
  printf '%s' '{"type":"result","subtype":"success","is_error":false,"result":"Implementei a solução.","num_turns":7,"duration_ms":4200,"usage":{"input_tokens":120,"output_tokens":900},"modelUsage":{"claude-sonnet-5":{"inputTokens":1500,"outputTokens":900},"claude-haiku-4-5-20251001":{"inputTokens":800,"outputTokens":16}}}'
  ;;
sucesso_com_saida_1)
  printf '%s' '{"type":"result","subtype":"success","is_error":false,"result":"feito","num_turns":1,"modelUsage":{"claude-sonnet-5":{"inputTokens":1,"outputTokens":1}}}'
  exit 1
  ;;
erro)
  printf '%s' '{"type":"result","subtype":"error_max_turns","is_error":true,"result":"","num_turns":40,"modelUsage":{"claude-sonnet-5":{"inputTokens":10,"outputTokens":5}}}'
  ;;
ilegivel)
  echo 'isto não é JSON'
  ;;
limite)
  printf '%s' '{"type":"result","subtype":"success","is_error":true,"result":"You'"'"'ve hit your limit · resets 3am (America/Sao_Paulo)","num_turns":1,"modelUsage":{}}'
  ;;
limite_no_stderr)
  echo 'Claude AI usage limit reached|1731618000' >&2
  exit 1
  ;;
trava)
  sleep 300 &
  echo $! > "$FALSO_REGISTRO/neto"
  sleep 300
  ;;
esac
`

type cenarioDoFalso struct {
	semExecucao bool
	modo        string
	registro    string
	tarefa      string
	ambiente    []string
	prazo       time.Duration
}

func agenteFalso(t *testing.T) (*Claude, string) {
	t.Helper()
	binarios := t.TempDir()
	caminho := filepath.Join(binarios, "claude")
	if err := os.WriteFile(caminho, []byte(scriptDoAgenteFalso), 0o700); err != nil { //nolint:gosec // G306: o script de teste precisa ser executável.
		t.Fatal(err)
	}
	return &Claude{Binario: caminho, Modelo: "claude-sonnet-5"}, binarios
}

func rodar(t *testing.T, cenario cenarioDoFalso) (Execucao, cenarioDoFalso, error) {
	t.Helper()
	claude, _ := agenteFalso(t)
	claude.SemExecucao = cenario.semExecucao
	cenario.registro = t.TempDir()
	cenario.tarefa = t.TempDir()
	if cenario.prazo == 0 {
		cenario.prazo = 30 * time.Second
	}
	claude.Ambiente = append([]string{
		"FALSO_MODO=" + cenario.modo, "FALSO_REGISTRO=" + cenario.registro, "PATH=/usr/bin:/bin",
	}, cenario.ambiente...)

	execucao, err := claude.Executar(t.Context(), Pedido{Diretorio: cenario.tarefa, Instrucao: "Implemente a tarefa.", Prazo: cenario.prazo})
	return execucao, cenario, err
}

func lerRegistro(t *testing.T, cenario cenarioDoFalso, nome string) string {
	t.Helper()
	conteudo, err := os.ReadFile(filepath.Join(cenario.registro, nome))
	if err != nil {
		t.Fatalf("o agente falso não registrou %s: %v", nome, err)
	}
	return string(conteudo)
}

func TestSucessoRegistraModelosTokensETurnos(t *testing.T) {
	execucao, cenario, err := rodar(t, cenarioDoFalso{modo: "sucesso"})
	if err != nil {
		t.Fatal(err)
	}

	if execucao.Desfecho != DesfechoConcluida || execucao.Resposta != "Implementei a solução." || execucao.Turnos != 7 {
		t.Errorf("execução concluída: %+v", execucao)
	}
	if execucao.ModeloPrincipal != "claude-sonnet-5" || len(execucao.Modelos) != 2 ||
		execucao.Modelos["claude-haiku-4-5-20251001"].TokensDeSaida != 16 {
		t.Errorf("os dois modelos do modelUsage ficam registrados, e o principal é o que gerou a saída: %+v", execucao.Modelos)
	}
	if execucao.TokensDeEntrada != 2300 || execucao.TokensDeSaida != 916 {
		t.Errorf("tokens somados de todos os modelos: entrada %d, saída %d", execucao.TokensDeEntrada, execucao.TokensDeSaida)
	}
	if execucao.Duracao <= 0 || len(execucao.Saida) == 0 {
		t.Errorf("duração medida e saída bruta guardada: %v, %d bytes", execucao.Duracao, len(execucao.Saida))
	}
	if diretorio := strings.TrimSpace(lerRegistro(t, cenario, "diretorio")); !mesmoDiretorio(t, diretorio, cenario.tarefa) {
		t.Errorf("o cwd do agente é o diretório da tarefa: %s", diretorio)
	}
}

func TestCodigoDeSaidaNuncaDecideODesfecho(t *testing.T) {
	sucesso, _, err := rodar(t, cenarioDoFalso{modo: "sucesso_com_saida_1"})
	if err != nil || sucesso.Desfecho != DesfechoConcluida || sucesso.CodigoDeSaida != 1 {
		t.Errorf("JSON de sucesso com saída 1 é concluída, com o código registrado: %+v, %v", sucesso, err)
	}

	erro, _, err := rodar(t, cenarioDoFalso{modo: "erro"})
	if err != nil || erro.Desfecho != DesfechoErroReportado || erro.CodigoDeSaida != 0 || !strings.Contains(erro.Detalhe, "error_max_turns") {
		t.Errorf("erro reportado com saída 0 é erro, com o subtipo no detalhe: %+v, %v", erro, err)
	}
}

func TestSaidaIlegivelEFalhaDoAgente(t *testing.T) {
	execucao, _, err := rodar(t, cenarioDoFalso{modo: "ilegivel"})

	if err != nil || execucao.Desfecho != DesfechoSaidaIlegivel {
		t.Errorf("saída que não é o JSON do claude: %+v, %v", execucao, err)
	}
}

func TestLimiteDeUsoEDesfechoProprio(t *testing.T) {
	for _, modo := range []string{"limite", "limite_no_stderr"} {
		t.Run(modo, func(t *testing.T) {
			execucao, _, err := rodar(t, cenarioDoFalso{modo: modo})

			if err != nil || execucao.Desfecho != DesfechoLimiteDeUso {
				t.Errorf("limite de uso não é falha da tarefa: %+v, %v", execucao, err)
			}
		})
	}
}

func TestPrazoEsgotadoMataOGrupoDeProcessos(t *testing.T) {
	inicio := time.Now()

	execucao, cenario, err := rodar(t, cenarioDoFalso{modo: "trava", prazo: time.Second})

	if err != nil || execucao.Desfecho != DesfechoPrazoEsgotado {
		t.Fatalf("prazo esgotado: %+v, %v", execucao, err)
	}
	if decorrido := time.Since(inicio); decorrido > 10*time.Second {
		t.Errorf("o agente precisa ser morto no prazo, levou %v", decorrido)
	}
	neto, convErr := strconv.Atoi(strings.TrimSpace(lerRegistro(t, cenario, "neto")))
	if convErr != nil {
		t.Fatal(convErr)
	}
	prazo := time.Now().Add(5 * time.Second)
	for syscall.Kill(neto, 0) == nil && time.Now().Before(prazo) {
		time.Sleep(50 * time.Millisecond)
	}
	if syscall.Kill(neto, 0) == nil {
		_ = syscall.Kill(neto, syscall.SIGKILL)
		t.Error("o processo neto do agente sobreviveu ao prazo")
	}
}

func TestAmbienteDoAgenteNuncaLevaChavesDeAPI(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "chave-do-processo-pai")

	_, cenario, err := rodar(t, cenarioDoFalso{modo: "sucesso", ambiente: []string{
		"GEMINI_API_KEY=chave-gemini", "GOOGLE_API_KEY=chave-google", "ANTHROPIC_API_KEY=chave-anthropic", "LANG=C.UTF-8",
	}})
	if err != nil {
		t.Fatal(err)
	}

	ambiente := lerRegistro(t, cenario, "ambiente")
	for _, chave := range []string{"GEMINI_API_KEY", "GOOGLE_API_KEY", "ANTHROPIC_API_KEY"} {
		if strings.Contains(ambiente, chave+"=") {
			t.Errorf("%s chegou ao processo do agente", chave)
		}
	}
	if !strings.Contains(ambiente, "LANG=C.UTF-8") || strings.Contains(ambiente, "chave-do-processo-pai") {
		t.Errorf("o agente recebe só o ambiente montado, nada do processo pai:\n%s", ambiente)
	}
}

func TestArgumentosFixamModeloPermissoesESandbox(t *testing.T) {
	_, cenario, err := rodar(t, cenarioDoFalso{modo: "sucesso"})
	if err != nil {
		t.Fatal(err)
	}

	argumentos := strings.Split(strings.TrimSpace(lerRegistro(t, cenario, "argumentos")), "\n")
	exigirPar := func(flag, valor string) {
		t.Helper()
		indice := slices.Index(argumentos, flag)
		if indice < 0 || indice+1 >= len(argumentos) || argumentos[indice+1] != valor {
			t.Errorf("esperava %s %s em %q", flag, valor, argumentos)
		}
	}
	exigirPar("-p", "Implemente a tarefa.")
	exigirPar("--output-format", "json")
	exigirPar("--model", "claude-sonnet-5")
	exigirPar("--permission-mode", "acceptEdits")
	exigirPar("--setting-sources", "")
	if !slices.Contains(argumentos, "--strict-mcp-config") {
		t.Error("nenhum MCP do usuário entra no agente")
	}
	for _, proibido := range []string{"--dangerously-skip-permissions", "bypassPermissions", "Bash", "Bash(*)"} {
		if slices.Contains(argumentos, proibido) {
			t.Errorf("%q nunca vai para o agente", proibido)
		}
	}
	for _, ferramenta := range []string{"Bash(go test:*)", "Bash(go build:*)", "Bash(pytest:*)", "Bash(mvn:*)"} {
		if !slices.Contains(argumentos, ferramenta) {
			t.Errorf("a lista fechada precisa ter %s", ferramenta)
		}
	}

	indice := slices.Index(argumentos, "--settings")
	if indice < 0 {
		t.Fatal("sem --settings")
	}
	configuracao := argumentos[indice+1]
	for _, trecho := range []string{`"enabled":true`, `"allowUnsandboxedCommands":false`, `"denyRead":["~/"]`, `"Read(~/**)"`, `"Edit(~/**)"`, `"Write(~/**)"`} {
		if !strings.Contains(configuracao, trecho) {
			t.Errorf("a configuração do sandbox precisa de %s: %s", trecho, configuracao)
		}
	}
}

func TestSemExecucaoOAgenteSoTemFerramentasDeLerEEditar(t *testing.T) {
	_, cenario, err := rodar(t, cenarioDoFalso{modo: "sucesso", semExecucao: true})
	if err != nil {
		t.Fatal(err)
	}

	argumentos := strings.Split(strings.TrimSpace(lerRegistro(t, cenario, "argumentos")), "\n")
	indice := slices.Index(argumentos, "--tools")
	if indice < 0 || indice+1 >= len(argumentos) || argumentos[indice+1] != "Read,Edit,Write,Glob,Grep" {
		t.Errorf("sem execução, as ferramentas são só Read,Edit,Write,Glob,Grep: %q", argumentos)
	}
	for _, proibido := range []string{"--allowedTools", "Bash(go test:*)", "Bash(pytest:*)", "Bash(mvn:*)"} {
		if slices.Contains(argumentos, proibido) {
			t.Errorf("%q não vai para o agente sem execução: %q", proibido, argumentos)
		}
	}
	for _, mantido := range []string{"acceptEdits", "--strict-mcp-config", "--settings", configuracaoDoSandbox} {
		if !slices.Contains(argumentos, mantido) {
			t.Errorf("o resto da configuração fica igual; faltou %q", mantido)
		}
	}
}

func TestPedidoSemDiretorioOuSemPrazoERecusado(t *testing.T) {
	claude, _ := agenteFalso(t)

	if _, err := claude.Executar(context.Background(), Pedido{Instrucao: "x", Prazo: time.Second}); err == nil {
		t.Error("sem diretório da tarefa o agente não roda")
	}
	if _, err := claude.Executar(context.Background(), Pedido{Diretorio: t.TempDir(), Instrucao: "x"}); err == nil {
		t.Error("sem prazo o agente não roda")
	}
}

func mesmoDiretorio(t *testing.T, a, b string) bool {
	t.Helper()
	resolvidoA, errA := filepath.EvalSymlinks(a)
	resolvidoB, errB := filepath.EvalSymlinks(b)
	return errA == nil && errB == nil && resolvidoA == resolvidoB
}

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PI-GRUPO/aPRova/internal/benchmark/agente"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/conferencia"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/isolamento"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/oraculo"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/remoto"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/rodada"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/suite"
	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/repository"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

const (
	binarioDoClaude     = "/usr/local/bin/claude"
	modeloPadrao        = "claude-sonnet-5"
	raizDoBenchmark     = "/Users/Shared/aprova-bench"
	urlDaAPIDoGitHub    = "https://api.github.com"
	urlPadraoDoAPRova   = "http://localhost:8080"
	prazoDoAgente       = 15 * time.Minute
	prazoDaDecisao      = 20 * time.Minute
	intervaloDeConsulta = 15 * time.Second
	prazoDasChamadas    = 60 * time.Second
)

var (
	linguagensDaRodada = []string{"go", "python", "java"}
	modelosDoDesenho   = []string{modeloPadrao, "claude-haiku-4-5-20251001"}
)

type opcoesDaRodada struct {
	Data        string
	IDs         []string
	Modelo      string
	SemExecucao bool
}

func lerOpcoesDaRodada(argumentos []string) (opcoesDaRodada, error) {
	opcoes := opcoesDaRodada{Modelo: modeloPadrao}
	for len(argumentos) > 0 && strings.HasPrefix(argumentos[0], "--") {
		switch {
		case argumentos[0] == "--sem-execucao":
			opcoes.SemExecucao = true
			argumentos = argumentos[1:]
		case argumentos[0] == "--modelo" && len(argumentos) >= 2:
			opcoes.Modelo = argumentos[1]
			argumentos = argumentos[2:]
		default:
			return opcoesDaRodada{}, fmt.Errorf("opção inválida: %s: %w", argumentos[0], errUsoErrado)
		}
	}
	if !slices.Contains(modelosDoDesenho, opcoes.Modelo) {
		return opcoesDaRodada{}, fmt.Errorf("o modelo %s não está no desenho do benchmark (%s)", opcoes.Modelo, strings.Join(modelosDoDesenho, ", "))
	}
	if len(argumentos) == 0 {
		return opcoesDaRodada{}, errUsoErrado
	}
	opcoes.Data, opcoes.IDs = argumentos[0], argumentos[1:]
	return opcoes, nil
}

type buscadorDeTaskRun interface {
	BuscarTaskRun(ctx context.Context, id string) (domain.TaskRun, error)
}

type estadosDoBanco struct {
	buscador buscadorDeTaskRun
}

func (e estadosDoBanco) Estado(ctx context.Context, taskRunID string) (string, error) {
	taskRun, err := e.buscador.BuscarTaskRun(ctx, taskRunID)
	if err != nil {
		return "", fmt.Errorf("estado do task_run %s: %w", taskRunID, err)
	}
	return string(taskRun.Estado), nil
}

type agentePorDiretorio struct {
	ferramentas agente.Ferramentas
	modelo      string
	semExecucao bool
}

func (a agentePorDiretorio) Executar(ctx context.Context, pedido agente.Pedido) (agente.Execucao, error) {
	if err := a.ferramentas.PrepararDiretorio(pedido.Diretorio); err != nil {
		return agente.Execucao{}, err
	}
	claude := &agente.Claude{
		Binario: binarioDoClaude, Modelo: a.modelo, Ambiente: a.ferramentas.Ambiente(pedido.Diretorio), SemExecucao: a.semExecucao,
	}
	return claude.Executar(ctx, pedido)
}

func filtrarTarefas(montada suite.Suite, ids []string) (suite.Suite, error) {
	if len(ids) == 0 {
		return montada, nil
	}
	for _, id := range ids {
		if !slices.ContainsFunc(montada.Tarefas, func(t suite.Tarefa) bool { return t.ID == id }) {
			return suite.Suite{}, fmt.Errorf("a tarefa %s não está na suíte", id)
		}
	}
	filtrada := montada
	filtrada.Tarefas = nil
	for _, tarefa := range montada.Tarefas {
		if slices.Contains(ids, tarefa.ID) {
			filtrada.Tarefas = append(filtrada.Tarefas, tarefa)
		}
	}
	return filtrada, nil
}

func repositoriosDoAmbiente(ler func(string) string) (map[string]string, error) {
	repositorios := map[string]string{}
	for _, linguagem := range linguagensDaRodada {
		variavel := "BENCH_REPOSITORIO_" + strings.ToUpper(linguagem)
		valor := strings.TrimSpace(ler(variavel))
		if valor == "" || !strings.Contains(valor, "/") {
			return nil, fmt.Errorf("%s precisa ser dono/repositorio", variavel)
		}
		repositorios[linguagem] = valor
	}
	return repositorios, nil
}

func rodar(ctx context.Context, opcoes opcoesDaRodada, saida io.Writer) error {
	execucao, sondagem, encerrar, err := montarRodada(ctx, opcoes)
	if err != nil {
		return err
	}
	defer encerrar()

	if err := execucao.VerificarPrecondicoes(ctx, sondagem); err != nil {
		return fmt.Errorf("pré-condição da rodada: %w", err)
	}
	if err := execucao.Executar(ctx); err != nil {
		if errors.Is(err, rodada.ErrLimiteDeUso) {
			_, _ = fmt.Fprintln(saida, "limite de uso do agente atingido; rode o mesmo comando depois que o limite renovar")
		}
		return err
	}
	_, err = fmt.Fprintf(saida, "rodada %s concluída: %d tarefas em %s\n", opcoes.Data, len(execucao.Suite.Tarefas), execucao.Resultados)
	return err
}

func montarRodada(ctx context.Context, opcoes opcoesDaRodada) (rodada.Rodada, isolamento.Sondagem, func(), error) {
	montada, err := lerSuite()
	if err != nil {
		return rodada.Rodada{}, isolamento.Sondagem{}, nil, err
	}
	if montada, err = filtrarTarefas(montada, opcoes.IDs); err != nil {
		return rodada.Rodada{}, isolamento.Sondagem{}, nil, err
	}
	repositorios, err := repositoriosDoAmbiente(os.Getenv)
	if err != nil {
		return rodada.Rodada{}, isolamento.Sondagem{}, nil, err
	}
	env, err := os.ReadFile(".env")
	if err != nil {
		return rodada.Rodada{}, isolamento.Sondagem{}, nil, fmt.Errorf("ler o .env do aPRova para a conferência do diff: %w", err)
	}
	token, err := exec.CommandContext(ctx, "gh", "auth", "token").Output()
	if err != nil {
		return rodada.Rodada{}, isolamento.Sondagem{}, nil, fmt.Errorf("obter o token do gh: %w", err)
	}
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return rodada.Rodada{}, isolamento.Sondagem{}, nil, fmt.Errorf("conectar ao banco do aPRova: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return rodada.Rodada{}, isolamento.Sondagem{}, nil, err
	}
	ferramentas := agente.Ferramentas{
		Home: home, Usuario: os.Getenv("USER"), Venv: filepath.Join(raizDoBenchmark, "venv"),
		JavaHome: "/opt/homebrew/opt/openjdk@21", DiretorioDoGo: "/usr/local/go/bin", BinariosDoBrew: "/opt/homebrew/bin",
		RepositorioMaven: filepath.Join(raizDoBenchmark, "m2"),
	}
	cliente := &http.Client{Timeout: prazoDasChamadas}
	trabalhoDaRodada := filepath.Join(raizDoBenchmark, "trabalho", opcoes.Data)
	agenteDaRodada := agentePorDiretorio{ferramentas: ferramentas, modelo: opcoes.Modelo, semExecucao: opcoes.SemExecucao}
	execucao := rodada.Rodada{
		Nome: opcoes.Data, SemExecucao: opcoes.SemExecucao,
		Suite: montada, RaizDaSuite: diretorioDaSuite,
		Resultados: filepath.Join("bench", "resultados", opcoes.Data), Trabalho: trabalhoDaRodada,
		Agentes: func(string) agente.Agente { return agenteDaRodada },
		GitHub:  &remoto.GitHub{URLBase: urlDaAPIDoGitHub, Token: strings.TrimSpace(string(token)), Cliente: cliente},
		Recibos: remoto.APRova{URLBase: valorOuPadrao(os.Getenv("BENCH_URL_APROVA"), urlPadraoDoAPRova), Cliente: cliente},
		Estados: estadosDoBanco{buscador: repository.NovoRepositorioPostgres(pool, repository.OpcoesPostgres{})},
		Oraculo: oraculo.Oraculo{
			Publicador: oraculo.PublicadorDocker{Imagem: imagensLocais[sandbox.LinguagemGo]},
			Executor: func(montagens []string) sandbox.Executor {
				return sandbox.NovoExecutorLocal(sandbox.OpcoesExecutorLocal{ImagemPorLinguagem: imagensLocais, Montagens: montagens})
			},
			Temporario: filepath.Join(raizDoBenchmark, "oraculo"),
		},
		Segredos: conferencia.SegredosDoEnv(env), Repositorios: repositorios,
		PrazoDoAgente: prazoDoAgente, PrazoDaDecisao: prazoDaDecisao, IntervaloDeConsulta: intervaloDeConsulta,
	}
	if err := registrarAmbiente(ctx, execucao.Resultados, montada, repositorios, opcoes); err != nil {
		pool.Close()
		return rodada.Rodada{}, isolamento.Sondagem{}, nil, err
	}

	raizDoAPRova, err := os.Getwd()
	if err != nil {
		return rodada.Rodada{}, isolamento.Sondagem{}, nil, err
	}
	sondagem := isolamento.Sondagem{
		Agente: agentePorDiretorio{ferramentas: ferramentas, modelo: modeloPadrao}, Home: home, RaizDoAPRova: raizDoAPRova, DiretorioDeTrabalho: trabalhoDaRodada,
	}
	return execucao, sondagem, pool.Close, nil
}

func lerSuite() (suite.Suite, error) {
	var montada suite.Suite
	bruto, err := os.ReadFile(filepath.Join(diretorioDaSuite, "suite.json"))
	if err != nil {
		return montada, fmt.Errorf("ler o suite.json: %w", err)
	}
	if err := json.Unmarshal(bruto, &montada); err != nil {
		return montada, fmt.Errorf("suite.json ilegível: %w", err)
	}
	return montada, nil
}

func registrarAmbiente(ctx context.Context, resultados string, montada suite.Suite, repositorios map[string]string, opcoes opcoesDaRodada) error {
	ids := make([]string, 0, len(montada.Tarefas))
	for _, tarefa := range montada.Tarefas {
		ids = append(ids, tarefa.ID)
	}
	ambiente := map[string]any{
		"iniciada_em":        time.Now().Format(time.RFC3339),
		"modelo":             opcoes.Modelo,
		"sem_execucao":       opcoes.SemExecucao,
		"modelo_da_sondagem": modeloPadrao,
		"semente_da_suite":   montada.Semente,
		"commit_do_polyglot": montada.CommitDoPolyglot,
		"tarefas":            ids,
		"repositorios":       repositorios,
		"versoes": map[string]string{
			"claude": versao(ctx, binarioDoClaude, "--version"),
			"go":     versao(ctx, "/usr/local/go/bin/go", "version"),
			"maven":  versao(ctx, "/opt/homebrew/bin/mvn", "-v"),
			"python": versao(ctx, filepath.Join(raizDoBenchmark, "venv", "bin", "python"), "--version"),
			"pytest": versao(ctx, filepath.Join(raizDoBenchmark, "venv", "bin", "pytest"), "--version"),
		},
	}
	bruto, err := json.MarshalIndent(ambiente, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(resultados, 0o750); err != nil {
		return err
	}
	nome := filepath.Join(resultados, "rodada-"+time.Now().Format("20060102T150405")+".json")
	return os.WriteFile(nome, append(bruto, '\n'), 0o600) //nolint:gosec // G703: o nome é montado aqui, dentro de bench/resultados.
}

func versao(ctx context.Context, binario string, argumentos ...string) string {
	comando := exec.CommandContext(ctx, binario, argumentos...) //nolint:gosec // G204: binários fixos deste arquivo.
	comando.Env = append(os.Environ(), "JAVA_HOME=/opt/homebrew/opt/openjdk@21")
	saida, err := comando.CombinedOutput()
	if err != nil {
		return "indisponível: " + err.Error()
	}
	return strings.SplitN(strings.TrimSpace(string(saida)), "\n", 2)[0]
}

func valorOuPadrao(valor, padrao string) string {
	if strings.TrimSpace(valor) == "" {
		return padrao
	}
	return valor
}

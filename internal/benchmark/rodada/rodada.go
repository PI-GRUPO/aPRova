// Package rodada executa as tarefas da suíte do benchmark: agente, diff,
// conferência, PR, decisão do aPRova e oráculo, gravando o resultado de cada
// tarefa assim que ela termina e retomando de onde parou.
package rodada

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/benchmark/agente"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/conferencia"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/oraculo"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/remoto"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/resultado"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/suite"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/trabalho"
)

const (
	nomeDoCheckRun        = "aPRova"
	diretorioDosPendentes = "pendentes"
	diretorioDasExecucoes = "execucoes"
	arquivoDosEventos     = "eventos.jsonl"
)

// ErrLimiteDeUso indica que o agente bateu o limite de uso do plano. Não é
// falha da tarefa: a rodada para e retoma depois, com o mesmo modelo.
var ErrLimiteDeUso = errors.New("rodada: limite de uso do agente atingido; retome depois de o limite renovar")

// GitHub é o que a rodada usa do GitHub dos repositórios do benchmark.
type GitHub interface {
	PublicarCommit(ctx context.Context, repositorio, branch string, commit remoto.Commit) (remoto.CommitPublicado, error)
	AbrirPR(ctx context.Context, repositorio, head, base, titulo, corpo string) (remoto.PR, error)
	CheckRun(ctx context.Context, repositorio, sha, nome string) (*remoto.CheckRun, error)
}

// Recibos lê a decisão registrada no recibo do aPRova.
type Recibos interface {
	Recibo(ctx context.Context, id string) (remoto.Recibo, error)
}

// Estados lê o estado de um task_run. Opcional.
type Estados interface {
	Estado(ctx context.Context, taskRunID string) (string, error)
}

// Julgador é o oráculo.
type Julgador interface {
	Julgar(ctx context.Context, entrada oraculo.Entrada) (oraculo.Veredito, error)
}

// Rodada é uma execução da suíte. Resultados é bench/resultados/<data>;
// Trabalho recebe as áreas de trabalho, fora do HOME. Agentes constrói o
// agente para o diretório de cada tarefa.
type Rodada struct {
	Suite               suite.Suite
	RaizDaSuite         string
	Resultados          string
	Trabalho            string
	Agentes             func(diretorio string) agente.Agente
	GitHub              GitHub
	Recibos             Recibos
	Estados             Estados
	Oraculo             Julgador
	Segredos            conferencia.Segredos
	Repositorios        map[string]string
	PrazoDoAgente       time.Duration
	PrazoDaDecisao      time.Duration
	IntervaloDeConsulta time.Duration
}

type pendente struct {
	Tarefa  resultado.Tarefa `json:"tarefa"`
	SHAHead string           `json:"sha_head"`
}

// Executar roda as tarefas sem resultado gravado, na ordem do suite.json.
// Devolve ErrLimiteDeUso quando o agente bate o limite; qualquer outro erro
// deixa a tarefa para a próxima execução, que retoma de onde parou.
func (r Rodada) Executar(ctx context.Context) error {
	for _, tarefa := range r.Suite.Tarefas {
		if r.existe(filepath.Join(resultado.DiretorioDasTarefas, tarefa.ID+".json")) {
			continue
		}
		if err := r.executarTarefa(ctx, tarefa); err != nil {
			return err
		}
	}
	return nil
}

func (r Rodada) executarTarefa(ctx context.Context, tarefa suite.Tarefa) error {
	var emAndamento pendente
	if r.lerJSON(filepath.Join(diretorioDosPendentes, tarefa.ID+".json"), &emAndamento) {
		r.registrarEvento("retomada", tarefa.ID, "retomada depois do PR aberto, sem rodar o agente de novo")
		return r.concluir(ctx, tarefa, emAndamento)
	}

	registro, area, mudanca, err := r.rodarAgente(ctx, tarefa)
	if err != nil || registro.Desfecho != "" {
		if err == nil {
			err = r.gravarResultado(registro)
		}
		return err
	}

	if err := conferencia.Conferir(mudanca.Caminhos(), mudanca.Patch, r.Segredos); err != nil {
		registro.Desfecho = resultado.DesfechoDiffRecusado
		registro.Detalhe = "diff recusado antes do push: " + err.Error()
		return r.gravarResultado(registro)
	}

	inicio := time.Now()
	emAndamento, err = r.publicar(ctx, tarefa, registro, area, mudanca)
	if err != nil {
		return err
	}
	emAndamento.Tarefa.Tempos.PublicacaoMS = time.Since(inicio).Milliseconds()
	if err := r.gravarJSON(filepath.Join(diretorioDosPendentes, tarefa.ID+".json"), emAndamento); err != nil {
		return err
	}
	return r.concluir(ctx, tarefa, emAndamento)
}

func (r Rodada) rodarAgente(ctx context.Context, tarefa suite.Tarefa) (resultado.Tarefa, trabalho.Area, trabalho.Mudanca, error) {
	registro := resultado.Tarefa{ID: tarefa.ID, Grupo: tarefa.Grupo, Linguagem: tarefa.Linguagem, Oraculo: resultado.OraculoNaoRodou}
	if tarefa.Grupo == resultado.GrupoRisco {
		registro.Esperado = &resultado.Esperado{Nivel: tarefa.Esperado.Nivel, Motivo: tarefa.Esperado.Motivo}
	}

	diretorio := filepath.Join(r.Trabalho, tarefa.ID)
	if err := os.RemoveAll(diretorio); err != nil {
		return registro, trabalho.Area{}, trabalho.Mudanca{}, fmt.Errorf("rodada: limpar a área de %s: %w", tarefa.ID, err)
	}
	area, err := trabalho.Preparar(r.originais(tarefa), diretorio)
	if err != nil {
		return registro, area, trabalho.Mudanca{}, err
	}
	enunciado, err := os.ReadFile(filepath.Join(r.RaizDaSuite, tarefa.ID, "enunciado.md")) //nolint:gosec // G304: o caminho vem do suite.json versionado.
	if err != nil {
		return registro, area, trabalho.Mudanca{}, fmt.Errorf("rodada: ler o enunciado de %s: %w", tarefa.ID, err)
	}

	registro.ExecucoesDoAgente, err = r.contarExecucao(tarefa.ID)
	if err != nil {
		return registro, area, trabalho.Mudanca{}, err
	}
	execucao, err := r.Agentes(diretorio).Executar(ctx, agente.Pedido{
		Diretorio: diretorio, Prazo: r.PrazoDoAgente,
		Instrucao: agente.Instrucao(agente.Tarefa{Enunciado: string(enunciado), Linguagem: tarefa.Linguagem, Solucao: tarefa.ArquivosDaSolucao, Protegidos: tarefa.ArquivosProtegidos}),
	})
	if err != nil {
		return registro, area, trabalho.Mudanca{}, err
	}
	if execucao.Desfecho == agente.DesfechoLimiteDeUso {
		r.registrarEvento(string(agente.DesfechoLimiteDeUso), tarefa.ID, execucao.Detalhe)
		return registro, area, trabalho.Mudanca{}, ErrLimiteDeUso
	}
	registro.Agente = usoDoAgente(execucao)
	registro.Tempos.AgenteMS = execucao.Duracao.Milliseconds()

	mudanca, err := area.Diff(ctx)
	if err != nil {
		return registro, area, mudanca, err
	}
	registro.TestesAdulterados, err = area.TestesAdulterados(r.originais(tarefa), tarefa.ArquivosProtegidos)
	if err != nil {
		return registro, area, mudanca, err
	}
	if mudanca.Vazia() {
		registro.Desfecho = resultado.DesfechoDiffVazio
		if execucao.Desfecho != agente.DesfechoConcluida {
			registro.Desfecho = resultado.DesfechoFalhaDoAgente
			registro.Detalhe = string(execucao.Desfecho) + ": " + execucao.Detalhe
		}
	}
	return registro, area, mudanca, nil
}

func (r Rodada) publicar(ctx context.Context, tarefa suite.Tarefa, registro resultado.Tarefa, area trabalho.Area, mudanca trabalho.Mudanca) (pendente, error) {
	repositorio := r.Repositorios[tarefa.Linguagem]
	originais, err := lerArvore(r.originais(tarefa))
	if err != nil {
		return pendente{}, err
	}
	base, err := r.GitHub.PublicarCommit(ctx, repositorio, "base/"+tarefa.ID, remoto.Commit{Mensagem: "Base da tarefa " + tarefa.ID, Arquivos: originais})
	if err != nil {
		return pendente{}, err
	}

	solucao := remoto.Commit{Mensagem: "Solução da tarefa " + tarefa.ID, Pai: &base, Arquivos: map[string][]byte{}, Removidos: mudanca.Removidos()}
	for _, arquivo := range mudanca.Arquivos {
		if arquivo.Situacao == "D" {
			continue
		}
		conteudo, err := os.ReadFile(filepath.Join(area.Diretorio, filepath.FromSlash(arquivo.Caminho))) //nolint:gosec // G304: caminho do diff já conferido dentro da área.
		if err != nil {
			return pendente{}, fmt.Errorf("rodada: ler %s da solução: %w", arquivo.Caminho, err)
		}
		solucao.Arquivos[arquivo.Caminho] = conteudo
	}
	branch := "solucao/" + tarefa.ID
	publicado, err := r.GitHub.PublicarCommit(ctx, repositorio, branch, solucao)
	if err != nil {
		return pendente{}, err
	}
	pr, err := r.GitHub.AbrirPR(ctx, repositorio, branch, "base/"+tarefa.ID, "Tarefa "+tarefa.ID,
		"Solução gerada por um agente de código para a tarefa "+tarefa.ID+".")
	if err != nil {
		return pendente{}, err
	}
	registro.PR = &resultado.PR{Numero: pr.Numero, URL: pr.URL, Branch: branch, SHA: publicado.SHA}
	return pendente{Tarefa: registro, SHAHead: publicado.SHA}, nil
}

func (r Rodada) concluir(ctx context.Context, tarefa suite.Tarefa, emAndamento pendente) error {
	registro := emAndamento.Tarefa
	inicio := time.Now()
	if err := r.coletarDecisao(ctx, tarefa, emAndamento.SHAHead, &registro); err != nil {
		return err
	}
	registro.Tempos.DecisaoMS = time.Since(inicio).Milliseconds()
	if registro.Desfecho == resultado.DesfechoSemDecisao {
		registro.Tempos.DecisaoMS = 0
	}

	if tarefa.Grupo == resultado.GrupoCorrecao {
		veredito, err := r.Oraculo.Julgar(ctx, oraculo.Entrada{
			ID: tarefa.ID, Linguagem: tarefa.Linguagem, Solucao: filepath.Join(r.Trabalho, tarefa.ID),
			Originais: r.originais(tarefa), Protegidos: tarefa.ArquivosProtegidos,
		})
		if err != nil {
			return fmt.Errorf("rodada: oráculo de %s: %w", tarefa.ID, err)
		}
		registro.Oraculo = veredito.Oraculo
		registro.Tempos.OraculoMS = veredito.Duracao.Milliseconds()
		if veredito.Detalhe != "" && registro.Detalhe == "" {
			registro.Detalhe = veredito.Detalhe
		}
	}
	if err := r.gravarResultado(registro); err != nil {
		return err
	}
	return os.Remove(filepath.Join(r.Resultados, diretorioDosPendentes, tarefa.ID+".json"))
}

func (r Rodada) coletarDecisao(ctx context.Context, tarefa suite.Tarefa, sha string, registro *resultado.Tarefa) error {
	repositorio := r.Repositorios[tarefa.Linguagem]
	limite := time.Now().Add(r.PrazoDaDecisao)
	for {
		check, err := r.GitHub.CheckRun(ctx, repositorio, sha, nomeDoCheckRun)
		if err == nil && check != nil && check.Concluido() {
			return r.registrarDecisao(ctx, *check, registro)
		}
		if time.Now().After(limite) {
			registro.Desfecho = resultado.DesfechoSemDecisao
			registro.Detalhe = fmt.Sprintf("o Check Run %s não concluiu em %v", nomeDoCheckRun, r.PrazoDaDecisao)
			return nil
		}
		if err := dormir(ctx, r.IntervaloDeConsulta); err != nil {
			return err
		}
	}
}

func (r Rodada) registrarDecisao(ctx context.Context, check remoto.CheckRun, registro *resultado.Tarefa) error {
	id := check.IDDoRecibo()
	if id == "" {
		registro.Desfecho = resultado.DesfechoSemDecisao
		registro.Detalhe = "o Check Run concluiu sem recibo: " + check.Titulo
		return nil
	}
	recibo, err := r.Recibos.Recibo(ctx, id)
	if err != nil {
		return err
	}
	registro.Desfecho = resultado.DesfechoDecidido
	registro.Decisao = &resultado.Decisao{Nivel: recibo.Nivel, Motivos: recibo.Motivos, ReciboID: recibo.ID, ReciboHash: recibo.Hash}
	if r.Estados != nil {
		estado, err := r.Estados.Estado(ctx, recibo.TaskRunID)
		if err != nil {
			return err
		}
		registro.Decisao.EstadoDoTaskRun = estado
	}
	return nil
}

func (r Rodada) originais(tarefa suite.Tarefa) string {
	return filepath.Join(r.RaizDaSuite, tarefa.ID, "arquivos")
}

func (r Rodada) contarExecucao(id string) (int, error) {
	caminho := filepath.Join(r.Resultados, diretorioDasExecucoes, id)
	anteriores := 0
	if bruto, err := os.ReadFile(caminho); err == nil { //nolint:gosec // G304: caminho dentro do diretório da rodada.
		anteriores, _ = strconv.Atoi(strings.TrimSpace(string(bruto)))
	}
	atual := anteriores + 1
	if err := gravarArquivo(caminho, []byte(strconv.Itoa(atual)+"\n")); err != nil {
		return 0, err
	}
	return atual, nil
}

func (r Rodada) gravarResultado(registro resultado.Tarefa) error {
	return r.gravarJSON(filepath.Join(resultado.DiretorioDasTarefas, registro.ID+".json"), registro)
}

func (r Rodada) gravarJSON(relativo string, valor any) error {
	bruto, err := json.MarshalIndent(valor, "", "  ")
	if err != nil {
		return fmt.Errorf("rodada: serializar %s: %w", relativo, err)
	}
	return gravarArquivo(filepath.Join(r.Resultados, relativo), append(bruto, '\n'))
}

func (r Rodada) lerJSON(relativo string, destino any) bool {
	bruto, err := os.ReadFile(filepath.Join(r.Resultados, relativo)) //nolint:gosec // G304: caminho dentro do diretório da rodada.
	return err == nil && json.Unmarshal(bruto, destino) == nil
}

func (r Rodada) existe(relativo string) bool {
	_, err := os.Stat(filepath.Join(r.Resultados, relativo))
	return err == nil
}

func (r Rodada) registrarEvento(evento, tarefa, detalhe string) {
	linha, err := json.Marshal(map[string]string{"quando": time.Now().Format(time.RFC3339), "evento": evento, "tarefa": tarefa, "detalhe": detalhe})
	if err != nil {
		return
	}
	caminho := filepath.Join(r.Resultados, arquivoDosEventos)
	if err := os.MkdirAll(filepath.Dir(caminho), 0o750); err != nil {
		return
	}
	arquivo, err := os.OpenFile(caminho, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // G304: caminho dentro do diretório da rodada.
	if err != nil {
		return
	}
	defer func() { _ = arquivo.Close() }()
	_, _ = arquivo.Write(append(linha, '\n'))
}

func usoDoAgente(execucao agente.Execucao) resultado.UsoDoAgente {
	modelos := make(map[string]resultado.UsoDoModelo, len(execucao.Modelos))
	for modelo, uso := range execucao.Modelos {
		modelos[modelo] = resultado.UsoDoModelo{TokensDeEntrada: uso.TokensDeEntrada, TokensDeSaida: uso.TokensDeSaida}
	}
	return resultado.UsoDoAgente{
		ModeloPrincipal: execucao.ModeloPrincipal, Modelos: modelos, Turnos: execucao.Turnos,
		TokensDeEntrada: execucao.TokensDeEntrada, TokensDeSaida: execucao.TokensDeSaida,
		Desfecho: string(execucao.Desfecho), Detalhe: execucao.Detalhe,
	}
}

func lerArvore(raiz string) (map[string][]byte, error) {
	arquivos := map[string][]byte{}
	err := filepath.WalkDir(raiz, func(caminho string, entrada fs.DirEntry, err error) error {
		if err != nil || !entrada.Type().IsRegular() {
			return err
		}
		relativo, err := filepath.Rel(raiz, caminho)
		if err != nil {
			return err
		}
		conteudo, err := os.ReadFile(caminho) //nolint:gosec // G304: arquivo da suíte versionada.
		if err != nil {
			return err
		}
		arquivos[filepath.ToSlash(relativo)] = conteudo
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("rodada: ler os arquivos originais: %w", err)
	}
	return arquivos, nil
}

func gravarArquivo(caminho string, conteudo []byte) error {
	if err := os.MkdirAll(filepath.Dir(caminho), 0o750); err != nil {
		return fmt.Errorf("rodada: criar %s: %w", filepath.Dir(caminho), err)
	}
	temporario := caminho + ".tmp"
	if err := os.WriteFile(temporario, conteudo, 0o600); err != nil { //nolint:gosec // G703: caminho dentro do diretório da rodada.
		return fmt.Errorf("rodada: gravar %s: %w", caminho, err)
	}
	return os.Rename(temporario, caminho)
}

func dormir(ctx context.Context, intervalo time.Duration) error {
	temporizador := time.NewTimer(intervalo)
	defer temporizador.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-temporizador.C:
		return nil
	}
}

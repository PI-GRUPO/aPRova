package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/platform"
	"github.com/Azulpasta/aPRova/internal/rastro"
	"github.com/Azulpasta/aPRova/internal/sandbox"
)

const (
	maximoDeTentativasPadrao   = 3
	esperaBasePadrao           = 30 * time.Second
	maiorMultiplicadorDaEspera = 32
)

// ErrDependenciaAusente indica motor montado sem uma dependência obrigatória.
var ErrDependenciaAusente = errors.New("worker: dependência ausente")

// Opcoes reúne as dependências e os ajustes do motor. Diretorios mapeia
// dono/repositorio ao diretório do projeto dentro do repositório. Linguagens
// mapeia dono/repositorio à linguagem do projeto; sem entrada, é Go. URLPublica
// é a base dos links de recibo. MaxTentativas e EsperaBase limitam as novas
// tentativas de falhas transitórias, com espera que dobra a cada uma.
type Opcoes struct {
	Armazenamento Armazenamento
	LeitorDePR    LeitorDePR
	Permissoes    ConsultaDePermissao
	Sandbox       sandbox.Executor
	Avaliador     AvaliadorRisco
	Recibos       EmissorRecibo
	Publicador    PublicadorPR
	Eventos       EmissorEventos
	Diretorios    map[string]string
	Linguagens    map[string]sandbox.Linguagem
	URLPublica    string
	MaxTentativas int
	EsperaBase    time.Duration
	Diario        *slog.Logger
	Agora         func() time.Time
}

// Motor conduz task_runs pela máquina de estados de domain.EstadoTaskRun.
type Motor struct {
	opcoes Opcoes
}

type passo struct {
	parar    bool
	desfecho Desfecho
}

var seguir = passo{}

func pararCom(desfecho Desfecho) passo {
	return passo{parar: true, desfecho: desfecho}
}

type conducao struct {
	*Motor
	ctx         context.Context
	trabalhador string
	taskRun     domain.TaskRun
	diario      *slog.Logger
	contexto    *platform.ContextoPR
}

// NovoMotor constrói o motor. Dependência ausente devolve
// ErrDependenciaAusente nomeando qual.
func NovoMotor(opcoes Opcoes) (*Motor, error) {
	if ausentes := dependenciasAusentes(opcoes); len(ausentes) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrDependenciaAusente, strings.Join(ausentes, ", "))
	}
	if opcoes.MaxTentativas <= 0 {
		opcoes.MaxTentativas = maximoDeTentativasPadrao
	}
	if opcoes.EsperaBase <= 0 {
		opcoes.EsperaBase = esperaBasePadrao
	}
	if opcoes.Diario == nil {
		opcoes.Diario = slog.Default()
	}
	if opcoes.Agora == nil {
		opcoes.Agora = time.Now
	}
	return &Motor{opcoes: opcoes}, nil
}

func dependenciasAusentes(opcoes Opcoes) []string {
	exigidas := []struct {
		nome     string
		presente bool
	}{
		{"Armazenamento", opcoes.Armazenamento != nil},
		{"LeitorDePR", opcoes.LeitorDePR != nil},
		{"Permissoes", opcoes.Permissoes != nil},
		{"Sandbox", opcoes.Sandbox != nil},
		{"Avaliador", opcoes.Avaliador != nil},
		{"Recibos", opcoes.Recibos != nil},
		{"Publicador", opcoes.Publicador != nil},
		{"Eventos", opcoes.Eventos != nil},
	}

	var ausentes []string
	for _, exigida := range exigidas {
		if !exigida.presente {
			ausentes = append(ausentes, exigida.nome)
		}
	}
	return ausentes
}

// Conduzir leva adiante o task_run reivindicado pelo trabalhador, etapa por
// etapa, até um estado terminal, até aguardar um review ou uma nova
// tentativa, ou até parando ser fechado. O estado novo é gravado antes da
// chamada externa de cada etapa, e cada etapa reaproveita o que já está
// gravado, de modo que retomar um task_run interrompido não perde nem
// duplica trabalho.
func (m *Motor) Conduzir(ctx context.Context, trabalhador string, taskRun domain.TaskRun, parando <-chan struct{}) Desfecho {
	c := &conducao{
		Motor:       m,
		ctx:         rastro.Com(ctx, "task_run_id", taskRun.ID),
		trabalhador: trabalhador,
		taskRun:     taskRun,
		diario:      m.opcoes.Diario.With("task_run_id", taskRun.ID, "trabalhador", trabalhador),
	}

	for !c.taskRun.Estado.Terminal() && !fechado(parando) {
		resultado, err := c.avancar()
		if err != nil {
			return c.desfechoDoErro(err)
		}
		if resultado.parar {
			return resultado.desfecho
		}
	}
	return Desfecho{Liberar: true}
}

func fechado(sinal <-chan struct{}) bool {
	select {
	case <-sinal:
		return true
	default:
		return false
	}
}

func (c *conducao) avancar() (passo, error) {
	substituido, err := c.opcoes.Armazenamento.ExisteTaskRunMaisNovo(c.ctx, c.taskRun)
	if err != nil {
		return passo{}, err
	}
	if substituido {
		return c.encerrarSubstituido()
	}

	switch c.taskRun.Estado {
	case domain.EstadoPendente:
		return seguir, c.transicionar(domain.EstadoMapeandoContexto, "")
	case domain.EstadoMapeandoContexto:
		return c.mapearContexto()
	case domain.EstadoTestandoSandbox:
		return c.testarNoSandbox()
	case domain.EstadoAvaliandoRisco:
		return c.avaliarRisco()
	case domain.EstadoAguardandoAprovacao:
		return c.decidirAprovacao()
	case domain.EstadoAplicando:
		return c.aplicar()
	default:
		c.diario.Warn("estado fora do fluxo do motor, task_run devolvido sem mudança", "estado", c.taskRun.Estado)
		return pararCom(Desfecho{Liberar: true}), nil
	}
}

func (c *conducao) transicionar(para domain.EstadoTaskRun, detalhe string) error {
	atualizado, err := c.opcoes.Armazenamento.TransicionarReivindicado(c.ctx, c.trabalhador, domain.Transicao{
		TaskRunID: c.taskRun.ID, De: c.taskRun.Estado, Para: para, Detalhe: detalhe,
	})
	if err != nil {
		return err
	}

	c.diario.Info("task_run mudou de estado", "de", c.taskRun.Estado, "para", para, "detalhe", detalhe)
	c.taskRun = atualizado
	return nil
}

func (c *conducao) falhar(detalhe string) error {
	return c.transicionar(domain.EstadoFalhou, detalhe)
}

func (c *conducao) desfechoDoErro(err error) Desfecho {
	switch {
	case errors.Is(err, domain.ErrPrazoPerdido), errors.Is(err, domain.ErrTransicaoConcorrente):
		c.diario.Warn("o task_run não está mais com este trabalhador, condução abandonada", "erro", err)
		return Desfecho{}
	case c.ctx.Err() != nil:
		c.diario.Warn("condução interrompida", "erro", err)
		return Desfecho{}
	default:
		c.diario.Error("falha ao conduzir o task_run, nova tentativa depois da espera", "erro", err, "espera", c.opcoes.EsperaBase)
		return Desfecho{Liberar: true, Espera: c.opcoes.EsperaBase}
	}
}

func (c *conducao) espera(tentativa int) time.Duration {
	multiplicador := min(1<<max(tentativa-1, 0), maiorMultiplicadorDaEspera)
	return c.opcoes.EsperaBase * time.Duration(multiplicador)
}

func (c *conducao) donoERepositorio() (string, string) {
	dono, repositorio, _ := strings.Cut(c.taskRun.Repositorio, "/")
	return dono, repositorio
}

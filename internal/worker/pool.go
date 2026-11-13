package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"sync"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

const (
	prazoPadrao            = 2 * time.Minute
	intervaloOciosoPadrao  = 2 * time.Second
	tempoLimiteDaLiberacao = 10 * time.Second
	divisorDaRenovacao     = 3
)

// OpcoesPool configura o pool. Nome identifica o processo nas
// reivindicações; cada trabalhador é Nome#índice. Prazo é a duração da
// reivindicação, renovada a cada terço enquanto a etapa roda. Valores vazios
// assumem padrões: uma goroutine, prazo de 2 minutos, 2 segundos entre
// consultas à fila vazia e o hostname com o pid como nome.
type OpcoesPool struct {
	Motor           *Motor
	Armazenamento   Armazenamento
	Nome            string
	Concorrencia    int
	Prazo           time.Duration
	IntervaloOcioso time.Duration
	Diario          *slog.Logger
}

// Pool roda trabalhadores que reivindicam task_runs da fila e os conduzem
// pelo Motor.
type Pool struct {
	opcoes        OpcoesPool
	parando       chan struct{}
	pararUmaVez   sync.Once
	trabalhadores sync.WaitGroup

	mutex            sync.Mutex
	cancelarTrabalho context.CancelFunc
	emMaos           map[string]string
}

// NovoPool constrói o pool. Motor e Armazenamento são obrigatórios.
func NovoPool(opcoes OpcoesPool) (*Pool, error) {
	if opcoes.Motor == nil || opcoes.Armazenamento == nil {
		return nil, fmt.Errorf("%w: o pool exige Motor e Armazenamento", ErrDependenciaAusente)
	}
	if opcoes.Concorrencia <= 0 {
		opcoes.Concorrencia = 1
	}
	if opcoes.Prazo <= 0 {
		opcoes.Prazo = prazoPadrao
	}
	if opcoes.IntervaloOcioso <= 0 {
		opcoes.IntervaloOcioso = intervaloOciosoPadrao
	}
	if opcoes.Nome == "" {
		opcoes.Nome = nomeDoProcesso()
	}
	if opcoes.Diario == nil {
		opcoes.Diario = slog.Default()
	}
	return &Pool{opcoes: opcoes, parando: make(chan struct{}), emMaos: map[string]string{}}, nil
}

func nomeDoProcesso() string {
	maquina, err := os.Hostname()
	if err != nil {
		maquina = "worker"
	}
	return fmt.Sprintf("%s-%d", maquina, os.Getpid())
}

// Rodar põe os trabalhadores para pegar trabalho e só volta quando todos
// pararem: depois de Parar, ou quando ctx terminar. Terminar ctx sem chamar
// Parar abandona as reivindicações, que voltam à fila quando o prazo vencer.
func (p *Pool) Rodar(ctx context.Context) {
	ctxDoTrabalho, cancelar := context.WithCancel(ctx)
	defer cancelar()

	p.mutex.Lock()
	p.cancelarTrabalho = cancelar
	p.mutex.Unlock()

	for indice := range p.opcoes.Concorrencia {
		trabalhador := fmt.Sprintf("%s#%d", p.opcoes.Nome, indice)
		p.trabalhadores.Go(func() { p.trabalhar(ctxDoTrabalho, trabalhador) })
	}
	p.trabalhadores.Wait()
}

// Parar desliga o pool: ninguém pega trabalho novo, as etapas em andamento
// têm até o limite para terminar, e as que não terminarem são interrompidas
// e devolvidas à fila na hora, sem mudar de estado.
func (p *Pool) Parar(limite time.Duration) {
	p.pararUmaVez.Do(func() { close(p.parando) })

	terminaram := make(chan struct{})
	go func() {
		p.trabalhadores.Wait()
		close(terminaram)
	}()

	select {
	case <-terminaram:
	case <-time.After(limite):
		p.opcoes.Diario.Warn("etapas em andamento passaram do limite do desligamento, interrompendo", "limite", limite)
		p.interromper()
		<-terminaram
	}
	p.liberarOQueFicouEmMaos()
}

func (p *Pool) interromper() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.cancelarTrabalho != nil {
		p.cancelarTrabalho()
	}
}

func (p *Pool) liberarOQueFicouEmMaos() {
	p.mutex.Lock()
	pendentes := maps.Clone(p.emMaos)
	p.mutex.Unlock()

	for taskRunID, trabalhador := range pendentes {
		ctx, cancelar := context.WithTimeout(context.Background(), tempoLimiteDaLiberacao)
		err := p.opcoes.Armazenamento.Liberar(ctx, taskRunID, trabalhador, 0)
		cancelar()

		diario := p.opcoes.Diario.With("task_run_id", taskRunID, "trabalhador", trabalhador)
		if err != nil {
			diario.Error("falha ao devolver o task_run à fila no desligamento", "erro", err)
			continue
		}
		diario.Info("task_run interrompido no desligamento devolvido à fila")
	}
}

func (p *Pool) trabalhar(ctx context.Context, trabalhador string) {
	for !p.encerrando(ctx) {
		taskRun, encontrado, err := p.opcoes.Armazenamento.Reivindicar(ctx, trabalhador, p.opcoes.Prazo)
		if err != nil && ctx.Err() == nil {
			p.opcoes.Diario.Error("falha ao reivindicar trabalho", "trabalhador", trabalhador, "erro", err)
		}
		if err != nil || !encontrado {
			p.aguardar(ctx)
			continue
		}
		p.conduzir(ctx, trabalhador, taskRun)
	}
}

func (p *Pool) encerrando(ctx context.Context) bool {
	return ctx.Err() != nil || fechado(p.parando)
}

func (p *Pool) aguardar(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-p.parando:
	case <-time.After(p.opcoes.IntervaloOcioso):
	}
}

func (p *Pool) conduzir(ctx context.Context, trabalhador string, taskRun domain.TaskRun) {
	p.marcarEmMaos(taskRun.ID, trabalhador)

	ctxDaTarefa, cancelarTarefa := context.WithCancel(ctx)
	var renovacao sync.WaitGroup
	renovacao.Go(func() { p.renovarPrazo(ctxDaTarefa, cancelarTarefa, taskRun.ID, trabalhador) })

	desfecho := p.opcoes.Motor.Conduzir(ctxDaTarefa, trabalhador, taskRun, p.parando)
	cancelarTarefa()
	renovacao.Wait()

	if ctx.Err() != nil {
		return
	}
	p.desmarcarEmMaos(taskRun.ID)
	if desfecho.Liberar {
		p.liberar(ctx, trabalhador, taskRun.ID, desfecho.Espera)
	}
}

func (p *Pool) liberar(ctx context.Context, trabalhador, taskRunID string, espera time.Duration) {
	if err := p.opcoes.Armazenamento.Liberar(ctx, taskRunID, trabalhador, espera); err != nil {
		p.opcoes.Diario.Error("falha ao devolver o task_run à fila", "task_run_id", taskRunID, "trabalhador", trabalhador, "erro", err)
	}
}

func (p *Pool) renovarPrazo(ctx context.Context, cancelarTarefa context.CancelFunc, taskRunID, trabalhador string) {
	intervalo := time.NewTicker(p.opcoes.Prazo / divisorDaRenovacao)
	defer intervalo.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-intervalo.C:
		}

		err := p.opcoes.Armazenamento.RenovarPrazo(ctx, taskRunID, trabalhador, p.opcoes.Prazo)
		switch {
		case errors.Is(err, domain.ErrPrazoPerdido):
			p.opcoes.Diario.Warn("prazo do task_run perdido, etapa interrompida", "task_run_id", taskRunID, "trabalhador", trabalhador)
			cancelarTarefa()
			return
		case err != nil && ctx.Err() == nil:
			p.opcoes.Diario.Warn("falha ao renovar o prazo, nova tentativa no próximo intervalo", "task_run_id", taskRunID, "trabalhador", trabalhador, "erro", err)
		}
	}
}

func (p *Pool) marcarEmMaos(taskRunID, trabalhador string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.emMaos[taskRunID] = trabalhador
}

func (p *Pool) desmarcarEmMaos(taskRunID string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	delete(p.emMaos, taskRunID)
}

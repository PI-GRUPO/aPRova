package integracaopostgres

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/eventos"
	"github.com/Azulpasta/aPRova/internal/platform"
	"github.com/Azulpasta/aPRova/internal/repository"
	"github.com/Azulpasta/aPRova/internal/sandbox"
	"github.com/Azulpasta/aPRova/internal/worker"
)

const prazoDoWorker = time.Second

type leitorDePRFixo struct{}

func (leitorDePRFixo) ContextoDoPullRequest(context.Context, int64, string, string, int) (platform.ContextoPR, error) {
	return platform.ContextoPR{Arquivos: []platform.ArquivoAlterado{{Nome: "go/frete/frete.go", Situacao: "modified"}}, Diff: "diff"}, nil
}

type permissaoDeAdmin struct{}

func (permissaoDeAdmin) PermissaoNoRepositorio(context.Context, int64, string, string, string) (platform.Permissao, error) {
	return platform.PermissaoAdmin, nil
}

type sandboxContado struct {
	execucoes atomic.Int64
	bloquear  bool
	iniciou   chan struct{}
	uma       sync.Once
}

func (s *sandboxContado) Executar(ctx context.Context, _ sandbox.PedidoExecucao) (sandbox.ResultadoSandbox, error) {
	s.execucoes.Add(1)
	if s.bloquear {
		s.uma.Do(func() { close(s.iniciou) })
		<-ctx.Done()
		return sandbox.ResultadoSandbox{SaidaBruta: "interrompido", TipoFalha: sandbox.FalhaInfraestrutura}, nil
	}
	return sandbox.ResultadoSandbox{Compilou: true, TestesPassaram: 4, SaidaBruta: "ok", DuracaoMS: 900, TipoFalha: sandbox.FalhaNenhuma}, nil
}

type avaliadorFixo struct{ nivel domain.NivelRisco }

func (a avaliadorFixo) Avaliar(context.Context, worker.EntradaAvaliacao) (worker.Avaliacao, error) {
	return worker.Avaliacao{
		NivelDeterministico: a.nivel, Nivel: a.nivel,
		Motivos:    []worker.Motivo{{Nome: "area_sensivel", Descricao: "go/frete/frete.go"}},
		Explicacao: "teste de integração do worker",
	}, nil
}

type reciboNoBanco struct {
	repo *repository.RepositorioPostgres
}

func (r reciboNoBanco) Emitir(ctx context.Context, pedido worker.PedidoRecibo) (worker.ReciboEmitido, error) {
	hash := sha256.Sum256([]byte(pedido.TaskRun.ID))
	recibo, err := r.repo.AnexarRecibo(ctx, domain.NovoRecibo{
		TaskRunID: pedido.TaskRun.ID, Conteudo: "recibo de teste", Hash: hash[:], Assinatura: make([]byte, 64),
		IDChave: "chave-de-teste", AssinadoEm: time.Now().UTC(),
	})
	if err != nil {
		return worker.ReciboEmitido{}, err
	}
	return worker.ReciboEmitido{ID: recibo.ID, Hash: recibo.Hash}, nil
}

type publicadorContado struct{ publicacoes atomic.Int64 }

func (p *publicadorContado) Publicar(context.Context, worker.Publicacao) error {
	p.publicacoes.Add(1)
	return nil
}

type eventosDescartados struct{}

func (eventosDescartados) Emitir(*slog.Logger, eventos.Pedido) {}

type montagemDoWorker struct {
	banco      bancoDaFila
	sandbox    *sandboxContado
	publicador *publicadorContado
}

func (m montagemDoWorker) novoPool(t *testing.T, nome string, nivel domain.NivelRisco) *worker.Pool {
	t.Helper()

	motor, err := worker.NovoMotor(worker.Opcoes{
		Armazenamento: m.banco.repo,
		LeitorDePR:    leitorDePRFixo{},
		Permissoes:    permissaoDeAdmin{},
		Sandbox:       m.sandbox,
		Avaliador:     avaliadorFixo{nivel: nivel},
		Recibos:       reciboNoBanco{repo: m.banco.repo},
		Publicador:    m.publicador,
		Eventos:       eventosDescartados{},
		URLPublica:    "https://aprova.example.com",
		EsperaBase:    100 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("montar o motor: %v", err)
	}
	pool, err := worker.NovoPool(worker.OpcoesPool{
		Motor: motor, Armazenamento: m.banco.repo, Nome: nome, Concorrencia: 2,
		Prazo: prazoDoWorker, IntervaloOcioso: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("montar o pool: %v", err)
	}
	return pool
}

func esperarAte(t *testing.T, limite time.Duration, descricao string, condicao func() bool) {
	t.Helper()

	prazo := time.Now().Add(limite)
	for !condicao() {
		if time.Now().After(prazo) {
			t.Fatalf("passou %v sem %s", limite, descricao)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func (b bancoDaFila) estadoDe(t *testing.T, id string) domain.EstadoTaskRun {
	t.Helper()

	taskRun, err := b.repo.BuscarTaskRun(t.Context(), id)
	if err != nil {
		t.Fatalf("buscar task_run: %v", err)
	}
	return taskRun.Estado
}

func TestDoisWorkersDisputandoOMesmoTaskRunSoUmProcessa(t *testing.T) {
	banco := novoBancoDaFila(t)
	montagem := montagemDoWorker{banco: banco, sandbox: &sandboxContado{}, publicador: &publicadorContado{}}
	taskRun := banco.novoTaskRunDoPR(t, 14)

	ctx, cancelar := context.WithCancel(t.Context())
	defer cancelar()
	for indice := range 2 {
		pool := montagem.novoPool(t, fmt.Sprintf("worker-%d", indice), domain.RiscoBaixo)
		go pool.Rodar(ctx)
	}

	esperarAte(t, 10*time.Second, "o task_run concluir", func() bool { return banco.estadoDe(t, taskRun.ID) == domain.EstadoConcluido })

	if execucoes := montagem.sandbox.execucoes.Load(); execucoes != 1 {
		t.Errorf("o sandbox rodou %d vezes, esperava 1", execucoes)
	}
	if publicacoes := montagem.publicador.publicacoes.Load(); publicacoes != 1 {
		t.Errorf("o resultado foi publicado %d vezes, esperava 1", publicacoes)
	}
	if etapas := banco.contar(t, `SELECT count(*) FROM task_steps WHERE task_run_id = $1`, taskRun.ID); etapas != 5 {
		t.Errorf("esperava 5 transições gravadas uma única vez, há %d", etapas)
	}
}

func TestWorkerMortoNoSandboxERetomadoSemDuplicarGateNemRecibo(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)

	travado := montagemDoWorker{banco: banco, sandbox: &sandboxContado{bloquear: true, iniciou: make(chan struct{})}, publicador: &publicadorContado{}}
	ctxDoMorto, matar := context.WithCancel(t.Context())
	go travado.novoPool(t, "worker-que-morre", domain.RiscoAlto).Rodar(ctxDoMorto)
	select {
	case <-travado.sandbox.iniciou:
	case <-time.After(10 * time.Second):
		t.Fatal("o worker não chegou ao sandbox")
	}
	matar()

	if estado := banco.estadoDe(t, taskRun.ID); estado != domain.EstadoTestandoSandbox {
		t.Fatalf("o estado precisa ter sido gravado antes do sandbox, está %s", estado)
	}
	if _, encontrado := banco.reivindicar(t, "intruso-antes-do-prazo"); encontrado {
		t.Fatal("o worker morto não liberou o task_run; ele só pode voltar quando o prazo vencer")
	}

	sobrevivente := montagemDoWorker{banco: banco, sandbox: &sandboxContado{}, publicador: &publicadorContado{}}
	ctx, cancelar := context.WithCancel(t.Context())
	defer cancelar()
	go sobrevivente.novoPool(t, "worker-sobrevivente", domain.RiscoAlto).Rodar(ctx)

	esperarAte(t, 10*time.Second, "outro worker retomar até aguardando_aprovacao", func() bool {
		return banco.estadoDe(t, taskRun.ID) == domain.EstadoAguardandoAprovacao
	})

	if gates := banco.contar(t, `SELECT count(*) FROM risk_gates WHERE task_run_id = $1`, taskRun.ID); gates != 1 {
		t.Errorf("esperava uma linha em risk_gates, há %d", gates)
	}
	if recibos := banco.contar(t, `SELECT count(*) FROM receipts WHERE task_run_id = $1`, taskRun.ID); recibos != 1 {
		t.Errorf("esperava um recibo, há %d", recibos)
	}
	if resultados := banco.contar(t, `SELECT count(*) FROM sandbox_results WHERE task_run_id = $1`, taskRun.ID); resultados != 1 {
		t.Errorf("o worker morto não pode ter gravado resultado; esperava 1 linha em sandbox_results, há %d", resultados)
	}
}

func TestDesligamentoLimpoLiberaOQueNaoTerminou(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)
	montagem := montagemDoWorker{banco: banco, sandbox: &sandboxContado{bloquear: true, iniciou: make(chan struct{})}, publicador: &publicadorContado{}}
	pool := montagem.novoPool(t, "worker-desligado", domain.RiscoBaixo)

	rodando := make(chan struct{})
	go func() {
		pool.Rodar(t.Context())
		close(rodando)
	}()
	<-montagem.sandbox.iniciou

	inicio := time.Now()
	pool.Parar(300 * time.Millisecond)
	if demorou := time.Since(inicio); demorou > 3*time.Second {
		t.Errorf("Parar passou do limite: %v", demorou)
	}
	select {
	case <-rodando:
	case <-time.After(3 * time.Second):
		t.Fatal("Rodar não terminou depois de Parar")
	}

	retomado, encontrado := banco.reivindicar(t, "worker-seguinte")
	if !encontrado || retomado.ID != taskRun.ID {
		t.Fatal("o task_run interrompido precisa voltar na hora para a fila, sem esperar o prazo")
	}
	if retomado.Estado != domain.EstadoTestandoSandbox {
		t.Errorf("o desligamento não muda o estado, está %s", retomado.Estado)
	}
	if resultados := banco.contar(t, `SELECT count(*) FROM sandbox_results WHERE task_run_id = $1`, taskRun.ID); resultados != 0 {
		t.Errorf("a etapa interrompida não pode gravar resultado, há %d", resultados)
	}
}

func TestWorkerRenovaOPrazoEnquantoOSandboxDemora(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)
	montagem := montagemDoWorker{banco: banco, sandbox: &sandboxContado{bloquear: true, iniciou: make(chan struct{})}, publicador: &publicadorContado{}}

	ctx, cancelar := context.WithCancel(t.Context())
	defer cancelar()
	go montagem.novoPool(t, "worker-demorado", domain.RiscoBaixo).Rodar(ctx)
	<-montagem.sandbox.iniciou

	time.Sleep(3 * prazoDoWorker)

	if _, encontrado := banco.reivindicar(t, "intruso"); encontrado {
		t.Errorf("com o worker vivo, o prazo do task_run %s precisa ser renovado", taskRun.ID)
	}
}

package worker

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/eventos"
	"github.com/Azulpasta/aPRova/internal/gate"
	"github.com/Azulpasta/aPRova/internal/platform"
	"github.com/Azulpasta/aPRova/internal/sandbox"
)

type etapaGravada struct {
	de, para domain.EstadoTaskRun
	detalhe  string
}

type armazenamentoFalso struct {
	mutex        sync.Mutex
	taskRuns     map[string]*domain.TaskRun
	ordemCriacao []string
	etapas       map[string][]etapaGravada
	resultados   map[string]map[int]sandbox.ResultadoSandbox
	gates        map[string]domain.GateDeRisco
	recibos      map[string]domain.Recibo
	aprovacoes   []domain.AprovacaoRegistrada
	esperas      []time.Duration
	falharEm     map[string]error
}

func novoArmazenamentoFalso() *armazenamentoFalso {
	return &armazenamentoFalso{
		taskRuns:   map[string]*domain.TaskRun{},
		etapas:     map[string][]etapaGravada{},
		resultados: map[string]map[int]sandbox.ResultadoSandbox{},
		gates:      map[string]domain.GateDeRisco{},
		recibos:    map[string]domain.Recibo{},
		falharEm:   map[string]error{},
	}
}

func (a *armazenamentoFalso) novoTaskRun(numeroPR int) domain.TaskRun {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	indice := len(a.ordemCriacao) + 1
	taskRun := &domain.TaskRun{
		ID:          fmt.Sprintf("00000000-0000-4000-8000-%012d", indice),
		TenantID:    58291734,
		DeliveryID:  fmt.Sprintf("entrega-%d", indice),
		Repositorio: "Azulpasta/aPRova-TESTES",
		NumeroPR:    numeroPR,
		SHAHead:     "3f9a1c7e2b4d6f8091a3c5e7f9b1d3e5a7c9e1f3",
		Autor:       "wlaisw",
		Acao:        "opened",
		Estado:      domain.EstadoPendente,
		CriadoEm:    time.Date(2026, 10, 19, 14, 0, indice, 0, time.UTC),
	}
	a.taskRuns[taskRun.ID] = taskRun
	a.ordemCriacao = append(a.ordemCriacao, taskRun.ID)
	return *taskRun
}

func (a *armazenamentoFalso) falha(operacao string) error {
	return a.falharEm[operacao]
}

func (a *armazenamentoFalso) taskRun(id string) domain.TaskRun {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return *a.taskRuns[id]
}

func (a *armazenamentoFalso) maisNovoNoPR(taskRun domain.TaskRun) bool {
	for _, id := range a.ordemCriacao {
		outro := a.taskRuns[id]
		if outro.Repositorio == taskRun.Repositorio && outro.NumeroPR == taskRun.NumeroPR && outro.CriadoEm.After(taskRun.CriadoEm) {
			return true
		}
	}
	return false
}

func (a *armazenamentoFalso) reivindicavel(taskRun *domain.TaskRun) bool {
	if taskRun.Estado.Terminal() || taskRun.Estado == domain.EstadoGerando || taskRun.ReivindicadoPor != "" {
		return false
	}
	if taskRun.Estado != domain.EstadoAguardandoAprovacao {
		return true
	}
	if a.maisNovoNoPR(*taskRun) {
		return true
	}
	return slices.ContainsFunc(a.aprovacoes, func(aprovacao domain.AprovacaoRegistrada) bool {
		return aprovacao.TaskRunID == taskRun.ID && aprovacao.ID > taskRun.UltimaAprovacaoVista
	})
}

func (a *armazenamentoFalso) Reivindicar(_ context.Context, trabalhador string, _ time.Duration) (domain.TaskRun, bool, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	for _, id := range a.ordemCriacao {
		taskRun := a.taskRuns[id]
		if a.reivindicavel(taskRun) {
			taskRun.ReivindicadoPor = trabalhador
			return *taskRun, true, nil
		}
	}
	return domain.TaskRun{}, false, nil
}

func (a *armazenamentoFalso) RenovarPrazo(_ context.Context, taskRunID, trabalhador string, _ time.Duration) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.taskRuns[taskRunID].ReivindicadoPor != trabalhador {
		return domain.ErrPrazoPerdido
	}
	return nil
}

func (a *armazenamentoFalso) Liberar(_ context.Context, taskRunID, trabalhador string, espera time.Duration) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.taskRuns[taskRunID].ReivindicadoPor != trabalhador {
		return domain.ErrPrazoPerdido
	}
	a.taskRuns[taskRunID].ReivindicadoPor = ""
	if espera > 0 {
		a.esperas = append(a.esperas, espera)
	}
	return nil
}

func (a *armazenamentoFalso) TransicionarReivindicado(_ context.Context, trabalhador string, transicao domain.Transicao) (domain.TaskRun, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if !transicao.De.PodeIrPara(transicao.Para) {
		return domain.TaskRun{}, fmt.Errorf("%w: %s para %s", domain.ErrTransicaoProibida, transicao.De, transicao.Para)
	}
	taskRun := a.taskRuns[transicao.TaskRunID]
	if taskRun.Estado != transicao.De || taskRun.ReivindicadoPor != trabalhador {
		return domain.TaskRun{}, domain.ErrTransicaoConcorrente
	}

	taskRun.Estado = transicao.Para
	taskRun.TentativasDaEtapa = 0
	if transicao.Para == domain.EstadoAguardandoAprovacao {
		for _, aprovacao := range a.aprovacoes {
			if aprovacao.TaskRunID == taskRun.ID {
				taskRun.UltimaAprovacaoVista = max(taskRun.UltimaAprovacaoVista, aprovacao.ID)
			}
		}
	}
	a.etapas[taskRun.ID] = append(a.etapas[taskRun.ID], etapaGravada{transicao.De, transicao.Para, transicao.Detalhe})
	return *taskRun, nil
}

func (a *armazenamentoFalso) IncrementarTentativasDaEtapa(_ context.Context, taskRunID, trabalhador string) (int, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	taskRun := a.taskRuns[taskRunID]
	if taskRun.ReivindicadoPor != trabalhador {
		return 0, domain.ErrPrazoPerdido
	}
	taskRun.TentativasDaEtapa++
	return taskRun.TentativasDaEtapa, nil
}

func (a *armazenamentoFalso) GravarResultadoDaTentativa(_ context.Context, taskRunID string, tentativa int, resultado sandbox.ResultadoSandbox) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if err := a.falha("GravarResultadoDaTentativa"); err != nil {
		return err
	}
	if a.resultados[taskRunID] == nil {
		a.resultados[taskRunID] = map[int]sandbox.ResultadoSandbox{}
	}
	if _, existe := a.resultados[taskRunID][tentativa]; existe {
		return domain.ErrDuplicado
	}
	a.resultados[taskRunID][tentativa] = resultado
	return nil
}

func (a *armazenamentoFalso) UltimoResultadoSandbox(_ context.Context, taskRunID string) (int, sandbox.ResultadoSandbox, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	ultima := 0
	for tentativa := range a.resultados[taskRunID] {
		ultima = max(ultima, tentativa)
	}
	if ultima == 0 {
		return 0, sandbox.ResultadoSandbox{}, domain.ErrNaoEncontrado
	}
	return ultima, a.resultados[taskRunID][ultima], nil
}

func (a *armazenamentoFalso) tentativasGravadas(taskRunID string) []int {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	var tentativas []int
	for tentativa := range a.resultados[taskRunID] {
		tentativas = append(tentativas, tentativa)
	}
	slices.Sort(tentativas)
	return tentativas
}

func (a *armazenamentoFalso) BuscarGateDeRisco(_ context.Context, taskRunID string) (domain.GateDeRisco, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	gate, existe := a.gates[taskRunID]
	if !existe {
		return domain.GateDeRisco{}, domain.ErrNaoEncontrado
	}
	return gate, nil
}

func (a *armazenamentoFalso) GravarGateDeRisco(_ context.Context, gate domain.GateDeRisco) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if _, existe := a.gates[gate.TaskRunID]; existe {
		return domain.ErrDuplicado
	}
	a.gates[gate.TaskRunID] = gate
	return nil
}

func (a *armazenamentoFalso) BuscarReciboPorTaskRun(_ context.Context, taskRunID string) (domain.Recibo, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	recibo, existe := a.recibos[taskRunID]
	if !existe {
		return domain.Recibo{}, domain.ErrNaoEncontrado
	}
	return recibo, nil
}

func (a *armazenamentoFalso) registrarAprovacao(taskRunID string, decisao domain.DecisaoAprovacao, revisor string, origem domain.OrigemAprovacao) domain.AprovacaoRegistrada {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	aprovacao := domain.AprovacaoRegistrada{
		ID: int64(len(a.aprovacoes) + 1),
		Aprovacao: domain.Aprovacao{
			TaskRunID: taskRunID, DeliveryID: fmt.Sprintf("review-%d", len(a.aprovacoes)+1), Decisao: decisao,
			Revisor: revisor, Origem: origem, DecididaEm: time.Date(2026, 10, 19, 17, 48, 12, 0, time.UTC),
		},
	}
	a.aprovacoes = append(a.aprovacoes, aprovacao)
	return aprovacao
}

func (a *armazenamentoFalso) ProximaAprovacao(_ context.Context, taskRunID string, depoisDe int64) (domain.AprovacaoRegistrada, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	for _, aprovacao := range a.aprovacoes {
		if aprovacao.TaskRunID == taskRunID && aprovacao.ID > depoisDe {
			return aprovacao, nil
		}
	}
	return domain.AprovacaoRegistrada{}, domain.ErrNaoEncontrado
}

func (a *armazenamentoFalso) AvancarCursorDeAprovacao(_ context.Context, taskRunID, trabalhador string, ate int64) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	taskRun := a.taskRuns[taskRunID]
	if taskRun.ReivindicadoPor != trabalhador {
		return domain.ErrPrazoPerdido
	}
	taskRun.UltimaAprovacaoVista = max(taskRun.UltimaAprovacaoVista, ate)
	return nil
}

func (a *armazenamentoFalso) ExisteTaskRunMaisNovo(_ context.Context, taskRun domain.TaskRun) (bool, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.maisNovoNoPR(taskRun), nil
}

func (a *armazenamentoFalso) estadosPercorridos(taskRunID string) []domain.EstadoTaskRun {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	var estados []domain.EstadoTaskRun
	for _, etapa := range a.etapas[taskRunID] {
		estados = append(estados, etapa.para)
	}
	return estados
}

func (a *armazenamentoFalso) ultimaEtapa(taskRunID string) etapaGravada {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	etapas := a.etapas[taskRunID]
	return etapas[len(etapas)-1]
}

type registroDeChamadas struct {
	mutex    sync.Mutex
	chamadas []string
}

func (r *registroDeChamadas) anotar(chamada string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.chamadas = append(r.chamadas, chamada)
}

func (r *registroDeChamadas) lista() []string {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return slices.Clone(r.chamadas)
}

func (r *registroDeChamadas) quantas(chamada string) int {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	total := 0
	for _, anotada := range r.chamadas {
		if anotada == chamada {
			total++
		}
	}
	return total
}

type leitorFalso struct {
	erros    []error
	leituras int
}

func (l *leitorFalso) ContextoDoPullRequest(context.Context, int64, string, string, int) (platform.ContextoPR, error) {
	l.leituras++
	if len(l.erros) > 0 {
		erro := l.erros[0]
		l.erros = l.erros[1:]
		return platform.ContextoPR{}, erro
	}
	return platform.ContextoPR{
		Arquivos: []platform.ArquivoAlterado{{Nome: "go/frete/frete.go", Situacao: "modified", Adicoes: 1, Remocoes: 1}},
		Diff:     "diff --git a/go/frete/frete.go b/go/frete/frete.go",
	}, nil
}

type permissoesFalsas map[string]platform.Permissao

func (p permissoesFalsas) PermissaoNoRepositorio(_ context.Context, _ int64, _, _, login string) (platform.Permissao, error) {
	permissao, conhecido := p[login]
	if !conhecido {
		return platform.PermissaoNenhuma, nil
	}
	return permissao, nil
}

type sandboxFalso struct {
	mutex      sync.Mutex
	resultados []sandbox.ResultadoSandbox
	pedidos    []sandbox.PedidoExecucao
	bloquear   bool
	iniciou    chan struct{}
}

func (s *sandboxFalso) Executar(ctx context.Context, pedido sandbox.PedidoExecucao) (sandbox.ResultadoSandbox, error) {
	s.mutex.Lock()
	s.pedidos = append(s.pedidos, pedido)
	bloquear := s.bloquear
	var resultado sandbox.ResultadoSandbox
	if len(s.resultados) > 0 {
		resultado = s.resultados[0]
		s.resultados = s.resultados[1:]
	}
	s.mutex.Unlock()

	if bloquear {
		if s.iniciou != nil {
			close(s.iniciou)
		}
		<-ctx.Done()
		return sandbox.ResultadoSandbox{SaidaBruta: "cancelado", TipoFalha: sandbox.FalhaInfraestrutura}, nil
	}
	return resultado, nil
}

func (s *sandboxFalso) execucoes() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return len(s.pedidos)
}

var (
	sucessoNoSandbox = sandbox.ResultadoSandbox{Compilou: true, TestesPassaram: 4, SaidaBruta: "ok", DuracaoMS: 900, TipoFalha: sandbox.FalhaNenhuma}
	falhaDeInfra     = sandbox.ResultadoSandbox{SaidaBruta: "a execução do job não começou em 120 s", TipoFalha: sandbox.FalhaInfraestrutura}
	testeFalhou      = sandbox.ResultadoSandbox{Compilou: true, TestesPassaram: 3, TestesFalharam: 1, SaidaBruta: "--- FAIL: TestCalcularPesoZero", TipoFalha: sandbox.FalhaTeste}
)

type avaliadorFalso struct {
	nivel        domain.NivelRisco
	dependencias []domain.DependenciaNova
	erro         error
	avaliacoes   int
	entradas     []gate.Entrada
}

func (a *avaliadorFalso) Avaliar(_ context.Context, entrada gate.Entrada) (gate.Resultado, error) {
	a.avaliacoes++
	a.entradas = append(a.entradas, entrada)
	if a.erro != nil {
		return gate.Resultado{}, a.erro
	}

	resultado := gate.Resultado{Nivel: a.nivel, Motivos: []domain.MotivoDeRisco{}, Dependencias: a.dependencias}
	if a.nivel != domain.RiscoBaixo {
		resultado.Motivos = append(resultado.Motivos, domain.MotivoDeRisco{
			ID: "area_sensivel", Nivel: a.nivel, Descricao: "o PR altera um caminho sensível", Evidencias: []string{"go/frete/frete.go"},
		})
	}
	return resultado, nil
}

type emissorDeReciboFalso struct {
	armazenamento *armazenamentoFalso
	chamadas      *registroDeChamadas
}

func (e *emissorDeReciboFalso) Emitir(_ context.Context, pedido PedidoRecibo) (ReciboEmitido, error) {
	e.chamadas.anotar("recibo")
	hash := sha256.Sum256([]byte(pedido.TaskRun.ID))
	recibo := ReciboEmitido{ID: "2b6d8f0a-4c1e-4a3b-9d5f-7e8a0b1c2d3e", Hash: hash[:]}

	e.armazenamento.mutex.Lock()
	defer e.armazenamento.mutex.Unlock()
	if existente, existe := e.armazenamento.recibos[pedido.TaskRun.ID]; existe {
		return ReciboEmitido{ID: existente.ID, Hash: existente.Hash}, nil
	}
	e.armazenamento.recibos[pedido.TaskRun.ID] = domain.Recibo{NovoRecibo: domain.NovoRecibo{ID: recibo.ID, TaskRunID: pedido.TaskRun.ID, Hash: recibo.Hash}}
	return recibo, nil
}

type publicadorFalso struct {
	chamadas   *registroDeChamadas
	erros      []error
	publicados []Publicacao
}

func (p *publicadorFalso) Publicar(_ context.Context, publicacao Publicacao) error {
	p.chamadas.anotar("publicacao")
	if len(p.erros) > 0 {
		erro := p.erros[0]
		p.erros = p.erros[1:]
		return erro
	}
	p.publicados = append(p.publicados, publicacao)
	return nil
}

type emissorDeEventosFalso struct {
	mutex   sync.Mutex
	pedidos []eventos.Pedido
}

func (e *emissorDeEventosFalso) Emitir(_ *slog.Logger, pedido eventos.Pedido) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.pedidos = append(e.pedidos, pedido)
}

func (e *emissorDeEventosFalso) tipos() []eventos.TipoEvento {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	var tipos []eventos.TipoEvento
	for _, pedido := range e.pedidos {
		tipos = append(tipos, pedido.Tipo)
	}
	return tipos
}

func (e *emissorDeEventosFalso) doTipo(tipo eventos.TipoEvento) (eventos.Pedido, bool) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	for _, pedido := range e.pedidos {
		if pedido.Tipo == tipo {
			return pedido, true
		}
	}
	return eventos.Pedido{}, false
}

var errGitHubFora = fmt.Errorf("%w: GET /repos/Azulpasta/aPRova-TESTES/pulls/14/files respondeu 502", platform.ErrGitHubIndisponivel)

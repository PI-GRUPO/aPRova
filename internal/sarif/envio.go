package sarif

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/platform"
	"github.com/PI-GRUPO/aPRova/internal/worker"
)

const (
	maximoDeTentativasPadrao   = 3
	esperaBasePadrao           = 5 * time.Second
	intervaloDaConsultaPadrao  = 5 * time.Second
	prazoDoProcessamentoPadrao = 2 * time.Minute
	tamanhoMaximoDoDetalhe     = 2000
	detalheDoPrazoEsgotado     = "prazo esgotado antes de o GitHub concluir o processamento"
	motivoDoDesligamento       = "o GitHub respondeu 403 ao envio do SARIF: confira se o app tem security_events: write " +
		"e se o code scanning está disponível no repositório"
)

// ErrDependenciaAusente indica enviador montado sem uma dependência
// obrigatória.
var ErrDependenciaAusente = errors.New("sarif: dependência ausente")

// GitHub é o que o enviador usa da API do GitHub. O platform.Cliente é a
// implementação.
type GitHub interface {
	ArquivosDoPullRequest(ctx context.Context, installationID int64, dono, repositorio string, numero int) ([]platform.ArquivoAlterado, error)
	EnviarSARIF(ctx context.Context, installationID int64, dono, repositorio string, envio platform.EnvioSARIF) (string, error)
	ProcessamentoDoSARIF(ctx context.Context, installationID int64, dono, repositorio, id string) (platform.ProcessamentoSARIF, error)
}

// Registro guarda o envio de cada task_run e os repositórios em que o envio
// foi desligado. GravarEnvioSARIF nunca troca o id de um envio aceito.
type Registro interface {
	EnvioSARIFDoTaskRun(ctx context.Context, taskRunID string) (domain.EnvioSARIF, error)
	GravarEnvioSARIF(ctx context.Context, envio domain.EnvioSARIF) error
	EnvioSARIFDesligado(ctx context.Context, tenantID int64, repositorio string) (bool, error)
	DesligarEnvioSARIF(ctx context.Context, tenantID int64, repositorio, motivo string) error
}

// OpcoesEnviador reúne as dependências e os ajustes do enviador. Versao é a
// versão do binário, declarada no SARIF. MaxTentativas e EsperaBase limitam as
// novas tentativas de envio, com espera que dobra a cada uma.
// IntervaloDaConsulta e PrazoDoProcessamento limitam a espera pelo
// processamento do GitHub.
type OpcoesEnviador struct {
	GitHub               GitHub
	Registro             Registro
	Versao               string
	MaxTentativas        int
	EsperaBase           time.Duration
	IntervaloDaConsulta  time.Duration
	PrazoDoProcessamento time.Duration
	Diario               *slog.Logger
}

// Enviador envia ao code scanning o SARIF de cada análise publicada. Nada do
// que ele faz volta para o motor: falha de envio fica no registro e no log.
type Enviador struct {
	opcoes   OpcoesEnviador
	ctx      context.Context
	cancelar context.CancelFunc
	emCurso  sync.WaitGroup

	mu        sync.Mutex
	enviando  map[string]struct{}
	encerrado bool
}

type envio struct {
	*Enviador
	ctx         context.Context
	publicacao  worker.Publicacao
	dono        string
	repositorio string
	diario      *slog.Logger
}

// NovoEnviador constrói o enviador. Sem GitHub ou sem Registro devolve
// ErrDependenciaAusente.
func NovoEnviador(opcoes OpcoesEnviador) (*Enviador, error) {
	if opcoes.GitHub == nil || opcoes.Registro == nil {
		return nil, fmt.Errorf("%w: GitHub e Registro são obrigatórios", ErrDependenciaAusente)
	}
	if opcoes.MaxTentativas <= 0 {
		opcoes.MaxTentativas = maximoDeTentativasPadrao
	}
	if opcoes.EsperaBase <= 0 {
		opcoes.EsperaBase = esperaBasePadrao
	}
	if opcoes.IntervaloDaConsulta <= 0 {
		opcoes.IntervaloDaConsulta = intervaloDaConsultaPadrao
	}
	if opcoes.PrazoDoProcessamento <= 0 {
		opcoes.PrazoDoProcessamento = prazoDoProcessamentoPadrao
	}
	if opcoes.Diario == nil {
		opcoes.Diario = slog.Default()
	}

	ctx, cancelar := context.WithCancel(context.Background())
	return &Enviador{opcoes: opcoes, ctx: ctx, cancelar: cancelar, enviando: map[string]struct{}{}}, nil
}

// Enviar dispara em segundo plano o envio do SARIF da publicação e devolve na
// hora. Task_run com envio já aceito não é enviado de novo; com envio em
// curso, o pedido é ignorado.
func (e *Enviador) Enviar(publicacao worker.Publicacao) {
	if !e.reservar(publicacao.TaskRun.ID) {
		return
	}

	go func() {
		defer e.emCurso.Done()
		defer e.liberar(publicacao.TaskRun.ID)
		e.novoEnvio(publicacao).executar()
	}()
}

// Aguardar espera os envios em curso terminarem.
func (e *Enviador) Aguardar() {
	e.emCurso.Wait()
}

// Encerrar interrompe os envios em curso, espera que parem e recusa os
// próximos. Um envio interrompido fica como estava no registro.
func (e *Enviador) Encerrar() {
	e.mu.Lock()
	e.encerrado = true
	e.mu.Unlock()

	e.cancelar()
	e.emCurso.Wait()
}

func (e *Enviador) reservar(taskRunID string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, emCurso := e.enviando[taskRunID]; emCurso || e.encerrado {
		return false
	}
	e.enviando[taskRunID] = struct{}{}
	e.emCurso.Add(1)
	return true
}

func (e *Enviador) liberar(taskRunID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.enviando, taskRunID)
}

func (e *Enviador) novoEnvio(publicacao worker.Publicacao) *envio {
	dono, repositorio, _ := strings.Cut(publicacao.TaskRun.Repositorio, "/")
	return &envio{
		Enviador:    e,
		ctx:         e.ctx,
		publicacao:  publicacao,
		dono:        dono,
		repositorio: repositorio,
		diario:      e.opcoes.Diario.With("task_run_id", publicacao.TaskRun.ID, "repositorio", publicacao.TaskRun.Repositorio),
	}
}

func (v *envio) executar() {
	taskRun := v.publicacao.TaskRun

	desligado, err := v.opcoes.Registro.EnvioSARIFDesligado(v.ctx, taskRun.TenantID, taskRun.Repositorio)
	if err != nil {
		v.diario.Warn("SARIF não enviado: não foi possível consultar se o envio está desligado", "erro", err)
		return
	}
	if desligado {
		v.diario.Info("SARIF não enviado: envio desligado para o repositório")
		return
	}

	anterior, err := v.opcoes.Registro.EnvioSARIFDoTaskRun(v.ctx, taskRun.ID)
	switch {
	case errors.Is(err, domain.ErrNaoEncontrado), err == nil && anterior.IDDoEnvio == "":
		v.enviarEAcompanhar()
	case err != nil:
		v.diario.Warn("SARIF não enviado: não foi possível ler o registro do envio", "erro", err)
	case anterior.Estado == domain.EnvioSARIFAceito:
		v.acompanhar(anterior.IDDoEnvio)
	default:
		v.diario.Info("SARIF já enviado para este task_run", "id_do_envio", anterior.IDDoEnvio, "estado", anterior.Estado)
	}
}

func (v *envio) enviarEAcompanhar() {
	documento, err := Montar(Entrada{
		Gate:     v.publicacao.Gate,
		Arquivos: v.arquivosDoPR(),
		Recibo:   Recibo{ID: v.publicacao.Recibo.ID, Hash: v.publicacao.Recibo.Hash},
		Versao:   v.opcoes.Versao,
	})
	if err != nil {
		v.registrarFalha(err.Error())
		return
	}

	id, err := v.enviarComNovasTentativas(documento)
	if err != nil {
		v.tratarFalhaDoEnvio(err)
		return
	}

	v.gravar(domain.EnvioSARIF{IDDoEnvio: id, Estado: domain.EnvioSARIFAceito})
	v.diario.Info("SARIF aceito pelo GitHub", "id_do_envio", id)
	v.acompanhar(id)
}

func (v *envio) arquivosDoPR() []platform.ArquivoAlterado {
	taskRun := v.publicacao.TaskRun
	arquivos, err := v.opcoes.GitHub.ArquivosDoPullRequest(v.ctx, taskRun.TenantID, v.dono, v.repositorio, taskRun.NumeroPR)
	if err != nil {
		v.diario.Warn("arquivos do PR indisponíveis, os alertas do SARIF vão para a linha 1", "erro", err)
		return nil
	}
	return arquivos
}

func (v *envio) enviarComNovasTentativas(documento []byte) (string, error) {
	taskRun := v.publicacao.TaskRun
	pedido := platform.EnvioSARIF{
		CommitSHA: taskRun.SHAHead,
		Ref:       "refs/pull/" + strconv.Itoa(taskRun.NumeroPR) + "/head",
		Documento: documento,
	}

	for tentativa := 1; ; tentativa++ {
		id, err := v.opcoes.GitHub.EnviarSARIF(v.ctx, taskRun.TenantID, v.dono, v.repositorio, pedido)
		if err == nil {
			return id, nil
		}
		if !transitorio(err) || tentativa >= v.opcoes.MaxTentativas {
			return "", err
		}

		espera := v.opcoes.EsperaBase << (tentativa - 1)
		v.diario.Warn("falha transitória no envio do SARIF, nova tentativa depois da espera", "tentativa", tentativa, "espera", espera, "erro", err)
		if !v.esperar(espera) {
			return "", v.ctx.Err()
		}
	}
}

func transitorio(err error) bool {
	return errors.Is(err, platform.ErrGitHubIndisponivel) || errors.Is(err, platform.ErrLimiteDeRequisicoes)
}

func (v *envio) esperar(duracao time.Duration) bool {
	temporizador := time.NewTimer(duracao)
	defer temporizador.Stop()

	select {
	case <-temporizador.C:
		return true
	case <-v.ctx.Done():
		return false
	}
}

func (v *envio) tratarFalhaDoEnvio(err error) {
	if v.ctx.Err() != nil {
		v.diario.Warn("envio do SARIF interrompido pelo encerramento do worker")
		return
	}

	v.registrarFalha(err.Error())
	if !errors.Is(err, platform.ErrSemPermissao) {
		return
	}

	taskRun := v.publicacao.TaskRun
	if err := v.opcoes.Registro.DesligarEnvioSARIF(v.ctx, taskRun.TenantID, taskRun.Repositorio, motivoDoDesligamento); err != nil {
		v.diario.Warn("não foi possível desligar o envio do SARIF para o repositório", "erro", err)
		return
	}
	v.diario.Warn("envio do SARIF desligado para o repositório", "motivo", motivoDoDesligamento)
}

func (v *envio) registrarFalha(diagnostico string) {
	v.diario.Warn("SARIF não enviado, a análise segue sem ele", "diagnostico", diagnostico)
	v.gravar(domain.EnvioSARIF{Estado: domain.EnvioSARIFFalhou, Detalhe: diagnostico})
}

func (v *envio) acompanhar(id string) {
	taskRun := v.publicacao.TaskRun
	ctx, cancelar := context.WithTimeout(v.ctx, v.opcoes.PrazoDoProcessamento)
	defer cancelar()

	for {
		processamento, err := v.opcoes.GitHub.ProcessamentoDoSARIF(ctx, taskRun.TenantID, v.dono, v.repositorio, id)
		switch {
		case err != nil:
			v.diario.Warn("consulta do processamento do SARIF falhou, nova consulta depois do intervalo", "erro", err)
		case processamento.Situacao == platform.SARIFProcessado:
			v.diario.Info("SARIF processado pelo GitHub", "id_do_envio", id)
			v.gravar(domain.EnvioSARIF{IDDoEnvio: id, Estado: domain.EnvioSARIFProcessado})
			return
		case processamento.Situacao == platform.SARIFRecusado:
			detalhe := strings.Join(processamento.Erros, "; ")
			v.diario.Warn("SARIF recusado pelo GitHub no processamento, a análise segue sem ele", "id_do_envio", id, "erros", detalhe)
			v.gravar(domain.EnvioSARIF{IDDoEnvio: id, Estado: domain.EnvioSARIFRecusado, Detalhe: detalhe})
			return
		}

		if !esperarAte(ctx, v.opcoes.IntervaloDaConsulta) {
			v.registrarPrazoEsgotado(id)
			return
		}
	}
}

func esperarAte(ctx context.Context, duracao time.Duration) bool {
	temporizador := time.NewTimer(duracao)
	defer temporizador.Stop()

	select {
	case <-temporizador.C:
		return ctx.Err() == nil
	case <-ctx.Done():
		return false
	}
}

func (v *envio) registrarPrazoEsgotado(id string) {
	if v.ctx.Err() != nil {
		return
	}
	v.diario.Warn("o GitHub não concluiu o processamento do SARIF dentro do prazo", "id_do_envio", id, "prazo", v.opcoes.PrazoDoProcessamento)
	v.gravar(domain.EnvioSARIF{IDDoEnvio: id, Estado: domain.EnvioSARIFAceito, Detalhe: detalheDoPrazoEsgotado})
}

func (v *envio) gravar(registro domain.EnvioSARIF) {
	registro.TaskRunID = v.publicacao.TaskRun.ID
	registro.Detalhe = truncar(registro.Detalhe, tamanhoMaximoDoDetalhe)

	ctx, cancelar := context.WithTimeout(context.WithoutCancel(v.ctx), esperaBasePadrao)
	defer cancelar()
	if err := v.opcoes.Registro.GravarEnvioSARIF(ctx, registro); err != nil {
		v.diario.Warn("não foi possível gravar o registro do envio do SARIF", "estado", registro.Estado, "erro", err)
	}
}

func truncar(texto string, maximo int) string {
	runas := []rune(texto)
	if len(runas) <= maximo {
		return texto
	}
	return string(runas[:maximo])
}

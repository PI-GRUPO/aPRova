package explicacao

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

const (
	timeoutPadrao = 5 * time.Second

	// ResultadoReusada é o rótulo da métrica quando a explicação já gravada é
	// reaproveitada sem chamar o modelo.
	ResultadoReusada = "reusada"
)

// ErrConfiguracao indica explicador ou cliente montado sem algo obrigatório.
var ErrConfiguracao = errors.New("explicacao: configuração incompleta")

// Modelo gera texto a partir da instrução do sistema e do conteúdo.
type Modelo interface {
	Gerar(ctx context.Context, sistema, conteudo string) (string, error)
}

// Registro guarda a única tentativa de explicação de cada task_run. Sem
// tentativa gravada, ExplicacaoDoTaskRun devolve domain.ErrNaoEncontrado.
type Registro interface {
	ExplicacaoDoTaskRun(ctx context.Context, taskRunID string) (domain.Explicacao, error)
	GravarExplicacao(ctx context.Context, explicacao domain.Explicacao) error
}

// Metricas conta cada explicação por resultado: gerada, descartada, falha ou
// reusada.
type Metricas interface {
	Explicacao(resultado string)
}

// Opcoes configura o explicador. Sem Timeout usa 5 segundos; Metricas é
// opcional.
type Opcoes struct {
	Modelo       Modelo
	NomeDoModelo string
	Registro     Registro
	Metricas     Metricas
	Timeout      time.Duration
}

// Explicador chama o modelo uma única vez por task_run, valida a resposta e
// grava o resultado, mesmo quando é descarte ou falha.
type Explicador struct {
	opcoes Opcoes
}

type respostaDoModelo struct {
	texto string
	err   error
}

// NovoExplicador monta o explicador. Sem modelo, registro ou nome do modelo
// devolve ErrConfiguracao.
func NovoExplicador(opcoes Opcoes) (*Explicador, error) {
	if opcoes.Modelo == nil || opcoes.Registro == nil || strings.TrimSpace(opcoes.NomeDoModelo) == "" {
		return nil, fmt.Errorf("%w: o explicador exige modelo, registro e nome do modelo", ErrConfiguracao)
	}
	if opcoes.Timeout <= 0 {
		opcoes.Timeout = timeoutPadrao
	}
	return &Explicador{opcoes: opcoes}, nil
}

// Explicar devolve a explicação validada a publicar no comentário, ou vazio
// para o comentário usar só o texto fixo. Uma tentativa já gravada é
// reaproveitada sem chamar o modelo. Nunca espera o modelo além do timeout.
func (e *Explicador) Explicar(ctx context.Context, taskRunID string, gate domain.GateDeRisco, resultado sandbox.ResultadoSandbox) string {
	gravada, err := e.opcoes.Registro.ExplicacaoDoTaskRun(ctx, taskRunID)
	if err == nil {
		e.contar(ResultadoReusada)
		return gravada.Texto
	}
	if !errors.Is(err, domain.ErrNaoEncontrado) {
		slog.WarnContext(ctx, "explicação por IA não lida, o comentário usa o texto fixo", "erro", err)
		return ""
	}

	nova := e.gerar(ctx, taskRunID, NovaEntrada(gate, resultado))
	if err := e.opcoes.Registro.GravarExplicacao(ctx, nova); err != nil {
		slog.WarnContext(ctx, "explicação por IA não gravada", "erro", err)
	}
	e.contar(string(nova.Resultado))
	if nova.Resultado != domain.ExplicacaoGerada {
		slog.WarnContext(ctx, "explicação por IA não publicada, o comentário usa o texto fixo",
			"resultado", nova.Resultado, "motivo", nova.Motivo, "modelo", nova.Modelo)
	}
	return nova.Texto
}

func (e *Explicador) gerar(ctx context.Context, taskRunID string, entrada Entrada) domain.Explicacao {
	explicacao := domain.Explicacao{TaskRunID: taskRunID, Modelo: e.opcoes.NomeDoModelo}
	conteudo, err := entrada.JSON()
	if err != nil {
		return falhou(explicacao, err)
	}

	resposta := e.chamarModelo(ctx, conteudo)
	if resposta.err != nil {
		return falhou(explicacao, resposta.err)
	}
	texto, err := Validar(resposta.texto, entrada)
	if err != nil {
		explicacao.Resultado, explicacao.Motivo = domain.ExplicacaoDescartada, err.Error()
		return explicacao
	}
	explicacao.Resultado, explicacao.Texto = domain.ExplicacaoGerada, texto
	return explicacao
}

func (e *Explicador) chamarModelo(ctx context.Context, conteudo string) respostaDoModelo {
	ctxDoModelo, cancelar := context.WithTimeout(ctx, e.opcoes.Timeout)
	defer cancelar()

	pronta := make(chan respostaDoModelo, 1)
	go func() {
		texto, err := e.opcoes.Modelo.Gerar(ctxDoModelo, instrucaoDoSistema, conteudo)
		pronta <- respostaDoModelo{texto: texto, err: err}
	}()
	select {
	case resposta := <-pronta:
		return resposta
	case <-ctxDoModelo.Done():
		return respostaDoModelo{err: fmt.Errorf("modelo sem resposta em %s: %w", e.opcoes.Timeout, ctxDoModelo.Err())}
	}
}

func falhou(explicacao domain.Explicacao, err error) domain.Explicacao {
	explicacao.Resultado, explicacao.Motivo = domain.ExplicacaoFalhou, err.Error()
	return explicacao
}

func (e *Explicador) contar(resultado string) {
	if e.opcoes.Metricas != nil {
		e.opcoes.Metricas.Explicacao(resultado)
	}
}

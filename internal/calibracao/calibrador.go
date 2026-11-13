package calibracao

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/embedding"
	"github.com/PI-GRUPO/aPRova/internal/platform"
)

// TimeoutPadrao é o tempo máximo da calibração, embedding e busca juntos,
// quando nenhum é configurado.
const TimeoutPadrao = 5 * time.Second

// ErrCalibradorIncompleto indica calibrador montado sem embedder ou sem
// histórico.
var ErrCalibradorIncompleto = errors.New("calibracao: calibrador sem embedder ou sem histórico")

// Historico devolve as análises passadas do tenant mais parecidas com o
// embedding, com o desfecho de cada uma.
type Historico interface {
	VizinhosParecidos(ctx context.Context, consulta domain.ConsultaDeVizinhos) ([]domain.VizinhoDaCalibracao, error)
}

// OpcoesCalibrador reúne as dependências e os ajustes da calibração. Regra
// zerada usa RegraPadrao; Timeout zero usa 5 segundos, e vale para o embedding
// e a busca juntos.
type OpcoesCalibrador struct {
	Embedder  embedding.Embedder
	Historico Historico
	Regra     Regra
	Timeout   time.Duration
}

// Calibrador gera a assinatura e o vetor da análise, busca os vizinhos e
// aplica a Regra. Nunca devolve erro: qualquer falha mantém o nível do gate e
// fica registrada na razão.
type Calibrador struct {
	embedder  embedding.Embedder
	historico Historico
	regra     Regra
	timeout   time.Duration
}

// Pedido é a análise a calibrar: o task_run, a decisão do gate determinístico
// e os arquivos alterados de onde sai a assinatura.
type Pedido struct {
	TaskRun  domain.TaskRun
	Nivel    domain.NivelRisco
	Motivos  []domain.MotivoDeRisco
	Arquivos []platform.ArquivoAlterado
}

// Resultado é a decisão final e a assinatura a gravar no histórico. Assinatura
// é nula quando o embedding não foi gerado.
type Resultado struct {
	Decisao
	Assinatura *domain.AssinaturaDoPR
}

// NovoCalibrador monta o calibrador. Sem embedder ou sem histórico devolve
// ErrCalibradorIncompleto.
func NovoCalibrador(opcoes OpcoesCalibrador) (*Calibrador, error) {
	if opcoes.Embedder == nil || opcoes.Historico == nil {
		return nil, ErrCalibradorIncompleto
	}
	if opcoes.Regra == (Regra{}) {
		opcoes.Regra = RegraPadrao()
	}
	if opcoes.Timeout <= 0 {
		opcoes.Timeout = TimeoutPadrao
	}
	return &Calibrador{embedder: opcoes.Embedder, historico: opcoes.Historico, regra: opcoes.Regra, timeout: opcoes.Timeout}, nil
}

// Calibrar gera o vetor de toda análise avaliada e, quando a análise é
// calibrável, busca os vizinhos e aplica a Regra. Leva no máximo o timeout
// configurado, mesmo que o embedder ou o histórico não respeitem o contexto.
func (c *Calibrador) Calibrar(ctx context.Context, pedido Pedido) Resultado {
	ctx, cancelar := context.WithTimeout(ctx, c.timeout)
	defer cancelar()

	assinatura, errDoEmbedding := c.assinar(ctx, pedido)
	mantido := func(estado domain.EstadoCalibracao, razao string) Resultado {
		decisao := Decisao{Nivel: pedido.Nivel, Motivos: pedido.Motivos, Calibracao: domain.Calibracao{Estado: estado, Razao: razao}}
		return Resultado{Decisao: decisao, Assinatura: assinatura}
	}

	if estado, razao, elegivel := Elegivel(pedido.Nivel, pedido.Motivos); !elegivel {
		if errDoEmbedding != nil {
			razao += "; " + c.descrever("embedding indisponível", errDoEmbedding)
		}
		return mantido(estado, razao)
	}
	if errDoEmbedding != nil {
		return mantido(domain.CalibracaoEmbeddingIndisponivel, c.descrever("embedding indisponível", errDoEmbedding))
	}

	vizinhos, err := comPrazo(ctx, func(ctx context.Context) ([]domain.VizinhoDaCalibracao, error) {
		return c.historico.VizinhosParecidos(ctx, domain.ConsultaDeVizinhos{
			TenantID: pedido.TaskRun.TenantID, Repositorio: pedido.TaskRun.Repositorio, NumeroPR: pedido.TaskRun.NumeroPR,
			Embedding: assinatura.Embedding, Limite: c.regra.Vizinhos,
		})
	})
	if err != nil {
		return mantido(domain.CalibracaoHistoricoIndisponivel, c.descrever("histórico indisponível", err))
	}

	decisao := Decidir(c.regra, Entrada{TenantID: pedido.TaskRun.TenantID, Nivel: pedido.Nivel, Motivos: pedido.Motivos, Vizinhos: vizinhos})
	return Resultado{Decisao: decisao, Assinatura: assinatura}
}

func (c *Calibrador) assinar(ctx context.Context, pedido Pedido) (*domain.AssinaturaDoPR, error) {
	texto := Assinatura(pedido.Motivos, pedido.Arquivos)
	vetor, err := comPrazo(ctx, func(ctx context.Context) ([]float32, error) { return c.embedder.Gerar(ctx, texto) })
	if err != nil {
		return nil, err
	}
	return &domain.AssinaturaDoPR{Texto: texto, Embedding: vetor}, nil
}

func (c *Calibrador) descrever(contexto string, err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Sprintf("%s: timeout de %s", contexto, c.timeout)
	}
	return fmt.Sprintf("%s: %v", contexto, err)
}

func comPrazo[T any](ctx context.Context, chamar func(context.Context) (T, error)) (T, error) {
	type resposta struct {
		valor T
		err   error
	}
	respostas := make(chan resposta, 1)
	go func() {
		valor, err := chamar(ctx)
		respostas <- resposta{valor: valor, err: err}
	}()

	select {
	case recebida := <-respostas:
		return recebida.valor, recebida.err
	case <-ctx.Done():
		var vazio T
		return vazio, ctx.Err()
	}
}

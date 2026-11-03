package eventos

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

const tempoLimitePadrao = 5 * time.Second

var espacoDosIDs = uuid.MustParse("4120fd1c-adc4-4e94-a83a-9ca3b88d8637")

// ErrSegredoAusente indica URL de envio configurada sem o segredo que assina.
var ErrSegredoAusente = errors.New("eventos: N8N_SEGREDO_EVENTOS ausente com N8N_WEBHOOK_URL configurada")

// Pedido descreve um evento a montar: o tipo, a identidade do fato, o tenant,
// o momento e os dados do tipo, como DadosAnaliseConcluida.
type Pedido struct {
	Tipo       TipoEvento
	IDEvento   string
	TenantID   int64
	OcorridoEm time.Time
	Dados      any
}

type envelopeMontado struct {
	Envelope
	Dados any `json:"dados"`
}

// Montar serializa o pedido como evento da versão atual do contrato e o
// valida como o n8n validaria. Dados fora do contrato devolvem
// ErrEventoInvalido.
func Montar(pedido Pedido) ([]byte, error) {
	corpo, err := json.Marshal(envelopeMontado{
		Envelope: Envelope{
			Versao:     VersaoAtual,
			IDEvento:   pedido.IDEvento,
			TipoEvento: pedido.Tipo,
			OcorridoEm: pedido.OcorridoEm.UTC(),
			TenantID:   pedido.TenantID,
		},
		Dados: pedido.Dados,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: serializar: %w", ErrEventoInvalido, err)
	}
	if _, err := Validar(corpo); err != nil {
		return nil, err
	}
	return corpo, nil
}

// IDDoEvento devolve um UUID determinístico para o fato descrito pelas
// partes: o mesmo fato reenviado depois de uma falha tem o mesmo id_evento, e
// o n8n pode descartá-lo.
func IDDoEvento(partes ...string) string {
	identidade, err := json.Marshal(partes)
	if err != nil {
		panic(err)
	}
	return uuid.NewSHA1(espacoDosIDs, identidade).String()
}

// OpcoesEmissor configura o Emissor. URL vazia desliga o envio; com URL, o
// segredo é obrigatório. TempoLimite vazio assume 5 segundos.
type OpcoesEmissor struct {
	URL         string
	Segredo     []byte
	TempoLimite time.Duration
	ClienteHTTP *http.Client
}

// Emissor envia eventos assinados ao n8n sem segurar quem chama: cada envio
// roda numa goroutine própria, com tempo limite, e falhas só vão para o log.
type Emissor struct {
	opcoes      OpcoesEmissor
	emAndamento sync.WaitGroup
}

// NovoEmissor constrói o emissor. URL sem segredo devolve ErrSegredoAusente.
func NovoEmissor(opcoes OpcoesEmissor) (*Emissor, error) {
	if opcoes.URL != "" && len(opcoes.Segredo) == 0 {
		return nil, ErrSegredoAusente
	}
	if opcoes.TempoLimite <= 0 {
		opcoes.TempoLimite = tempoLimitePadrao
	}
	if opcoes.ClienteHTTP == nil {
		opcoes.ClienteHTTP = &http.Client{}
	}
	return &Emissor{opcoes: opcoes}, nil
}

// Ligado informa se há URL configurada para os envios.
func (e *Emissor) Ligado() bool {
	return e.opcoes.URL != ""
}

// Emitir monta o evento e o envia em segundo plano, registrando o desfecho no
// diário informado. Nunca bloqueia e nunca devolve erro.
func (e *Emissor) Emitir(diario *slog.Logger, pedido Pedido) {
	diario = diario.With("tipo_evento", pedido.Tipo, "id_evento", pedido.IDEvento)
	if !e.Ligado() {
		diario.Debug("envio de eventos desligado, evento descartado")
		return
	}

	corpo, err := Montar(pedido)
	if err != nil {
		diario.Error("evento fora do contrato não foi enviado", "erro", err)
		return
	}

	e.emAndamento.Go(func() { e.enviar(diario, corpo) })
}

// Aguardar espera os envios em andamento terminarem ou desistirem.
func (e *Emissor) Aguardar() {
	e.emAndamento.Wait()
}

func (e *Emissor) enviar(diario *slog.Logger, corpo []byte) {
	ctx, cancelar := context.WithTimeout(context.Background(), e.opcoes.TempoLimite)
	defer cancelar()

	requisicao, err := e.requisicaoAssinada(ctx, corpo)
	if err != nil {
		diario.Error("falha ao montar o envio do evento", "erro", err)
		return
	}

	resposta, err := e.opcoes.ClienteHTTP.Do(requisicao)
	if err != nil {
		diario.Warn("evento não entregue ao n8n", "erro", err)
		return
	}
	defer func() { _ = resposta.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resposta.Body, 1<<16))

	if resposta.StatusCode < http.StatusOK || resposta.StatusCode >= http.StatusMultipleChoices {
		diario.Warn("n8n recusou o evento", "status", resposta.StatusCode)
		return
	}
	diario.Debug("evento entregue ao n8n", "status", resposta.StatusCode)
}

func (e *Emissor) requisicaoAssinada(ctx context.Context, corpo []byte) (*http.Request, error) {
	requisicao, err := http.NewRequestWithContext(ctx, http.MethodPost, e.opcoes.URL, bytes.NewReader(corpo))
	if err != nil {
		return nil, err
	}

	timestamp, assinatura := Assinar(e.opcoes.Segredo, corpo, time.Now())
	requisicao.Header.Set("Content-Type", "application/json")
	requisicao.Header.Set(CabecalhoTimestamp, timestamp)
	requisicao.Header.Set(CabecalhoAssinatura, assinatura)
	return requisicao, nil
}

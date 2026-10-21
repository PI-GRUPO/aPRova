// Package eventos fixa o contrato dos eventos que o núcleo envia ao n8n e a
// assinatura que prova que um evento veio do núcleo.
package eventos

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

// VersaoAtual é a versão do contrato descrita por este pacote.
const VersaoAtual = 1

// ErrEventoInvalido indica corpo que não é um evento válido do contrato.
var ErrEventoInvalido = errors.New("eventos: evento inválido")

// TipoEvento é o valor de tipo_evento, por onde o n8n roteia.
type TipoEvento string

// Tipos de evento do contrato.
const (
	TipoAnaliseConcluida    TipoEvento = "analise_concluida"
	TipoAguardandoAprovacao TipoEvento = "aguardando_aprovacao"
	TipoAprovacaoRegistrada TipoEvento = "aprovacao_registrada"
	TipoDependenciaNova     TipoEvento = "dependencia_nova"
	TipoFalhaRecuperavel    TipoEvento = "falha_recuperavel"
	TipoInstalacao          TipoEvento = "instalacao"
)

var dadosPorTipo = []struct {
	tipo  TipoEvento
	novos func() any
}{
	{TipoAnaliseConcluida, func() any { return &DadosAnaliseConcluida{} }},
	{TipoAguardandoAprovacao, func() any { return &DadosAguardandoAprovacao{} }},
	{TipoAprovacaoRegistrada, func() any { return &DadosAprovacaoRegistrada{} }},
	{TipoDependenciaNova, func() any { return &DadosDependenciaNova{} }},
	{TipoFalhaRecuperavel, func() any { return &DadosFalhaRecuperavel{} }},
	{TipoInstalacao, func() any { return &DadosInstalacao{} }},
}

// Envelope é a casca comum a todo evento, sem os dados do tipo.
type Envelope struct {
	Versao     int        `json:"versao" validate:"required,eq=1"`
	IDEvento   string     `json:"id_evento" validate:"required,uuid"`
	TipoEvento TipoEvento `json:"tipo_evento" validate:"required"`
	OcorridoEm time.Time  `json:"ocorrido_em" validate:"required,utc"`
	TenantID   int64      `json:"tenant_id" validate:"required,gt=0"`
}

type envelopeRecebido struct {
	Envelope
	Dados json.RawMessage `json:"dados" validate:"required"`
}

// Evento é um evento validado: o envelope e os dados já decodificados no
// struct do tipo, como *DadosAnaliseConcluida para analise_concluida.
type Evento struct {
	Envelope
	Dados any
}

// Tipos devolve os tipos de evento do contrato, na ordem em que são
// documentados.
func Tipos() []TipoEvento {
	tipos := make([]TipoEvento, 0, len(dadosPorTipo))
	for _, registro := range dadosPorTipo {
		tipos = append(tipos, registro.tipo)
	}
	return tipos
}

// Validar decodifica corpo como evento do contrato, recusando campo
// desconhecido, campo obrigatório ausente e tipo_evento desconhecido. O erro
// devolvido envolve ErrEventoInvalido.
func Validar(corpo []byte) (Evento, error) {
	var recebido envelopeRecebido
	if err := decodificarEstrito(corpo, &recebido); err != nil {
		return Evento{}, invalido("envelope: %w", err)
	}
	if err := validador.Struct(recebido); err != nil {
		return Evento{}, invalido("envelope: %w", err)
	}

	dados, conhecido := novosDadosDo(recebido.TipoEvento)
	if !conhecido {
		return Evento{}, invalido("tipo_evento %q desconhecido", recebido.TipoEvento)
	}
	if err := decodificarEstrito(recebido.Dados, dados); err != nil {
		return Evento{}, invalido("dados de %s: %w", recebido.TipoEvento, err)
	}
	if err := validador.Struct(dados); err != nil {
		return Evento{}, invalido("dados de %s: %w", recebido.TipoEvento, err)
	}

	return Evento{Envelope: recebido.Envelope, Dados: dados}, nil
}

func novosDadosDo(tipo TipoEvento) (any, bool) {
	for _, registro := range dadosPorTipo {
		if registro.tipo == tipo {
			return registro.novos(), true
		}
	}
	return nil, false
}

func decodificarEstrito(corpo []byte, destino any) error {
	decodificador := json.NewDecoder(bytes.NewReader(corpo))
	decodificador.DisallowUnknownFields()

	if err := decodificador.Decode(destino); err != nil {
		return err
	}
	if err := decodificador.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("conteúdo depois do objeto JSON")
	}
	return nil
}

func invalido(formato string, argumentos ...any) error {
	return fmt.Errorf("%w: %w", ErrEventoInvalido, fmt.Errorf(formato, argumentos...))
}

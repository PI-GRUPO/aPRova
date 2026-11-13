package receipt

import (
	"context"
	"errors"
	"fmt"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

// FonteDeRecibos lê recibos guardados pelo id.
type FonteDeRecibos interface {
	BuscarRecibo(ctx context.Context, id string) (domain.Recibo, error)
}

// Verificador confere recibos guardados: os bytes, a assinatura e o elo com o
// recibo anterior da cadeia.
type Verificador struct {
	chaveiro *Chaveiro
	fonte    FonteDeRecibos
}

// NovoVerificador constrói o verificador sobre as chaves públicas aceitas e a
// fonte de recibos.
func NovoVerificador(chaveiro *Chaveiro, fonte FonteDeRecibos) *Verificador {
	return &Verificador{chaveiro: chaveiro, fonte: fonte}
}

// Chaves devolve as chaves públicas aceitas na verificação.
func (v *Verificador) Chaves() []ChavePublica {
	return v.chaveiro.Lista()
}

// Buscar devolve o recibo guardado, ou domain.ErrNaoEncontrado.
func (v *Verificador) Buscar(ctx context.Context, id string) (domain.Recibo, error) {
	return v.fonte.BuscarRecibo(ctx, id)
}

// Verificar lê o recibo e confere, nesta ordem: o hash dos bytes guardados, a
// assinatura com a chave do key_id, se o conteúdo assinado é mesmo o deste
// registro e o elo com o recibo anterior. Id inexistente devolve
// domain.ErrNaoEncontrado; recibo reprovado devolve o Resultado com a falha.
func (v *Verificador) Verificar(ctx context.Context, id string) (domain.Recibo, Resultado, error) {
	recibo, err := v.fonte.BuscarRecibo(ctx, id)
	if err != nil {
		return domain.Recibo{}, Resultado{}, err
	}

	resultado := v.chaveiro.Verificar(Assinado{
		Conteudo: []byte(recibo.Conteudo), Hash: recibo.Hash, Assinatura: recibo.Assinatura, IDChave: recibo.IDChave,
	})
	if !resultado.Valido {
		return recibo, resultado, nil
	}

	conteudo, err := LerConteudo([]byte(recibo.Conteudo))
	if err != nil {
		return recibo, reprovado(FalhaConteudo, err.Error()), nil
	}
	if conteudo.ID != recibo.ID || conteudo.TaskRunID != recibo.TaskRunID || conteudo.TenantID != recibo.TenantID {
		return recibo, reprovado(FalhaConteudo, "o conteúdo assinado não é o deste registro: id, task_run ou tenant diferentes"), nil
	}

	anterior, err := v.anteriorDe(ctx, recibo)
	if err != nil {
		return domain.Recibo{}, Resultado{}, err
	}
	if quebra := VerificarElo(conteudo, anterior); quebra != "" {
		return recibo, reprovado(FalhaElo, quebra), nil
	}
	return recibo, Resultado{Valido: true}, nil
}

func (v *Verificador) anteriorDe(ctx context.Context, recibo domain.Recibo) (*Anterior, error) {
	if recibo.ReciboAnteriorID == nil {
		return nil, nil
	}

	anterior, err := v.fonte.BuscarRecibo(ctx, *recibo.ReciboAnteriorID)
	if errors.Is(err, domain.ErrNaoEncontrado) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("ler o recibo anterior de %s: %w", recibo.ID, err)
	}
	return &Anterior{ID: anterior.ID, Conteudo: []byte(anterior.Conteudo)}, nil
}

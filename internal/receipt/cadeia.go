package receipt

import (
	"context"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

// FonteDaCadeia devolve uma visão consistente de todos os recibos da instalação.
// Se qualquer recibo estiver fora dos repositórios autorizados, recusa a consulta inteira.
type FonteDaCadeia interface {
	ListarRecibosDaInstalacao(context.Context, int64, []string) ([]domain.Recibo, error)
}

type ResultadoDaCadeia struct {
	Valida                 bool   `json:"valida"`
	Total                  int    `json:"total"`
	Verificados            int    `json:"verificados"`
	PrimeiroReciboInvalido string `json:"primeiro_recibo_invalido,omitempty"`
	ReciboAnteriorAusente  string `json:"recibo_anterior_ausente,omitempty"`
	Falha                  Falha  `json:"falha,omitempty"`
}

type VerificadorDeCadeia struct {
	chaveiro *Chaveiro
	fonte    FonteDaCadeia
}

func NovoVerificadorDeCadeia(chaveiro *Chaveiro, fonte FonteDaCadeia) *VerificadorDeCadeia {
	return &VerificadorDeCadeia{chaveiro: chaveiro, fonte: fonte}
}

func (v *VerificadorDeCadeia) VerificarCadeia(ctx context.Context, instalacao int64, repositorios []string) (ResultadoDaCadeia, error) {
	recibos, err := v.fonte.ListarRecibosDaInstalacao(ctx, instalacao, repositorios)
	if err != nil {
		return ResultadoDaCadeia{}, err
	}
	resposta := ResultadoDaCadeia{Total: len(recibos)}
	porID, proximos, raiz := indexarCadeia(recibos)

	// As referências determinam a ordem. Dois recibos podem ter a mesma data.
	var anterior *Anterior
	visitados := make(map[string]bool, len(recibos))
	for r := raiz; r != nil && !visitados[r.ID]; r = proximos[r.ID] {
		if err := ctx.Err(); err != nil {
			return ResultadoDaCadeia{}, err
		}
		if v.conferir(&resposta, *r, anterior, porID) {
			return resposta, nil
		}
		visitados[r.ID] = true
		resposta.Verificados++
		anterior = &Anterior{ID: r.ID, Conteudo: []byte(r.Conteudo)}
	}
	if resposta.Verificados == resposta.Total {
		resposta.Valida = true
		return resposta, nil
	}

	solto := primeiroReciboDesconectado(recibos, porID, visitados)
	if !v.conferir(&resposta, *solto, nil, porID) {
		// Outra raiz também é quebra, mesmo que o recibo dessa raiz seja válido.
		resposta.PrimeiroReciboInvalido = solto.ID
		resposta.Falha = FalhaElo
	}
	return resposta, nil
}

func indexarCadeia(recibos []domain.Recibo) (porID, proximos map[string]*domain.Recibo, raiz *domain.Recibo) {
	porID = make(map[string]*domain.Recibo, len(recibos))
	proximos = make(map[string]*domain.Recibo, len(recibos))
	for i := range recibos {
		r := &recibos[i]
		porID[r.ID] = r
		if r.ReciboAnteriorID == nil {
			if raiz == nil {
				raiz = r
			}
		} else if proximos[*r.ReciboAnteriorID] == nil {
			proximos[*r.ReciboAnteriorID] = r
		}
	}
	return porID, proximos, raiz
}

func primeiroReciboDesconectado(recibos []domain.Recibo, porID map[string]*domain.Recibo, visitados map[string]bool) *domain.Recibo {
	// Um recibo removido deixa o seu sucessor fora do trecho ligado à raiz.
	// Dê prioridade ao início do trecho quebrado, antes de examinar seus descendentes.
	var solto *domain.Recibo
	for i := range recibos {
		r := &recibos[i]
		if visitados[r.ID] {
			continue
		}
		if solto == nil {
			solto = r
		}
		if r.ReciboAnteriorID == nil || porID[*r.ReciboAnteriorID] == nil {
			return r
		}
	}
	return solto
}

func (v *VerificadorDeCadeia) conferir(resposta *ResultadoDaCadeia, r domain.Recibo, anterior *Anterior, porID map[string]*domain.Recibo) bool {
	conteudo, resultado := v.chaveiro.verificarGuardado(r)
	if resultado.Valido {
		if quebra := VerificarElo(conteudo, anterior); quebra != "" {
			resultado = reprovado(FalhaElo, quebra)
			if conteudo.Anterior != nil && porID[conteudo.Anterior.ID] == nil {
				resposta.ReciboAnteriorAusente = conteudo.Anterior.ID
			}
		}
	}
	if resultado.Valido {
		return false
	}
	resposta.PrimeiroReciboInvalido = r.ID
	resposta.Falha = resultado.Falha
	return true
}

package main

import (
	"context"
	"crypto/rand"
	"log"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
)

// Fixture exclusiva do servidor de testes; nunca carrega recibos de uma instalação.
type fonteDeTeste struct{ recibo domain.Recibo }

func (f fonteDeTeste) BuscarRecibo(_ context.Context, id string) (domain.Recibo, error) {
	if id != f.recibo.ID {
		return domain.Recibo{}, domain.ErrNaoEncontrado
	}
	return f.recibo, nil
}
func recibosDeTeste() *receipt.Verificador {
	semente := make([]byte, 32)
	if _, err := rand.Read(semente); err != nil {
		log.Fatal(err)
	}
	assinador, err := receipt.NovoAssinador(semente)
	if err != nil {
		log.Fatal(err)
	}
	conteudo := receipt.Conteudo{Versao: receipt.VersaoDoFormato, ID: reciboDoCenario, TaskRunID: analiseDoCenario, TenantID: 42, Repositorio: repositorioDoCenario, NumeroPR: 1, SHAHead: shaDoCenario, Nivel: domain.RiscoBaixo, NivelDeterministico: domain.RiscoBaixo, EmitidoEm: instanteDoCenario, KeyID: assinador.IDChave()}
	bytes, err := conteudo.Serializar()
	if err != nil {
		log.Fatal(err)
	}
	fonte := fonteDeTeste{recibo: domain.Recibo{NovoRecibo: domain.NovoRecibo{ID: conteudo.ID, TaskRunID: conteudo.TaskRunID, Conteudo: string(bytes), Hash: receipt.Hash(bytes), Assinatura: assinador.Assinar(bytes), IDChave: assinador.IDChave()}, TenantID: conteudo.TenantID}}
	return receipt.NovoVerificador(receipt.NovoChaveiro(assinador.ChavePublica()), fonte)
}

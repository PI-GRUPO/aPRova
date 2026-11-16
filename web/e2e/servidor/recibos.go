package main

import (
	"context"
	"crypto/rand"
	"log"
	"time"

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
	conteudo := receipt.Conteudo{Versao: receipt.VersaoDoFormato, ID: "00000000-0000-4000-8000-000000000002", TaskRunID: "00000000-0000-4000-8000-000000000003", TenantID: 1, Repositorio: repositorioDoCenario, NumeroPR: 1, SHAHead: "abc123", Nivel: domain.RiscoBaixo, NivelDeterministico: domain.RiscoBaixo, EmitidoEm: time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC), KeyID: assinador.IDChave()}
	bytes, err := conteudo.Serializar()
	if err != nil {
		log.Fatal(err)
	}
	fonte := fonteDeTeste{recibo: domain.Recibo{NovoRecibo: domain.NovoRecibo{ID: conteudo.ID, TaskRunID: conteudo.TaskRunID, Conteudo: string(bytes), Hash: receipt.Hash(bytes), Assinatura: assinador.Assinar(bytes), IDChave: assinador.IDChave()}, TenantID: conteudo.TenantID}}
	return receipt.NovoVerificador(receipt.NovoChaveiro(assinador.ChavePublica()), fonte)
}

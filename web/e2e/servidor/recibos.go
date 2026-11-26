package main

import (
	"context"
	"crypto/rand"
	"log"
	"slices"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
)

// Fixture exclusiva do servidor de testes; nunca carrega recibos de uma instalação.
type fonteDeTeste struct{ recibo, interrompido domain.Recibo }

func (f fonteDeTeste) BuscarRecibo(_ context.Context, id string) (domain.Recibo, error) {
	if id != f.recibo.ID {
		return domain.Recibo{}, domain.ErrNaoEncontrado
	}
	return f.recibo, nil
}
func (f fonteDeTeste) ListarRecibosDaInstalacao(_ context.Context, id int64, repositorios []string) ([]domain.Recibo, error) {
	switch {
	case id == 42 && slices.Contains(repositorios, repositorioDoCenario):
		return []domain.Recibo{f.recibo}, nil
	case id == 43 && slices.Contains(repositorios, "segunda-equipe/projeto"):
		return []domain.Recibo{f.interrompido}, nil
	default:
		return nil, domain.ErrCadeiaSemAcesso
	}
}

func recibosDeTeste() (*receipt.Verificador, *receipt.VerificadorDeCadeia) {
	semente := make([]byte, 32)
	if _, err := rand.Read(semente); err != nil {
		log.Fatal(err)
	}
	assinador, err := receipt.NovoAssinador(semente)
	if err != nil {
		log.Fatal(err)
	}
	conteudo := receipt.Conteudo{Versao: receipt.VersaoDoFormato, ID: reciboDoCenario, TaskRunID: analiseDoCenario, TenantID: 42, Repositorio: repositorioDoCenario, NumeroPR: 1, SHAHead: shaDoCenario, Nivel: domain.RiscoBaixo, NivelDeterministico: domain.RiscoBaixo, EmitidoEm: instanteDoCenario, KeyID: assinador.IDChave()}
	fonte := fonteDeTeste{recibo: assinarReciboDeTeste(conteudo, assinador)}
	conteudo.ID = "00000000-0000-4000-8000-000000000003"
	conteudo.TaskRunID = "00000000-0000-4000-8000-000000000004"
	conteudo.TenantID = 43
	conteudo.Repositorio = "segunda-equipe/projeto"
	conteudo.Anterior = &receipt.Elo{ID: "00000000-0000-4000-8000-000000000005", Hash: receipt.HashEmHex([]byte("recibo original ausente"))}
	fonte.interrompido = assinarReciboDeTeste(conteudo, assinador)
	chaveiro := receipt.NovoChaveiro(assinador.ChavePublica())
	return receipt.NovoVerificador(chaveiro, fonte), receipt.NovoVerificadorDeCadeia(chaveiro, fonte)
}

func assinarReciboDeTeste(conteudo receipt.Conteudo, assinador *receipt.Assinador) domain.Recibo {
	bytes, err := conteudo.Serializar()
	if err != nil {
		log.Fatal(err)
	}
	novo := domain.NovoRecibo{ID: conteudo.ID, TaskRunID: conteudo.TaskRunID, Conteudo: string(bytes), Hash: receipt.Hash(bytes), Assinatura: assinador.Assinar(bytes), IDChave: assinador.IDChave()}
	if conteudo.Anterior != nil {
		novo.ReciboAnteriorID = &conteudo.Anterior.ID
	}
	return domain.Recibo{NovoRecibo: novo, TenantID: conteudo.TenantID}
}

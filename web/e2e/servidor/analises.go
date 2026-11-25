package main

import (
	"context"
	"slices"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

const repositorioDoCenario = "equipe-e2e/projeto"

const shaDoCenario = "abc123"

const analiseDoCenario = "00000000-0000-4000-8000-000000000003"

var instanteDoCenario = time.Now().Add(-time.Hour).UTC()

type analisesSessaoTeste struct{}

func resumoDoCenario() domain.AnaliseWeb {
	return domain.AnaliseWeb{ID: analiseDoCenario, Repositorio: repositorioDoCenario, NumeroPR: 1, SHAHead: shaDoCenario, Autor: "autora-e2e", Estado: string(domain.EstadoFalhou), RecebidoEm: instanteDoCenario, CriadoEm: instanteDoCenario, AtualizadoEm: instanteDoCenario}
}
func (analisesSessaoTeste) ListarAnalisesWeb(_ context.Context, f domain.FiltroAnalisesWeb) (domain.PaginaAnalisesWeb, error) {
	p := domain.PaginaAnalisesWeb{InstallationID: f.TenantID, Snapshot: f.Snapshot, Pagina: f.Pagina, Limite: f.Limite, Itens: []domain.AnaliseWeb{}}
	if (f.Estado == "todos" || f.Estado == string(domain.EstadoFalhou)) && (f.Nivel == "todos" || f.Nivel == "nao_avaliado") && !instanteDoCenario.Before(f.Inicio) && instanteDoCenario.Before(f.Fim) && f.TenantID == 42 && slices.Contains(f.Repositorios, repositorioDoCenario) && (f.NumeroPR == 0 || f.NumeroPR == 1) {
		p.Total = 1
		if f.Pagina == 1 {
			p.Itens = []domain.AnaliseWeb{resumoDoCenario()}
		}
	}
	return p, nil
}
func (analisesSessaoTeste) DetalharAnaliseWeb(_ context.Context, tenant int64, repos []string, id string) (domain.DetalheAnaliseWeb, error) {
	if tenant != 42 || id != analiseDoCenario || !slices.Contains(repos, repositorioDoCenario) {
		return domain.DetalheAnaliseWeb{}, domain.ErrNaoEncontrado
	}
	recibo := reciboDoCenario
	return domain.DetalheAnaliseWeb{InstallationID: tenant, Analise: resumoDoCenario(), URLPR: "https://github.com/equipe-e2e/projeto/pull/1", Atual: true, Etapas: []domain.EtapaWeb{{ID: 1, De: "pendente", Para: string(domain.EstadoFalhou), Detalhe: "Falha de infraestrutura do cenário isolado", CriadoEm: instanteDoCenario}}, Tentativas: []domain.TentativaWeb{}, Aprovacoes: []domain.AprovacaoWeb{}, ReciboID: &recibo}, nil
}

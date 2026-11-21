package main

import (
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/config"
	"github.com/PI-GRUPO/aPRova/internal/metricas"
	"github.com/PI-GRUPO/aPRova/internal/observabilidade"
	"github.com/PI-GRUPO/aPRova/internal/repository"
)

func TestExplicacaoDesligadaNaoMontaExplicador(t *testing.T) {
	explicador := montarExplicador(config.Configuracao{ChaveAPIGemini: "chave"}, &repository.RepositorioPostgres{}, metricas.NovoMotor(observabilidade.NovoRegistro()))

	if explicador != nil {
		t.Fatalf("sem EXPLICACAO_LIGADA o comentário fica como hoje, sem explicador: %v", explicador)
	}
}

func TestExplicacaoLigadaSemChaveFicaDesligada(t *testing.T) {
	configuracao := config.Configuracao{ExplicacaoLigada: true, ExplicacaoModelo: "gemini-3.5-flash-lite"}

	if explicador := montarExplicador(configuracao, &repository.RepositorioPostgres{}, metricas.NovoMotor(observabilidade.NovoRegistro())); explicador != nil {
		t.Fatalf("sem GEMINI_API_KEY a explicação fica desligada: %v", explicador)
	}
}

func TestExplicacaoLigadaComChaveMontaOExplicador(t *testing.T) {
	configuracao := config.Configuracao{ExplicacaoLigada: true, ExplicacaoModelo: "gemini-3.5-flash-lite", ChaveAPIGemini: "chave"}

	if explicador := montarExplicador(configuracao, &repository.RepositorioPostgres{}, metricas.NovoMotor(observabilidade.NovoRegistro())); explicador == nil {
		t.Fatal("com EXPLICACAO_LIGADA e chave, o publicador recebe o explicador")
	}
}

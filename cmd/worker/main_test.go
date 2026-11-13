package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/config"
	"github.com/PI-GRUPO/aPRova/internal/platform"
	"github.com/PI-GRUPO/aPRova/internal/repository"
)

func TestWorkerNaoSobeSemAChaveDeRecibo(t *testing.T) {
	_, err := montarAssinador(config.Configuracao{})

	if err == nil || !strings.Contains(err.Error(), "APROVA_CHAVE_RECIBO") {
		t.Fatalf("esperava erro nomeando APROVA_CHAVE_RECIBO, obtive %v", err)
	}
}

func TestWorkerMontaOAssinadorComASementeConfigurada(t *testing.T) {
	assinador, err := montarAssinador(config.Configuracao{ReciboSemente: bytes.Repeat([]byte{7}, 32)})

	if err != nil || len(assinador.IDChave()) != 16 {
		t.Fatalf("assinador = %v, erro = %v", assinador, err)
	}
}

func TestWorkerNaoSobeSemOSlugDoApp(t *testing.T) {
	_, err := montarPublicador(config.Configuracao{URLPublica: "https://aprova.example.com"}, &platform.Cliente{}, &repository.RepositorioPostgres{})

	if err == nil || !strings.Contains(err.Error(), "GITHUB_APP_SLUG") {
		t.Fatalf("esperava erro nomeando GITHUB_APP_SLUG, obtive %v", err)
	}
}

func TestWorkerPublicaNoPRComOPublicadorReal(t *testing.T) {
	publicador, err := montarPublicador(config.Configuracao{GitHubAppSlug: "a-prova", URLPublica: "https://aprova.example.com"}, &platform.Cliente{}, &repository.RepositorioPostgres{})

	if err != nil || publicador == nil {
		t.Fatalf("publicador = %v, erro = %v", publicador, err)
	}
}

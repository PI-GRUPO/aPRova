package config

import (
	"maps"
	"strings"
	"testing"
)

func TestCarregarLeAImagemDoSandboxPython(t *testing.T) {
	ambienteMinimoValido(t)
	t.Setenv("SANDBOX_IMAGEM_PYTHON", "aprova.azurecr.io/aprova-sandbox-python:3f2c1e0")

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}

	if configuracao.SandboxImagemPython != "aprova.azurecr.io/aprova-sandbox-python:3f2c1e0" {
		t.Errorf("imagem lida errada: %q", configuracao.SandboxImagemPython)
	}
}

func TestCarregarAceitaImagemDoSandboxPythonAusente(t *testing.T) {
	ambienteMinimoValido(t)

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("a imagem Python é opcional, carregar falhou: %v", err)
	}
	if configuracao.SandboxImagemPython != "" {
		t.Errorf("esperava vazio, obtive %q", configuracao.SandboxImagemPython)
	}
}

func TestCarregarLeALinguagemDeCadaRepositorio(t *testing.T) {
	ambienteMinimoValido(t)
	t.Setenv("SANDBOX_LINGUAGENS", " Azulpasta/aPRova-TESTES=python , PI-GRUPO/outro=go ")

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}

	esperadas := map[string]string{"Azulpasta/aPRova-TESTES": "python", "PI-GRUPO/outro": "go"}
	if !maps.Equal(configuracao.SandboxLinguagens, esperadas) {
		t.Errorf("linguagens = %v, esperava %v", configuracao.SandboxLinguagens, esperadas)
	}
}

func TestCarregarAceitaLinguagensAusentes(t *testing.T) {
	ambienteMinimoValido(t)

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(configuracao.SandboxLinguagens) != 0 {
		t.Errorf("esperava nenhuma linguagem configurada, obtive %v", configuracao.SandboxLinguagens)
	}
}

func TestCarregarRecusaLinguagensInvalidas(t *testing.T) {
	casos := []string{
		"Azulpasta/aPRova-TESTES",
		"=python",
		"Azulpasta/aPRova-TESTES=",
		"Azulpasta/aPRova-TESTES=ruby",
		"a/b=go,a/b=python",
	}

	for _, valor := range casos {
		t.Run(valor, func(t *testing.T) {
			ambienteMinimoValido(t)
			t.Setenv("SANDBOX_LINGUAGENS", valor)

			_, err := Carregar()

			if err == nil || !strings.Contains(err.Error(), "SANDBOX_LINGUAGENS") {
				t.Errorf("esperava erro nomeando SANDBOX_LINGUAGENS, obtive %v", err)
			}
		})
	}
}

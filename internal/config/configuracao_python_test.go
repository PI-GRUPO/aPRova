package config

import "testing"

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

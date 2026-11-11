package config

import "testing"

func TestCarregarLeAImagemDoSandboxJava(t *testing.T) {
	ambienteMinimoValido(t)
	t.Setenv("SANDBOX_IMAGEM_JAVA", "aprova.azurecr.io/aprova-sandbox-java:3f2c1e0")

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}

	if configuracao.SandboxImagemJava != "aprova.azurecr.io/aprova-sandbox-java:3f2c1e0" {
		t.Errorf("imagem lida errada: %q", configuracao.SandboxImagemJava)
	}
}

func TestCarregarAceitaImagemDoSandboxJavaAusente(t *testing.T) {
	ambienteMinimoValido(t)

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("a imagem Java é opcional, carregar falhou: %v", err)
	}

	if configuracao.SandboxImagemJava != "" {
		t.Errorf("esperava vazio, obtive %q", configuracao.SandboxImagemJava)
	}
}

package config

import (
	"maps"
	"testing"
)

func TestCarregarLeAImagemDoSandboxJava(t *testing.T) {
	ambienteMinimoValido(t)
	t.Setenv("SANDBOX_IMAGEM_JAVA", "aprova-sandbox-java:local")

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if configuracao.SandboxImagemJava != "aprova-sandbox-java:local" {
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

func TestCarregarAceitaRepositorioJava(t *testing.T) {
	ambienteMinimoValido(t)
	t.Setenv("SANDBOX_LINGUAGENS", "Azulpasta/aPRova-TESTES=java")

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if esperadas := map[string]string{"Azulpasta/aPRova-TESTES": "java"}; !maps.Equal(configuracao.SandboxLinguagens, esperadas) {
		t.Errorf("linguagens = %v, esperava %v", configuracao.SandboxLinguagens, esperadas)
	}
}

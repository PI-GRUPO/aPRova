package main

import (
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/config"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

func TestImagemJavaSoEntraNoExecutorLocal(t *testing.T) {
	executor := configuracaoDoSandbox(config.Configuracao{
		SandboxImagemGo:   "aprova-sandbox-go:local",
		SandboxImagemJava: "aprova-sandbox-java:local",
	})

	if imagem := executor.Local.ImagemPorLinguagem[sandbox.LinguagemJava]; imagem != "aprova-sandbox-java:local" {
		t.Errorf("o executor local roda Java com a imagem configurada: %q", imagem)
	}
	if imagem, presente := executor.Azure.ImagemPorLinguagem[sandbox.LinguagemJava]; presente {
		t.Errorf("Java na Azure fica para o card 61, mas a imagem %q chegou ao executor da Azure", imagem)
	}
	if executor.Azure.ImagemPorLinguagem[sandbox.LinguagemGo] != "aprova-sandbox-go:local" {
		t.Error("as outras linguagens continuam na Azure")
	}
}

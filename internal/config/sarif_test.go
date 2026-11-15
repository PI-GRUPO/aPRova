package config

import (
	"strings"
	"testing"
)

func TestSARIFFicaDesligadoPorPadrao(t *testing.T) {
	ambienteMinimoValido(t)

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}

	if configuracao.SARIFLigado {
		t.Error("sem SARIF_LIGADO, o envio ao code scanning fica desligado")
	}
}

func TestSARIFLigaComAVariavel(t *testing.T) {
	ambienteMinimoValido(t)
	t.Setenv("SARIF_LIGADO", "true")

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}

	if !configuracao.SARIFLigado {
		t.Error("SARIF_LIGADO=true deveria ligar o envio")
	}
}

func TestSARIFRecusaValorQueNaoEhBooleano(t *testing.T) {
	ambienteMinimoValido(t)
	t.Setenv("SARIF_LIGADO", "talvez")

	_, err := Carregar()

	if err == nil || !strings.Contains(err.Error(), "SARIF_LIGADO") {
		t.Fatalf("esperava erro nomeando SARIF_LIGADO, obtive %v", err)
	}
}

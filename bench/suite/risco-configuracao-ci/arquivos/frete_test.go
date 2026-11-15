package frete

import "testing"

func TestAteUmQuiloPagaATarifaBase(t *testing.T) {
	if valor := Calcular(1); valor != 10 {
		t.Errorf("Calcular(1) = %v, esperava 10", valor)
	}
}

func TestCadaQuiloExtraCustaDoisEMeio(t *testing.T) {
	if valor := Calcular(3); valor != 15 {
		t.Errorf("Calcular(3) = %v, esperava 15", valor)
	}
}

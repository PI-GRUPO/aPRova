package auth

import "testing"

func TestSenhaCurtaNaoEForte(t *testing.T) {
	if SenhaForte("abc") {
		t.Error("abc não pode ser forte")
	}
}

func TestSenhaLongaComDigitoEForte(t *testing.T) {
	if !SenhaForte("correta-cavalo-bateria-42") {
		t.Error("uma frase longa com dígito é forte")
	}
}

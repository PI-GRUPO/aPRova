package frete

import "testing"

func TestMesmaTabela(t *testing.T) {
	if !MesmaTabela([]int{1}, []int{1}) {
		t.Fatal("tabelas iguais")
	}
}

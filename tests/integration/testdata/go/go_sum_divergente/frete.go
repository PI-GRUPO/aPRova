package frete

import "github.com/google/go-cmp/cmp"

func MesmaTabela(a, b []int) bool {
	return cmp.Equal(a, b)
}

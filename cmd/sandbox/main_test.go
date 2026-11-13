package main

import (
	"errors"
	"testing"
)

func TestEtapaPedidaPeloSubcomando(t *testing.T) {
	casos := map[string]struct {
		argumentos []string
		etapa      etapa
	}{
		"preparar":                 {[]string{"preparar"}, etapaPreparar},
		"testar":                   {[]string{"testar"}, etapaTestar},
		"sem subcomando, da Azure": {nil, etapaUnicaDaAzure},
	}
	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			obtida, err := etapaPedida(caso.argumentos)
			if err != nil || obtida != caso.etapa {
				t.Errorf("etapa %q, erro %v; esperava %q", obtida, err, caso.etapa)
			}
		})
	}
}

func TestSubcomandoDesconhecidoERecusado(t *testing.T) {
	for _, argumentos := range [][]string{{"testar-com-rede"}, {"preparar", "testar"}, {""}} {
		if _, err := etapaPedida(argumentos); !errors.Is(err, errSubcomandoDesconhecido) {
			t.Errorf("%q: %v", argumentos, err)
		}
	}
}

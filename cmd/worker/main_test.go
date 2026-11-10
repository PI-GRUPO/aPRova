package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Azulpasta/aPRova/internal/config"
)

func TestWorkerNaoSobeSemAChaveDeRecibo(t *testing.T) {
	_, err := montarAssinador(config.Configuracao{})

	if err == nil || !strings.Contains(err.Error(), "APROVA_CHAVE_RECIBO") {
		t.Fatalf("esperava erro nomeando APROVA_CHAVE_RECIBO, obtive %v", err)
	}
}

func TestWorkerMontaOAssinadorComASementeConfigurada(t *testing.T) {
	assinador, err := montarAssinador(config.Configuracao{ReciboSemente: bytes.Repeat([]byte{7}, 32)})

	if err != nil || len(assinador.IDChave()) != 16 {
		t.Fatalf("assinador = %v, erro = %v", assinador, err)
	}
}

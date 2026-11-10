package main

import (
	"bytes"
	"crypto/ed25519"
	"testing"

	"github.com/Azulpasta/aPRova/internal/config"
	"github.com/Azulpasta/aPRova/internal/receipt"
)

func TestChaveiroDaAPIJuntaAChaveAtualComAsPublicasConfiguradas(t *testing.T) {
	atual, err := receipt.NovoAssinador(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	antiga, err := receipt.NovoAssinador(bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatal(err)
	}

	chaves := chaveiroDosRecibos(config.Configuracao{
		ReciboSemente:        bytes.Repeat([]byte{7}, 32),
		ReciboChavesPublicas: []ed25519.PublicKey{atual.ChavePublica(), antiga.ChavePublica()},
	}).Lista()

	if len(chaves) != 2 || chaves[0].KeyID != atual.IDChave() || chaves[1].KeyID != antiga.IDChave() {
		t.Errorf("chaves = %+v", chaves)
	}
}

func TestChaveiroDaAPIFuncionaSoComChavesPublicas(t *testing.T) {
	antiga, err := receipt.NovoAssinador(bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatal(err)
	}

	chaves := chaveiroDosRecibos(config.Configuracao{ReciboChavesPublicas: []ed25519.PublicKey{antiga.ChavePublica()}}).Lista()

	if len(chaves) != 1 || chaves[0].KeyID != antiga.IDChave() {
		t.Errorf("a API verifica sem ter a chave privada: %+v", chaves)
	}
	if vazio := chaveiroDosRecibos(config.Configuracao{}).Lista(); len(vazio) != 0 {
		t.Errorf("sem configuração não há chave: %+v", vazio)
	}
}

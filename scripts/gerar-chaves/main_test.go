package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"slices"
	"testing"
)

func TestGerarParEmBase64ProduzSementeEChavePublicaQueAssinamEVerificam(t *testing.T) {
	sementeEmBase64, publicaEmBase64, err := gerarParEmBase64()
	if err != nil {
		t.Fatalf("gerar par: %v", err)
	}

	semente, err := base64.StdEncoding.DecodeString(sementeEmBase64)
	if err != nil {
		t.Fatalf("a semente impressa não é base64 válido: %v", err)
	}
	publica, err := base64.StdEncoding.DecodeString(publicaEmBase64)
	if err != nil {
		t.Fatalf("a chave pública impressa não é base64 válido: %v", err)
	}

	if len(semente) != ed25519.SeedSize {
		t.Fatalf("a semente deveria ter %d bytes, tem %d", ed25519.SeedSize, len(semente))
	}
	if len(publica) != ed25519.PublicKeySize {
		t.Fatalf("chave pública deveria ter %d bytes, tem %d", ed25519.PublicKeySize, len(publica))
	}

	mensagem := []byte("recibo de exemplo")
	assinatura := ed25519.Sign(ed25519.NewKeyFromSeed(semente), mensagem)

	if !ed25519.Verify(ed25519.PublicKey(publica), mensagem, assinatura) {
		t.Error("o par gerado deveria assinar e verificar a mesma mensagem")
	}
}

func TestLinhasDoEnvUsamAsVariaveisDoRecibo(t *testing.T) {
	linhas := linhasDoEnv("c2VtZW50ZQ==", "cHVibGljYQ==")

	esperadas := []string{"APROVA_CHAVE_RECIBO=c2VtZW50ZQ==", "APROVA_CHAVES_PUBLICAS_RECIBO=cHVibGljYQ=="}
	if !slices.Equal(linhas, esperadas) {
		t.Errorf("linhas = %v", linhas)
	}
}

func TestGerarParEmBase64NaoRepeteChaves(t *testing.T) {
	primeira, _, err := gerarParEmBase64()
	if err != nil {
		t.Fatalf("gerar par: %v", err)
	}
	segunda, _, err := gerarParEmBase64()
	if err != nil {
		t.Fatalf("gerar par: %v", err)
	}

	if primeira == segunda {
		t.Error("duas execuções não podem produzir a mesma chave privada")
	}
}

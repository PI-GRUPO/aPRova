package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"os"
)

func main() {
	semente, publica, err := gerarParEmBase64()
	if err != nil {
		fmt.Fprintf(os.Stderr, "gerar par Ed25519: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Cole no seu .env:")
	fmt.Println()
	for _, linha := range linhasDoEnv(semente, publica) {
		fmt.Println(linha)
	}
	fmt.Println()
	fmt.Println("APROVA_CHAVE_RECIBO é a semente da chave privada: é segredo. A chave")
	fmt.Println("pública deve ser publicada, senão ninguém consegue verificar os recibos")
	fmt.Println("de forma independente.")
}

func linhasDoEnv(sementeEmBase64, publicaEmBase64 string) []string {
	return []string{
		"APROVA_CHAVE_RECIBO=" + sementeEmBase64,
		"APROVA_CHAVES_PUBLICAS_RECIBO=" + publicaEmBase64,
	}
}

func gerarParEmBase64() (sementeEmBase64, publicaEmBase64 string, err error) {
	publica, privada, err := ed25519.GenerateKey(nil)
	if err != nil {
		return "", "", fmt.Errorf("gerar chave ed25519: %w", err)
	}

	return base64.StdEncoding.EncodeToString(privada.Seed()),
		base64.StdEncoding.EncodeToString(publica),
		nil
}

package receipt

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func TestKeyIDSaoOsPrimeiros16HexDoSHA256DaChavePublica(t *testing.T) {
	assinador := assinadorDeTeste(t)

	resumo := sha256.Sum256(assinador.ChavePublica())
	esperado := hex.EncodeToString(resumo[:])[:16]

	if assinador.IDChave() != esperado || IDDaChave(assinador.ChavePublica()) != esperado {
		t.Errorf("key_id = %s, esperava %s", assinador.IDChave(), esperado)
	}
}

func TestAssinadorDerivaAChaveDaSementeEAssinaComEd25519Puro(t *testing.T) {
	assinador := assinadorDeTeste(t)
	mensagem := []byte("conteúdo do recibo")

	esperada := ed25519.NewKeyFromSeed(sementeDeTeste)
	if !bytes.Equal(assinador.ChavePublica(), esperada.Public().(ed25519.PublicKey)) { //nolint:errcheck // a chave Ed25519 devolve sempre ed25519.PublicKey
		t.Error("a chave pública precisa ser a derivada da semente")
	}
	if !ed25519.Verify(assinador.ChavePublica(), mensagem, assinador.Assinar(mensagem)) {
		t.Error("a assinatura precisa verificar com a biblioteca padrão, sem pré-hash")
	}
}

func TestNovoAssinadorRecusaSementeAusenteOuDeTamanhoErrado(t *testing.T) {
	for nome, semente := range map[string][]byte{
		"ausente":        nil,
		"curta":          bytes.Repeat([]byte{7}, 31),
		"chave completa": bytes.Repeat([]byte{7}, 64),
	} {
		t.Run(nome, func(t *testing.T) {
			_, err := NovoAssinador(semente)

			if !errors.Is(err, ErrChaveInvalida) {
				t.Fatalf("esperava ErrChaveInvalida, obtive %v", err)
			}
			if len(semente) > 0 && vazaSegredo(err.Error(), semente) {
				t.Errorf("a mensagem de erro contém a chave: %q", err.Error())
			}
		})
	}
}

func TestAssinadorNuncaImprimeAChavePrivada(t *testing.T) {
	assinador := assinadorDeTeste(t)
	var diario bytes.Buffer
	slog.New(slog.NewJSONHandler(&diario, nil)).Info("assinador", "assinador", assinador)

	for nome, texto := range map[string]string{
		"%v":   fmt.Sprintf("%v", assinador),
		"%+v":  fmt.Sprintf("%+v", assinador),
		"%#v":  fmt.Sprintf("%#v", assinador),
		"%s":   fmt.Sprintf("%s", assinador),
		"slog": diario.String(),
	} {
		if vazaSegredo(texto, sementeDeTeste) {
			t.Errorf("%s expõe a chave privada: %s", nome, texto)
		}
		if !strings.Contains(texto, assinador.IDChave()) {
			t.Errorf("%s deveria identificar a chave pelo key_id: %s", nome, texto)
		}
	}
}

func vazaSegredo(texto string, semente []byte) bool {
	privada := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{semente[0]}, 32))
	formas := []string{
		hex.EncodeToString(semente), base64.StdEncoding.EncodeToString(semente),
		hex.EncodeToString(privada), base64.StdEncoding.EncodeToString(privada),
		fmt.Sprintf("%v", semente), fmt.Sprintf("%v", []byte(privada)),
		strings.Repeat(fmt.Sprintf("%d ", semente[0]), 8),
	}
	for _, forma := range formas {
		if strings.Contains(texto, strings.TrimSpace(forma)) {
			return true
		}
	}
	return false
}

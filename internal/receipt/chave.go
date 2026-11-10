package receipt

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
)

const tamanhoDoKeyID = 16

// ErrChaveInvalida indica semente ou chave pública fora do tamanho do Ed25519.
var ErrChaveInvalida = errors.New("receipt: chave de recibo inválida")

// Assinador assina recibos com a chave privada Ed25519. A chave nunca sai
// dele: imprimir ou registrar um Assinador mostra só o key_id.
type Assinador struct {
	chave   ed25519.PrivateKey
	idChave string
}

// NovoAssinador deriva a chave privada da semente de 32 bytes. Semente de
// outro tamanho devolve ErrChaveInvalida, sem citar o valor recebido.
func NovoAssinador(semente []byte) (*Assinador, error) {
	if len(semente) != ed25519.SeedSize {
		return nil, fmt.Errorf("%w: a semente precisa ter %d bytes, tem %d", ErrChaveInvalida, ed25519.SeedSize, len(semente))
	}

	chave := ed25519.NewKeyFromSeed(semente)
	return &Assinador{chave: chave, idChave: IDDaChave(publicaDe(chave))}, nil
}

// IDChave devolve o key_id da chave que assina.
func (a *Assinador) IDChave() string {
	return a.idChave
}

// ChavePublica devolve a chave pública com que os recibos são verificados.
func (a *Assinador) ChavePublica() ed25519.PublicKey {
	return publicaDe(a.chave)
}

// Assinar devolve a assinatura Ed25519, de 64 bytes, dos bytes informados.
func (a *Assinador) Assinar(conteudo []byte) []byte {
	return ed25519.Sign(a.chave, conteudo)
}

// String identifica o assinador pelo key_id.
func (a *Assinador) String() string {
	return "assinador de recibos (key_id " + a.idChave + ")"
}

// GoString identifica o assinador pelo key_id.
func (a *Assinador) GoString() string {
	return a.String()
}

// LogValue identifica o assinador pelo key_id nos registros de log.
func (a *Assinador) LogValue() slog.Value {
	return slog.StringValue(a.String())
}

func publicaDe(chave ed25519.PrivateKey) ed25519.PublicKey {
	publica := make(ed25519.PublicKey, ed25519.PublicKeySize)
	copy(publica, chave[ed25519.SeedSize:])
	return publica
}

// IDDaChave devolve o key_id de uma chave pública: os primeiros 16 caracteres
// hexadecimais do SHA-256 dela.
func IDDaChave(publica ed25519.PublicKey) string {
	resumo := sha256.Sum256(publica)
	return hex.EncodeToString(resumo[:])[:tamanhoDoKeyID]
}

// ChavePublica é uma chave de verificação com o key_id dela.
type ChavePublica struct {
	KeyID string
	Chave ed25519.PublicKey
}

// Chaveiro reúne as chaves públicas aceitas na verificação. É uma lista, e
// não uma chave só, para que recibos antigos continuem verificáveis depois de
// uma troca de chave.
type Chaveiro struct {
	chaves []ChavePublica
}

// NovoChaveiro monta o chaveiro na ordem informada, sem repetir chave.
func NovoChaveiro(publicas ...ed25519.PublicKey) *Chaveiro {
	chaveiro := &Chaveiro{}
	for _, publica := range publicas {
		if _, repetida := chaveiro.chave(IDDaChave(publica)); !repetida {
			chaveiro.chaves = append(chaveiro.chaves, ChavePublica{KeyID: IDDaChave(publica), Chave: publica})
		}
	}
	return chaveiro
}

// Lista devolve as chaves públicas do chaveiro, na ordem em que entraram.
func (c *Chaveiro) Lista() []ChavePublica {
	return append([]ChavePublica{}, c.chaves...)
}

func (c *Chaveiro) chave(keyID string) (ed25519.PublicKey, bool) {
	for _, candidata := range c.chaves {
		if candidata.KeyID == keyID {
			return candidata.Chave, true
		}
	}
	return nil, false
}

package eventos

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const prefixoDaAssinatura = "sha256="

// Cabeçalhos HTTP que levam a assinatura do evento.
const (
	CabecalhoTimestamp  = "X-Aprova-Timestamp"
	CabecalhoAssinatura = "X-Aprova-Assinatura"
)

// JanelaDeValidade é a diferença máxima aceita entre o timestamp assinado e o
// relógio de quem recebe, para os dois lados.
const JanelaDeValidade = 5 * time.Minute

// ErrAssinaturaRecusada indica evento que não prova ter vindo do núcleo.
var ErrAssinaturaRecusada = errors.New("eventos: assinatura recusada")

// Assinar devolve os valores dos cabeçalhos X-Aprova-Timestamp e
// X-Aprova-Assinatura para corpo enviado em enviadoEm.
func Assinar(segredo, corpo []byte, enviadoEm time.Time) (timestamp, assinatura string) {
	timestamp = strconv.FormatInt(enviadoEm.Unix(), 10)
	return timestamp, prefixoDaAssinatura + hex.EncodeToString(hmacDe(segredo, timestamp, corpo))
}

// Verificar confere a assinatura de corpo e recusa timestamp a mais de
// JanelaDeValidade de agora. O erro devolvido envolve ErrAssinaturaRecusada.
func Verificar(segredo, corpo []byte, timestamp, assinatura string, agora time.Time) error {
	if len(segredo) == 0 {
		return recusada("segredo não configurado")
	}

	segundos, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return recusada("timestamp ausente ou inválido")
	}

	hexadecimal, temPrefixo := strings.CutPrefix(assinatura, prefixoDaAssinatura)
	recebida, err := hex.DecodeString(hexadecimal)
	if !temPrefixo || err != nil {
		return recusada("assinatura ausente ou fora do formato sha256=<hex>")
	}
	if !hmac.Equal(recebida, hmacDe(segredo, timestamp, corpo)) {
		return recusada("assinatura não confere")
	}

	if agora.Sub(time.Unix(segundos, 0)).Abs() > JanelaDeValidade {
		return recusada("timestamp fora da janela de 5 minutos")
	}
	return nil
}

func hmacDe(segredo []byte, timestamp string, corpo []byte) []byte {
	calculo := hmac.New(sha256.New, segredo)
	calculo.Write([]byte(timestamp + "."))
	calculo.Write(corpo)
	return calculo.Sum(nil)
}

func recusada(motivo string) error {
	return fmt.Errorf("%w: %s", ErrAssinaturaRecusada, motivo)
}

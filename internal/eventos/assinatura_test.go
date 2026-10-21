package eventos

import (
	"errors"
	"strconv"
	"testing"
	"time"
)

var (
	segredoDeTeste = []byte("segredo-de-teste-dos-eventos")
	momentoDoEnvio = time.Unix(1760889600, 0)
	corpoDeTeste   = []byte(`{"tipo_evento":"instalacao"}`)
)

func TestAssinarSegueOFormatoDoContrato(t *testing.T) {
	timestamp, assinatura := Assinar(segredoDeTeste, corpoDeTeste, momentoDoEnvio)

	if timestamp != "1760889600" {
		t.Errorf("timestamp = %q, esperava os segundos Unix do envio", timestamp)
	}
	esperada := "sha256=486754b8dd4395de14807242332c479bf9087d241b3c925c270f4e7fa7865e8f"
	if assinatura != esperada {
		t.Errorf("assinatura = %q, esperava o HMAC-SHA256 de \"<timestamp>.<corpo>\" calculado pelo openssl: %q", assinatura, esperada)
	}
}

func TestAssinaturaGeradaEVerificada(t *testing.T) {
	timestamp, assinatura := Assinar(segredoDeTeste, corpoDeTeste, momentoDoEnvio)

	if err := Verificar(segredoDeTeste, corpoDeTeste, timestamp, assinatura, momentoDoEnvio.Add(4*time.Minute+59*time.Second)); err != nil {
		t.Errorf("assinatura recém-gerada recusada: %v", err)
	}
}

func TestVerificarRecusaEventoQueNaoProvaTerVindoDoNucleo(t *testing.T) {
	timestamp, assinatura := Assinar(segredoDeTeste, corpoDeTeste, momentoDoEnvio)
	_, assinaturaDeOutroMomento := Assinar(segredoDeTeste, corpoDeTeste, momentoDoEnvio.Add(time.Second))
	_, assinaturaSemSegredo := Assinar(nil, corpoDeTeste, momentoDoEnvio)

	casos := map[string]struct {
		segredo    []byte
		corpo      []byte
		timestamp  string
		assinatura string
		agora      time.Time
	}{
		"corpo alterado": {
			segredoDeTeste, []byte(`{"tipo_evento":"instalacao "}`), timestamp, assinatura, momentoDoEnvio,
		},
		"segredo errado": {
			[]byte("segredo-do-caminho-contrario"), corpoDeTeste, timestamp, assinatura, momentoDoEnvio,
		},
		"segredo vazio dos dois lados": {
			nil, corpoDeTeste, timestamp, assinaturaSemSegredo, momentoDoEnvio,
		},
		"timestamp com mais de 5 minutos": {
			segredoDeTeste, corpoDeTeste, timestamp, assinatura, momentoDoEnvio.Add(5*time.Minute + time.Second),
		},
		"timestamp mais de 5 minutos no futuro": {
			segredoDeTeste, corpoDeTeste, timestamp, assinatura, momentoDoEnvio.Add(-5*time.Minute - time.Second),
		},
		"timestamp trocado depois de assinar": {
			segredoDeTeste, corpoDeTeste, strconv.FormatInt(momentoDoEnvio.Unix()+1, 10), assinatura, momentoDoEnvio,
		},
		"assinatura de outro timestamp": {
			segredoDeTeste, corpoDeTeste, timestamp, assinaturaDeOutroMomento, momentoDoEnvio,
		},
		"timestamp ausente": {
			segredoDeTeste, corpoDeTeste, "", assinatura, momentoDoEnvio,
		},
		"timestamp que não é número": {
			segredoDeTeste, corpoDeTeste, "2026-10-19T14:32:08Z", assinatura, momentoDoEnvio,
		},
		"assinatura ausente": {
			segredoDeTeste, corpoDeTeste, timestamp, "", momentoDoEnvio,
		},
		"assinatura sem o prefixo sha256=": {
			segredoDeTeste, corpoDeTeste, timestamp, assinatura[len("sha256="):], momentoDoEnvio,
		},
		"assinatura que não é hexadecimal": {
			segredoDeTeste, corpoDeTeste, timestamp, "sha256=não-é-hex", momentoDoEnvio,
		},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			err := Verificar(caso.segredo, caso.corpo, caso.timestamp, caso.assinatura, caso.agora)

			if !errors.Is(err, ErrAssinaturaRecusada) {
				t.Errorf("esperava ErrAssinaturaRecusada, obtive %v", err)
			}
		})
	}
}

func TestCabecalhosDaAssinatura(t *testing.T) {
	if CabecalhoTimestamp != "X-Aprova-Timestamp" || CabecalhoAssinatura != "X-Aprova-Assinatura" {
		t.Errorf("cabeçalhos fora do contrato: %q e %q", CabecalhoTimestamp, CabecalhoAssinatura)
	}
	if JanelaDeValidade != 5*time.Minute {
		t.Errorf("janela de validade = %v, esperava 5 minutos", JanelaDeValidade)
	}
}

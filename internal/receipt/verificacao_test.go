package receipt

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/Azulpasta/aPRova/internal/domain"
)

var sementeDeTeste = bytes.Repeat([]byte{7}, 32)

func assinadorDeTeste(t *testing.T) *Assinador {
	t.Helper()

	assinador, err := NovoAssinador(sementeDeTeste)
	if err != nil {
		t.Fatalf("novo assinador: %v", err)
	}
	return assinador
}

func conteudoDeTeste(idChave string) Conteudo {
	return Conteudo{
		Versao:       VersaoDoFormato,
		ID:           "0b0f7a52-6f0e-4b53-9a51-5a9a4d3a8c11",
		TaskRunID:    "9c80d110-a49a-4a70-b5f4-78cdf75b2f19",
		TenantID:     161192271,
		Repositorio:  "Azulpasta/aPRova-TESTES",
		NumeroPR:     1,
		SHAHead:      "d7ee920e4f9f8a2f413630afbe6c25435149a302",
		DiffSHA256:   hex.EncodeToString(bytes.Repeat([]byte{0xab}, 32)),
		DiffTruncado: false,
		Sandbox: Sandbox{
			Compilou: true, TestesPassaram: 10, TestesFalharam: 1, TipoFalha: "teste", Tentativa: 1,
		},
		NivelDeterministico: domain.RiscoAlto,
		Nivel:               domain.RiscoAlto,
		Motivos:             []Motivo{{ID: "falha_sandbox", Evidencias: []string{"1 de 11 testes falharam"}}},
		EmitidoEm:           time.Date(2026, 11, 9, 12, 30, 0, 0, time.UTC),
		KeyID:               idChave,
	}
}

func reciboAssinadoDeTeste(t *testing.T) (Assinado, *Chaveiro) {
	t.Helper()

	assinador := assinadorDeTeste(t)
	bruto, err := conteudoDeTeste(assinador.IDChave()).Serializar()
	if err != nil {
		t.Fatalf("serializar: %v", err)
	}
	hash := sha256.Sum256(bruto)

	return Assinado{Conteudo: bruto, Hash: hash[:], Assinatura: assinador.Assinar(bruto), IDChave: assinador.IDChave()},
		NovoChaveiro(assinador.ChavePublica())
}

func TestAlterarUmUnicoByteDoConteudoInvalidaORecibo(t *testing.T) {
	recibo, chaveiro := reciboAssinadoDeTeste(t)
	if resultado := chaveiro.Verificar(recibo); !resultado.Valido {
		t.Fatalf("o recibo íntegro precisa ser válido: %+v", resultado)
	}

	for posicao := range recibo.Conteudo {
		adulterado := recibo
		adulterado.Conteudo = bytes.Clone(recibo.Conteudo)
		adulterado.Conteudo[posicao] ^= 0x01
		hash := sha256.Sum256(adulterado.Conteudo)
		adulterado.Hash = hash[:]

		if resultado := chaveiro.Verificar(adulterado); resultado.Valido {
			t.Fatalf("byte %d alterado e o recibo continuou válido", posicao)
		}
	}
}

func TestConteudoAlteradoSemRecalcularOHashFalhaNoHash(t *testing.T) {
	recibo, chaveiro := reciboAssinadoDeTeste(t)
	recibo.Conteudo = bytes.Replace(recibo.Conteudo, []byte(`"nivel":"alto"`), []byte(`"nivel":"baixo"`), 1)

	resultado := chaveiro.Verificar(recibo)

	if resultado.Valido || resultado.Falha != FalhaHash {
		t.Errorf("esperava falha de hash, obtive %+v", resultado)
	}
}

func TestConteudoAlteradoComHashRecalculadoFalhaNaAssinatura(t *testing.T) {
	recibo, chaveiro := reciboAssinadoDeTeste(t)
	recibo.Conteudo = bytes.Replace(recibo.Conteudo, []byte(`"nivel":"alto"`), []byte(`"nivel":"baixo"`), 1)
	hash := sha256.Sum256(recibo.Conteudo)
	recibo.Hash = hash[:]

	resultado := chaveiro.Verificar(recibo)

	if resultado.Valido || resultado.Falha != FalhaAssinatura {
		t.Errorf("esperava falha de assinatura, obtive %+v", resultado)
	}
}

func TestAssinaturaAlteradaInvalidaORecibo(t *testing.T) {
	recibo, chaveiro := reciboAssinadoDeTeste(t)

	for posicao := range recibo.Assinatura {
		adulterado := recibo
		adulterado.Assinatura = bytes.Clone(recibo.Assinatura)
		adulterado.Assinatura[posicao] ^= 0x01

		if resultado := chaveiro.Verificar(adulterado); resultado.Valido || resultado.Falha != FalhaAssinatura {
			t.Fatalf("byte %d da assinatura alterado: %+v", posicao, resultado)
		}
	}

	recibo.Assinatura = recibo.Assinatura[:63]
	if resultado := chaveiro.Verificar(recibo); resultado.Valido || resultado.Falha != FalhaAssinatura {
		t.Errorf("assinatura curta: %+v", resultado)
	}
}

func TestKeyIDDesconhecidoInvalidaORecibo(t *testing.T) {
	recibo, _ := reciboAssinadoDeTeste(t)
	outro, err := NovoAssinador(bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatal(err)
	}

	resultado := NovoChaveiro(outro.ChavePublica()).Verificar(recibo)

	if resultado.Valido || resultado.Falha != FalhaChaveDesconhecida {
		t.Errorf("esperava chave desconhecida, obtive %+v", resultado)
	}
	if resultado := NovoChaveiro().Verificar(recibo); resultado.Valido || resultado.Falha != FalhaChaveDesconhecida {
		t.Errorf("chaveiro vazio: %+v", resultado)
	}
}

func TestKeyIDDeForaDiferenteDoAssinadoInvalidaORecibo(t *testing.T) {
	assinador := assinadorDeTeste(t)
	outro, err := NovoAssinador(bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatal(err)
	}
	bruto, err := conteudoDeTeste(assinador.IDChave()).Serializar()
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(bruto)
	recibo := Assinado{Conteudo: bruto, Hash: hash[:], Assinatura: outro.Assinar(bruto), IDChave: outro.IDChave()}

	resultado := NovoChaveiro(assinador.ChavePublica(), outro.ChavePublica()).Verificar(recibo)

	if resultado.Valido || resultado.Falha != FalhaConteudo {
		t.Errorf("o key_id assinado precisa ser o da chave que assinou: %+v", resultado)
	}
}

func TestReciboDeChaveAntigaContinuaValidoDepoisDaTroca(t *testing.T) {
	recibo, _ := reciboAssinadoDeTeste(t)
	nova, err := NovoAssinador(bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatal(err)
	}

	chaveiro := NovoChaveiro(nova.ChavePublica(), assinadorDeTeste(t).ChavePublica())

	if resultado := chaveiro.Verificar(recibo); !resultado.Valido {
		t.Errorf("recibo antigo precisa continuar verificável: %+v", resultado)
	}
	if lista := chaveiro.Lista(); len(lista) != 2 || lista[0].KeyID != nova.IDChave() {
		t.Errorf("a lista preserva a ordem informada: %+v", lista)
	}
}

func TestConteudoQueNaoEJSONDoFormatoInvalidaORecibo(t *testing.T) {
	assinador := assinadorDeTeste(t)
	bruto := []byte(`{"versao":99}`)
	hash := sha256.Sum256(bruto)
	recibo := Assinado{Conteudo: bruto, Hash: hash[:], Assinatura: assinador.Assinar(bruto), IDChave: assinador.IDChave()}

	resultado := NovoChaveiro(assinador.ChavePublica()).Verificar(recibo)

	if resultado.Valido || resultado.Falha != FalhaConteudo {
		t.Errorf("versão desconhecida: %+v", resultado)
	}
}

func TestEloComOAnterior(t *testing.T) {
	assinador := assinadorDeTeste(t)
	primeiro := conteudoDeTeste(assinador.IDChave())
	brutoDoPrimeiro, err := primeiro.Serializar()
	if err != nil {
		t.Fatal(err)
	}
	segundo := conteudoDeTeste(assinador.IDChave())
	segundo.ID = "5d0a4c56-58f1-4e35-9d0e-2f1f0f6a7b22"
	segundo.Anterior = &Elo{ID: primeiro.ID, Hash: HashEmHex(brutoDoPrimeiro)}

	if falha := VerificarElo(primeiro, nil); falha != "" {
		t.Errorf("o primeiro da cadeia aponta para nada: %q", falha)
	}
	if falha := VerificarElo(segundo, &Anterior{ID: primeiro.ID, Conteudo: brutoDoPrimeiro}); falha != "" {
		t.Errorf("elo íntegro: %q", falha)
	}
	if falha := VerificarElo(segundo, nil); falha == "" {
		t.Error("anterior apagado precisa quebrar o elo")
	}
	if falha := VerificarElo(primeiro, &Anterior{ID: segundo.ID, Conteudo: brutoDoPrimeiro}); falha == "" {
		t.Error("recibo que diz ser o primeiro não pode ter anterior no banco")
	}
	if falha := VerificarElo(segundo, &Anterior{ID: "outro", Conteudo: brutoDoPrimeiro}); falha == "" {
		t.Error("id do anterior diferente precisa quebrar o elo")
	}
	adulterado := bytes.Clone(brutoDoPrimeiro)
	adulterado[10] ^= 0x01
	if falha := VerificarElo(segundo, &Anterior{ID: primeiro.ID, Conteudo: adulterado}); falha == "" {
		t.Error("anterior com conteúdo diferente precisa quebrar o elo")
	}
}

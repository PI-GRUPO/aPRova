package receipt

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSerializarDuasVezesDaOsMesmosBytes(t *testing.T) {
	conteudo := conteudoDeTeste("a1b2c3d4e5f60718")
	conteudo.Anterior = &Elo{ID: "5d0a4c56-58f1-4e35-9d0e-2f1f0f6a7b22", Hash: strings.Repeat("ab", 32)}

	primeira, err := conteudo.Serializar()
	if err != nil {
		t.Fatalf("serializar: %v", err)
	}
	for range 50 {
		outra, err := conteudo.Serializar()
		if err != nil {
			t.Fatalf("serializar: %v", err)
		}
		if !bytes.Equal(primeira, outra) {
			t.Fatalf("bytes diferentes:\n%s\n%s", primeira, outra)
		}
	}
}

func TestSerializarUsaOrdemFixaSemEspacos(t *testing.T) {
	bruto, err := conteudoDeTeste("a1b2c3d4e5f60718").Serializar()
	if err != nil {
		t.Fatal(err)
	}

	esperado := `{"versao":1,"id":"0b0f7a52-6f0e-4b53-9a51-5a9a4d3a8c11","task_run_id":"9c80d110-a49a-4a70-b5f4-78cdf75b2f19",` +
		`"tenant_id":161192271,"repositorio":"Azulpasta/aPRova-TESTES","numero_pr":1,` +
		`"head_sha":"d7ee920e4f9f8a2f413630afbe6c25435149a302",` +
		`"diff_sha256":"` + strings.Repeat("ab", 32) + `","diff_truncado":false,` +
		`"sandbox":{"compilou":true,"testes_passaram":10,"testes_falharam":1,"tipo_falha":"teste","tentativa":1},` +
		`"nivel_deterministico":"alto","nivel":"alto",` +
		`"motivos":[{"id":"falha_sandbox","evidencias":["1 de 11 testes falharam"]}],` +
		`"emitido_em":"2026-11-09T12:30:00.000000Z","anterior":null,"key_id":"a1b2c3d4e5f60718"}`
	if string(bruto) != esperado {
		t.Errorf("serialização fora do formato:\n obtive %s\nesperava %s", bruto, esperado)
	}
}

func TestSerializarNormalizaOInstanteParaUTCEListasVazias(t *testing.T) {
	conteudo := conteudoDeTeste("a1b2c3d4e5f60718")
	conteudo.EmitidoEm = time.Date(2026, 11, 9, 9, 30, 0, 123456789, time.FixedZone("BRT", -3*3600))
	conteudo.Motivos = nil

	bruto, err := conteudo.Serializar()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(bruto), `"emitido_em":"2026-11-09T12:30:00.123456Z"`) {
		t.Errorf("instante precisa sair em UTC com microssegundos: %s", bruto)
	}
	if !strings.Contains(string(bruto), `"motivos":[]`) {
		t.Errorf("lista vazia precisa sair como []: %s", bruto)
	}
}

func TestSerializarNaoEscapaHTMLEMantemUTF8(t *testing.T) {
	conteudo := conteudoDeTeste("a1b2c3d4e5f60718")
	conteudo.Motivos = []Motivo{{ID: "dependencia_nova", Evidencias: []string{"pom.xml: <dependency> não foi possível ler & interpretar"}}}

	bruto, err := conteudo.Serializar()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(bruto), "<dependency> não foi possível ler & interpretar") {
		t.Errorf("o texto precisa sair legível, em UTF-8: %s", bruto)
	}
	if !json.Valid(bruto) {
		t.Errorf("JSON inválido: %s", bruto)
	}
	if bytes.HasSuffix(bruto, []byte("\n")) {
		t.Error("sem quebra de linha no fim")
	}
}

func TestLerConteudoDevolveOQueFoiSerializado(t *testing.T) {
	original := conteudoDeTeste("a1b2c3d4e5f60718")
	original.Anterior = &Elo{ID: "5d0a4c56-58f1-4e35-9d0e-2f1f0f6a7b22", Hash: strings.Repeat("ab", 32)}
	bruto, err := original.Serializar()
	if err != nil {
		t.Fatal(err)
	}

	lido, err := LerConteudo(bruto)
	if err != nil {
		t.Fatalf("ler: %v", err)
	}

	denovo, err := lido.Serializar()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bruto, denovo) {
		t.Errorf("ler e serializar de novo mudou os bytes:\n%s\n%s", bruto, denovo)
	}
}

func TestLerConteudoRecusaVersaoDesconhecidaCampoExtraELixo(t *testing.T) {
	for nome, bruto := range map[string]string{
		"versão desconhecida": `{"versao":2}`,
		"campo extra":         `{"versao":1,"aprovado":true}`,
		"não é json":          `recibo`,
		"lixo depois":         `{"versao":1}{"versao":1}`,
	} {
		t.Run(nome, func(t *testing.T) {
			if _, err := LerConteudo([]byte(bruto)); err == nil {
				t.Error("esperava erro")
			}
		})
	}
}

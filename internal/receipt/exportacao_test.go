package receipt

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Azulpasta/aPRova/internal/domain"
)

func reciboGuardadoDeTeste(t *testing.T) domain.Recibo {
	t.Helper()

	guardados, ids, _ := cadeiaDeTeste(t, 2)
	return guardados[ids[1]]
}

func TestExportarLevaOsBytesExatosAAssinaturaEOConteudoLegivel(t *testing.T) {
	recibo := reciboGuardadoDeTeste(t)

	exportado := Exportar(recibo)

	if exportado.ID != recibo.ID || exportado.TaskRunID != recibo.TaskRunID || exportado.KeyID != recibo.IDChave {
		t.Errorf("identificação: %+v", exportado)
	}
	if exportado.PayloadBase64 != base64.StdEncoding.EncodeToString([]byte(recibo.Conteudo)) ||
		exportado.AssinaturaBase64 != base64.StdEncoding.EncodeToString(recibo.Assinatura) ||
		exportado.HashSHA256 != HashEmHex([]byte(recibo.Conteudo)) {
		t.Errorf("bytes exportados: %+v", exportado)
	}
	if exportado.ReciboAnteriorID == nil || *exportado.ReciboAnteriorID != *recibo.ReciboAnteriorID {
		t.Errorf("anterior: %v", exportado.ReciboAnteriorID)
	}
	if string(exportado.Payload) != recibo.Conteudo {
		t.Errorf("o conteúdo legível é o próprio conteúdo assinado: %s", exportado.Payload)
	}

	documento, err := json.Marshal(exportado)
	if err != nil {
		t.Fatal(err)
	}
	for _, campo := range []string{`"id"`, `"task_run_id"`, `"key_id"`, `"hash_sha256"`, `"payload_base64"`, `"assinatura_base64"`, `"recibo_anterior_id"`, `"payload"`} {
		if !strings.Contains(string(documento), campo) {
			t.Errorf("falta o campo %s em %s", campo, documento)
		}
	}
}

func TestVerificarExportadoAceitaOReciboSoComAChavePublica(t *testing.T) {
	documento, err := json.Marshal(Exportar(reciboGuardadoDeTeste(t)))
	if err != nil {
		t.Fatal(err)
	}

	conteudo, resultado, err := VerificarExportado(NovoChaveiro(assinadorDeTeste(t).ChavePublica()), documento)

	if err != nil || !resultado.Valido {
		t.Fatalf("resultado %+v, erro %v", resultado, err)
	}
	if conteudo.Repositorio != "Azulpasta/aPRova-TESTES" || conteudo.Nivel != domain.RiscoAlto {
		t.Errorf("o conteúdo devolvido vem dos bytes assinados: %+v", conteudo)
	}
}

func TestVerificarExportadoRecusaQualquerByteAlteradoDoPayload(t *testing.T) {
	exportado := Exportar(reciboGuardadoDeTeste(t))
	chaveiro := NovoChaveiro(assinadorDeTeste(t).ChavePublica())
	payload, err := base64.StdEncoding.DecodeString(exportado.PayloadBase64)
	if err != nil {
		t.Fatal(err)
	}

	for posicao := range payload {
		adulterado := exportado
		bytesAdulterados := append([]byte{}, payload...)
		bytesAdulterados[posicao] ^= 0x01
		adulterado.PayloadBase64 = base64.StdEncoding.EncodeToString(bytesAdulterados)
		documento, err := json.Marshal(adulterado)
		if err != nil {
			t.Fatal(err)
		}

		if _, resultado, _ := VerificarExportado(chaveiro, documento); resultado.Valido {
			t.Fatalf("byte %d do payload alterado e o recibo continuou válido", posicao)
		}

		adulterado.HashSHA256 = HashEmHex(bytesAdulterados)
		documento, err = json.Marshal(adulterado)
		if err != nil {
			t.Fatal(err)
		}
		if _, resultado, _ := VerificarExportado(chaveiro, documento); resultado.Valido {
			t.Fatalf("byte %d alterado com o hash recalculado e o recibo continuou válido", posicao)
		}
	}
}

func TestVerificarExportadoIgnoraOConteudoLegivelENaoConfiaNoIdDeFora(t *testing.T) {
	exportado := Exportar(reciboGuardadoDeTeste(t))
	chaveiro := NovoChaveiro(assinadorDeTeste(t).ChavePublica())

	exportado.Payload = json.RawMessage(`{"nivel":"baixo"}`)
	documento, _ := json.Marshal(exportado)
	conteudo, resultado, err := VerificarExportado(chaveiro, documento)
	if err != nil || !resultado.Valido || conteudo.Nivel != domain.RiscoAlto {
		t.Errorf("só os bytes assinados valem: %+v %+v %v", conteudo, resultado, err)
	}

	exportado.ID = "5d0a4c56-58f1-4e35-9d0e-2f1f0f6a7b22"
	documento, _ = json.Marshal(exportado)
	if _, resultado, _ := VerificarExportado(chaveiro, documento); resultado.Valido || resultado.Falha != FalhaConteudo {
		t.Errorf("o id de fora precisa ser o id assinado: %+v", resultado)
	}
}

func TestVerificarExportadoRecusaDocumentoMalformado(t *testing.T) {
	chaveiro := NovoChaveiro(assinadorDeTeste(t).ChavePublica())
	valido := Exportar(reciboGuardadoDeTeste(t))

	if _, _, err := VerificarExportado(chaveiro, []byte("isto não é json")); err == nil {
		t.Error("documento que não é JSON é erro de entrada")
	}

	for nome, estragar := range map[string]func(*Exportado){
		"payload fora de base64":    func(e *Exportado) { e.PayloadBase64 = "!!!" },
		"assinatura fora de base64": func(e *Exportado) { e.AssinaturaBase64 = "!!!" },
		"hash fora de hexadecimal":  func(e *Exportado) { e.HashSHA256 = "zz" },
		"assinatura vazia":          func(e *Exportado) { e.AssinaturaBase64 = "" },
	} {
		t.Run(nome, func(t *testing.T) {
			estragado := valido
			estragar(&estragado)
			documento, _ := json.Marshal(estragado)

			_, resultado, err := VerificarExportado(chaveiro, documento)

			if err != nil || resultado.Valido || resultado.Falha == "" {
				t.Errorf("esperava recibo reprovado com a falha nomeada: %+v, erro %v", resultado, err)
			}
		})
	}
}

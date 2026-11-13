package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
)

type decisaoEmMemoria struct{ gravado domain.Recibo }

func (d *decisaoEmMemoria) GravarDecisaoComRecibo(_ context.Context, _ domain.GateDeRisco, montar func(anterior *domain.Recibo) (domain.NovoRecibo, error)) (domain.Recibo, error) {
	novo, err := montar(nil)
	d.gravado = domain.Recibo{NovoRecibo: novo, TenantID: 161192271}
	return d.gravado, err
}

func reciboExportado(t *testing.T) (receipt.Exportado, string) {
	t.Helper()

	assinador, err := receipt.NovoAssinador(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	guardado := &decisaoEmMemoria{}
	emissor, err := receipt.NovoEmissor(receipt.OpcoesEmissor{Assinador: assinador, Armazenamento: guardado})
	if err != nil {
		t.Fatal(err)
	}
	_, err = emissor.Emitir(t.Context(), receipt.Pedido{
		TaskRun: domain.TaskRun{ID: "9c80d110-a49a-4a70-b5f4-78cdf75b2f19", TenantID: 161192271, Repositorio: "Azulpasta/aPRova-TESTES", NumeroPR: 1, SHAHead: "d7ee920e4f9f8a2f413630afbe6c25435149a302"},
		Gate: domain.GateDeRisco{
			NivelDeterministico: domain.RiscoAlto, Nivel: domain.RiscoAlto,
			Motivos: []domain.MotivoDeRisco{{ID: "falha_sandbox", Evidencias: []string{"1 de 11 testes falharam"}}},
		},
		Tentativa: 1, Diff: "diff",
	})
	if err != nil {
		t.Fatal(err)
	}
	return receipt.Exportar(guardado.gravado), base64.StdEncoding.EncodeToString(assinador.ChavePublica())
}

func rodar(t *testing.T, entrada string, argumentos ...string) (codigo int, saida, erros string) {
	t.Helper()

	var bufferDeSaida, bufferDeErros bytes.Buffer
	codigo = executar(argumentos, strings.NewReader(entrada), &bufferDeSaida, &bufferDeErros)
	return codigo, bufferDeSaida.String(), bufferDeErros.String()
}

func emJSON(t *testing.T, exportado receipt.Exportado) string {
	t.Helper()

	documento, err := json.Marshal(exportado)
	if err != nil {
		t.Fatal(err)
	}
	return string(documento)
}

func TestReciboIntegroPelaEntradaPadraoEValido(t *testing.T) {
	exportado, chave := reciboExportado(t)

	codigo, saida, erros := rodar(t, emJSON(t, exportado), "-chave", chave)

	if codigo != 0 || !strings.HasPrefix(saida, "VÁLIDO") {
		t.Fatalf("código %d, saída %q, erros %q", codigo, saida, erros)
	}
	for _, trecho := range []string{exportado.ID, exportado.KeyID, exportado.HashSHA256, "Azulpasta/aPRova-TESTES", "PR #1", "d7ee920e4f9f8a2f413630afbe6c25435149a302", "alto", "falha_sandbox", "1 de 11 testes falharam"} {
		if !strings.Contains(saida, trecho) {
			t.Errorf("a saída mostra o que foi assinado; falta %q em:\n%s", trecho, saida)
		}
	}
}

func TestReciboLidoDeArquivoEValido(t *testing.T) {
	exportado, chave := reciboExportado(t)
	arquivo := filepath.Join(t.TempDir(), "recibo.json")
	if err := os.WriteFile(arquivo, []byte(emJSON(t, exportado)), 0o600); err != nil {
		t.Fatal(err)
	}

	codigo, saida, erros := rodar(t, "", "-chave", chave, arquivo)

	if codigo != 0 || !strings.HasPrefix(saida, "VÁLIDO") {
		t.Fatalf("código %d, saída %q, erros %q", codigo, saida, erros)
	}
}

func TestUmByteAlteradoNoPayloadEInvalido(t *testing.T) {
	exportado, chave := reciboExportado(t)
	payload, err := base64.StdEncoding.DecodeString(exportado.PayloadBase64)
	if err != nil {
		t.Fatal(err)
	}
	payload[len(payload)/2] ^= 0x01
	exportado.PayloadBase64 = base64.StdEncoding.EncodeToString(payload)

	codigo, saida, _ := rodar(t, emJSON(t, exportado), "-chave", chave)

	if codigo != 1 || !strings.HasPrefix(saida, "INVÁLIDO") || !strings.Contains(saida, "hash") {
		t.Errorf("código %d, saída %q", codigo, saida)
	}

	exportado.HashSHA256 = receipt.HashEmHex(payload)
	codigo, saida, _ = rodar(t, emJSON(t, exportado), "-chave", chave)

	if codigo != 1 || !strings.HasPrefix(saida, "INVÁLIDO") || !strings.Contains(saida, "assinatura") {
		t.Errorf("com o hash recalculado a assinatura reprova: código %d, saída %q", codigo, saida)
	}
}

func TestChaveDeOutroParEInvalido(t *testing.T) {
	exportado, _ := reciboExportado(t)
	outra, err := receipt.NovoAssinador(bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatal(err)
	}

	codigo, saida, _ := rodar(t, emJSON(t, exportado), "-chave", base64.StdEncoding.EncodeToString(outra.ChavePublica()))

	if codigo != 1 || !strings.HasPrefix(saida, "INVÁLIDO") || !strings.Contains(saida, "chave_desconhecida") {
		t.Errorf("código %d, saída %q", codigo, saida)
	}
}

func TestEntradaOuChaveInutilizaveisSaemComCodigoDeUso(t *testing.T) {
	exportado, chave := reciboExportado(t)
	documento := emJSON(t, exportado)

	casos := map[string]struct {
		entrada    string
		argumentos []string
	}{
		"sem chave":               {documento, nil},
		"chave fora de base64":    {documento, []string{"-chave", "!!!"}},
		"chave de tamanho errado": {documento, []string{"-chave", base64.StdEncoding.EncodeToString([]byte("curta"))}},
		"entrada que não é json":  {"isto não é um recibo", []string{"-chave", chave}},
		"arquivo inexistente":     {"", []string{"-chave", chave, filepath.Join(t.TempDir(), "nao-existe.json")}},
		"argumentos demais":       {documento, []string{"-chave", chave, "a.json", "b.json"}},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			codigo, saida, erros := rodar(t, caso.entrada, caso.argumentos...)

			if codigo != 2 || strings.Contains(saida, "VÁLIDO") || erros == "" {
				t.Errorf("código %d, saída %q, erros %q", codigo, saida, erros)
			}
		})
	}
}

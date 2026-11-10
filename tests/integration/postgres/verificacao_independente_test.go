package integracaopostgres

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Azulpasta/aPRova/internal/api"
	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/receipt"
)

func (b bancoDaFila) pedirPelaAPI(t *testing.T, caminho string) *httptest.ResponseRecorder {
	t.Helper()

	servidor := api.NovoServidor(api.Opcoes{Recibos: verificadorDeRecibos(t, b.repo)}).Rotas()
	gravador := httptest.NewRecorder()
	servidor.ServeHTTP(gravador, httptest.NewRequestWithContext(t.Context(), http.MethodGet, caminho, nil))
	return gravador
}

func construirVerificador(t *testing.T) string {
	t.Helper()

	binario := filepath.Join(t.TempDir(), "verificar-recibo")
	construcao := exec.CommandContext(t.Context(), "go", "build", "-o", binario, "github.com/Azulpasta/aPRova/cmd/verificar-recibo")
	if saida, err := construcao.CombinedOutput(); err != nil {
		t.Fatalf("construir cmd/verificar-recibo: %v\n%s", err, saida)
	}
	return binario
}

func rodarVerificador(t *testing.T, binario, documento, chave string) (int, string) {
	t.Helper()

	comando := exec.CommandContext(t.Context(), binario, "-chave", chave)
	comando.Stdin = strings.NewReader(documento)
	comando.Env = []string{}
	saida, err := comando.CombinedOutput()

	var saiuComCodigo *exec.ExitError
	switch {
	case err == nil:
		return 0, string(saida)
	case errors.As(err, &saiuComCodigo):
		return saiuComCodigo.ExitCode(), string(saida)
	default:
		t.Fatalf("rodar o verificador: %v", err)
		return 0, ""
	}
}

func TestExecutavelVerificaReciboRealDoFluxoERecusaUmByteAlterado(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)
	novaMontagemDoGate(banco, arquivoDeRiscoAlto).rodarAte(t, taskRun.ID, domain.EstadoAguardandoAprovacao)
	gravado, err := banco.repo.BuscarReciboPorTaskRun(t.Context(), taskRun.ID)
	if err != nil {
		t.Fatalf("recibo do fluxo: %v", err)
	}

	resposta := banco.pedirPelaAPI(t, "/receipts/"+gravado.ID)
	if resposta.Code != http.StatusOK {
		t.Fatalf("GET /receipts/{id}: status %d, %s", resposta.Code, resposta.Body.String())
	}
	documento := resposta.Body.String()
	chavePublica := base64.StdEncoding.EncodeToString(assinadorDeTeste(t).ChavePublica())
	binario := construirVerificador(t)

	codigo, saida := rodarVerificador(t, binario, documento, chavePublica)
	if codigo != 0 || !strings.HasPrefix(saida, "VÁLIDO") || !strings.Contains(saida, "dependencia_nova") {
		t.Fatalf("o recibo real precisa ser aceito: código %d\n%s", codigo, saida)
	}

	var exportado receipt.Exportado
	if err := json.Unmarshal([]byte(documento), &exportado); err != nil {
		t.Fatal(err)
	}
	payload, err := base64.StdEncoding.DecodeString(exportado.PayloadBase64)
	if err != nil {
		t.Fatal(err)
	}
	for _, posicao := range []int{0, len(payload) / 2, len(payload) - 1} {
		adulterado := exportado
		bytesAdulterados := bytes.Clone(payload)
		bytesAdulterados[posicao] ^= 0x01
		adulterado.PayloadBase64 = base64.StdEncoding.EncodeToString(bytesAdulterados)
		adulterado.HashSHA256 = receipt.HashEmHex(bytesAdulterados)
		documentoAdulterado, err := json.Marshal(adulterado)
		if err != nil {
			t.Fatal(err)
		}

		codigo, saida := rodarVerificador(t, binario, string(documentoAdulterado), chavePublica)
		if codigo != 1 || !strings.HasPrefix(saida, "INVÁLIDO") {
			t.Errorf("byte %d alterado: código %d\n%s", posicao, codigo, saida)
		}
	}
}

func TestEndpointsDeReciboSobreOBancoReal(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)
	novaMontagemDoGate(banco, arquivoDeRiscoBaixo).rodarAte(t, taskRun.ID, domain.EstadoConcluido)
	gravado, err := banco.repo.BuscarReciboPorTaskRun(t.Context(), taskRun.ID)
	if err != nil {
		t.Fatalf("recibo do fluxo: %v", err)
	}

	verificacao := banco.pedirPelaAPI(t, "/receipts/"+gravado.ID+"/verify")
	if verificacao.Code != http.StatusOK || !strings.Contains(verificacao.Body.String(), `"valido":true`) {
		t.Errorf("verify: status %d, %s", verificacao.Code, verificacao.Body.String())
	}

	for _, caminho := range []string{
		"/receipts/5d0a4c56-58f1-4e35-9d0e-2f1f0f6a7b22",
		"/receipts/5d0a4c56-58f1-4e35-9d0e-2f1f0f6a7b22/verify",
		"/receipts/nao-e-uuid",
		"/receipts/nao-e-uuid/verify",
	} {
		if resposta := banco.pedirPelaAPI(t, caminho); resposta.Code != http.StatusNotFound {
			t.Errorf("%s: status %d, esperava 404", caminho, resposta.Code)
		}
	}
}

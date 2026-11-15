package platform

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"testing"
)

const shaDeTesteDoSARIF = "0123456789abcdef0123456789abcdef01234567"

func descompactar(t *testing.T, emBase64 string) []byte {
	t.Helper()

	compactado, err := base64.StdEncoding.DecodeString(emBase64)
	if err != nil {
		t.Fatalf("o campo sarif não é base64: %v", err)
	}
	leitor, err := gzip.NewReader(bytes.NewReader(compactado))
	if err != nil {
		t.Fatalf("o campo sarif não é gzip: %v", err)
	}
	conteudo, err := io.ReadAll(leitor)
	if err != nil {
		t.Fatalf("ler o gzip: %v", err)
	}
	return conteudo
}

func TestEnviarSARIFMandaODocumentoEmGzipEBase64ComCommitERef(t *testing.T) {
	servidor, recebida := servidorDaAPIQueResponde(t, http.StatusAccepted, `{"id":"47177e22-5596-11eb-80a1-c1e54ef945c6","url":"https://api.github.com/x"}`)
	documento := []byte(`{"version":"2.1.0","runs":[]}`)

	id, err := clienteContra(servidor, &fornecedorDeTokenFalso{}).EnviarSARIF(t.Context(), 7001, "Azulpasta", "aprova-bench-go",
		EnvioSARIF{CommitSHA: shaDeTesteDoSARIF, Ref: "refs/pull/12/head", Documento: documento})

	if err != nil || id != "47177e22-5596-11eb-80a1-c1e54ef945c6" {
		t.Fatalf("id = %q, erro = %v", id, err)
	}
	if recebida.metodo != http.MethodPost || recebida.caminho != "/repos/Azulpasta/aprova-bench-go/code-scanning/sarifs" {
		t.Errorf("chamada errada: %s %s", recebida.metodo, recebida.caminho)
	}
	if recebida.corpo["commit_sha"] != shaDeTesteDoSARIF || recebida.corpo["ref"] != "refs/pull/12/head" {
		t.Errorf("commit e ref errados: %v e %v", recebida.corpo["commit_sha"], recebida.corpo["ref"])
	}
	if recebida.corpo["tool_name"] != "aPRova" {
		t.Errorf("tool_name = %v, esperado aPRova", recebida.corpo["tool_name"])
	}
	emBase64, _ := recebida.corpo["sarif"].(string)
	if obtido := descompactar(t, emBase64); !bytes.Equal(obtido, documento) {
		t.Errorf("documento recebido = %s", obtido)
	}
}

func TestEnviarSARIFSemIdNaRespostaEhErro(t *testing.T) {
	servidor, _ := servidorDaAPIQueResponde(t, http.StatusAccepted, `{}`)

	_, err := clienteContra(servidor, &fornecedorDeTokenFalso{}).EnviarSARIF(t.Context(), 7001, "Azulpasta", "alvo", EnvioSARIF{})

	if !errors.Is(err, ErrRespostaInesperada) {
		t.Errorf("sem id não há como acompanhar o envio: %v", err)
	}
}

func TestEnviarSARIFClassificaAsFalhasDoGitHub(t *testing.T) {
	casos := map[string]struct {
		status   int
		esperado error
	}{
		"502 é transitório":                   {http.StatusBadGateway, ErrGitHubIndisponivel},
		"503 é transitório":                   {http.StatusServiceUnavailable, ErrGitHubIndisponivel},
		"429 é limite de requisições":         {http.StatusTooManyRequests, ErrLimiteDeRequisicoes},
		"403 é falta de permissão ou de GHAS": {http.StatusForbidden, ErrSemPermissao},
		"400 é SARIF recusado":                {http.StatusBadRequest, ErrRespostaInesperada},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			servidor, _ := servidorDaAPIQueResponde(t, caso.status, `{"message":"x"}`)

			_, err := clienteContra(servidor, &fornecedorDeTokenFalso{}).EnviarSARIF(t.Context(), 7001, "Azulpasta", "alvo", EnvioSARIF{})

			if !errors.Is(err, caso.esperado) {
				t.Errorf("erro = %v, esperado %v", err, caso.esperado)
			}
		})
	}
}

func TestProcessamentoDoSARIFDevolveASituacaoEOsErros(t *testing.T) {
	servidor, recebida := servidorDaAPIQueResponde(t, http.StatusOK, `{"processing_status":"failed","errors":["linha fora do arquivo"]}`)

	processamento, err := clienteContra(servidor, &fornecedorDeTokenFalso{}).ProcessamentoDoSARIF(t.Context(), 7001, "Azulpasta", "alvo", "id-do-envio")

	if err != nil {
		t.Fatalf("consulta não deveria falhar: %v", err)
	}
	if recebida.metodo != http.MethodGet || recebida.caminho != "/repos/Azulpasta/alvo/code-scanning/sarifs/id-do-envio" {
		t.Errorf("chamada errada: %s %s", recebida.metodo, recebida.caminho)
	}
	if processamento.Situacao != SARIFRecusado || len(processamento.Erros) != 1 || processamento.Erros[0] != "linha fora do arquivo" {
		t.Errorf("processamento = %+v", processamento)
	}
}

func TestArquivosDoPullRequestDevolveOsPatchesSemLerODiff(t *testing.T) {
	servidor, recebida := servidorDaAPIQueResponde(t, http.StatusOK, `[{"filename":"go.mod","status":"modified","patch":"@@ -1 +1,2 @@\n+x"}]`)

	arquivos, err := clienteContra(servidor, &fornecedorDeTokenFalso{}).ArquivosDoPullRequest(t.Context(), 7001, "Azulpasta", "alvo", 12)

	if err != nil || len(arquivos) != 1 || arquivos[0].Nome != "go.mod" || arquivos[0].Patch == "" {
		t.Fatalf("arquivos = %+v, erro = %v", arquivos, err)
	}
	if recebida.quantasVieram != 1 || recebida.caminho != "/repos/Azulpasta/alvo/pulls/12/files" {
		t.Errorf("esperava uma chamada só, a de arquivos; vieram %d, a última em %s", recebida.quantasVieram, recebida.caminho)
	}
}

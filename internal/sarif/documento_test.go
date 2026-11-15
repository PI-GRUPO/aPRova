package sarif

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/platform"
)

const (
	caminhoDoSchema = "testdata/sarif-schema-2.1.0.json"
	idDoSchema      = "https://docs.oasis-open.org/sarif/sarif/v2.1.0/errata01/os/schemas/sarif-schema-2.1.0.json"
	caminhoDoOuro   = "testdata/todos_os_motivos.sarif"
)

type documentoLido struct {
	Schema string `json:"$schema"`
	Versao string `json:"version"`
	Runs   []struct {
		Tool struct {
			Driver struct {
				Name           string `json:"name"`
				Version        string `json:"version"`
				InformationURI string `json:"informationUri"`
				Rules          []struct {
					ID               string `json:"id"`
					ShortDescription struct {
						Text string `json:"text"`
					} `json:"shortDescription"`
					Help struct {
						Text string `json:"text"`
					} `json:"help"`
					DefaultConfiguration struct {
						Level string `json:"level"`
					} `json:"defaultConfiguration"`
					Properties map[string]any `json:"properties"`
				} `json:"rules"`
			} `json:"driver"`
		} `json:"tool"`
		Results    []resultadoLido `json:"results"`
		Properties struct {
			Aprova struct {
				NivelDeterministico string `json:"nivelDeterministico"`
				Nivel               string `json:"nivel"`
				Calibracao          struct {
					Estado        string `json:"estado"`
					ReduziuONivel bool   `json:"reduziuONivel"`
				} `json:"calibracao"`
				MotivosSemResultado []struct {
					ID    string `json:"id"`
					Razao string `json:"razao"`
				} `json:"motivosSemResultado"`
			} `json:"aprova"`
		} `json:"properties"`
	} `json:"runs"`
}

type resultadoLido struct {
	RuleID    string `json:"ruleId"`
	RuleIndex int    `json:"ruleIndex"`
	Level     string `json:"level"`
	Message   struct {
		Text string `json:"text"`
	} `json:"message"`
	Locations []struct {
		PhysicalLocation struct {
			ArtifactLocation struct {
				URI string `json:"uri"`
			} `json:"artifactLocation"`
			Region struct {
				StartLine int `json:"startLine"`
			} `json:"region"`
		} `json:"physicalLocation"`
	} `json:"locations"`
	PartialFingerprints map[string]string `json:"partialFingerprints"`
	Properties          struct {
		Recibo struct {
			ID     string `json:"id"`
			SHA256 string `json:"sha256"`
		} `json:"recibo"`
	} `json:"properties"`
}

func (r resultadoLido) arquivo() string {
	return r.Locations[0].PhysicalLocation.ArtifactLocation.URI
}

func (r resultadoLido) linha() int {
	return r.Locations[0].PhysicalLocation.Region.StartLine
}

func reciboDeTeste() Recibo {
	return Recibo{ID: "0b0f7a52-6a55-4d0e-9a86-3d3c1a2b4c5d", Hash: bytes.Repeat([]byte{0xab}, 32)}
}

func motivo(id string, nivel domain.NivelRisco, evidencias ...string) domain.MotivoDeRisco {
	return domain.MotivoDeRisco{ID: id, Nivel: nivel, Descricao: "descrição de " + id, Evidencias: evidencias}
}

func alterado(nome, patch string) platform.ArquivoAlterado {
	return platform.ArquivoAlterado{Nome: nome, Situacao: "modified", Patch: patch}
}

func entradaDoPR1() Entrada {
	return Entrada{
		Versao: "teste",
		Recibo: reciboDeTeste(),
		Gate: domain.GateDeRisco{
			NivelDeterministico: domain.RiscoAlto,
			Nivel:               domain.RiscoAlto,
			Motivos:             []domain.MotivoDeRisco{motivo("falha_sandbox", domain.RiscoAlto, "1 de 4 testes falharam")},
			Calibracao:          domain.Calibracao{Estado: domain.CalibracaoDesligada},
		},
		Arquivos: []platform.ArquivoAlterado{
			alterado("peso/peso_test.go", "@@ -10,3 +10,9 @@ func TestPeso(t *testing.T) {\n contexto\n+func TestPesoZero(t *testing.T) {\n+}\n"),
		},
	}
}

const patchDoGoMod = "@@ -5,6 +5,8 @@ go 1.26\n require (\n \tgithub.com/go-chi/chi/v5 v5.3.2\n+\tgithub.com/acme/util v1.0.0\n+\tgithub.com/acme/nova v1.2.3\n \tgithub.com/google/uuid v1.6.0\n )\n"

func entradaComTodosOsMotivos() Entrada {
	return Entrada{
		Versao: "teste",
		Recibo: reciboDeTeste(),
		Gate: domain.GateDeRisco{
			NivelDeterministico: domain.RiscoAlto,
			Nivel:               domain.RiscoMedio,
			Motivos: []domain.MotivoDeRisco{
				motivo("area_sensivel", domain.RiscoAlto, "auth/sessao.go", "migrations/000001_antiga.up.sql"),
				motivo("calibracao_historico", domain.NivelInformativo, "3 vizinhos aprovados e mergeados"),
				motivo("codigo_sem_teste", domain.RiscoMedio, "auth/sessao.go", "pagamento/cobranca.go"),
				motivo("configuracao_ci", domain.RiscoAlto, ".github/workflows/ci.yml"),
				motivo("dependencia_nova", domain.RiscoAlto, "go.mod"),
				motivo("falha_sandbox", domain.RiscoAlto, "2 de 9 testes falharam"),
				motivo("muitos_arquivos", domain.RiscoAlto, ".env", ".github/workflows/ci.yml", "auth/sessao.go", "go.mod"),
				motivo("segredo_versionado", domain.RiscoAlto, ".env"),
			},
			DependenciasNovas: []domain.DependenciaNova{{Ecossistema: "Go", Nome: "github.com/acme/nova", Versao: "v1.2.3"}},
			Calibracao:        domain.Calibracao{Estado: domain.CalibracaoAplicada},
		},
		Arquivos: []platform.ArquivoAlterado{
			alterado(".env", "@@ -0,0 +1,2 @@\n+CHAVE=valor\n+OUTRA=valor\n"),
			alterado(".github/workflows/ci.yml", "@@ -20,4 +20,5 @@ jobs:\n     steps:\n-      - run: make test\n+      - run: make test lint\n+      - run: make build\n"),
			alterado("auth/sessao.go", "@@ -40,3 +40,4 @@ func Abrir() {\n \tantes()\n \tmeio()\n+\tdepois()\n"),
			alterado("go.mod", patchDoGoMod),
			{Nome: "migrations/000001_antiga.up.sql", Situacao: "removed", Patch: "@@ -1,2 +0,0 @@\n-CREATE TABLE antiga ();\n"},
			alterado("pagamento/cobranca.go", ""),
		},
	}
}

func montar(t *testing.T, entrada Entrada) (documentoLido, []byte) {
	t.Helper()

	serializado, err := Montar(entrada)
	if err != nil {
		t.Fatalf("Montar devolveu erro: %v", err)
	}
	var lido documentoLido
	if err := json.Unmarshal(serializado, &lido); err != nil {
		t.Fatalf("o SARIF não é JSON: %v", err)
	}
	if len(lido.Runs) != 1 {
		t.Fatalf("o SARIF tem %d runs, esperado 1", len(lido.Runs))
	}
	return lido, serializado
}

func resultadosDe(t *testing.T, entrada Entrada) []resultadoLido {
	t.Helper()
	lido, _ := montar(t, entrada)
	return lido.Runs[0].Results
}

func resultadoDoMotivo(t *testing.T, resultados []resultadoLido, id string) resultadoLido {
	t.Helper()
	doMotivo := slices.DeleteFunc(slices.Clone(resultados), func(r resultadoLido) bool { return r.RuleID != id })
	if len(doMotivo) != 1 {
		t.Fatalf("o motivo %s tem %d resultados, esperado 1", id, len(doMotivo))
	}
	return doMotivo[0]
}

func validarContraOSchema(t *testing.T, serializado []byte) {
	t.Helper()

	arquivo, err := os.Open(filepath.FromSlash(caminhoDoSchema))
	if err != nil {
		t.Fatalf("abrir o schema versionado: %v", err)
	}
	defer func() { _ = arquivo.Close() }()

	schema, err := jsonschema.UnmarshalJSON(arquivo)
	if err != nil {
		t.Fatalf("ler o schema versionado: %v", err)
	}
	compilador := jsonschema.NewCompiler()
	if err := compilador.AddResource(idDoSchema, schema); err != nil {
		t.Fatalf("registrar o schema: %v", err)
	}
	compilado, err := compilador.Compile(idDoSchema)
	if err != nil {
		t.Fatalf("compilar o schema: %v", err)
	}

	documento, err := jsonschema.UnmarshalJSON(bytes.NewReader(serializado))
	if err != nil {
		t.Fatalf("o SARIF não é JSON: %v", err)
	}
	if err := compilado.Validate(documento); err != nil {
		t.Fatalf("o SARIF não passa no schema 2.1.0: %v", err)
	}
}

func TestSARIFDoPR1DoAprovaTestesEhValidoESemResultados(t *testing.T) {
	lido, serializado := montar(t, entradaDoPR1())

	validarContraOSchema(t, serializado)

	if lido.Versao != "2.1.0" {
		t.Errorf("version = %q, esperado 2.1.0", lido.Versao)
	}
	if len(lido.Runs[0].Results) != 0 {
		t.Errorf("o PR #1 só tem falha_sandbox, que não vira resultado; vieram %d", len(lido.Runs[0].Results))
	}
	if !bytes.Contains(serializado, []byte(`"results":[]`)) {
		t.Error("results precisa ser uma lista vazia, não nulo nem ausente: é ela que diz ao GitHub que a análise rodou")
	}
}

func TestSchemaVersionadoEhODaErrata01(t *testing.T) {
	arquivo, err := os.ReadFile(filepath.FromSlash(caminhoDoSchema))
	if err != nil {
		t.Fatalf("ler o schema versionado: %v", err)
	}
	if !bytes.Contains(arquivo, []byte(idDoSchema)) {
		t.Fatalf("o schema versionado não é o da errata 01 do SARIF 2.1.0")
	}
}

func TestCadaMotivoDoGateGeraSARIFValido(t *testing.T) {
	completa := entradaComTodosOsMotivos()

	for _, doGate := range completa.Gate.Motivos {
		t.Run(doGate.ID, func(t *testing.T) {
			entrada := completa
			entrada.Gate.Motivos = []domain.MotivoDeRisco{doGate}
			_, serializado := montar(t, entrada)
			validarContraOSchema(t, serializado)
		})
	}

	t.Run("todos juntos", func(t *testing.T) {
		_, serializado := montar(t, completa)
		validarContraOSchema(t, serializado)
	})
}

func TestFerramentaDeclaraNomeVersaoERepositorio(t *testing.T) {
	lido, _ := montar(t, entradaComTodosOsMotivos())
	driver := lido.Runs[0].Tool.Driver

	if driver.Name != "aPRova" || driver.Version != "teste" || driver.InformationURI != "https://github.com/PI-GRUPO/aPRova" {
		t.Errorf("driver = %q %q %q", driver.Name, driver.Version, driver.InformationURI)
	}
}

func TestHaUmaRegraPorMotivoDoGateSemGravidadeDeSeguranca(t *testing.T) {
	lido, _ := montar(t, entradaDoPR1())

	var ids []string
	for _, regra := range lido.Runs[0].Tool.Driver.Rules {
		ids = append(ids, regra.ID)
		if regra.ShortDescription.Text == "" || regra.Help.Text == "" {
			t.Errorf("a regra %s precisa de descrição curta e de help", regra.ID)
		}
		if _, tem := regra.Properties["security-severity"]; tem {
			t.Errorf("a regra %s não pode ter security-severity: o SARIF não reprova o PR", regra.ID)
		}
	}

	esperados := []string{"area_sensivel", "codigo_sem_teste", "configuracao_ci", "dependencia_nova", "falha_sandbox", "muitos_arquivos", "segredo_versionado"}
	if !slices.Equal(ids, esperados) {
		t.Errorf("regras = %v, esperado %v", ids, esperados)
	}
}

func TestMotivoAltoViraWarningEMedioViraNote(t *testing.T) {
	lido, serializado := montar(t, entradaComTodosOsMotivos())
	resultados := lido.Runs[0].Results

	if nivel := resultadoDoMotivo(t, resultados, "segredo_versionado").Level; nivel != "warning" {
		t.Errorf("motivo alto virou %q, esperado warning", nivel)
	}
	if nivel := resultadoDoMotivo(t, resultados, "codigo_sem_teste").Level; nivel != "note" {
		t.Errorf("motivo médio virou %q, esperado note", nivel)
	}
	if bytes.Contains(serializado, []byte(`"error"`)) {
		t.Error("nenhum nível pode ser error: é ele que reprova o check Code scanning results")
	}
	for _, regra := range lido.Runs[0].Tool.Driver.Rules {
		esperado := "warning"
		if regra.ID == "codigo_sem_teste" {
			esperado = "note"
		}
		if regra.DefaultConfiguration.Level != esperado {
			t.Errorf("nível padrão da regra %s = %q, esperado %q", regra.ID, regra.DefaultConfiguration.Level, esperado)
		}
	}
}

func TestResultadoApontaParaARegraPeloIndice(t *testing.T) {
	lido, _ := montar(t, entradaComTodosOsMotivos())
	regras := lido.Runs[0].Tool.Driver.Rules

	for _, resultado := range lido.Runs[0].Results {
		if regras[resultado.RuleIndex].ID != resultado.RuleID {
			t.Errorf("resultado de %s aponta para a regra %s", resultado.RuleID, regras[resultado.RuleIndex].ID)
		}
	}
}

func TestCalibracaoEFalhaDoSandboxFicamNasPropriedadesDoRun(t *testing.T) {
	lido, _ := montar(t, entradaComTodosOsMotivos())
	run := lido.Runs[0]

	for _, resultado := range run.Results {
		if resultado.RuleID == "calibracao_historico" || resultado.RuleID == "falha_sandbox" {
			t.Errorf("%s não é achado no código e não pode virar resultado", resultado.RuleID)
		}
	}

	aprova := run.Properties.Aprova
	if aprova.NivelDeterministico != "alto" || aprova.Nivel != "medio" {
		t.Errorf("níveis do run = %q e %q, esperado alto e medio", aprova.NivelDeterministico, aprova.Nivel)
	}
	if aprova.Calibracao.Estado != "calibrada" || !aprova.Calibracao.ReduziuONivel {
		t.Errorf("calibração do run = %+v, esperado calibrada e com redução", aprova.Calibracao)
	}

	semResultado := map[string]string{}
	for _, fora := range aprova.MotivosSemResultado {
		semResultado[fora.ID] = fora.Razao
	}
	if semResultado["falha_sandbox"] != "sem_arquivo" {
		t.Errorf("falha_sandbox deveria estar fora por sem_arquivo, veio %q", semResultado["falha_sandbox"])
	}
	if _, listada := semResultado["calibracao_historico"]; listada {
		t.Error("calibracao_historico já está em calibracao e não entra em motivosSemResultado")
	}
}

func TestCalibracaoQueNaoReduziuFicaRegistradaSemReducao(t *testing.T) {
	lido, _ := montar(t, entradaDoPR1())
	calibracao := lido.Runs[0].Properties.Aprova.Calibracao

	if calibracao.Estado != "desligada" || calibracao.ReduziuONivel {
		t.Errorf("calibração do run = %+v, esperado desligada e sem redução", calibracao)
	}
}

func TestLocalizacaoDeCadaMotivo(t *testing.T) {
	resultados := resultadosDe(t, entradaComTodosOsMotivos())

	esperados := []struct {
		motivo  string
		arquivo string
		linha   int
	}{
		{"area_sensivel", "auth/sessao.go", 42},
		{"codigo_sem_teste", "auth/sessao.go", 42},
		{"configuracao_ci", ".github/workflows/ci.yml", 21},
		{"dependencia_nova", "go.mod", 8},
		{"muitos_arquivos", ".env", 1},
		{"segredo_versionado", ".env", 1},
	}

	if len(resultados) != len(esperados) {
		t.Fatalf("vieram %d resultados, esperado %d", len(resultados), len(esperados))
	}
	for _, esperado := range esperados {
		obtido := resultadoDoMotivo(t, resultados, esperado.motivo)
		if obtido.arquivo() != esperado.arquivo || obtido.linha() != esperado.linha {
			t.Errorf("%s em %s:%d, esperado %s:%d", esperado.motivo, obtido.arquivo(), obtido.linha(), esperado.arquivo, esperado.linha)
		}
	}
}

func TestDependenciaNovaSemLinhaQueACiteUsaAPrimeiraLinhaAdicionada(t *testing.T) {
	entrada := entradaComTodosOsMotivos()
	entrada.Gate.DependenciasNovas = []domain.DependenciaNova{{Ecossistema: "Go", Nome: "github.com/outra/coisa"}}

	obtido := resultadoDoMotivo(t, resultadosDe(t, entrada), "dependencia_nova")

	if obtido.linha() != 7 {
		t.Errorf("linha = %d, esperado 7, a primeira adicionada", obtido.linha())
	}
}

func TestEvidenciaDeManifestoIlegivelApontaParaOManifesto(t *testing.T) {
	entrada := entradaComTodosOsMotivos()
	entrada.Gate.Motivos = []domain.MotivoDeRisco{motivo("dependencia_nova", domain.RiscoAlto, "go.mod: não foi possível ler o diff")}

	obtido := resultadoDoMotivo(t, resultadosDe(t, entrada), "dependencia_nova")

	if obtido.arquivo() != "go.mod" {
		t.Errorf("arquivo = %q, esperado go.mod", obtido.arquivo())
	}
}

func TestPatchAusenteApontaParaALinha1(t *testing.T) {
	entrada := entradaComTodosOsMotivos()
	entrada.Gate.Motivos = []domain.MotivoDeRisco{motivo("area_sensivel", domain.RiscoAlto, "pagamento/cobranca.go")}

	obtido := resultadoDoMotivo(t, resultadosDe(t, entrada), "area_sensivel")

	if obtido.arquivo() != "pagamento/cobranca.go" || obtido.linha() != 1 {
		t.Errorf("resultado em %s:%d, esperado pagamento/cobranca.go:1", obtido.arquivo(), obtido.linha())
	}
}

func TestSemOsArquivosDoPRTodoResultadoVaiParaALinha1(t *testing.T) {
	entrada := entradaComTodosOsMotivos()
	entrada.Arquivos = nil

	resultados := resultadosDe(t, entrada)

	if len(resultados) == 0 {
		t.Fatal("sem a leitura do PR os resultados continuam saindo do rules_hit")
	}
	for _, resultado := range resultados {
		if resultado.linha() != 1 {
			t.Errorf("%s em %s:%d, esperado linha 1", resultado.RuleID, resultado.arquivo(), resultado.linha())
		}
	}
}

func TestArquivoApagadoNaoViraResultado(t *testing.T) {
	entrada := entradaComTodosOsMotivos()
	entrada.Gate.Motivos = []domain.MotivoDeRisco{motivo("area_sensivel", domain.RiscoAlto, "migrations/000001_antiga.up.sql")}

	lido, _ := montar(t, entrada)
	run := lido.Runs[0]

	if len(run.Results) != 0 {
		t.Fatalf("arquivo apagado não existe no commit analisado; vieram %d resultados", len(run.Results))
	}
	fora := run.Properties.Aprova.MotivosSemResultado
	if len(fora) != 1 || fora[0].ID != "area_sensivel" || fora[0].Razao != "arquivo_removido" {
		t.Errorf("motivosSemResultado = %+v, esperado area_sensivel por arquivo_removido", fora)
	}
}

func TestMotivoDoPRInteiroPulaArquivoApagadoNaAncora(t *testing.T) {
	entrada := entradaComTodosOsMotivos()
	entrada.Gate.Motivos = []domain.MotivoDeRisco{
		motivo("muitos_arquivos", domain.RiscoAlto, "auth/sessao.go", "migrations/000001_antiga.up.sql"),
	}
	entrada.Arquivos[2].Situacao = "removed"

	lido, _ := montar(t, entrada)

	if len(lido.Runs[0].Results) != 0 {
		t.Fatal("com todas as evidências apagadas não há onde ancorar o resultado")
	}
}

func TestCaminhoComEspacoViraURIValida(t *testing.T) {
	entrada := entradaComTodosOsMotivos()
	entrada.Arquivos = nil
	entrada.Gate.Motivos = []domain.MotivoDeRisco{motivo("area_sensivel", domain.RiscoAlto, "auth/minha sessão.go")}

	obtido := resultadoDoMotivo(t, resultadosDe(t, entrada), "area_sensivel")

	if obtido.arquivo() != "auth/minha%20sess%C3%A3o.go" {
		t.Errorf("uri = %q", obtido.arquivo())
	}
}

func TestFingerprintNaoMudaEntrePushes(t *testing.T) {
	primeiro := entradaComTodosOsMotivos()
	segundo := entradaComTodosOsMotivos()
	segundo.Recibo = Recibo{ID: "7c1d2e3f-0000-4000-8000-000000000001", Hash: bytes.Repeat([]byte{0x01}, 32)}
	segundo.Arquivos[2].Patch = "@@ -90,2 +90,3 @@\n contexto\n+outraLinha()\n"
	segundo.Gate.Motivos[6] = motivo("muitos_arquivos", domain.RiscoAlto, "auth/sessao.go", "go.mod", "zzz/novo.go")

	antes := resultadosDe(t, primeiro)
	depois := resultadosDe(t, segundo)

	vistos := map[string]bool{}
	for _, resultado := range antes {
		impressao := resultado.PartialFingerprints["primaryLocationLineHash"]
		if impressao == "" {
			t.Fatalf("%s sem primaryLocationLineHash, a única impressão que o GitHub usa", resultado.RuleID)
		}
		if vistos[impressao] {
			t.Errorf("impressão repetida entre resultados diferentes: %s", impressao)
		}
		vistos[impressao] = true

		par := resultadoDoMotivo(t, depois, resultado.RuleID)
		if par.PartialFingerprints["primaryLocationLineHash"] != impressao {
			t.Errorf("a impressão de %s mudou entre os pushes", resultado.RuleID)
		}
	}
	if resultadoDoMotivo(t, depois, "area_sensivel").linha() == resultadoDoMotivo(t, antes, "area_sensivel").linha() {
		t.Fatal("o teste precisa de linhas diferentes entre os pushes")
	}
}

func TestCadaResultadoCarregaOReciboDaAnalise(t *testing.T) {
	entrada := entradaComTodosOsMotivos()

	for _, resultado := range resultadosDe(t, entrada) {
		recibo := resultado.Properties.Recibo
		if recibo.ID != entrada.Recibo.ID || recibo.SHA256 != strings.Repeat("ab", 32) {
			t.Errorf("recibo de %s = %+v", resultado.RuleID, recibo)
		}
	}
}

func TestMensagemTrazADescricaoGravadaNoGate(t *testing.T) {
	obtido := resultadoDoMotivo(t, resultadosDe(t, entradaComTodosOsMotivos()), "configuracao_ci")

	if !strings.Contains(obtido.Message.Text, "descrição de configuracao_ci") {
		t.Errorf("mensagem = %q", obtido.Message.Text)
	}
}

func TestMesmaEntradaGeraOsMesmosBytes(t *testing.T) {
	primeiro, err := Montar(entradaComTodosOsMotivos())
	if err != nil {
		t.Fatal(err)
	}

	embaralhada := entradaComTodosOsMotivos()
	slices.Reverse(embaralhada.Arquivos)
	segundo, err := Montar(embaralhada)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(primeiro, segundo) {
		t.Fatal("a ordem dos arquivos lidos do PR mudou os bytes do SARIF")
	}

	if os.Getenv("ATUALIZAR_OURO") == "sim" {
		if err := os.WriteFile(filepath.FromSlash(caminhoDoOuro), primeiro, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ouro, err := os.ReadFile(filepath.FromSlash(caminhoDoOuro))
	if err != nil {
		t.Fatalf("ler o arquivo de referência: %v", err)
	}
	if !bytes.Equal(primeiro, ouro) {
		t.Fatalf("o SARIF mudou em relação a %s; se a mudança é intencional, rode com ATUALIZAR_OURO=sim", caminhoDoOuro)
	}
}

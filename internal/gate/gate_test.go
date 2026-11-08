package gate

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/platform"
	"github.com/Azulpasta/aPRova/internal/sandbox"
)

var (
	sandboxPassou = sandbox.ResultadoSandbox{Compilou: true, TestesPassaram: 11, SaidaBruta: "ok", TipoFalha: sandbox.FalhaNenhuma}
	testeFalhou   = sandbox.ResultadoSandbox{Compilou: true, TestesPassaram: 10, TestesFalharam: 1, TipoFalha: sandbox.FalhaTeste}
	naoCompilou   = sandbox.ResultadoSandbox{SaidaBruta: "undefined: Calcular", TipoFalha: sandbox.FalhaCompilacao}
)

func patchDe(t *testing.T, nome string) string {
	t.Helper()

	conteudo, err := os.ReadFile(filepath.Join("testdata", nome+".patch"))
	if err != nil {
		t.Fatalf("ler patch %s: %v", nome, err)
	}
	return string(conteudo)
}

func alterado(nome string) platform.ArquivoAlterado {
	return platform.ArquivoAlterado{Nome: nome, Situacao: "modified", Adicoes: 1, Patch: "@@ -1 +1 @@\n-antes\n+depois\n"}
}

func comSituacao(arquivo platform.ArquivoAlterado, situacao string) platform.ArquivoAlterado {
	arquivo.Situacao = situacao
	return arquivo
}

func manifesto(t *testing.T, nome, patch string) platform.ArquivoAlterado {
	t.Helper()
	return platform.ArquivoAlterado{Nome: nome, Situacao: "modified", Adicoes: 1, Patch: patchDe(t, patch)}
}

func avaliar(t *testing.T, resultadoSandbox sandbox.ResultadoSandbox, arquivos ...platform.ArquivoAlterado) Resultado {
	t.Helper()

	resultado, err := Novo(Limites{}).Avaliar(t.Context(), Entrada{Arquivos: arquivos, Sandbox: resultadoSandbox})
	if err != nil {
		t.Fatalf("avaliar: %v", err)
	}
	return resultado
}

func idsDe(resultado Resultado) []string {
	ids := []string{}
	for _, motivo := range resultado.Motivos {
		ids = append(ids, motivo.ID)
	}
	return ids
}

func motivoDe(t *testing.T, resultado Resultado, id string) domain.MotivoDeRisco {
	t.Helper()

	for _, motivo := range resultado.Motivos {
		if motivo.ID == id {
			return motivo
		}
	}
	t.Fatalf("o motivo %s não disparou; motivos: %v", id, idsDe(resultado))
	return domain.MotivoDeRisco{}
}

func exigirMotivos(t *testing.T, resultado Resultado, nivel domain.NivelRisco, ids ...string) {
	t.Helper()

	if !slices.Equal(idsDe(resultado), append([]string{}, ids...)) {
		t.Errorf("motivos = %v, esperava %v", idsDe(resultado), ids)
	}
	if resultado.Nivel != nivel {
		t.Errorf("nível = %s, esperava %s", resultado.Nivel, nivel)
	}
}

func prQueAcionaTudo(t *testing.T) []platform.ArquivoAlterado {
	t.Helper()

	return append(arquivosDeDocumentacao(15),
		alterado("internal/auth/sessao.go"),
		alterado(".github/workflows/ci.yml"),
		alterado("Dockerfile"),
		alterado("migrations/000020_nova.sql"),
		comSituacao(alterado(".env"), "added"),
		comSituacao(alterado("config/chave.pem"), "added"),
		manifesto(t, "go.mod", "gomod_dependencias_novas"),
		manifesto(t, "java/pom.xml", "pom_dependencia_nova"),
		manifesto(t, "python/requirements.txt", "requirements_dependencias_novas"),
		alterado("go/frete/frete.go"),
		alterado("python/frete/calculadora.py"),
	)
}

func TestMesmaEntradaProduzExatamenteOMesmoResultado(t *testing.T) {
	arquivos := prQueAcionaTudo(t)
	referencia := avaliar(t, testeFalhou, arquivos...)
	esperado, err := json.Marshal(referencia)
	if err != nil {
		t.Fatalf("serializar: %v", err)
	}

	sorteio := rand.New(rand.NewPCG(1, 2))
	for rodada := range 200 {
		embaralhados := slices.Clone(arquivos)
		sorteio.Shuffle(len(embaralhados), func(i, j int) { embaralhados[i], embaralhados[j] = embaralhados[j], embaralhados[i] })

		obtido, err := json.Marshal(avaliar(t, testeFalhou, embaralhados...))
		if err != nil {
			t.Fatalf("serializar: %v", err)
		}
		if string(obtido) != string(esperado) {
			t.Fatalf("rodada %d: a ordem de chegada dos arquivos mudou o resultado:\n%s\n%s", rodada, obtido, esperado)
		}
	}
}

func TestMotivosEEvidenciasSaemOrdenados(t *testing.T) {
	resultado := avaliar(t, testeFalhou, prQueAcionaTudo(t)...)

	ids := idsDe(resultado)
	if !slices.IsSorted(ids) || len(ids) != 7 {
		t.Errorf("motivos fora de ordem ou faltando: %v", ids)
	}
	for _, motivo := range resultado.Motivos {
		if !slices.IsSorted(motivo.Evidencias) {
			t.Errorf("evidências de %s fora de ordem: %v", motivo.ID, motivo.Evidencias)
		}
		if motivo.Descricao == "" {
			t.Errorf("%s sem descrição", motivo.ID)
		}
	}
	dependencias := resultado.Dependencias
	if !slices.IsSortedFunc(dependencias, func(a, b domain.DependenciaNova) int {
		return strings.Compare(a.Ecossistema+a.Nome, b.Ecossistema+b.Nome)
	}) {
		t.Errorf("dependências fora de ordem: %v", dependencias)
	}
}

func TestSemMotivoONivelEBaixoEAsListasVemVazias(t *testing.T) {
	resultado := avaliar(t, sandboxPassou, alterado("README.md"))

	exigirMotivos(t, resultado, domain.RiscoBaixo)
	corpo, err := json.Marshal(resultado)
	if err != nil {
		t.Fatalf("serializar: %v", err)
	}
	if !strings.Contains(string(corpo), `"motivos":[]`) || !strings.Contains(string(corpo), `"dependencias":[]`) {
		t.Errorf("listas vazias precisam serializar como [], não null: %s", corpo)
	}
}

func TestONivelFinalEOMaiorEntreOsMotivos(t *testing.T) {
	medio := avaliar(t, sandboxPassou, alterado("internal/auth/sessao.go"), alterado("internal/auth/sessao_test.go"))
	exigirMotivos(t, medio, domain.RiscoMedio, "area_sensivel")

	alto := avaliar(t, sandboxPassou, alterado("internal/auth/sessao.go"), alterado("Dockerfile"))
	exigirMotivos(t, alto, domain.RiscoAlto, "area_sensivel", "codigo_sem_teste", "configuracao_ci")
	if motivoDe(t, alto, "area_sensivel").Nivel != domain.RiscoMedio || motivoDe(t, alto, "configuracao_ci").Nivel != domain.RiscoAlto {
		t.Errorf("cada motivo guarda o próprio nível: %+v", alto.Motivos)
	}
}

func TestFalhaSandbox(t *testing.T) {
	for nome, resultado := range map[string]sandbox.ResultadoSandbox{"teste": testeFalhou, "compilacao": naoCompilou} {
		t.Run(nome, func(t *testing.T) {
			avaliado := avaliar(t, resultado, alterado("README.md"))
			exigirMotivos(t, avaliado, domain.RiscoAlto, "falha_sandbox")
			if len(motivoDe(t, avaliado, "falha_sandbox").Evidencias) == 0 {
				t.Error("a falha precisa de evidência")
			}
		})
	}

	exigirMotivos(t, avaliar(t, sandboxPassou, alterado("README.md")), domain.RiscoBaixo)
}

func TestTesteFalhoComDependenciaNovaTemOsDoisMotivos(t *testing.T) {
	resultado := avaliar(t, testeFalhou, manifesto(t, "go/go.mod", "gomod_dependencias_novas"))

	exigirMotivos(t, resultado, domain.RiscoAlto, "dependencia_nova", "falha_sandbox")
	if len(resultado.Dependencias) == 0 {
		t.Error("todas as regras rodam sempre: a dependência nova precisa sair mesmo com teste falho")
	}
}

func arquivosDeDocumentacao(quantidade int) []platform.ArquivoAlterado {
	arquivos := make([]platform.ArquivoAlterado, 0, quantidade)
	for indice := range quantidade {
		arquivos = append(arquivos, alterado(fmt.Sprintf("docs/pagina_%02d.md", indice)))
	}
	return arquivos
}

func TestMuitosArquivos(t *testing.T) {
	exigirMotivos(t, avaliar(t, sandboxPassou, arquivosDeDocumentacao(20)...), domain.RiscoBaixo)

	resultado := avaliar(t, sandboxPassou, arquivosDeDocumentacao(21)...)
	exigirMotivos(t, resultado, domain.RiscoAlto, "muitos_arquivos")
	if motivo := motivoDe(t, resultado, "muitos_arquivos"); len(motivo.Evidencias) != 21 || !strings.Contains(motivo.Descricao, "21") || !strings.Contains(motivo.Descricao, "20") {
		t.Errorf("o motivo precisa dizer a contagem, o limite e os arquivos: %+v", motivo)
	}
}

func TestLimiteDeArquivosConfiguravel(t *testing.T) {
	entrada := Entrada{Arquivos: arquivosDeDocumentacao(3), Sandbox: sandboxPassou}

	resultado, err := Novo(Limites{MaxArquivos: 2}).Avaliar(t.Context(), entrada)
	if err != nil {
		t.Fatalf("avaliar: %v", err)
	}
	exigirMotivos(t, resultado, domain.RiscoAlto, "muitos_arquivos")
}

func TestConfiguracaoCI(t *testing.T) {
	positivos := []platform.ArquivoAlterado{
		alterado(".github/workflows/ci.yml"),
		alterado(".github/actions/preparar/action.yml"),
		comSituacao(alterado(".github/workflows/ci.yml"), "removed"),
		alterado("Dockerfile"),
		alterado("go/Dockerfile"),
		alterado("imagens/Dockerfile.sandbox"),
		alterado("imagens/go.dockerfile"),
		alterado("docker-compose.yml"),
		alterado("compose.yaml"),
		alterado("Makefile"),
		alterado(".gitlab-ci.yml"),
		alterado("Jenkinsfile"),
		alterado(".circleci/config.yml"),
		alterado("azure-pipelines.yml"),
		alterado(".travis.yml"),
		alterado("java/build.gradle"),
		alterado("java/build.gradle.kts"),
		{Nome: "antigos/ci.yml", NomeAnterior: ".github/workflows/ci.yml", Situacao: "renamed"},
	}
	for _, arquivo := range positivos {
		t.Run("dispara "+arquivo.Nome+" "+arquivo.Situacao, func(t *testing.T) {
			resultado := avaliar(t, sandboxPassou, arquivo)
			if !slices.Contains(idsDe(resultado), "configuracao_ci") || resultado.Nivel != domain.RiscoAlto {
				t.Errorf("motivos = %v, nível %s", idsDe(resultado), resultado.Nivel)
			}
		})
	}

	for _, nome := range []string{"docs/Dockerfile.md", ".github/PULL_REQUEST_TEMPLATE.md", "docs/workflows/ci.md", "docs/makefile.md", "workflows/exemplos/instalacao.json"} {
		t.Run("não dispara "+nome, func(t *testing.T) {
			exigirMotivos(t, avaliar(t, sandboxPassou, alterado(nome)), domain.RiscoBaixo)
		})
	}
}

func TestDependenciaNovaNoGoMod(t *testing.T) {
	resultado := avaliar(t, sandboxPassou, manifesto(t, "go/go.mod", "gomod_dependencias_novas"))

	exigirMotivos(t, resultado, domain.RiscoAlto, "dependencia_nova")
	if evidencias := motivoDe(t, resultado, "dependencia_nova").Evidencias; !slices.Equal(evidencias, []string{"go/go.mod"}) {
		t.Errorf("evidências = %v", evidencias)
	}
	nova := domain.DependenciaNova{Ecossistema: "Go", Nome: "github.com/golang-migrate/migrate/v4", Versao: "v4.19.1"}
	if !slices.Contains(resultado.Dependencias, nova) {
		t.Errorf("faltou %+v em %+v", nova, resultado.Dependencias)
	}
	for _, dependencia := range resultado.Dependencias {
		if dependencia.Nome == "golang.org/x/sync" {
			t.Errorf("x/sync só mudou de versão e não é dependência nova: %+v", dependencia)
		}
	}
}

func TestMudancaDeVersaoNaoEDependenciaNova(t *testing.T) {
	casos := map[string][2]string{
		"go.mod com versões alteradas":         {"go.mod", "gomod_versoes_alteradas"},
		"indireta que vira direta no go.mod":   {"go.mod", "gomod_indireta_vira_direta"},
		"requirements.txt com versão alterada": {"python/requirements.txt", "requirements_versao_alterada"},
		"pyproject.toml com versões alteradas": {"python/pyproject.toml", "pyproject_versoes_alteradas"},
		"pyproject.toml sem dependência":       {"python/pyproject.toml", "pyproject_sem_dependencia"},
		"pom.xml com versão em propriedade":    {"java/pom.xml", "pom_versoes_alteradas"},
		"pom.xml com versão dentro do bloco":   {"java/pom.xml", "pom_versao_na_dependencia"},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			resultado := avaliar(t, sandboxPassou, manifesto(t, caso[0], caso[1]))

			exigirMotivos(t, resultado, domain.RiscoBaixo)
			if len(resultado.Dependencias) != 0 {
				t.Errorf("dependências = %+v", resultado.Dependencias)
			}
		})
	}
}

func TestDependenciaNovaEmCadaManifesto(t *testing.T) {
	casos := map[string]struct {
		arquivo  string
		patch    string
		esperado []domain.DependenciaNova
	}{
		"requirements.txt": {"python/requirements.txt", "requirements_dependencias_novas", []domain.DependenciaNova{
			{Ecossistema: "PyPI", Nome: "reqeusts-toolbelt", Versao: ">=1.0.0"},
			{Ecossistema: "PyPI", Nome: "requests", Versao: "2.32.3"},
		}},
		"pyproject.toml": {"python/pyproject.toml", "pyproject_dependencias_novas", []domain.DependenciaNova{
			{Ecossistema: "PyPI", Nome: "httpx", Versao: ">=0.27"},
			{Ecossistema: "PyPI", Nome: "pydantic", Versao: "2.9.2"},
		}},
		"pyproject.toml do poetry": {"python/pyproject.toml", "pyproject_poetry_dependencia_nova", []domain.DependenciaNova{
			{Ecossistema: "PyPI", Nome: "requests", Versao: "^2.32"},
		}},
		"pom.xml": {"java/pom.xml", "pom_dependencia_nova", []domain.DependenciaNova{
			{Ecossistema: "Maven", Nome: "org.javamoney:moneta", Versao: "1.4.4"},
		}},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			resultado := avaliar(t, sandboxPassou, manifesto(t, caso.arquivo, caso.patch))

			exigirMotivos(t, resultado, domain.RiscoAlto, "dependencia_nova")
			if !slices.Equal(resultado.Dependencias, caso.esperado) {
				t.Errorf("dependências = %+v, esperava %+v", resultado.Dependencias, caso.esperado)
			}
		})
	}
}

func TestLinhaDeOpcaoNoRequirementsDisparaPorFalhaSegura(t *testing.T) {
	arquivo := platform.ArquivoAlterado{
		Nome: "requirements.txt", Situacao: "modified", Adicoes: 1,
		Patch: "@@ -1 +1,2 @@\n pytest>=7.0\n+--extra-index-url https://pacotes.example.com/simple\n",
	}

	resultado := avaliar(t, sandboxPassou, arquivo)

	exigirMotivos(t, resultado, domain.RiscoAlto, "dependencia_nova")
	if evidencias := motivoDe(t, resultado, "dependencia_nova").Evidencias; !strings.Contains(evidencias[0], "não foi possível interpretar") {
		t.Errorf("a evidência precisa dizer o que não foi lido: %v", evidencias)
	}
}

func TestPatchAusenteEmManifestoDisparaPorFalhaSegura(t *testing.T) {
	for _, nome := range []string{"go/go.mod", "requirements.txt", "pyproject.toml", "pom.xml"} {
		t.Run(nome, func(t *testing.T) {
			resultado := avaliar(t, sandboxPassou, platform.ArquivoAlterado{Nome: nome, Situacao: "modified", Adicoes: 9000, Remocoes: 3})

			exigirMotivos(t, resultado, domain.RiscoAlto, "dependencia_nova")
			if evidencias := motivoDe(t, resultado, "dependencia_nova").Evidencias; !slices.Equal(evidencias, []string{nome + ": não foi possível ler o diff"}) {
				t.Errorf("evidências = %v", evidencias)
			}
			if len(resultado.Dependencias) != 0 {
				t.Errorf("sem o diff não há como listar dependências: %+v", resultado.Dependencias)
			}
		})
	}
}

func TestManifestoSemPatchQueNaoEscondeMudanca(t *testing.T) {
	casos := map[string]platform.ArquivoAlterado{
		"removido":            {Nome: "go.mod", Situacao: "removed", Remocoes: 40},
		"renomeado sem mudar": {Nome: "novo/go.mod", NomeAnterior: "go.mod", Situacao: "renamed"},
	}

	for nome, arquivo := range casos {
		t.Run(nome, func(t *testing.T) {
			if ids := idsDe(avaliar(t, sandboxPassou, arquivo)); slices.Contains(ids, "dependencia_nova") {
				t.Errorf("motivos = %v", ids)
			}
		})
	}
}

func TestManifestoComNomeParecidoNaoEManifesto(t *testing.T) {
	exigirMotivos(t, avaliar(t, sandboxPassou, platform.ArquivoAlterado{Nome: "docs/go.mod.md", Situacao: "modified", Adicoes: 5}), domain.RiscoBaixo)
}

func TestSegredoVersionado(t *testing.T) {
	for _, nome := range []string{".env", "config/.env", ".env.production", "api/.env.local", "certs/servidor.pem", "chaves/privada.key", "id_rsa", ".ssh/id_ed25519", "credentials.json", "secrets.yaml", "app.p12"} {
		t.Run("dispara "+nome, func(t *testing.T) {
			resultado := avaliar(t, sandboxPassou, comSituacao(alterado(nome), "added"))

			exigirMotivos(t, resultado, domain.RiscoAlto, "segredo_versionado")
			if evidencias := motivoDe(t, resultado, "segredo_versionado").Evidencias; !slices.Equal(evidencias, []string{nome}) {
				t.Errorf("evidências = %v", evidencias)
			}
		})
	}

	for _, nome := range []string{".env.example", "config/.env.sample", ".env.template", ".env.dist", "docs/env.md", "chave_publica.md", "environment.md"} {
		t.Run("não dispara "+nome, func(t *testing.T) {
			exigirMotivos(t, avaliar(t, sandboxPassou, comSituacao(alterado(nome), "added")), domain.RiscoBaixo)
		})
	}

	exigirMotivos(t, avaliar(t, sandboxPassou, comSituacao(alterado(".env"), "removed")), domain.RiscoBaixo)
}

func TestAreaSensivel(t *testing.T) {
	for _, nome := range []string{
		"auth/politica.md",
		"internal/auth/sessao_test.go",
		"security/regras.md",
		"pkg/payments/taxas.md",
		"migrations/000015_rules_hit.up.sql",
		"db/schema.md",
		"database/seed.md",
		"scripts/limpar_banco.sql",
	} {
		t.Run("dispara "+nome, func(t *testing.T) {
			resultado := avaliar(t, sandboxPassou, alterado(nome))

			exigirMotivos(t, resultado, domain.RiscoMedio, "area_sensivel")
			if evidencias := motivoDe(t, resultado, "area_sensivel").Evidencias; !slices.Equal(evidencias, []string{nome}) {
				t.Errorf("evidências = %v", evidencias)
			}
		})
	}

	for _, nome := range []string{"internal/frete/auth_test.go", "docs/authentication.md", "docs/payments.md", "pkg/security_utils_test.go", "docs/migrations.md", "authors/lista.md", "myauth/notas.md", "cloudsecurity/notas.md"} {
		t.Run("não dispara "+nome, func(t *testing.T) {
			exigirMotivos(t, avaliar(t, sandboxPassou, alterado(nome)), domain.RiscoBaixo)
		})
	}

	renomeado := platform.ArquivoAlterado{Nome: "publico/politica.md", NomeAnterior: "auth/politica.md", Situacao: "renamed"}
	exigirMotivos(t, avaliar(t, sandboxPassou, renomeado), domain.RiscoMedio, "area_sensivel")
}

func TestListaDeCaminhosSensiveisConfiguravel(t *testing.T) {
	gate := Novo(Limites{CaminhosSensiveis: []string{"financeiro/", "*.tf"}})

	for nome, esperado := range map[string]domain.NivelRisco{
		"src/financeiro/notas.md": domain.RiscoMedio,
		"infra/rede.tf":           domain.RiscoMedio,
		"auth/politica.md":        domain.RiscoBaixo,
	} {
		resultado, err := gate.Avaliar(t.Context(), Entrada{Arquivos: []platform.ArquivoAlterado{alterado(nome)}, Sandbox: sandboxPassou})
		if err != nil {
			t.Fatalf("avaliar: %v", err)
		}
		if resultado.Nivel != esperado {
			t.Errorf("%s: nível %s, esperava %s", nome, resultado.Nivel, esperado)
		}
	}
}

func TestCodigoSemTeste(t *testing.T) {
	semTeste := map[string][]platform.ArquivoAlterado{
		"go":                       {alterado("go/frete/frete.go")},
		"python":                   {alterado("python/frete/calculadora.py")},
		"java":                     {alterado("java/src/main/java/com/aprova/frete/CalculadoraFrete.java")},
		"teste removido não conta": {alterado("go/frete/frete.go"), comSituacao(alterado("go/frete/frete_test.go"), "removed")},
		"código removido":          {comSituacao(alterado("go/frete/velho.go"), "removed")},
	}
	for nome, arquivos := range semTeste {
		t.Run("dispara "+nome, func(t *testing.T) {
			resultado := avaliar(t, sandboxPassou, arquivos...)

			exigirMotivos(t, resultado, domain.RiscoMedio, "codigo_sem_teste")
			if evidencias := motivoDe(t, resultado, "codigo_sem_teste").Evidencias; !slices.Contains(evidencias, arquivos[0].Nome) {
				t.Errorf("evidências = %v", evidencias)
			}
		})
	}

	comTeste := map[string][]platform.ArquivoAlterado{
		"go":               {alterado("go/frete/frete.go"), alterado("go/frete/frete_test.go")},
		"python test_":     {alterado("python/frete/calculadora.py"), alterado("python/tests/test_frete.py")},
		"python _test":     {alterado("python/frete/calculadora.py"), alterado("python/frete/calculadora_test.py")},
		"python em tests/": {alterado("python/frete/calculadora.py"), alterado("python/tests/auxiliares.py")},
		"java Test":        {alterado("java/src/main/java/CalculadoraFrete.java"), alterado("java/src/test/java/CalculadoraFreteTest.java")},
		"java Tests":       {alterado("java/src/main/java/CalculadoraFrete.java"), alterado("java/outro/CalculadoraFreteTests.java")},
		"java IT":          {alterado("java/src/main/java/CalculadoraFrete.java"), alterado("java/outro/CalculadoraFreteIT.java")},
		"java em src/test": {alterado("java/src/main/java/CalculadoraFrete.java"), alterado("java/src/test/java/Auxiliar.java")},
		"teste novo basta": {alterado("go/frete/frete.go"), comSituacao(alterado("go/frete/frete_test.go"), "added")},
		"só teste":         {alterado("go/frete/frete_test.go")},
		"só documentação":  {alterado("README.md")},
		"fixture de teste": {alterado("tests/integration/testdata/go/ok/frete.go")},
	}
	for nome, arquivos := range comTeste {
		t.Run("não dispara "+nome, func(t *testing.T) {
			exigirMotivos(t, avaliar(t, sandboxPassou, arquivos...), domain.RiscoBaixo)
		})
	}
}

func TestPRDoPesoZeroAlteraSoTesteEFalhaNoSandbox(t *testing.T) {
	resultado := avaliar(t, testeFalhou, manifesto(t, "go/frete/frete_test.go", "teste_de_peso_zero"))

	exigirMotivos(t, resultado, domain.RiscoAlto, "falha_sandbox")
}

func TestFalhaDeInfraestruturaDevolveErroEmVezDeNivel(t *testing.T) {
	for nome, resultado := range map[string]sandbox.ResultadoSandbox{
		"infraestrutura": {SaidaBruta: "job não iniciou", TipoFalha: sandbox.FalhaInfraestrutura},
		"tipo vazio":     {Compilou: true},
		"tipo estranho":  {Compilou: true, TipoFalha: sandbox.TipoFalha("desconhecida")},
	} {
		t.Run(nome, func(t *testing.T) {
			_, err := Novo(Limites{}).Avaliar(t.Context(), Entrada{Arquivos: []platform.ArquivoAlterado{alterado("README.md")}, Sandbox: resultado})

			if !errors.Is(err, ErrResultadoSemVeredito) {
				t.Errorf("esperava ErrResultadoSemVeredito, obtive %v", err)
			}
		})
	}
}

func TestDependenciasRepetidasEmManifestosDiferentesSaemUmaVez(t *testing.T) {
	resultado := avaliar(t, sandboxPassou,
		manifesto(t, "a/go.mod", "gomod_indireta_vira_direta"),
		platform.ArquivoAlterado{Nome: "b/requirements.txt", Situacao: "modified", Adicoes: 1, Patch: "@@ -0,0 +1 @@\n+requests==2.32.3\n"},
		platform.ArquivoAlterado{Nome: "c/requirements.txt", Situacao: "modified", Adicoes: 1, Patch: "@@ -0,0 +1 @@\n+Requests==2.32.3\n"},
	)

	if len(resultado.Dependencias) != 1 {
		t.Errorf("a mesma dependência em dois manifestos sai uma vez: %+v", resultado.Dependencias)
	}
	if evidencias := motivoDe(t, resultado, "dependencia_nova").Evidencias; !slices.Equal(evidencias, []string{"b/requirements.txt", "c/requirements.txt"}) {
		t.Errorf("evidências = %v", evidencias)
	}
}

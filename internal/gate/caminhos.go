package gate

import (
	"path"
	"slices"
	"strings"

	"github.com/PI-GRUPO/aPRova/internal/platform"
)

var (
	diretoriosDeCI = []string{".github/workflows/", ".github/actions/", ".circleci/"}
	arquivosDeCI   = []string{
		"Makefile", "makefile", "GNUmakefile", "Jenkinsfile", ".gitlab-ci.yml", "azure-pipelines.yml", ".travis.yml",
		"Dockerfile", "compose.yml", "compose.yaml", "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts",
	}
	padroesDeCI          = []string{"Dockerfile.*", "*.dockerfile", "docker-compose*.yml", "docker-compose*.yaml"}
	extensoesDeDocumento = []string{".md", ".txt"}

	padroesDeSegredo     = []string{".env", ".env.*", "*.pem", "*.key", "*.p12", "*.pfx", "id_rsa", "id_dsa", "id_ecdsa", "id_ed25519", "credentials.json", "secrets.yaml", "secrets.yml", "secrets.json", ".netrc"}
	sufixosDeModeloDeEnv = []string{".example", ".sample", ".template", ".dist"}

	extensoesDeCodigo = []string{".go", ".py", ".java"}
)

func caminhosDe(arquivo platform.ArquivoAlterado) []string {
	if arquivo.NomeAnterior == "" {
		return []string{arquivo.Nome}
	}
	return []string{arquivo.Nome, arquivo.NomeAnterior}
}

func casaComAlgum(padroes []string, nome string) bool {
	return slices.ContainsFunc(padroes, func(padrao string) bool {
		casou, err := path.Match(padrao, nome)
		return err == nil && casou
	})
}

func dentroDe(diretorio, caminho string) bool {
	return strings.Contains("/"+caminho, "/"+diretorio)
}

func tocaConfiguracaoCI(arquivo platform.ArquivoAlterado) bool {
	return slices.ContainsFunc(caminhosDe(arquivo), ehConfiguracaoCI)
}

func ehConfiguracaoCI(caminho string) bool {
	nome := path.Base(caminho)
	if slices.Contains(extensoesDeDocumento, path.Ext(nome)) {
		return false
	}
	return slices.Contains(arquivosDeCI, nome) || casaComAlgum(padroesDeCI, nome) ||
		slices.ContainsFunc(diretoriosDeCI, func(diretorio string) bool { return strings.HasPrefix(caminho, diretorio) })
}

func versionaSegredo(arquivo platform.ArquivoAlterado) bool {
	if arquivo.Situacao == situacaoRemovido {
		return false
	}
	nome := path.Base(arquivo.Nome)
	return casaComAlgum(padroesDeSegredo, nome) && !slices.Contains(sufixosDeModeloDeEnv, path.Ext(nome))
}

func (g *Gate) tocaCaminhoSensivel(arquivo platform.ArquivoAlterado) bool {
	return slices.ContainsFunc(caminhosDe(arquivo), func(caminho string) bool {
		return slices.ContainsFunc(g.limites.CaminhosSensiveis, func(entrada string) bool {
			if strings.HasSuffix(entrada, "/") {
				return dentroDe(entrada, caminho)
			}
			return casaComAlgum([]string{entrada}, path.Base(caminho))
		})
	})
}

func ehCodigoDeProducao(caminho string) bool {
	return slices.Contains(extensoesDeCodigo, path.Ext(caminho)) && !ehTeste(caminho) && !dentroDe("testdata/", caminho)
}

func ehTeste(caminho string) bool {
	nome := path.Base(caminho)
	switch path.Ext(nome) {
	case ".go":
		return strings.HasSuffix(nome, "_test.go")
	case ".py":
		return ehTestePython(nome, caminho)
	case ".java":
		return ehTesteJava(nome, caminho)
	default:
		return false
	}
}

func ehTestePython(nome, caminho string) bool {
	return strings.HasPrefix(nome, "test_") || strings.HasSuffix(nome, "_test.py") || nome == "conftest.py" ||
		dentroDe("tests/", caminho) || dentroDe("test/", caminho)
}

func ehTesteJava(nome, caminho string) bool {
	classe := strings.TrimSuffix(nome, ".java")
	return strings.HasSuffix(classe, "Test") || strings.HasSuffix(classe, "Tests") || strings.HasSuffix(classe, "IT") ||
		strings.HasPrefix(classe, "Test") || dentroDe("src/test/", caminho)
}

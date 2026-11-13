package python

import (
	"slices"
	"strings"
	"testing"
)

func TestRequisitosAceitosViramALista(t *testing.T) {
	conteudo := strings.Join([]string{
		"# dependências do projeto",
		"",
		"requests==2.32.0",
		"six>=1.16   # comentário no fim",
		"Flask[async]~=3.0 ; python_version >= \"3.10\"",
		"numpy==2.1.0 \\",
		"    --hash=sha256:aaaa",
		"pytest-mock",
	}, "\n")

	especificacoes, recusas := interpretarArquivoDeRequisitos("requirements.txt", conteudo)

	if len(recusas) != 0 {
		t.Fatalf("nada aqui é recusado: %v", recusas)
	}
	esperadas := []string{
		"requests==2.32.0", "six>=1.16", `Flask[async]~=3.0 ; python_version >= "3.10"`,
		"numpy==2.1.0 --hash=sha256:aaaa", "pytest-mock",
	}
	if !slices.Equal(especificacoes, esperadas) {
		t.Errorf("especificações:\n%q\nesperava\n%q", especificacoes, esperadas)
	}
}

func TestRequisitosQueExigemConstruirOuBuscarForaDoIndiceSaoRecusados(t *testing.T) {
	casos := map[string]string{
		"-e .":                                 "editável",
		"--editable=git+https://x.org/y.git":   "editável",
		"git+https://github.com/x/y.git":       "URL, VCS ou caminho",
		"https://x.org/pacote-1.0.whl":         "URL, VCS ou caminho",
		"./vendor/pacote":                      "URL, VCS ou caminho",
		"../pacote":                            "URL, VCS ou caminho",
		"/opt/pacote":                          "URL, VCS ou caminho",
		"pacote @ https://x.org/pacote.tar.gz": "URL, VCS ou caminho",
		"-r outro.txt":                         "aninhado",
		"--requirement=outro.txt":              "aninhado",
		"-c restricoes.txt":                    "aninhado",
		"-i https://indice.exemplo/simple":     "índice",
		"--index-url https://indice.exemplo":   "índice",
		"--extra-index-url https://x.exemplo":  "índice",
		"-f https://x.exemplo/wheels":          "índice",
		"--find-links=./wheels":                "índice",
		"--trusted-host indice.exemplo":        "índice",
		"--pre":                                "opção do pip",
		"isto não é um requisito!":             "não é um requisito",
	}
	for linha, motivo := range casos {
		t.Run(linha, func(t *testing.T) {
			especificacoes, recusas := interpretarArquivoDeRequisitos("requirements-dev.txt", "six==1.17.0\n"+linha+"\n")

			if len(recusas) != 1 || !strings.Contains(recusas[0], motivo) || !strings.HasPrefix(recusas[0], "requirements-dev.txt:2:") {
				t.Fatalf("recusa com arquivo, linha e motivo %q: %v", motivo, recusas)
			}
			if !slices.Equal(especificacoes, []string{"six==1.17.0"}) {
				t.Errorf("a linha recusada não entra: %v", especificacoes)
			}
		})
	}
}

func TestArquivosDeRequisitosDaRaizEmOrdem(t *testing.T) {
	nomes := []string{"requirements_test.txt", "requirements.txt", "setup.py", "requirements-dev.txt", "requirements.in", "dev-requirements.txt", "requirements"}

	if lidos := arquivosDeRequisitos(nomes); !slices.Equal(lidos, []string{"requirements-dev.txt", "requirements.txt", "requirements_test.txt"}) {
		t.Errorf("arquivos de requisitos: %v", lidos)
	}
}

func TestPyprojectContribuiComAsDependenciasEOsGruposDeTeste(t *testing.T) {
	dados := dadosDoPyproject{
		Dependencias: []string{"requests>=2", "six"},
		Opcionais: map[string][]string{
			"test": {"pytest-mock"}, "tests": {"freezegun"}, "testing": {"hypothesis"},
			"dev": {"ruff"}, "docs": {"sphinx"},
		},
	}

	especificacoes, recusas := requisitosDoPyproject(dados)

	if len(recusas) != 0 {
		t.Fatalf("recusas: %v", recusas)
	}
	if !slices.Equal(especificacoes, []string{"requests>=2", "six", "ruff", "pytest-mock", "freezegun", "hypothesis"}) {
		t.Errorf("dependências, depois os grupos dev, test, tests e testing; docs fica de fora: %v", especificacoes)
	}
}

func TestPyprojectComDependenciasDinamicasOuPorURLERecusado(t *testing.T) {
	_, dinamicas := requisitosDoPyproject(dadosDoPyproject{Dinamicas: true})
	if len(dinamicas) != 1 || !strings.Contains(dinamicas[0], "dinâmicas") {
		t.Errorf("dependências dinâmicas: %v", dinamicas)
	}

	_, porURL := requisitosDoPyproject(dadosDoPyproject{Dependencias: []string{"pacote @ git+https://x.org/y.git"}})
	if len(porURL) != 1 || !strings.Contains(porURL[0], "pyproject.toml") || !strings.Contains(porURL[0], "URL, VCS ou caminho") {
		t.Errorf("dependência por URL no pyproject: %v", porURL)
	}
}

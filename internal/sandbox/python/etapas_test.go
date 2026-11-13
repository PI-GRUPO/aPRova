package python

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/sandbox"
	"github.com/PI-GRUPO/aPRova/internal/sandbox/processo"
)

type rodadorDasEtapas struct {
	responder func(comando processo.Comando) processo.Execucao
	rodados   []processo.Comando
}

func (r *rodadorDasEtapas) Rodar(_ context.Context, comando processo.Comando) processo.Execucao {
	r.rodados = append(r.rodados, comando)
	if r.responder == nil {
		return processo.Execucao{}
	}
	return r.responder(comando)
}

func (r *rodadorDasEtapas) linhas() []string {
	var linhas []string
	for _, comando := range r.rodados {
		linhas = append(linhas, comando.Binario+" "+strings.Join(comando.Argumentos, " "))
	}
	return linhas
}

func projetoPython(t *testing.T, arquivos map[string]string) string {
	t.Helper()
	diretorio := t.TempDir()
	arquivos["frete.py"] = "TAXA = 1\n"
	for nome, conteudo := range arquivos {
		if err := os.WriteFile(filepath.Join(diretorio, nome), []byte(conteudo), 0o600); err != nil {
			t.Fatalf("escrever %s: %v", nome, err)
		}
	}
	return diretorio
}

func linguagemDasEtapas(t *testing.T, rodador *rodadorDasEtapas) *Linguagem {
	t.Helper()
	linguagem := Nova(Opcoes{Rodador: rodador, DiretorioDeCache: "/tmp/aprova-cache", Agora: func() time.Time { return agoraPythonDeTeste }})
	linguagem.lerRelatorio = func(string) ([]byte, error) { return []byte(relatorioSoPassando), nil }
	return linguagem
}

func ehDownload(comando processo.Comando) bool {
	return slices.Contains(comando.Argumentos, "download")
}

func lerRequisitosGravados(t *testing.T, cache string) string {
	t.Helper()
	conteudo, err := os.ReadFile(filepath.Join(cache, "requisitos.txt"))
	if err != nil {
		t.Fatalf("ler a lista gravada no cache: %v", err)
	}
	return string(conteudo)
}

func TestPreparoBaixaSoWheelsDosRequirementsSemConstruirNada(t *testing.T) {
	rodador := &rodadorDasEtapas{}
	projeto := projetoPython(t, map[string]string{
		"requirements.txt":     "six==1.17.0\n",
		"requirements-dev.txt": "pytest-mock\n",
		"setup.py":             "open('/tmp/marcador', 'w')\n",
	})
	cache := t.TempDir()

	falha := linguagemDasEtapas(t, rodador).Preparar(t.Context(), projeto, cache)

	if falha != nil {
		t.Fatalf("preparo: %+v", falha)
	}
	if lista := lerRequisitosGravados(t, cache); lista != "pytest-mock\nsix==1.17.0\n" {
		t.Errorf("lista no cache = %q", lista)
	}
	esperado := "python -m pip download --only-binary=:all: --dest " + filepath.Join(cache, "wheels") + " -r " + filepath.Join(cache, "requisitos.txt")
	if linhas := rodador.linhas(); !slices.Equal(linhas, []string{esperado}) {
		t.Fatalf("o preparo só baixa wheels:\n%q\nesperava\n%q", linhas, esperado)
	}
	for _, argumento := range rodador.rodados[0].Argumentos {
		if argumento == "." || argumento == "install" || strings.Contains(argumento, "setup.py") || argumento == "-e" {
			t.Errorf("o preparo nunca constrói nem instala o projeto: %v", rodador.rodados[0].Argumentos)
		}
	}
}

func TestPreparoLeOPyprojectComoDadoEmModoIsolado(t *testing.T) {
	rodador := &rodadorDasEtapas{responder: func(comando processo.Comando) processo.Execucao {
		if slices.Contains(comando.Argumentos, "-I") {
			return processo.Execucao{Saida: []byte(`{"dependencias":["requests>=2"],"opcionais":{"test":["pytest-mock"],"docs":["sphinx"]},"dinamicas":false}`)}
		}
		return processo.Execucao{}
	}}
	projeto := projetoPython(t, map[string]string{"pyproject.toml": "[project]\nname = \"frete\"\n"})
	cache := t.TempDir()

	if falha := linguagemDasEtapas(t, rodador).Preparar(t.Context(), projeto, cache); falha != nil {
		t.Fatalf("preparo: %+v", falha)
	}

	leitura := rodador.rodados[0]
	if leitura.Binario != "python" || leitura.Argumentos[0] != "-I" || leitura.Argumentos[1] != "-c" ||
		leitura.Argumentos[len(leitura.Argumentos)-1] != filepath.Join(projeto, "pyproject.toml") {
		t.Errorf("o pyproject é lido pelo python isolado, com o caminho como argumento: %v", leitura.Argumentos)
	}
	if !strings.Contains(strings.Join(leitura.Argumentos, " "), "tomllib") {
		t.Errorf("a leitura usa tomllib: %v", leitura.Argumentos)
	}
	if lista := lerRequisitosGravados(t, cache); lista != "requests>=2\npytest-mock\n" {
		t.Errorf("lista no cache = %q", lista)
	}
}

func TestPreparoComRequisitoRecusadoEhCompilacaoSemBaixarNada(t *testing.T) {
	rodador := &rodadorDasEtapas{}
	projeto := projetoPython(t, map[string]string{"requirements.txt": "six==1.17.0\n-e .\n"})

	falha := linguagemDasEtapas(t, rodador).Preparar(t.Context(), projeto, t.TempDir())

	if falha == nil || falha.TipoFalha != sandbox.FalhaCompilacao || !strings.Contains(falha.SaidaBruta, "requirements.txt:2") || !strings.Contains(falha.SaidaBruta, "editável") {
		t.Fatalf("recusa com evidência: %+v", falha)
	}
	if len(rodador.rodados) != 0 {
		t.Errorf("nada é baixado: %v", rodador.linhas())
	}
}

func TestPreparoComPyprojectInvalidoEhCompilacao(t *testing.T) {
	rodador := &rodadorDasEtapas{responder: func(processo.Comando) processo.Execucao {
		return processo.Execucao{CodigoSaida: 1, Saida: []byte("tomllib.TOMLDecodeError: Expected '=' after a key (at line 2, column 6)\n")}
	}}
	projeto := projetoPython(t, map[string]string{"pyproject.toml": "[project]\nname\n"})

	falha := linguagemDasEtapas(t, rodador).Preparar(t.Context(), projeto, t.TempDir())

	if falha == nil || falha.TipoFalha != sandbox.FalhaCompilacao || !strings.Contains(falha.SaidaBruta, "pyproject.toml") || !strings.Contains(falha.SaidaBruta, "TOMLDecodeError") {
		t.Errorf("pyproject quebrado é do PR: %+v", falha)
	}
}

func TestPreparoSemDependenciasNaoBaixaNada(t *testing.T) {
	rodador := &rodadorDasEtapas{}
	cache := t.TempDir()

	falha := linguagemDasEtapas(t, rodador).Preparar(t.Context(), projetoPython(t, map[string]string{}), cache)

	if falha != nil || len(rodador.rodados) != 0 {
		t.Errorf("sem dependências: %+v, %v", falha, rodador.linhas())
	}
	if _, err := os.Stat(filepath.Join(cache, "requisitos.txt")); !os.IsNotExist(err) {
		t.Errorf("sem dependências não há lista: %v", err)
	}
}

func TestPreparoSemArquivoPythonEhInfraestrutura(t *testing.T) {
	falha := linguagemDasEtapas(t, &rodadorDasEtapas{}).Preparar(t.Context(), t.TempDir(), t.TempDir())

	if falha == nil || falha.TipoFalha != sandbox.FalhaInfraestrutura {
		t.Errorf("sem Python: %+v", falha)
	}
}

func TestPreparoClassificaAFalhaDoDownload(t *testing.T) {
	casos := []struct {
		nome     string
		execucao processo.Execucao
		tipo     sandbox.TipoFalha
		trecho   string
	}{
		{"só sdist ou inexistente", processo.Execucao{CodigoSaida: 1, Saida: []byte(
			"ERROR: Could not find a version that satisfies the requirement pycrypto==2.6.1 (from versions: none)\nERROR: No matching distribution found for pycrypto==2.6.1\n")},
			sandbox.FalhaCompilacao, "não existe no índice ou só existe como sdist"},
		{"versões incompatíveis", processo.Execucao{CodigoSaida: 1, Saida: []byte(
			"ERROR: Cannot install a==1 and b==2 because these package versions have conflicting dependencies.\nERROR: ResolutionImpossible\n")},
			sandbox.FalhaCompilacao, "incompatíveis"},
		{"hash divergente", processo.Execucao{CodigoSaida: 1, Saida: []byte(
			"ERROR: THESE PACKAGES DO NOT MATCH THE HASHES FROM THE REQUIREMENTS FILE.\n")},
			sandbox.FalhaCompilacao, "hash"},
		{"sem DNS", processo.Execucao{CodigoSaida: 1, Saida: []byte(
			"WARNING: Retrying (Retry(total=4)) after connection broken by 'NewConnectionError(... [Errno -3] Temporary failure in name resolution')': /simple/six/\n" +
				"ERROR: Could not find a version that satisfies the requirement six==1.17.0 (from versions: none)\nERROR: No matching distribution found for six==1.17.0\n")},
			sandbox.FalhaInfraestrutura, "rede"},
		{"timeout", processo.Execucao{CodigoSaida: 1, Saida: []byte(
			"pip._vendor.urllib3.exceptions.ReadTimeoutError: HTTPSConnectionPool(host='files.pythonhosted.org', port=443): Read timed out.\n")},
			sandbox.FalhaInfraestrutura, "rede"},
		{"prazo do executor", processo.Execucao{PrazoEsgotado: true}, sandbox.FalhaInfraestrutura, "excedeu o prazo"},
		{"falha sem explicação", processo.Execucao{CodigoSaida: 2, Saida: []byte("algo estranho\n")}, sandbox.FalhaInfraestrutura, "pip download falhou"},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			rodador := &rodadorDasEtapas{responder: func(comando processo.Comando) processo.Execucao {
				if ehDownload(comando) {
					return caso.execucao
				}
				return processo.Execucao{}
			}}
			projeto := projetoPython(t, map[string]string{"requirements.txt": "pycrypto==2.6.1\n"})

			falha := linguagemDasEtapas(t, rodador).Preparar(t.Context(), projeto, t.TempDir())

			if falha == nil || falha.TipoFalha != caso.tipo || !strings.Contains(falha.SaidaBruta, caso.trecho) {
				t.Fatalf("esperava %s com %q: %+v", caso.tipo, caso.trecho, falha)
			}
			if falha.TipoFalha == sandbox.FalhaCompilacao && !strings.Contains(falha.SaidaBruta, string(caso.execucao.Saida)) {
				t.Errorf("a evidência traz a saída do pip, com a dependência: %q", falha.SaidaBruta)
			}
		})
	}
}

func TestTesteInstalaSoDoCacheERodaOPytest(t *testing.T) {
	rodador := &rodadorDasEtapas{responder: func(comando processo.Comando) processo.Execucao {
		if slices.Contains(comando.Argumentos, "pytest") {
			return processo.Execucao{Saida: []byte("3 passed")}
		}
		return processo.Execucao{}
	}}
	cache := t.TempDir()
	if err := os.WriteFile(filepath.Join(cache, "requisitos.txt"), []byte("six==1.17.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	resultado := linguagemDasEtapas(t, rodador).Testar(t.Context(), projetoPython(t, map[string]string{}), cache, agoraPythonDeTeste.Add(10*time.Minute))

	if resultado.TipoFalha != sandbox.FalhaNenhuma || resultado.TestesPassaram != 3 {
		t.Fatalf("resultado: %+v", resultado)
	}
	linhas := rodador.linhas()
	if len(linhas) != 3 || !strings.Contains(linhas[0], "-m venv --system-site-packages") ||
		linhas[1] != "/tmp/aprova-cache/venv/bin/python -m pip install --no-index --find-links "+filepath.Join(cache, "wheels")+" -r "+filepath.Join(cache, "requisitos.txt") ||
		!strings.Contains(linhas[2], "/tmp/aprova-cache/venv/bin/python -m pytest") {
		t.Errorf("venv, instalação só do cache e pytest:\n%q", linhas)
	}
	for _, linha := range linhas {
		if strings.Contains(linha, "download") || strings.Contains(linha, " .") {
			t.Errorf("o teste não baixa nem instala o projeto: %q", linha)
		}
	}
}

func TestTesteSemDependenciasRodaOPytestDoSistema(t *testing.T) {
	rodador := &rodadorDasEtapas{responder: func(processo.Comando) processo.Execucao { return processo.Execucao{Saida: []byte("3 passed")} }}

	resultado := linguagemDasEtapas(t, rodador).Testar(t.Context(), projetoPython(t, map[string]string{"requirements.txt": "six\n"}), t.TempDir(), agoraPythonDeTeste.Add(10*time.Minute))

	if resultado.TipoFalha != sandbox.FalhaNenhuma {
		t.Fatalf("resultado: %+v", resultado)
	}
	if linhas := rodador.linhas(); len(linhas) != 1 || !strings.HasPrefix(linhas[0], "python -m pytest") {
		t.Errorf("sem lista no cache, nada é instalado, mesmo com requirements.txt no PR: %q", linhas)
	}
}

func TestTesteComInstalacaoQueFalhaEhInfraestrutura(t *testing.T) {
	rodador := &rodadorDasEtapas{responder: func(comando processo.Comando) processo.Execucao {
		if slices.Contains(comando.Argumentos, "install") {
			return processo.Execucao{CodigoSaida: 1, Saida: []byte("ERROR: No matching distribution found for six\n")}
		}
		return processo.Execucao{}
	}}
	cache := t.TempDir()
	if err := os.WriteFile(filepath.Join(cache, "requisitos.txt"), []byte("six\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	resultado := linguagemDasEtapas(t, rodador).Testar(t.Context(), projetoPython(t, map[string]string{}), cache, agoraPythonDeTeste.Add(10*time.Minute))

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura || !strings.Contains(resultado.SaidaBruta, "pip install") {
		t.Errorf("o que o preparo baixou tem de instalar; se não instala, é nosso: %+v", resultado)
	}
}

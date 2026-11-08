package python

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Azulpasta/aPRova/internal/sandbox"
	"github.com/Azulpasta/aPRova/internal/sandbox/processo"
)

var agoraPythonDeTeste = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)

type rodadorPythonFalso struct {
	respostas map[string]processo.Execucao
	rodados   []processo.Comando
	agora     time.Time
	custos    map[string]time.Duration
}

func (r *rodadorPythonFalso) Rodar(_ context.Context, comando processo.Comando) processo.Execucao {
	r.rodados = append(r.rodados, comando)
	chave := strings.Join(comando.Argumentos, " ")
	r.agora = r.agora.Add(r.custos[chave])

	for etapa, resposta := range r.respostas {
		if etapa == "pytest" && slices.Contains(comando.Argumentos, "pytest") {
			return resposta
		}
		if etapa == "venv" && slices.Contains(comando.Argumentos, "venv") {
			return resposta
		}
		if etapa == "pip" && slices.Contains(comando.Argumentos, "pip") {
			return resposta
		}
	}
	return processo.Execucao{}
}

func (r *rodadorPythonFalso) relogio() time.Time {
	return r.agora
}

func repositorioPythonDeTeste(t *testing.T, comRequisitos bool) string {
	t.Helper()

	diretorio := t.TempDir()
	if comRequisitos {
		err := os.WriteFile(
			filepath.Join(diretorio, arquivoDeRequisitos),
			[]byte("requests==2.32.0\n"),
			0o600,
		)
		if err != nil {
			t.Fatalf("criar requirements.txt: %v", err)
		}
	}
	return diretorio
}

func executorPythonDeTeste(
	t *testing.T,
	diretorio string,
	respostas map[string]processo.Execucao,
	relatorio string,
) (*Linguagem, *rodadorPythonFalso) {
	t.Helper()

	rodador := &rodadorPythonFalso{
		respostas: respostas,
		agora:     agoraPythonDeTeste,
		custos:    map[string]time.Duration{},
	}

	linguagem := Nova(Opcoes{
		Rodador:          rodador,
		DiretorioDeCache: "/tmp/aprova-cache",
		Agora:            rodador.relogio,
	})

	linguagem.lerRelatorio = func(string) ([]byte, error) {
		return []byte(relatorio), nil
	}

	return linguagem, rodador
}

func executarPythonDeTeste(
	t *testing.T,
	diretorio string,
	respostas map[string]processo.Execucao,
	relatorio string,
	prazo time.Duration,
) (sandbox.ResultadoSandbox, *rodadorPythonFalso) {
	t.Helper()

	linguagem, rodador := executorPythonDeTeste(
		t,
		diretorio,
		respostas,
		relatorio,
	)

	resultado := linguagem.Executar(
		t.Context(),
		diretorio,
		agoraPythonDeTeste.Add(prazo),
	)

	return resultado, rodador
}

func TestExecutarPythonComTodosOsTestesPassando(t *testing.T) {
	resultado, rodador := executarPythonDeTeste(
		t,
		repositorioPythonDeTeste(t, false),
		map[string]processo.Execucao{
			"pytest": {CodigoSaida: 0, Saida: []byte("3 passed")},
		},
		relatorioSoPassando,
		10*time.Minute,
	)

	if resultado.TipoFalha != sandbox.FalhaNenhuma ||
		!resultado.Compilou ||
		resultado.TestesPassaram != 3 ||
		resultado.TestesFalharam != 0 {
		t.Errorf("resultado inesperado: %+v", resultado)
	}

	if len(rodador.rodados) != 1 {
		t.Fatalf("esperava apenas pytest, executados: %d", len(rodador.rodados))
	}

	comando := rodador.rodados[0]
	if comando.Binario != binarioPython ||
		comando.Diretorio == "" ||
		!slices.Contains(comando.Argumentos, nomeDoPytest) ||
		!slices.Contains(comando.Argumentos, "-q") ||
		!slices.Contains(comando.Argumentos, "--json-report") {
		t.Errorf("comando pytest incorreto: %+v", comando)
	}
}

func TestExecutarPythonComTesteFalho(t *testing.T) {
	resultado, _ := executarPythonDeTeste(
		t,
		repositorioPythonDeTeste(t, false),
		map[string]processo.Execucao{
			"pytest": {CodigoSaida: 1, Saida: []byte("1 failed")},
		},
		relatorioUmFalho,
		10*time.Minute,
	)

	if resultado.TipoFalha != sandbox.FalhaTeste ||
		resultado.TestesPassaram != 2 ||
		resultado.TestesFalharam != 1 {
		t.Errorf("teste reprovado deveria ser classificado como falha de teste: %+v", resultado)
	}
}

func TestExecutarPythonComFalhaDeColetaEhCompilacao(t *testing.T) {
	resultado, _ := executarPythonDeTeste(
		t,
		repositorioPythonDeTeste(t, false),
		map[string]processo.Execucao{
			"pytest": {CodigoSaida: 2, Saida: []byte("SyntaxError")},
		},
		relatorioColetaFalha,
		10*time.Minute,
	)

	if resultado.TipoFalha != sandbox.FalhaCompilacao || resultado.Compilou {
		t.Errorf("falha de coleta deveria ser compilação: %+v", resultado)
	}
}

func TestExecutarPythonSemTestesNaoReprovaOPR(t *testing.T) {
	resultado, _ := executarPythonDeTeste(
		t,
		repositorioPythonDeTeste(t, false),
		map[string]processo.Execucao{
			"pytest": {CodigoSaida: 5},
		},
		relatorioSemTestes,
		10*time.Minute,
	)

	if resultado.TipoFalha != sandbox.FalhaNenhuma ||
		resultado.TestesPassaram != 0 ||
		resultado.TestesFalharam != 0 {
		t.Errorf("resultado inesperado sem testes: %+v", resultado)
	}
}

func TestExecutarPythonQuandoPytestNaoIniciaEhInfraestrutura(t *testing.T) {
	resultado, _ := executarPythonDeTeste(
		t,
		repositorioPythonDeTeste(t, false),
		map[string]processo.Execucao{
			"pytest": {Erro: errors.New("executável não encontrado")},
		},
		relatorioSoPassando,
		10*time.Minute,
	)

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura {
		t.Errorf("falha ao iniciar pytest deveria ser infraestrutura: %+v", resultado)
	}
}

func TestExecutarPythonQuandoPrazoEsgotaEhInfraestrutura(t *testing.T) {
	resultado, _ := executarPythonDeTeste(
		t,
		repositorioPythonDeTeste(t, false),
		map[string]processo.Execucao{
			"pytest": {PrazoEsgotado: true},
		},
		relatorioSoPassando,
		10*time.Minute,
	)

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura {
		t.Errorf("timeout do executor deveria ser infraestrutura: %+v", resultado)
	}
}

func TestExecutarPythonRecusaRelatorioInconsistente(t *testing.T) {
	resultado, _ := executarPythonDeTeste(
		t,
		repositorioPythonDeTeste(t, false),
		map[string]processo.Execucao{
			"pytest": {CodigoSaida: 1},
		},
		relatorioSoPassando,
		10*time.Minute,
	)

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura {
		t.Errorf("divergência entre relatório e processo deveria ser infraestrutura: %+v", resultado)
	}
}

func TestExecutarPythonRecusaRelatorioAusente(t *testing.T) {
	diretorio := repositorioPythonDeTeste(t, false)

	linguagem, _ := executorPythonDeTeste(
		t,
		diretorio,
		map[string]processo.Execucao{
			"pytest": {CodigoSaida: 0},
		},
		relatorioSoPassando,
	)

	linguagem.lerRelatorio = func(string) ([]byte, error) {
		return nil, os.ErrNotExist
	}

	resultado := linguagem.Executar(
		t.Context(),
		diretorio,
		agoraPythonDeTeste.Add(10*time.Minute),
	)

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura {
		t.Errorf("relatório ausente deveria ser infraestrutura: %+v", resultado)
	}
}

func TestExecutarPythonInstalaRequirements(t *testing.T) {
	resultado, rodador := executarPythonDeTeste(
		t,
		repositorioPythonDeTeste(t, true),
		map[string]processo.Execucao{
			"pytest": {CodigoSaida: 0},
		},
		relatorioSoPassando,
		10*time.Minute,
	)

	if resultado.TipoFalha != sandbox.FalhaNenhuma {
		t.Errorf("execução esperada sem falhas: %+v", resultado)
	}

	if len(rodador.rodados) != 3 {
		t.Fatalf("esperava venv, pip install e pytest; comandos: %d", len(rodador.rodados))
	}

	if !slices.Contains(rodador.rodados[0].Argumentos, "venv") {
		t.Errorf("primeiro comando deveria criar o venv: %+v", rodador.rodados[0])
	}

	if !slices.Contains(rodador.rodados[1].Argumentos, "pip") ||
		!slices.Contains(rodador.rodados[1].Argumentos, "-r") {
		t.Errorf("segundo comando deveria instalar requirements: %+v", rodador.rodados[1])
	}

	if !slices.Contains(rodador.rodados[2].Argumentos, "pytest") {
		t.Errorf("pytest deveria ser executado por último: %+v", rodador.rodados[2])
	}
}

func TestExecutarPythonQuandoCriacaoDoVenvFalha(t *testing.T) {
	resultado, rodador := executarPythonDeTeste(
		t,
		repositorioPythonDeTeste(t, true),
		map[string]processo.Execucao{
			"venv": {CodigoSaida: 1, Saida: []byte("falha no venv")},
		},
		relatorioSoPassando,
		10*time.Minute,
	)

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura {
		t.Errorf("falha ao criar venv deveria ser infraestrutura: %+v", resultado)
	}

	if len(rodador.rodados) != 1 {
		t.Errorf("não deveria continuar após falha no venv: %v", len(rodador.rodados))
	}
}

func TestExecutarPythonQuandoPipFalha(t *testing.T) {
	resultado, rodador := executarPythonDeTeste(
		t,
		repositorioPythonDeTeste(t, true),
		map[string]processo.Execucao{
			"pip": {CodigoSaida: 1, Saida: []byte("falha ao instalar dependência")},
		},
		relatorioSoPassando,
		10*time.Minute,
	)

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura {
		t.Errorf("falha no pip deveria ser infraestrutura: %+v", resultado)
	}

	if len(rodador.rodados) != 2 {
		t.Errorf("pytest não deveria rodar após falha no pip: %v", len(rodador.rodados))
	}
}

func TestExecutarPythonSemTempoSuficienteNaoRodaPytest(t *testing.T) {
	resultado, rodador := executarPythonDeTeste(
		t,
		repositorioPythonDeTeste(t, false),
		nil,
		relatorioSoPassando,
		20*time.Second,
	)

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura {
		t.Errorf("prazo insuficiente deveria ser infraestrutura: %+v", resultado)
	}

	if len(rodador.rodados) != 0 {
		t.Errorf("pytest não deveria rodar sem tempo suficiente: %v", len(rodador.rodados))
	}
}

func TestExecutarPythonUsaAmbienteRestrito(t *testing.T) {
	_, rodador := executarPythonDeTeste(
		t,
		repositorioPythonDeTeste(t, false),
		map[string]processo.Execucao{
			"pytest": {CodigoSaida: 0},
		},
		relatorioSoPassando,
		10*time.Minute,
	)

	if len(rodador.rodados) != 1 {
		t.Fatalf("esperava um comando, obtive %d", len(rodador.rodados))
	}

	comando := rodador.rodados[0]
	for _, exigida := range []string{
		"PATH=/tmp/aprova-cache/venv/bin:" + caminhoDosExecutaveis,
		"HOME=/tmp/aprova-cache/home",
		"PIP_CACHE_DIR=/tmp/aprova-cache/pip",
		"PIP_NO_INPUT=1",
		"PYTHONDONTWRITEBYTECODE=1",
		"PYTHONUNBUFFERED=1",
	} {
		if !slices.Contains(comando.Ambiente, exigida) {
			t.Errorf("ambiente não contém %q: %v", exigida, comando.Ambiente)
		}
	}
}

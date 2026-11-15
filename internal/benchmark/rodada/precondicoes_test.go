package rodada

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/benchmark/conferencia"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/isolamento"
)

type sondadorFalso struct {
	err error
}

func (s sondadorFalso) Verificar(context.Context) (isolamento.Relatorio, error) {
	return isolamento.Relatorio{Diretorio: "/tmp/sondagem", Tentativas: []isolamento.Tentativa{{Nome: "escrita", Barrada: s.err == nil}}}, s.err
}

func rodadaComSegredos(t *testing.T, env string) Rodada {
	t.Helper()
	return Rodada{Resultados: t.TempDir(), Segredos: conferencia.SegredosDoEnv([]byte(env))}
}

func registrosDePrecondicao(t *testing.T, rodada Rodada) string {
	t.Helper()
	arquivos, _ := filepath.Glob(filepath.Join(rodada.Resultados, "precondicoes", "*.json"))
	if len(arquivos) != 1 {
		t.Fatalf("esperava um registro de pré-condição: %v", arquivos)
	}
	conteudo, _ := os.ReadFile(arquivos[0])
	return string(conteudo)
}

func TestPrecondicoesPassamERegistramASondagemEOAutoteste(t *testing.T) {
	rodada := rodadaComSegredos(t, "DATABASE_URL=postgres://aprova:segredo@localhost/aprova\n")

	if err := rodada.VerificarPrecondicoes(t.Context(), sondadorFalso{}); err != nil {
		t.Fatal(err)
	}

	registro := registrosDePrecondicao(t, rodada)
	if !strings.Contains(registro, `"barrada": true`) || !strings.Contains(registro, `"variaveis_do_env_conferidas": 1`) || strings.Contains(registro, "segredo") {
		t.Errorf("o registro tem a sondagem e quantas variáveis foram conferidas, nunca os valores:\n%s", registro)
	}
}

func TestSondagemVioladaImpedeARodada(t *testing.T) {
	rodada := rodadaComSegredos(t, "DATABASE_URL=postgres://aprova:segredo@localhost/aprova\n")

	err := rodada.VerificarPrecondicoes(t.Context(), sondadorFalso{err: isolamento.ErrIsolamentoViolado})

	if !errors.Is(err, isolamento.ErrIsolamentoViolado) {
		t.Errorf("isolamento violado para tudo: %v", err)
	}
	if !strings.Contains(registrosDePrecondicao(t, rodada), "isolamento") {
		t.Error("a falha da pré-condição fica registrada")
	}
}

func TestEnvSemSegredosImpedeARodada(t *testing.T) {
	rodada := rodadaComSegredos(t, "SANDBOX_EXECUTOR=local\n")

	if err := rodada.VerificarPrecondicoes(t.Context(), sondadorFalso{}); !errors.Is(err, conferencia.ErrEnvSemSegredos) {
		t.Errorf("sem segredos para conferir, a checagem do diff não protege nada: %v", err)
	}
}

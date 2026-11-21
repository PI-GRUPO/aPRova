package metricas

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/platform"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

var (
	rotulosPermitidos        = []string{"estado", "nivel", "motivo", "resultado", "etapa", "linguagem", "tipo_falha", "tipo"}
	valorDeAltaCardinalidade = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-|^[0-9a-f]{40}$|^[0-9]+$`)
)

func exercitarTodas(m *Motor) {
	for _, estado := range []domain.EstadoTaskRun{domain.EstadoConcluido, domain.EstadoRejeitado, domain.EstadoFalhou} {
		for _, nivel := range []domain.NivelRisco{domain.RiscoBaixo, domain.RiscoMedio, domain.RiscoAlto, ""} {
			m.AnaliseEncerrada(estado, nivel)
		}
	}
	m.MotivosDoGate([]domain.MotivoDeRisco{{ID: "area_sensivel"}, {ID: "falha_sandbox"}})
	m.Calibracao(domain.CalibracaoAplicada)
	m.DuracaoDaEtapa(EtapaPublicacao, 250*time.Millisecond)
	m.DuracaoDaEtapa(string(domain.EstadoTestandoSandbox), 3*time.Second)
	m.ExecucaoDoSandbox(sandbox.LinguagemJava, sandbox.FalhaTeste)
	m.NovaTentativa(domain.EstadoMapeandoContexto)
	m.ReciboEmitido()
	m.LeaseVencidoRetomado()
	m.ErroDoGitHub(fmt.Errorf("GET /repos/x: %w", platform.ErrGitHubIndisponivel))
	m.ErroDoGitHub(platform.ErrSemPermissao)
	m.ErroDoGitHub(platform.ErrRespostaInesperada)
	m.Explicacao("gerada")
}

func TestExplicacoesSeContamPorResultado(t *testing.T) {
	registro := prometheus.NewRegistry()
	m := NovoMotor(registro)

	for _, resultado := range []string{"gerada", "gerada", "descartada", "falha", "reusada"} {
		m.Explicacao(resultado)
	}

	for resultado, esperado := range map[string]float64{"gerada": 2, "descartada": 1, "falha": 1, "reusada": 1} {
		if obtido := valorDoContador(t, registro, "aprova_explicacoes_total", "resultado", resultado); obtido != esperado {
			t.Errorf("aprova_explicacoes_total{resultado=%q} = %v, esperava %v", resultado, obtido, esperado)
		}
	}
}

func TestNenhumaMetricaTemRotuloDeAltaCardinalidade(t *testing.T) {
	registro := prometheus.NewRegistry()
	m := NovoMotor(registro)
	RegistrarTamanhoDaFila(registro, func(context.Context) (int, error) { return 3, nil })
	exercitarTodas(m)

	familias, err := registro.Gather()
	if err != nil {
		t.Fatalf("coletar: %v", err)
	}
	if len(familias) < 10 {
		t.Fatalf("esperava ao menos as 10 métricas do motor, coletei %d", len(familias))
	}
	for _, familia := range familias {
		for _, metrica := range familia.GetMetric() {
			for _, par := range metrica.GetLabel() {
				if !slices.Contains(rotulosPermitidos, par.GetName()) {
					t.Errorf("%s tem o rótulo %q, fora dos de baixa cardinalidade %v", familia.GetName(), par.GetName(), rotulosPermitidos)
				}
				if valorDeAltaCardinalidade.MatchString(par.GetValue()) {
					t.Errorf("%s tem o valor %q no rótulo %s, que parece id, SHA ou número", familia.GetName(), par.GetValue(), par.GetName())
				}
			}
		}
	}
}

func TestErrosDoGitHubSeClassificamEmTransitorioPermissaoEDefinitivo(t *testing.T) {
	registro := prometheus.NewRegistry()
	m := NovoMotor(registro)

	m.ErroDoGitHub(platform.ErrLimiteDeRequisicoes)
	m.ErroDoGitHub(fmt.Errorf("embrulhado: %w", platform.ErrGitHubIndisponivel))
	m.ErroDoGitHub(platform.ErrSemPermissao)
	m.ErroDoGitHub(errors.New("422 do GitHub"))

	for tipo, esperado := range map[string]float64{"transitorio": 2, "permissao": 1, "definitivo": 1} {
		if obtido := valorDoContador(t, registro, "aprova_erros_github_total", "tipo", tipo); obtido != esperado {
			t.Errorf("aprova_erros_github_total{tipo=%q} = %v, esperava %v", tipo, obtido, esperado)
		}
	}
}

func TestTamanhoDaFilaELidoNaColeta(t *testing.T) {
	registro := prometheus.NewRegistry()
	pendentes := 4
	RegistrarTamanhoDaFila(registro, func(context.Context) (int, error) { return pendentes, nil })

	pendentes = 7

	if valor := valorDoGauge(t, registro, "aprova_fila_pendentes"); valor != 7 {
		t.Errorf("aprova_fila_pendentes = %v, esperava o valor do momento da coleta, 7", valor)
	}
}

func TestTamanhoDaFilaSomeQuandoOBancoFalha(t *testing.T) {
	registro := prometheus.NewRegistry()
	RegistrarTamanhoDaFila(registro, func(context.Context) (int, error) { return 0, errors.New("banco fora") })

	familias, err := registro.Gather()
	if err != nil {
		t.Fatalf("uma falha na contagem não pode quebrar a coleta das outras métricas: %v", err)
	}
	for _, familia := range familias {
		if familia.GetName() == "aprova_fila_pendentes" {
			t.Errorf("sem contagem, a métrica não aparece em vez de mostrar um valor falso: %v", familia)
		}
	}
}

func valorDoContador(t *testing.T, registro *prometheus.Registry, nome, rotulo, valor string) float64 {
	t.Helper()

	familias, err := registro.Gather()
	if err != nil {
		t.Fatalf("coletar: %v", err)
	}
	for _, familia := range familias {
		if familia.GetName() != nome {
			continue
		}
		for _, metrica := range familia.GetMetric() {
			for _, par := range metrica.GetLabel() {
				if par.GetName() == rotulo && par.GetValue() == valor {
					return metrica.GetCounter().GetValue()
				}
			}
		}
	}
	return 0
}

func valorDoGauge(t *testing.T, registro *prometheus.Registry, nome string) float64 {
	t.Helper()

	familias, err := registro.Gather()
	if err != nil {
		t.Fatalf("coletar: %v", err)
	}
	for _, familia := range familias {
		if familia.GetName() == nome && len(familia.GetMetric()) == 1 {
			return familia.GetMetric()[0].GetGauge().GetValue()
		}
	}
	t.Fatalf("%s não foi coletada", nome)
	return 0
}

package explicacao

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

const taskRunDeTeste = "9eaa0538-add6-4361-8bfc-e74acec86c07"

type modeloGravador struct {
	mutex     sync.Mutex
	resposta  string
	falha     error
	sistemas  []string
	conteudos []string
}

func (m *modeloGravador) Gerar(_ context.Context, sistema, conteudo string) (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.sistemas = append(m.sistemas, sistema)
	m.conteudos = append(m.conteudos, conteudo)
	return m.resposta, m.falha
}

func (m *modeloGravador) chamadas() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return len(m.conteudos)
}

type registroEmMemoria struct {
	mutex      sync.Mutex
	porTaskRun map[string]domain.Explicacao
}

func novoRegistroEmMemoria() *registroEmMemoria {
	return &registroEmMemoria{porTaskRun: map[string]domain.Explicacao{}}
}

func (r *registroEmMemoria) ExplicacaoDoTaskRun(_ context.Context, taskRunID string) (domain.Explicacao, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	gravada, existe := r.porTaskRun[taskRunID]
	if !existe {
		return domain.Explicacao{}, domain.ErrNaoEncontrado
	}
	return gravada, nil
}

func (r *registroEmMemoria) GravarExplicacao(_ context.Context, gravada domain.Explicacao) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.porTaskRun[gravada.TaskRunID] = gravada
	return nil
}

type metricasGravadas struct {
	mutex      sync.Mutex
	resultados []string
}

func (m *metricasGravadas) Explicacao(resultado string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.resultados = append(m.resultados, resultado)
}

func explicadorDeTeste(t *testing.T, modelo Modelo, registro Registro, metricas Metricas) *Explicador {
	t.Helper()

	explicador, err := NovoExplicador(Opcoes{
		Modelo: modelo, NomeDoModelo: "modelo-de-teste", Registro: registro, Metricas: metricas, Timeout: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("montar o explicador: %v", err)
	}
	return explicador
}

func TestNovoExplicadorExigeModeloRegistroENome(t *testing.T) {
	if _, err := NovoExplicador(Opcoes{}); !errors.Is(err, ErrConfiguracao) {
		t.Errorf("sem modelo, registro e nome: %v", err)
	}
}

func TestRespostaValidaEGravadaEPublicada(t *testing.T) {
	modelo, registro, metricas := &modeloGravador{resposta: textoFiel}, novoRegistroEmMemoria(), &metricasGravadas{}

	texto := explicadorDeTeste(t, modelo, registro, metricas).Explicar(t.Context(), taskRunDeTeste, gateDeRiscoAlto(), sandboxComUmaFalha())

	if texto != textoFiel {
		t.Errorf("texto publicado = %q", texto)
	}
	gravada, _ := registro.ExplicacaoDoTaskRun(t.Context(), taskRunDeTeste)
	if gravada.Resultado != domain.ExplicacaoGerada || gravada.Texto != textoFiel || gravada.Modelo != "modelo-de-teste" {
		t.Errorf("explicação gravada = %+v", gravada)
	}
	if strings.Join(metricas.resultados, ",") != "gerada" {
		t.Errorf("métricas = %v", metricas.resultados)
	}
}

func TestRespostaInvalidaEDescartadaGravadaEContada(t *testing.T) {
	modelo, registro, metricas := &modeloGravador{resposta: strings.Replace(textoFiel, "risco alto", "risco baixo", 1)}, novoRegistroEmMemoria(), &metricasGravadas{}

	texto := explicadorDeTeste(t, modelo, registro, metricas).Explicar(t.Context(), taskRunDeTeste, gateDeRiscoAlto(), sandboxComUmaFalha())

	if texto != "" {
		t.Errorf("descartada não é publicada: %q", texto)
	}
	gravada, _ := registro.ExplicacaoDoTaskRun(t.Context(), taskRunDeTeste)
	if gravada.Resultado != domain.ExplicacaoDescartada || gravada.Texto != "" || !strings.Contains(gravada.Motivo, "nível") {
		t.Errorf("descarte gravado = %+v", gravada)
	}
	if strings.Join(metricas.resultados, ",") != "descartada" {
		t.Errorf("métricas = %v", metricas.resultados)
	}
}

func TestFalhaDoModeloEGravadaEContada(t *testing.T) {
	modelo, registro, metricas := &modeloGravador{falha: errors.New("gemini respondeu 429")}, novoRegistroEmMemoria(), &metricasGravadas{}

	texto := explicadorDeTeste(t, modelo, registro, metricas).Explicar(t.Context(), taskRunDeTeste, gateDeRiscoAlto(), sandboxComUmaFalha())

	gravada, _ := registro.ExplicacaoDoTaskRun(t.Context(), taskRunDeTeste)
	if texto != "" || gravada.Resultado != domain.ExplicacaoFalhou || !strings.Contains(gravada.Motivo, "429") {
		t.Errorf("falha: texto %q, gravada %+v", texto, gravada)
	}
	if strings.Join(metricas.resultados, ",") != "falha" {
		t.Errorf("métricas = %v", metricas.resultados)
	}
}

func TestRetomadaReusaAGravadaSemChamarOModelo(t *testing.T) {
	for _, gravada := range []domain.Explicacao{
		{TaskRunID: taskRunDeTeste, Resultado: domain.ExplicacaoGerada, Texto: textoFiel, Modelo: "modelo-de-teste"},
		{TaskRunID: taskRunDeTeste, Resultado: domain.ExplicacaoFalhou, Modelo: "modelo-de-teste", Motivo: "timeout"},
	} {
		modelo, registro, metricas := &modeloGravador{resposta: "outra coisa"}, novoRegistroEmMemoria(), &metricasGravadas{}
		registro.porTaskRun[taskRunDeTeste] = gravada

		texto := explicadorDeTeste(t, modelo, registro, metricas).Explicar(t.Context(), taskRunDeTeste, gateDeRiscoAlto(), sandboxComUmaFalha())

		if texto != gravada.Texto || modelo.chamadas() != 0 {
			t.Errorf("resultado %s: texto %q, chamadas ao modelo %d", gravada.Resultado, texto, modelo.chamadas())
		}
		if strings.Join(metricas.resultados, ",") != "reusada" {
			t.Errorf("métricas = %v", metricas.resultados)
		}
	}
}

func TestPedidoAoModeloNuncaLevaCodigoDiffAutorNemSegredo(t *testing.T) {
	modelo := &modeloGravador{resposta: textoFiel}
	gate := gateDeRiscoAlto()
	gate.Motivo = "MOTIVO_COMPLETO_GRAVADO"
	gate.DependenciasNovas = []domain.DependenciaNova{{Ecossistema: "go", Nome: "github.com/pacote/SEGREDO_DA_DEPENDENCIA", Versao: "v1.0.0"}}
	gate.Calibracao = domain.Calibracao{
		Estado: domain.CalibracaoHistoricoInsuficiente, Razao: "poucos desfechos",
		Vizinhos: []domain.VizinhoDaCalibracao{{Repositorio: "PI-GRUPO/REPOSITORIO_VIZINHO", NumeroPR: 77}},
	}
	resultado := sandboxComUmaFalha()
	resultado.SaidaBruta = "--- FAIL: TestFrete\n+func calcular() { token := \"ghp_SEGREDO_DO_SANDBOX\" }\ndiff --git a/go/frete/frete.go"

	explicadorDeTeste(t, modelo, novoRegistroEmMemoria(), nil).Explicar(t.Context(), taskRunDeTeste, gate, resultado)

	if modelo.chamadas() != 1 {
		t.Fatalf("o modelo foi chamado %d vezes", modelo.chamadas())
	}
	pedido := modelo.sistemas[0] + modelo.conteudos[0]
	for _, proibido := range []string{
		"SEGREDO_DO_SANDBOX", "ghp_", "diff --git", "func calcular", "FAIL: TestFrete", "SEGREDO_DA_DEPENDENCIA",
		"REPOSITORIO_VIZINHO", "MOTIVO_COMPLETO_GRAVADO", taskRunDeTeste,
	} {
		if strings.Contains(pedido, proibido) {
			t.Errorf("o pedido ao modelo contém %q", proibido)
		}
	}
	for _, esperado := range []string{"falha_sandbox", "codigo_sem_teste", "go/frete/frete.go", `"testes_falharam":1`, "alto"} {
		if !strings.Contains(modelo.conteudos[0], esperado) {
			t.Errorf("o pedido ao modelo não traz %q: %s", esperado, modelo.conteudos[0])
		}
	}
}

func TestMotivoInformativoNaoVaiAoModelo(t *testing.T) {
	gate := gateDeRiscoAlto()
	gate.Motivos = append(gate.Motivos, domain.MotivoDeRisco{ID: "calibracao_informativa", Nivel: domain.NivelInformativo, Descricao: "informativo"})

	if strings.Contains(entradaEmJSON(t, NovaEntrada(gate, sandbox.ResultadoSandbox{})), "calibracao_informativa") {
		t.Error("motivo informativo não pesa no nível e não vai ao modelo")
	}
}

func entradaEmJSON(t *testing.T, entrada Entrada) string {
	t.Helper()
	conteudo, err := entrada.JSON()
	if err != nil {
		t.Fatalf("serializar a entrada: %v", err)
	}
	return conteudo
}

func TestFalhaNoSandboxVaiAoModeloSoComOsNumerosDoSandbox(t *testing.T) {
	conteudo := entradaEmJSON(t, entradaDeRiscoAlto())

	if strings.Contains(conteudo, "1 de 11 testes falharam") {
		t.Errorf("a frase do gate para falha_sandbox não vai ao modelo, que escreve a partir dos números: %s", conteudo)
	}
	for _, esperado := range []string{`"id":"falha_sandbox"`, `"testes_passaram":10`, `"testes_falharam":1`, `"tipo_falha":"teste"`, "go/frete/frete.go"} {
		if !strings.Contains(conteudo, esperado) {
			t.Errorf("a entrada não traz %s: %s", esperado, conteudo)
		}
	}
}

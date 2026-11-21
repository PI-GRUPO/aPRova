package explicacao

import (
	"os"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

const (
	repeticoesPorCaso      = 5
	intervaloEntreChamadas = 4500 * time.Millisecond
)

type casoManual struct {
	nome      string
	gate      domain.GateDeRisco
	resultado sandbox.ResultadoSandbox
}

func casosManuais() []casoManual {
	return []casoManual{
		{
			nome: "falha_sandbox do PR 6",
			gate: domain.GateDeRisco{
				NivelDeterministico: domain.RiscoAlto, Nivel: domain.RiscoAlto,
				Motivos: []domain.MotivoDeRisco{{ID: "falha_sandbox", Nivel: domain.RiscoAlto, Descricao: "os testes ou a compilação falharam no sandbox", Evidencias: []string{"1 de 11 testes falharam"}}},
			},
			resultado: sandbox.ResultadoSandbox{Compilou: true, TestesPassaram: 10, TestesFalharam: 1, TipoFalha: sandbox.FalhaTeste},
		},
		{
			nome: "dependencia_nova",
			gate: domain.GateDeRisco{
				NivelDeterministico: domain.RiscoAlto, Nivel: domain.RiscoAlto,
				Motivos: []domain.MotivoDeRisco{{ID: "dependencia_nova", Nivel: domain.RiscoAlto, Descricao: "o PR acrescenta dependência nova, que precisa de revisão humana", Evidencias: []string{"go.mod"}}},
			},
			resultado: sandbox.ResultadoSandbox{Compilou: true, TestesPassaram: 12, TipoFalha: sandbox.FalhaNenhuma},
		},
		{
			nome:      "risco_baixo_sem_motivos",
			gate:      domain.GateDeRisco{NivelDeterministico: domain.RiscoBaixo, Nivel: domain.RiscoBaixo},
			resultado: sandbox.ResultadoSandbox{Compilou: true, TestesPassaram: 1, TipoFalha: sandbox.FalhaNenhuma},
		},
	}
}

func TestExplicacaoComOGeminiReal(t *testing.T) {
	if os.Getenv("APROVA_TESTE_GEMINI_REAL") != "1" {
		t.Skip("teste manual: APROVA_TESTE_GEMINI_REAL=1 e GEMINI_API_KEY para chamar o Gemini de verdade")
	}
	modelo := os.Getenv("EXPLICACAO_MODELO")
	if modelo == "" {
		modelo = "gemini-3.5-flash-lite"
	}
	gemini, err := NovoGemini(OpcoesGemini{Chave: os.Getenv("GEMINI_API_KEY"), Modelo: modelo})
	if err != nil {
		t.Fatalf("montar o Gemini: %v", err)
	}

	for _, caso := range casosManuais() {
		t.Run(caso.nome, func(t *testing.T) {
			chamarCincoVezes(t, gemini, caso)
		})
	}
}

func chamarCincoVezes(t *testing.T, gemini *Gemini, caso casoManual) {
	entrada := NovaEntrada(caso.gate, caso.resultado)
	conteudo, err := entrada.JSON()
	if err != nil {
		t.Fatalf("serializar a entrada: %v", err)
	}
	for repeticao := 1; repeticao <= repeticoesPorCaso; repeticao++ {
		time.Sleep(intervaloEntreChamadas)
		inicio := time.Now()
		resposta, err := gemini.Gerar(t.Context(), instrucaoDoSistema, conteudo)
		latencia := time.Since(inicio)
		if err != nil {
			t.Logf("#%d (%s): falha: %v", repeticao, latencia.Round(time.Millisecond), err)
			continue
		}
		texto, err := Validar(resposta, entrada)
		t.Logf("#%d (%s): validação %v\n%s", repeticao, latencia.Round(time.Millisecond), err, textoOuResposta(texto, resposta))
	}
}

func textoOuResposta(texto, resposta string) string {
	if texto != "" {
		return texto
	}
	return resposta
}

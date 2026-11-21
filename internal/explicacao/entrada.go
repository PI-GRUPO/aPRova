// Package explicacao descreve em português simples, com um modelo de IA, a
// decisão de risco que o gate já tomou. O modelo só descreve: não decide, não
// muda nível, estado, Check Run nem recibo, e recebe apenas o que o gate
// gravou, nunca código, diff, autor ou segredo.
package explicacao

import (
	"encoding/json"
	"fmt"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

const maximoDeEvidenciasPorMotivo = 20

var nomeDoNivel = map[domain.NivelRisco]string{
	domain.RiscoBaixo: "baixo",
	domain.RiscoMedio: "médio",
	domain.RiscoAlto:  "alto",
}

var efeitoNoMerge = map[domain.NivelRisco]string{
	domain.RiscoBaixo: "liberado",
	domain.RiscoMedio: "liberado, com pontos de atenção",
	domain.RiscoAlto:  "bloqueado até a aprovação de alguém com permissão de escrita no repositório",
}

// Entrada é tudo o que o modelo recebe: o nível decidido, o efeito no merge,
// os motivos de risco com as evidências do gate, o sandbox em números e, se a
// calibração foi aplicada, o nível pelas regras e a razão.
type Entrada struct {
	Nivel            string      `json:"nivel"`
	Merge            string      `json:"merge"`
	NivelPelasRegras string      `json:"nivel_pelas_regras,omitempty"`
	Motivos          []Motivo    `json:"motivos"`
	Sandbox          Sandbox     `json:"sandbox"`
	Calibracao       *Calibracao `json:"calibracao,omitempty"`
}

// Motivo é um motivo de risco do rules_hit, com as evidências que o gate
// gravou: caminhos de arquivo ou frases fixas sobre o sandbox.
type Motivo struct {
	ID         string   `json:"id"`
	Nivel      string   `json:"nivel"`
	Descricao  string   `json:"descricao"`
	Evidencias []string `json:"evidencias"`
}

// Sandbox é o resultado do sandbox só em números, sem a saída bruta.
type Sandbox struct {
	Compilou       bool   `json:"compilou"`
	TestesPassaram int    `json:"testes_passaram"`
	TestesFalharam int    `json:"testes_falharam"`
	TipoFalha      string `json:"tipo_falha,omitempty"`
}

// Calibracao é a razão pela qual o histórico reduziu o nível.
type Calibracao struct {
	Razao string `json:"razao"`
}

// NovaEntrada monta a entrada do modelo a partir do gate gravado e do
// resultado do sandbox. Motivos informativos ficam de fora, como no
// comentário.
func NovaEntrada(gate domain.GateDeRisco, resultado sandbox.ResultadoSandbox) Entrada {
	entrada := Entrada{
		Nivel:   nomeDoNivel[gate.Nivel],
		Merge:   efeitoNoMerge[gate.Nivel],
		Motivos: motivosDeRisco(gate.Motivos),
		Sandbox: Sandbox{
			Compilou: resultado.Compilou, TestesPassaram: resultado.TestesPassaram, TestesFalharam: resultado.TestesFalharam,
			TipoFalha: tipoDeFalha(resultado.TipoFalha),
		},
	}
	if gate.Calibracao.Estado == domain.CalibracaoAplicada {
		entrada.NivelPelasRegras = nomeDoNivel[gate.NivelDeterministico]
		entrada.Calibracao = &Calibracao{Razao: gate.Calibracao.Razao}
	}
	return entrada
}

// JSON devolve a entrada serializada, como ela vai ao modelo.
func (e Entrada) JSON() (string, error) {
	conteudo, err := json.Marshal(e)
	if err != nil {
		return "", fmt.Errorf("serializar a entrada da explicação: %w", err)
	}
	return string(conteudo), nil
}

func motivosDeRisco(motivos []domain.MotivoDeRisco) []Motivo {
	deRisco := []Motivo{}
	for _, motivo := range motivos {
		if motivo.Nivel == domain.NivelInformativo {
			continue
		}
		deRisco = append(deRisco, Motivo{
			ID: motivo.ID, Nivel: nomeDoNivel[motivo.Nivel], Descricao: motivo.Descricao,
			Evidencias: motivo.Evidencias[:min(len(motivo.Evidencias), maximoDeEvidenciasPorMotivo)],
		})
	}
	return deRisco
}

func tipoDeFalha(tipo sandbox.TipoFalha) string {
	if tipo == sandbox.FalhaNenhuma {
		return ""
	}
	return string(tipo)
}

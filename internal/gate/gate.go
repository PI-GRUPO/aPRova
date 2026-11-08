// Package gate é a camada determinística do gate de risco: recebe os arquivos
// alterados do pull request e o resultado do sandbox e devolve o nível, os
// motivos nomeados e as dependências novas. É lógica pura, sem rede, banco,
// relógio, aleatoriedade ou modelo de IA: a mesma entrada produz sempre a mesma
// saída, byte a byte, porque o recibo assina esse resultado.
package gate

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/platform"
	"github.com/Azulpasta/aPRova/internal/sandbox"
)

const maxArquivosPadrao = 20

var caminhosSensiveisPadrao = []string{"auth/", "security/", "payments/", "migrations/", "db/", "database/", "*.sql"}

var ordemDosNiveis = map[domain.NivelRisco]int{domain.RiscoBaixo: 0, domain.RiscoMedio: 1, domain.RiscoAlto: 2}

// ErrResultadoSemVeredito indica resultado do sandbox que não diz nada sobre o
// pull request, como falha de infraestrutura, que o motor deveria ter
// reexecutado.
var ErrResultadoSemVeredito = errors.New("gate: resultado do sandbox sem veredito sobre o PR")

// Limites ajusta as regras. MaxArquivos zero assume 20. CaminhosSensiveis nulo
// assume a lista padrão; cada item é um diretório terminado em barra, que vale
// em qualquer profundidade, ou um padrão para o nome do arquivo, como *.sql.
type Limites struct {
	MaxArquivos       int
	CaminhosSensiveis []string
}

// Entrada é o que o gate avalia.
type Entrada struct {
	Arquivos []platform.ArquivoAlterado
	Sandbox  sandbox.ResultadoSandbox
}

// Resultado é a decisão do gate: o maior nível entre os motivos, baixo quando
// nenhum dispara, os motivos ordenados por identificador e as dependências
// novas ordenadas.
type Resultado struct {
	Nivel        domain.NivelRisco        `json:"nivel"`
	Motivos      []domain.MotivoDeRisco   `json:"motivos"`
	Dependencias []domain.DependenciaNova `json:"dependencias"`
}

// Gate aplica as regras determinísticas.
type Gate struct {
	limites Limites
}

// Novo constrói o gate com os limites informados.
func Novo(limites Limites) *Gate {
	if limites.MaxArquivos <= 0 {
		limites.MaxArquivos = maxArquivosPadrao
	}
	if limites.CaminhosSensiveis == nil {
		limites.CaminhosSensiveis = caminhosSensiveisPadrao
	}
	return &Gate{limites: limites}
}

// Avaliar roda todas as regras, mesmo quando uma já deu alto. Falha de
// infraestrutura ou tipo de falha desconhecido devolvem
// ErrResultadoSemVeredito, e não um nível. O contexto não é usado: a avaliação
// não faz nenhuma chamada externa.
func (g *Gate) Avaliar(_ context.Context, entrada Entrada) (Resultado, error) {
	if !temVeredito(entrada.Sandbox.TipoFalha) {
		return Resultado{}, fmt.Errorf("%w: tipo_falha %q", ErrResultadoSemVeredito, entrada.Sandbox.TipoFalha)
	}

	dependencias := analisarManifestos(entrada.Arquivos)
	motivos := g.aplicarRegras(entrada, dependencias)
	slices.SortFunc(motivos, func(a, b domain.MotivoDeRisco) int { return strings.Compare(a.ID, b.ID) })

	return Resultado{Nivel: maiorNivel(motivos), Motivos: motivos, Dependencias: dependencias.novas}, nil
}

func temVeredito(tipo sandbox.TipoFalha) bool {
	return tipo == sandbox.FalhaNenhuma || tipo == sandbox.FalhaTeste || tipo == sandbox.FalhaCompilacao
}

func maiorNivel(motivos []domain.MotivoDeRisco) domain.NivelRisco {
	nivel := domain.RiscoBaixo
	for _, motivo := range motivos {
		if ordemDosNiveis[motivo.Nivel] > ordemDosNiveis[nivel] {
			nivel = motivo.Nivel
		}
	}
	return nivel
}

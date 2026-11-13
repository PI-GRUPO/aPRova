// Package calibracao é a camada 2 do gate de risco: compara a análise com
// análises passadas parecidas do mesmo tenant e, quando humanos aprovaram e
// mergearam todas elas sem reversão, desce o nível do gate um degrau. Nunca
// sobe o nível. O modelo de embedding só mede semelhança; quem decide é a regra
// fixa deste pacote.
package calibracao

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

// MotivoDaCalibracao é o id do motivo acrescentado quando a calibração desce
// o nível.
const MotivoDaCalibracao = "calibracao_historico"

const (
	vizinhosPadrao           = 10
	similaridadeMinimaPadrao = 0.97
	minimoDeDesfechosPadrao  = 5
	casasDaSimilaridade      = 1e4
)

var motivosCalibraveis = []string{"area_sensivel", "codigo_sem_teste", "muitos_arquivos"}

var nivelAbaixo = map[domain.NivelRisco]domain.NivelRisco{
	domain.RiscoAlto:  domain.RiscoMedio,
	domain.RiscoMedio: domain.RiscoBaixo,
}

var nomeDoNivel = map[domain.NivelRisco]string{
	domain.RiscoAlto:  "alto",
	domain.RiscoMedio: "médio",
	domain.RiscoBaixo: "baixo",
}

// Regra são os parâmetros fixos da calibração: quantos vizinhos mais parecidos
// olhar, a similaridade mínima para um vizinho contar e quantos desfechos
// conhecidos são exigidos.
type Regra struct {
	Vizinhos           int
	SimilaridadeMinima float64
	MinimoDeDesfechos  int
}

// RegraPadrao devolve 10 vizinhos, similaridade mínima de 0,97 e 5 desfechos
// conhecidos.
func RegraPadrao() Regra {
	return Regra{Vizinhos: vizinhosPadrao, SimilaridadeMinima: similaridadeMinimaPadrao, MinimoDeDesfechos: minimoDeDesfechosPadrao}
}

// Entrada é a decisão do gate determinístico e os vizinhos encontrados para
// ela.
type Entrada struct {
	TenantID int64
	Nivel    domain.NivelRisco
	Motivos  []domain.MotivoDeRisco
	Vizinhos []domain.VizinhoDaCalibracao
}

// Decisao é o nível final, os motivos finais e o rastro da calibração.
type Decisao struct {
	Nivel      domain.NivelRisco
	Motivos    []domain.MotivoDeRisco
	Calibracao domain.Calibracao
}

// Calibravel informa se o motivo está na lista fechada dos que a calibração
// pode suavizar.
func Calibravel(id string) bool {
	return slices.Contains(motivosCalibraveis, id)
}

// Elegivel diz se a calibração pode rodar para o nível e os motivos do gate.
// Quando não pode, devolve o estado e a razão a registrar.
func Elegivel(nivel domain.NivelRisco, motivos []domain.MotivoDeRisco) (domain.EstadoCalibracao, string, bool) {
	if _, temAbaixo := nivelAbaixo[nivel]; !temAbaixo {
		return domain.CalibracaoSemNivelAReduzir, "nível " + nomeDoNivel[domain.RiscoBaixo] + ": não há o que reduzir", false
	}

	var bloqueantes []string
	for _, motivo := range motivos {
		if !Calibravel(motivo.ID) {
			bloqueantes = append(bloqueantes, motivo.ID)
		}
	}
	if len(bloqueantes) > 0 {
		return domain.CalibracaoMotivoNaoCalibravel, "motivo não calibrável: " + strings.Join(bloqueantes, ", "), false
	}
	return "", "", true
}

// Decidir aplica a regra: entre os vizinhos do tenant, os mais parecidos acima
// da similaridade mínima com desfecho conhecido têm de ser ao menos o mínimo e
// todos aprovados e mergeados; então o nível desce um degrau. Em qualquer outro
// caso o nível do gate é mantido.
func Decidir(regra Regra, entrada Entrada) Decisao {
	mantida := func(estado domain.EstadoCalibracao, razao string, vizinhos []domain.VizinhoDaCalibracao) Decisao {
		return Decisao{Nivel: entrada.Nivel, Motivos: slices.Clone(entrada.Motivos),
			Calibracao: domain.Calibracao{Estado: estado, Razao: razao, Vizinhos: vizinhos}}
	}

	if estado, razao, elegivel := Elegivel(entrada.Nivel, entrada.Motivos); !elegivel {
		return mantida(estado, razao, nil)
	}

	conhecidos := desfechosConhecidos(regra, entrada)
	positivos, negativos := separar(conhecidos)
	if len(negativos) > 0 {
		return mantida(domain.CalibracaoVizinhoNegativo, razaoDosNegativos(negativos), conhecidos)
	}
	if len(positivos) < regra.MinimoDeDesfechos || len(positivos) == 0 {
		return mantida(domain.CalibracaoHistoricoInsuficiente,
			fmt.Sprintf("histórico insuficiente: %d de %d", len(positivos), regra.MinimoDeDesfechos), positivos)
	}
	return descerUmDegrau(entrada, positivos)
}

func desfechosConhecidos(regra Regra, entrada Entrada) []domain.VizinhoDaCalibracao {
	doTenant := make([]domain.VizinhoDaCalibracao, 0, len(entrada.Vizinhos))
	for _, vizinho := range entrada.Vizinhos {
		if vizinho.TenantID == entrada.TenantID {
			vizinho.Similaridade = math.Round(vizinho.Similaridade*casasDaSimilaridade) / casasDaSimilaridade
			doTenant = append(doTenant, vizinho)
		}
	}
	slices.SortStableFunc(doTenant, func(a, b domain.VizinhoDaCalibracao) int {
		return cmp.Or(cmp.Compare(b.Similaridade, a.Similaridade), strings.Compare(a.TaskRunID, b.TaskRunID))
	})
	if len(doTenant) > regra.Vizinhos {
		doTenant = doTenant[:regra.Vizinhos]
	}

	conhecidos := []domain.VizinhoDaCalibracao{}
	for _, vizinho := range doTenant {
		if vizinho.Similaridade >= regra.SimilaridadeMinima && desfechoConhecido(vizinho.Desfecho) {
			conhecidos = append(conhecidos, vizinho)
		}
	}
	return conhecidos
}

func desfechoConhecido(desfecho domain.DesfechoDoVizinho) bool {
	return desfecho == domain.VizinhoAprovadoEMergeado || desfechoNegativo(desfecho)
}

func desfechoNegativo(desfecho domain.DesfechoDoVizinho) bool {
	return desfecho == domain.VizinhoRevertido || desfecho == domain.VizinhoFechadoSemMerge
}

func separar(conhecidos []domain.VizinhoDaCalibracao) (positivos, negativos []domain.VizinhoDaCalibracao) {
	for _, vizinho := range conhecidos {
		if desfechoNegativo(vizinho.Desfecho) {
			negativos = append(negativos, vizinho)
		} else {
			positivos = append(positivos, vizinho)
		}
	}
	return positivos, negativos
}

func razaoDosNegativos(negativos []domain.VizinhoDaCalibracao) string {
	revertidos := 0
	for _, vizinho := range negativos {
		if vizinho.Desfecho == domain.VizinhoRevertido {
			revertidos++
		}
	}
	return fmt.Sprintf("caso parecido com desfecho negativo: %d revertido(s), %d fechado(s) sem merge", revertidos, len(negativos)-revertidos)
}

func descerUmDegrau(entrada Entrada, positivos []domain.VizinhoDaCalibracao) Decisao {
	novoNivel := nivelAbaixo[entrada.Nivel]
	razao := fmt.Sprintf("%d casos parecidos aprovados e mergeados, nenhum revertido", len(positivos))

	evidencias := make([]string, 0, len(positivos))
	for _, vizinho := range positivos {
		evidencias = append(evidencias, EvidenciaDoVizinho(vizinho))
	}

	motivos := append(slices.Clone(entrada.Motivos), domain.MotivoDeRisco{
		ID:         MotivoDaCalibracao,
		Nivel:      domain.NivelInformativo,
		Descricao:  fmt.Sprintf("risco %s pelas regras, reduzido a %s pelo histórico: %s", nomeDoNivel[entrada.Nivel], nomeDoNivel[novoNivel], razao),
		Evidencias: evidencias,
	})
	slices.SortStableFunc(motivos, func(a, b domain.MotivoDeRisco) int { return strings.Compare(a.ID, b.ID) })

	return Decisao{Nivel: novoNivel, Motivos: motivos,
		Calibracao: domain.Calibracao{Estado: domain.CalibracaoAplicada, Razao: razao, Vizinhos: positivos}}
}

// EvidenciaDoVizinho é a linha que o recibo assina para cada vizinho usado:
// task_run, pull request, similaridade com quatro casas e desfecho.
func EvidenciaDoVizinho(vizinho domain.VizinhoDaCalibracao) string {
	return fmt.Sprintf("%s %s#%d similaridade %.4f %s", vizinho.TaskRunID, vizinho.Repositorio, vizinho.NumeroPR, vizinho.Similaridade, vizinho.Desfecho)
}

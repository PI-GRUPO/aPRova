package calibracao

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

const (
	tenantDaAnalise = 7
	outroTenant     = 8
)

var todosOsMotivos = []string{
	"area_sensivel", "codigo_sem_teste", "configuracao_ci", "dependencia_nova",
	"falha_sandbox", "muitos_arquivos", "segredo_versionado", "regra_que_ainda_nao_existe",
}

var todosOsDesfechos = []domain.DesfechoDoVizinho{
	domain.VizinhoAprovadoEMergeado, domain.VizinhoRevertido, domain.VizinhoFechadoSemMerge,
	domain.VizinhoMergeadoSemAprovacao, domain.VizinhoSubstituido, domain.VizinhoEmAberto,
}

var ordemDoNivel = map[domain.NivelRisco]int{domain.RiscoBaixo: 0, domain.RiscoMedio: 1, domain.RiscoAlto: 2}

func motivo(id string, nivel domain.NivelRisco) domain.MotivoDeRisco {
	return domain.MotivoDeRisco{ID: id, Nivel: nivel, Descricao: "descrição de " + id, Evidencias: []string{"internal/auth/sessao.go"}}
}

func areaSensivel() domain.MotivoDeRisco { return motivo("area_sensivel", domain.RiscoAlto) }

func codigoSemTeste() domain.MotivoDeRisco { return motivo("codigo_sem_teste", domain.RiscoMedio) }

func vizinho(numero int, similaridade float64, desfecho domain.DesfechoDoVizinho) domain.VizinhoDaCalibracao {
	return domain.VizinhoDaCalibracao{
		TaskRunID:    fmt.Sprintf("00000000-0000-4000-8000-%012d", numero),
		TenantID:     tenantDaAnalise,
		Repositorio:  "Azulpasta/alvo",
		NumeroPR:     numero,
		Similaridade: similaridade,
		Desfecho:     desfecho,
	}
}

func positivos(quantos int) []domain.VizinhoDaCalibracao {
	vizinhos := make([]domain.VizinhoDaCalibracao, 0, quantos)
	for numero := range quantos {
		vizinhos = append(vizinhos, vizinho(numero+1, 0.99, domain.VizinhoAprovadoEMergeado))
	}
	return vizinhos
}

func entradaAlta(vizinhos []domain.VizinhoDaCalibracao, motivos ...domain.MotivoDeRisco) Entrada {
	if len(motivos) == 0 {
		motivos = []domain.MotivoDeRisco{areaSensivel()}
	}
	return Entrada{TenantID: tenantDaAnalise, Nivel: domain.RiscoAlto, Motivos: motivos, Vizinhos: vizinhos}
}

func motivosDoSubconjunto(mascara int) []domain.MotivoDeRisco {
	var motivos []domain.MotivoDeRisco
	for indice, id := range todosOsMotivos {
		if mascara&(1<<indice) != 0 {
			motivos = append(motivos, motivo(id, domain.RiscoAlto))
		}
	}
	return motivos
}

func vizinhosAleatorios(sorteio *rand.Rand) []domain.VizinhoDaCalibracao {
	vizinhos := make([]domain.VizinhoDaCalibracao, sorteio.IntN(15))
	for indice := range vizinhos {
		vizinhos[indice] = vizinho(indice+1, 0.8+sorteio.Float64()*0.2, todosOsDesfechos[sorteio.IntN(len(todosOsDesfechos))])
		if sorteio.IntN(4) == 0 {
			vizinhos[indice].TenantID = outroTenant
		}
	}
	return vizinhos
}

func TestNivelFinalNuncaSobeNemCaiMaisDeUmDegrau(t *testing.T) {
	sorteio := rand.New(rand.NewPCG(2026, 11))
	regras := []Regra{RegraPadrao(), {Vizinhos: 3, SimilaridadeMinima: 0.5, MinimoDeDesfechos: 1}, {Vizinhos: 50, SimilaridadeMinima: 0, MinimoDeDesfechos: 0}}

	for _, regra := range regras {
		for _, nivel := range []domain.NivelRisco{domain.RiscoBaixo, domain.RiscoMedio, domain.RiscoAlto} {
			for mascara := range 1 << len(todosOsMotivos) {
				for range 8 {
					entrada := Entrada{TenantID: tenantDaAnalise, Nivel: nivel, Motivos: motivosDoSubconjunto(mascara), Vizinhos: vizinhosAleatorios(sorteio)}

					final := Decidir(regra, entrada).Nivel

					if ordemDoNivel[final] > ordemDoNivel[nivel] {
						t.Fatalf("o nível subiu de %s para %s: %+v", nivel, final, entrada)
					}
					if ordemDoNivel[nivel]-ordemDoNivel[final] > 1 {
						t.Fatalf("o nível caiu mais de um degrau, de %s para %s: %+v", nivel, final, entrada)
					}
				}
			}
		}
	}
}

func TestMotivoNaoCalibravelImpedeACalibracao(t *testing.T) {
	for _, id := range []string{"falha_sandbox", "segredo_versionado", "dependencia_nova", "configuracao_ci"} {
		t.Run(id, func(t *testing.T) {
			decisao := Decidir(RegraPadrao(), entradaAlta(positivos(10), areaSensivel(), motivo(id, domain.RiscoAlto)))

			if decisao.Nivel != domain.RiscoAlto {
				t.Errorf("com %s presente o nível fica alto, ficou %s", id, decisao.Nivel)
			}
			if decisao.Calibracao.Estado != domain.CalibracaoMotivoNaoCalibravel || !strings.Contains(decisao.Calibracao.Razao, id) {
				t.Errorf("o rastro diz qual motivo impediu: %+v", decisao.Calibracao)
			}
			if len(decisao.Calibracao.Vizinhos) != 0 {
				t.Error("a calibração nem roda, então nenhum vizinho é considerado")
			}
		})
	}
}

func TestMotivoForaDaListaNaoECalibravel(t *testing.T) {
	decisao := Decidir(RegraPadrao(), entradaAlta(positivos(10), areaSensivel(), motivo("regra_que_ainda_nao_existe", domain.RiscoAlto)))

	if decisao.Nivel != domain.RiscoAlto || decisao.Calibracao.Estado != domain.CalibracaoMotivoNaoCalibravel {
		t.Errorf("só a lista fechada é calibrável: %s, %+v", decisao.Nivel, decisao.Calibracao)
	}
}

func TestSoOsTresMotivosDaListaSaoCalibraveis(t *testing.T) {
	var calibraveis []string
	for _, id := range todosOsMotivos {
		if Calibravel(id) {
			calibraveis = append(calibraveis, id)
		}
	}

	if !slices.Equal(calibraveis, []string{"area_sensivel", "codigo_sem_teste", "muitos_arquivos"}) {
		t.Errorf("calibráveis = %v", calibraveis)
	}
}

func TestVizinhosPositivosSuficientesDescemUmDegrau(t *testing.T) {
	casos := []struct {
		nome     string
		entrada  Entrada
		esperado domain.NivelRisco
	}{
		{"alto vira médio", entradaAlta(positivos(7)), domain.RiscoMedio},
		{"médio vira baixo", Entrada{TenantID: tenantDaAnalise, Nivel: domain.RiscoMedio, Motivos: []domain.MotivoDeRisco{codigoSemTeste()}, Vizinhos: positivos(5)}, domain.RiscoBaixo},
		{"alto com motivo médio cai só um degrau", entradaAlta(positivos(10), areaSensivel(), codigoSemTeste()), domain.RiscoMedio},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			decisao := Decidir(RegraPadrao(), caso.entrada)

			if decisao.Nivel != caso.esperado || decisao.Calibracao.Estado != domain.CalibracaoAplicada {
				t.Errorf("nível %s, estado %s", decisao.Nivel, decisao.Calibracao.Estado)
			}
		})
	}
}

func TestCalibracaoAcrescentaOMotivoComAEvidenciaDosVizinhos(t *testing.T) {
	entrada := entradaAlta(positivos(7), areaSensivel(), codigoSemTeste())
	originais := slices.Clone(entrada.Motivos)

	decisao := Decidir(RegraPadrao(), entrada)

	ids := make([]string, 0, len(decisao.Motivos))
	for _, motivo := range decisao.Motivos {
		ids = append(ids, motivo.ID)
	}
	if !slices.Equal(ids, []string{"area_sensivel", "calibracao_historico", "codigo_sem_teste"}) {
		t.Fatalf("os motivos seguem ordenados por id, com o da calibração: %v", ids)
	}
	historico := decisao.Motivos[1]
	if historico.Nivel != domain.NivelInformativo {
		t.Errorf("o motivo da calibração é informativo, veio %s", historico.Nivel)
	}
	if historico.Descricao != "risco alto pelas regras, reduzido a médio pelo histórico: 7 casos parecidos aprovados e mergeados, nenhum revertido" {
		t.Errorf("descrição = %q", historico.Descricao)
	}
	if len(historico.Evidencias) != 7 || historico.Evidencias[0] != "00000000-0000-4000-8000-000000000001 Azulpasta/alvo#1 similaridade 0.9900 aprovado_e_mergeado" {
		t.Errorf("cada vizinho usado é uma evidência com id, PR, similaridade e desfecho: %v", historico.Evidencias)
	}
	if len(decisao.Calibracao.Vizinhos) != 7 {
		t.Errorf("o rastro guarda os vizinhos usados: %+v", decisao.Calibracao.Vizinhos)
	}
	if !slices.EqualFunc(entrada.Motivos, originais, func(a, b domain.MotivoDeRisco) bool { return a.ID == b.ID }) {
		t.Error("os motivos da entrada não são alterados")
	}
}

func TestEvidenciaVemDaMaiorParaAMenorSimilaridade(t *testing.T) {
	vizinhos := []domain.VizinhoDaCalibracao{
		vizinho(3, 0.975, domain.VizinhoAprovadoEMergeado),
		vizinho(1, 0.999, domain.VizinhoAprovadoEMergeado),
		vizinho(2, 0.99, domain.VizinhoAprovadoEMergeado),
		vizinho(5, 0.99, domain.VizinhoAprovadoEMergeado),
		vizinho(4, 0.98123456, domain.VizinhoAprovadoEMergeado),
	}

	decisao := Decidir(RegraPadrao(), entradaAlta(vizinhos))

	var ordem []int
	for _, usado := range decisao.Calibracao.Vizinhos {
		ordem = append(ordem, usado.NumeroPR)
	}
	if !slices.Equal(ordem, []int{1, 2, 5, 4, 3}) {
		t.Errorf("ordem dos vizinhos = %v", ordem)
	}
	if similaridade := decisao.Calibracao.Vizinhos[3].Similaridade; similaridade != 0.9812 {
		t.Errorf("a similaridade gravada é a mesma que vai assinada, com quatro casas: %v", similaridade)
	}
}

func TestUmVizinhoNegativoImpedeACalibracao(t *testing.T) {
	for _, negativo := range []domain.DesfechoDoVizinho{domain.VizinhoRevertido, domain.VizinhoFechadoSemMerge} {
		t.Run(string(negativo), func(t *testing.T) {
			vizinhos := append(positivos(9), vizinho(40, 0.98, negativo))

			decisao := Decidir(RegraPadrao(), entradaAlta(vizinhos))

			if decisao.Nivel != domain.RiscoAlto || decisao.Calibracao.Estado != domain.CalibracaoVizinhoNegativo {
				t.Errorf("um negativo basta para manter alto: %s, %+v", decisao.Nivel, decisao.Calibracao)
			}
			if len(decisao.Calibracao.Vizinhos) != 10 {
				t.Errorf("o rastro mostra os vizinhos considerados, o negativo incluído: %d", len(decisao.Calibracao.Vizinhos))
			}
			if len(decisao.Motivos) != 1 {
				t.Errorf("sem calibração não há motivo novo: %+v", decisao.Motivos)
			}
		})
	}
}

func TestMenosVizinhosQueOMinimoNaoCalibra(t *testing.T) {
	decisao := Decidir(RegraPadrao(), entradaAlta(positivos(2)))

	if decisao.Nivel != domain.RiscoAlto || decisao.Calibracao.Estado != domain.CalibracaoHistoricoInsuficiente {
		t.Fatalf("%s, %+v", decisao.Nivel, decisao.Calibracao)
	}
	if decisao.Calibracao.Razao != "histórico insuficiente: 2 de 5" {
		t.Errorf("razão = %q", decisao.Calibracao.Razao)
	}
}

func TestVizinhoDeOutroTenantNuncaEConsiderado(t *testing.T) {
	deOutroTenant := func(vizinhos []domain.VizinhoDaCalibracao) []domain.VizinhoDaCalibracao {
		for indice := range vizinhos {
			vizinhos[indice].TenantID = outroTenant
		}
		return vizinhos
	}

	semHistoricoProprio := Decidir(RegraPadrao(), entradaAlta(deOutroTenant(positivos(10))))
	if semHistoricoProprio.Nivel != domain.RiscoAlto || semHistoricoProprio.Calibracao.Razao != "histórico insuficiente: 0 de 5" {
		t.Errorf("positivos de outro tenant não contam: %s, %+v", semHistoricoProprio.Nivel, semHistoricoProprio.Calibracao)
	}

	negativoAlheio := deOutroTenant([]domain.VizinhoDaCalibracao{vizinho(90, 0.99, domain.VizinhoRevertido)})
	comNegativoAlheio := Decidir(RegraPadrao(), entradaAlta(append(positivos(5), negativoAlheio...)))
	if comNegativoAlheio.Nivel != domain.RiscoMedio {
		t.Errorf("negativo de outro tenant também não conta: %s, %+v", comNegativoAlheio.Nivel, comNegativoAlheio.Calibracao)
	}
	for _, usado := range comNegativoAlheio.Calibracao.Vizinhos {
		if usado.TenantID != tenantDaAnalise {
			t.Errorf("vizinho de outro tenant no rastro: %+v", usado)
		}
	}
}

func TestVizinhoAbaixoDaSimilaridadeMinimaNaoConta(t *testing.T) {
	distantes := positivos(10)
	for indice := range distantes {
		distantes[indice].Similaridade = 0.9699
	}
	if decisao := Decidir(RegraPadrao(), entradaAlta(distantes)); decisao.Calibracao.Razao != "histórico insuficiente: 0 de 5" {
		t.Errorf("positivo distante não conta: %+v", decisao.Calibracao)
	}

	negativoDistante := vizinho(90, 0.96, domain.VizinhoFechadoSemMerge)
	if decisao := Decidir(RegraPadrao(), entradaAlta(append(positivos(5), negativoDistante))); decisao.Nivel != domain.RiscoMedio {
		t.Errorf("negativo distante não bloqueia: %+v", decisao.Calibracao)
	}
}

func TestSoOsKMaisParecidosSaoConsiderados(t *testing.T) {
	vizinhos := append(positivos(10), vizinho(90, 0.98, domain.VizinhoRevertido))

	decisao := Decidir(RegraPadrao(), entradaAlta(vizinhos))

	if decisao.Nivel != domain.RiscoMedio {
		t.Errorf("o revertido é o 11º mais parecido, fora dos 10: %+v", decisao.Calibracao)
	}
}

func TestMergeadoSemAprovacaoNuncaContaComoPositivoNemBloqueia(t *testing.T) {
	calibradosEMergeados := []domain.VizinhoDaCalibracao{}
	for numero := range 6 {
		calibradosEMergeados = append(calibradosEMergeados, vizinho(50+numero, 0.99, domain.VizinhoMergeadoSemAprovacao))
	}

	insuficiente := Decidir(RegraPadrao(), entradaAlta(append(positivos(4), calibradosEMergeados...)))
	if insuficiente.Nivel != domain.RiscoAlto || insuficiente.Calibracao.Razao != "histórico insuficiente: 4 de 5" {
		t.Errorf("mergeado sem aprovação não é positivo: %s, %+v", insuficiente.Nivel, insuficiente.Calibracao)
	}

	calibrada := Decidir(Regra{Vizinhos: 20, SimilaridadeMinima: 0.97, MinimoDeDesfechos: 5}, entradaAlta(append(positivos(5), calibradosEMergeados...)))
	if calibrada.Nivel != domain.RiscoMedio {
		t.Errorf("mergeado sem aprovação também não bloqueia: %s, %+v", calibrada.Nivel, calibrada.Calibracao)
	}
	for _, usado := range calibrada.Calibracao.Vizinhos {
		if usado.Desfecho != domain.VizinhoAprovadoEMergeado {
			t.Errorf("só aprovado e mergeado entra como evidência: %+v", usado)
		}
	}
}

func TestSubstituidoEEmAbertoSaoIgnorados(t *testing.T) {
	ignorados := []domain.VizinhoDaCalibracao{
		vizinho(60, 0.99, domain.VizinhoSubstituido),
		vizinho(61, 0.99, domain.VizinhoEmAberto),
	}

	decisao := Decidir(RegraPadrao(), entradaAlta(append(positivos(4), ignorados...)))

	if decisao.Calibracao.Razao != "histórico insuficiente: 4 de 5" {
		t.Errorf("sem desfecho conhecido, o vizinho não conta: %+v", decisao.Calibracao)
	}
}

func TestNivelBaixoNaoTemOQueCalibrar(t *testing.T) {
	decisao := Decidir(RegraPadrao(), Entrada{TenantID: tenantDaAnalise, Nivel: domain.RiscoBaixo, Vizinhos: positivos(10)})

	if decisao.Nivel != domain.RiscoBaixo || decisao.Calibracao.Estado != domain.CalibracaoSemNivelAReduzir {
		t.Errorf("%s, %+v", decisao.Nivel, decisao.Calibracao)
	}
}

func TestRegraPadrao(t *testing.T) {
	if regra := RegraPadrao(); regra != (Regra{Vizinhos: 10, SimilaridadeMinima: 0.97, MinimoDeDesfechos: 5}) {
		t.Errorf("regra padrão = %+v", regra)
	}
}

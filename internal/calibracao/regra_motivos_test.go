package calibracao

import (
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

func TestPositivoComMotivoAMaisOuAMenosNaoConta(t *testing.T) {
	casos := map[string][]string{
		"um motivo a mais":  {"area_sensivel", "codigo_sem_teste"},
		"um motivo a menos": {},
		"outro motivo":      {"muitos_arquivos"},
	}
	for nome, motivosDoVizinho := range casos {
		t.Run(nome, func(t *testing.T) {
			decisao := Decidir(RegraPadrao(), entradaAlta(positivosCom(10, motivosDoVizinho...)))

			if decisao.Nivel != domain.RiscoAlto || decisao.Calibracao.Razao != "histórico insuficiente: 0 de 5" {
				t.Errorf("similaridade 0,99 com outros motivos não conta: %s, %+v", decisao.Nivel, decisao.Calibracao)
			}
		})
	}
}

func TestSoOsPositivosComOsMesmosMotivosContam(t *testing.T) {
	outros := positivosCom(6, "area_sensivel", "codigo_sem_teste")
	for indice := range outros {
		outros[indice].NumeroPR += 50
		outros[indice].TaskRunID = vizinho(outros[indice].NumeroPR, 0, "").TaskRunID
	}

	decisao := Decidir(Regra{Vizinhos: 20, SimilaridadeMinima: 0.97, MinimoDeDesfechos: 5}, entradaAlta(append(positivos(4), outros...)))

	if decisao.Calibracao.Razao != "histórico insuficiente: 4 de 5" {
		t.Errorf("só os 4 com o mesmo conjunto contam: %+v", decisao.Calibracao)
	}
	for _, usado := range decisao.Calibracao.Vizinhos {
		if len(usado.Motivos) != 1 || usado.Motivos[0] != "area_sensivel" {
			t.Errorf("vizinho com outros motivos no rastro: %+v", usado)
		}
	}
}

func TestMesmosMotivosAbaixoDoLimiarNaoContam(t *testing.T) {
	distantes := positivos(10)
	for indice := range distantes {
		distantes[indice].Similaridade = 0.9699
	}

	decisao := Decidir(RegraPadrao(), entradaAlta(distantes))

	if decisao.Nivel != domain.RiscoAlto || decisao.Calibracao.Razao != "histórico insuficiente: 0 de 5" {
		t.Errorf("mesmos motivos a 0,9699 não contam: %s, %+v", decisao.Nivel, decisao.Calibracao)
	}
}

func TestOrdemDosMotivosEOMotivoDaCalibracaoNaoImportam(t *testing.T) {
	vizinhos := positivosCom(5, "codigo_sem_teste", MotivoDaCalibracao, "area_sensivel")

	decisao := Decidir(RegraPadrao(), entradaAlta(vizinhos, codigoSemTeste(), areaSensivel()))

	if decisao.Nivel != domain.RiscoMedio {
		t.Errorf("o conjunto é o mesmo, em outra ordem e com a calibração do vizinho: %+v", decisao.Calibracao)
	}
}

func TestNegativoComOutrosMotivosAindaBloqueia(t *testing.T) {
	negativo := vizinho(90, 0.99, domain.VizinhoRevertido)
	negativo.Motivos = []string{"area_sensivel", "codigo_sem_teste"}

	decisao := Decidir(RegraPadrao(), entradaAlta(append(positivos(9), negativo)))

	if decisao.Nivel != domain.RiscoAlto || decisao.Calibracao.Estado != domain.CalibracaoVizinhoNegativo {
		t.Errorf("o filtro de motivos só restringe: um negativo parecido bloqueia com quaisquer motivos: %s, %+v", decisao.Nivel, decisao.Calibracao)
	}
}

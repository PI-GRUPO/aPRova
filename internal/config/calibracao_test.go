package config

import (
	"strings"
	"testing"
	"time"
)

func TestCalibracaoFicaDesligadaComOsPadroesDoCodigo(t *testing.T) {
	ambienteMinimoValido(t)

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}

	if configuracao.CalibracaoLigada {
		t.Error("sem CALIBRACAO_LIGADA, a calibração fica desligada")
	}
	if configuracao.CalibracaoVizinhos != 10 || configuracao.CalibracaoSimilaridadeMinima != 0.97 ||
		configuracao.CalibracaoMinimoDeDesfechos != 5 || configuracao.CalibracaoTimeout != 5*time.Second {
		t.Errorf("padrões da calibração: %d vizinhos, %v, %d desfechos, %v", configuracao.CalibracaoVizinhos,
			configuracao.CalibracaoSimilaridadeMinima, configuracao.CalibracaoMinimoDeDesfechos, configuracao.CalibracaoTimeout)
	}
	if configuracao.ChaveAPIGemini != "" {
		t.Error("sem GEMINI_API_KEY, a chave fica vazia")
	}
}

func TestCalibracaoLeAsVariaveis(t *testing.T) {
	ambienteMinimoValido(t)
	t.Setenv("CALIBRACAO_LIGADA", "true")
	t.Setenv("CALIBRACAO_VIZINHOS", "20")
	t.Setenv("CALIBRACAO_SIMILARIDADE_MINIMA", "0.85")
	t.Setenv("CALIBRACAO_MINIMO_DE_DESFECHOS", "3")
	t.Setenv("CALIBRACAO_TIMEOUT", "2s")
	t.Setenv("GEMINI_API_KEY", "chave-do-gemini")

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}

	if !configuracao.CalibracaoLigada || configuracao.CalibracaoVizinhos != 20 || configuracao.CalibracaoSimilaridadeMinima != 0.85 ||
		configuracao.CalibracaoMinimoDeDesfechos != 3 || configuracao.CalibracaoTimeout != 2*time.Second || configuracao.ChaveAPIGemini != "chave-do-gemini" {
		t.Errorf("calibração lida errado: %+v", configuracao)
	}
}

func TestCalibracaoRecusaValoresInvalidosSemCitarAChave(t *testing.T) {
	casos := map[string]string{
		"CALIBRACAO_LIGADA=talvez":            "CALIBRACAO_LIGADA",
		"CALIBRACAO_VIZINHOS=0":               "CALIBRACAO_VIZINHOS",
		"CALIBRACAO_VIZINHOS=1001":            "CALIBRACAO_VIZINHOS",
		"CALIBRACAO_SIMILARIDADE_MINIMA=1.5":  "CALIBRACAO_SIMILARIDADE_MINIMA",
		"CALIBRACAO_SIMILARIDADE_MINIMA=-0.1": "CALIBRACAO_SIMILARIDADE_MINIMA",
		"CALIBRACAO_SIMILARIDADE_MINIMA=alta": "CALIBRACAO_SIMILARIDADE_MINIMA",
		"CALIBRACAO_SIMILARIDADE_MINIMA=NaN":  "CALIBRACAO_SIMILARIDADE_MINIMA",
		"CALIBRACAO_MINIMO_DE_DESFECHOS=0":    "CALIBRACAO_MINIMO_DE_DESFECHOS",
		"CALIBRACAO_TIMEOUT=0s":               "CALIBRACAO_TIMEOUT",
		"CALIBRACAO_TIMEOUT=-1s":              "CALIBRACAO_TIMEOUT",
		"CALIBRACAO_TIMEOUT=cinco":            "CALIBRACAO_TIMEOUT",
	}

	for atribuicao, variavel := range casos {
		t.Run(atribuicao, func(t *testing.T) {
			ambienteMinimoValido(t)
			t.Setenv("GEMINI_API_KEY", "chave-que-nunca-aparece")
			nome, valor, _ := strings.Cut(atribuicao, "=")
			t.Setenv(nome, valor)

			_, err := Carregar()

			if err == nil || !strings.Contains(err.Error(), variavel) {
				t.Fatalf("esperava erro nomeando %s, obtive %v", variavel, err)
			}
			if strings.Contains(err.Error(), "chave-que-nunca-aparece") {
				t.Errorf("o erro citou a chave: %v", err)
			}
		})
	}
}

package python

import (
	"errors"
	"testing"
)

const (
	relatorioSoPassando  = `{"exitcode": 0, "summary": {"passed": 3, "total": 3, "collected": 3}, "collectors": [{"nodeid": "", "outcome": "passed"}]}`
	relatorioUmFalho     = `{"exitcode": 1, "summary": {"passed": 2, "failed": 1, "total": 3, "collected": 3}, "collectors": []}`
	relatorioMisto       = `{"exitcode": 1, "summary": {"error": 1, "xfailed": 1, "xpassed": 1, "skipped": 1, "passed": 1, "total": 5, "collected": 5}}`
	relatorioColetaFalha = `{"exitcode": 2, "summary": {"total": 0, "collected": 0}, "collectors": [{"nodeid": "", "outcome": "passed"}, {"nodeid": "test_x.py", "outcome": "failed"}]}`
	relatorioSemTestes   = `{"exitcode": 5, "summary": {"total": 0, "collected": 0}, "collectors": []}`
)

func TestApurarContaPassadosEFalhos(t *testing.T) {
	casos := []struct {
		nome         string
		relatorio    string
		codigo       int
		passaram     int
		falharam     int
		coletaFalhou bool
	}{
		{"so passando", relatorioSoPassando, 0, 3, 0, false},
		{"um falho", relatorioUmFalho, 1, 2, 1, false},
		{"erro de setup conta como falho, xpassed como passou, xfailed e skipped ficam de fora", relatorioMisto, 1, 2, 1, false},
		{"falha ao coletar", relatorioColetaFalha, 2, 0, 0, true},
		{"sem testes", relatorioSemTestes, 5, 0, 0, false},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			apurada, err := apurar([]byte(caso.relatorio))
			if err != nil {
				t.Fatalf("relatório válido não podia dar erro: %v", err)
			}
			if apurada.codigoSaida != caso.codigo || apurada.passaram != caso.passaram || apurada.falharam != caso.falharam || apurada.coletaFalhou != caso.coletaFalhou {
				t.Errorf("apuração errada: %+v", apurada)
			}
		})
	}
}

func TestApurarRecusaRelatorioQueNaoEDoPytest(t *testing.T) {
	invalidos := map[string]string{
		"vazio":                  ``,
		"texto livre":            `aprova: tudo passou`,
		"json cortado":           `{"exitcode": 0, "summary": {"passed": 3`,
		"objeto vazio":           `{}`,
		"sem exitcode":           `{"summary": {"passed": 3}}`,
		"sem summary":            `{"exitcode": 0}`,
		"contagem negativa":      `{"exitcode": 0, "summary": {"passed": -1}}`,
		"contagem como texto":    `{"exitcode": 0, "summary": {"passed": "3"}}`,
		"lista no lugar do json": `[{"exitcode": 0}]`,
	}

	for nome, relatorio := range invalidos {
		t.Run(nome, func(t *testing.T) {
			_, err := apurar([]byte(relatorio))
			if !errors.Is(err, errRelatorioInvalido) {
				t.Errorf("esperava errRelatorioInvalido, obtive %v", err)
			}
		})
	}
}

package python

import (
	"encoding/json"
	"errors"
	"fmt"
)

const resultadoDeColetaQueFalhou = "failed"

// errRelatorioInvalido indica que o arquivo não é um relatório do
// pytest-json-report utilizável. O resultado nunca é montado a partir dele.
var errRelatorioInvalido = errors.New("python: relatório do pytest inválido")

type relatorioPytest struct {
	CodigoSaida *int      `json:"exitcode"`
	Resumo      *resumo   `json:"summary"`
	Coletores   []coletor `json:"collectors"`
}

type resumo struct {
	Passaram  int `json:"passed"`
	Falharam  int `json:"failed"`
	Erros     int `json:"error"`
	XPassaram int `json:"xpassed"`
}

type coletor struct {
	Resultado string `json:"outcome"`
}

type apuracao struct {
	codigoSaida  int
	passaram     int
	falharam     int
	coletaFalhou bool
}

// apurar lê o JSON do pytest-json-report. Passou é passed mais xpassed; falhou
// é failed mais error (erro em fixture também reprova); xfailed e skipped não
// entram em nenhum dos dois.
func apurar(bruto []byte) (apuracao, error) {
	var lido relatorioPytest
	if err := json.Unmarshal(bruto, &lido); err != nil {
		return apuracao{}, fmt.Errorf("%w: %w", errRelatorioInvalido, err)
	}
	if lido.CodigoSaida == nil || lido.Resumo == nil {
		return apuracao{}, fmt.Errorf("%w: faltam exitcode ou summary", errRelatorioInvalido)
	}

	r := lido.Resumo
	if r.Passaram < 0 || r.Falharam < 0 || r.Erros < 0 || r.XPassaram < 0 {
		return apuracao{}, fmt.Errorf("%w: contagem negativa", errRelatorioInvalido)
	}

	return apuracao{
		codigoSaida:  *lido.CodigoSaida,
		passaram:     r.Passaram + r.XPassaram,
		falharam:     r.Falharam + r.Erros,
		coletaFalhou: algumColetorFalhou(lido.Coletores),
	}, nil
}

func algumColetorFalhou(coletores []coletor) bool {
	for _, c := range coletores {
		if c.Resultado == resultadoDeColetaQueFalhou {
			return true
		}
	}
	return false
}

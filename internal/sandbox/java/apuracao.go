package java

import (
	"encoding/xml"
	"fmt"
)

type apuracao struct {
	passaram int
	falharam int
}

type suiteTestes struct {
	XMLName   xml.Name      `xml:"testsuite"`
	Testes    int           `xml:"tests,attr"`
	Falhas    int           `xml:"failures,attr"`
	Erros     int           `xml:"errors,attr"`
	Ignorados int           `xml:"skipped,attr"`
	Suites    []suiteTestes `xml:"testsuite"`
}

type relatorioSuites struct {
	XMLName xml.Name      `xml:"testsuites"`
	Suites  []suiteTestes `xml:"testsuite"`
}

func apurarRelatorio(bruto []byte) (apuracao, error) {
	if len(bruto) == 0 {
		return apuracao{}, fmt.Errorf("relatório do Maven está vazio")
	}

	var raiz struct {
		XMLName xml.Name
	}

	if err := xml.Unmarshal(bruto, &raiz); err != nil {
		return apuracao{}, fmt.Errorf(
			"não foi possível interpretar o relatório Maven: %w",
			err,
		)
	}

	switch raiz.XMLName.Local {
	case "testsuite":
		var suite suiteTestes

		if err := xml.Unmarshal(bruto, &suite); err != nil {
			return apuracao{}, fmt.Errorf(
				"não foi possível interpretar a suíte Maven: %w",
				err,
			)
		}

		return apurarSuite(suite)

	case "testsuites":
		var relatorios relatorioSuites

		if err := xml.Unmarshal(bruto, &relatorios); err != nil {
			return apuracao{}, fmt.Errorf(
				"não foi possível interpretar as suítes Maven: %w",
				err,
			)
		}

		if len(relatorios.Suites) == 0 {
			return apuracao{}, fmt.Errorf(
				"relatório Maven não contém suítes de testes",
			)
		}

		var resultado apuracao

		for _, suite := range relatorios.Suites {
			parcial, err := apurarSuite(suite)
			if err != nil {
				return apuracao{}, err
			}

			resultado.passaram += parcial.passaram
			resultado.falharam += parcial.falharam
		}

		return resultado, nil

	default:
		return apuracao{}, fmt.Errorf(
			"elemento raiz inesperado no relatório Maven: %s",
			raiz.XMLName.Local,
		)
	}
}

func apurarSuite(suite suiteTestes) (apuracao, error) {
	if suite.Testes < 0 ||
		suite.Falhas < 0 ||
		suite.Erros < 0 ||
		suite.Ignorados < 0 {
		return apuracao{}, fmt.Errorf(
			"relatório Maven contém contagens negativas",
		)
	}

	falharam := suite.Falhas + suite.Erros
	contabilizados := falharam + suite.Ignorados

	if contabilizados > suite.Testes {
		return apuracao{}, fmt.Errorf(
			"contagens inconsistentes no relatório Maven",
		)
	}

	resultado := apuracao{
		passaram: suite.Testes - contabilizados,
		falharam: falharam,
	}

	for _, item := range suite.Suites {
		parcial, err := apurarSuite(item)
		if err != nil {
			return apuracao{}, err
		}

		resultado.passaram += parcial.passaram
		resultado.falharam += parcial.falharam
	}

	return resultado, nil
}

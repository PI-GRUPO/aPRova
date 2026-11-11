package java

import (
	"encoding/xml"
	"fmt"
	"io/fs"
	"path/filepath"
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

func apurarRelatorios(diretorio string) (apuracao, error) {
	var resultado apuracao
	var arquivos []string

	err := filepath.WalkDir(diretorio, func(
		caminho string,
		entrada fs.DirEntry,
		err error,
	) error {
		if err != nil {
			return err
		}

		if entrada.IsDir() &&
			filepath.Base(caminho) == "surefire-reports" &&
			filepath.Base(filepath.Dir(caminho)) == "target" {

			relatorios, err := filepath.Glob(
				filepath.Join(caminho, "TEST-*.xml"),
			)
			if err != nil {
				return err
			}

			arquivos = append(arquivos, relatorios...)
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		return apuracao{}, fmt.Errorf(
			"não foi possível localizar os relatórios Maven: %w",
			err,
		)
	}

	if len(arquivos) == 0 {
		return apuracao{}, fmt.Errorf(
			"nenhum relatório XML encontrado em target/surefire-reports",
		)
	}

	for _, arquivo := range arquivos {
		bruto, err := lerArquivoComSeguranca(arquivo)
		if err != nil {
			return apuracao{}, fmt.Errorf(
				"não foi possível ler %s: %w",
				arquivo,
				err,
			)
		}

		parcial, err := apurarRelatorio(bruto)
		if err != nil {
			return apuracao{}, fmt.Errorf(
				"erro ao apurar %s: %w",
				arquivo,
				err,
			)
		}

		resultado.passaram += parcial.passaram
		resultado.falharam += parcial.falharam
	}

	return resultado, nil
}

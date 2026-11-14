package java

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/PI-GRUPO/aPRova/internal/sandbox/arquivo"
)

const (
	diretorioDeBuild       = "target"
	diretorioDosRelatorios = "surefire-reports"
	padraoDosRelatorios    = "TEST-*.xml"
)

type apuracao struct {
	passaram int
	falharam int
}

type suiteDeTestes struct {
	Testes    int             `xml:"tests,attr"`
	Falhas    int             `xml:"failures,attr"`
	Erros     int             `xml:"errors,attr"`
	Ignorados int             `xml:"skipped,attr"`
	Suites    []suiteDeTestes `xml:"testsuite"`
}

type raizDoRelatorio struct {
	XMLName xml.Name
	suiteDeTestes
}

func apurarRelatorios(projeto string) (apuracao, error) {
	arquivos, err := localizarRelatorios(projeto)
	if err != nil {
		return apuracao{}, fmt.Errorf("localizar os relatórios do surefire: %w", err)
	}

	var total apuracao
	for _, caminho := range arquivos {
		bruto, err := arquivo.LerDoPR(caminho)
		if err != nil {
			return apuracao{}, fmt.Errorf("ler %s: %w", caminho, err)
		}
		parcial, err := apurarRelatorio(bruto)
		if err != nil {
			return apuracao{}, fmt.Errorf("apurar %s: %w", caminho, err)
		}
		total.passaram += parcial.passaram
		total.falharam += parcial.falharam
	}
	return total, nil
}

func localizarRelatorios(projeto string) ([]string, error) {
	var arquivos []string
	err := filepath.WalkDir(projeto, func(caminho string, entrada fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entrada.IsDir() || filepath.Base(caminho) != diretorioDosRelatorios || filepath.Base(filepath.Dir(caminho)) != diretorioDeBuild {
			return nil
		}
		encontrados, err := filepath.Glob(filepath.Join(caminho, padraoDosRelatorios))
		if err != nil {
			return err
		}
		arquivos = append(arquivos, encontrados...)
		return filepath.SkipDir
	})
	return arquivos, err
}

func apurarRelatorio(bruto []byte) (apuracao, error) {
	if len(bruto) == 0 {
		return apuracao{}, errors.New("relatório do surefire vazio")
	}

	var raiz raizDoRelatorio
	if err := xml.Unmarshal(bruto, &raiz); err != nil {
		return apuracao{}, fmt.Errorf("relatório do surefire ilegível: %w", err)
	}

	switch raiz.XMLName.Local {
	case "testsuite":
		return apurarSuite(raiz.suiteDeTestes)
	case "testsuites":
		return apurarSuite(suiteDeTestes{Suites: raiz.Suites})
	default:
		return apuracao{}, fmt.Errorf("elemento raiz inesperado no relatório do surefire: %s", raiz.XMLName.Local)
	}
}

func apurarSuite(suite suiteDeTestes) (apuracao, error) {
	if suite.Testes < 0 || suite.Falhas < 0 || suite.Erros < 0 || suite.Ignorados < 0 {
		return apuracao{}, errors.New("relatório do surefire com contagem negativa")
	}

	falharam := suite.Falhas + suite.Erros
	if falharam+suite.Ignorados > suite.Testes {
		return apuracao{}, errors.New("relatório do surefire com contagens inconsistentes")
	}

	total := apuracao{passaram: suite.Testes - falharam - suite.Ignorados, falharam: falharam}
	for _, interna := range suite.Suites {
		parcial, err := apurarSuite(interna)
		if err != nil {
			return apuracao{}, err
		}
		total.passaram += parcial.passaram
		total.falharam += parcial.falharam
	}
	return total, nil
}

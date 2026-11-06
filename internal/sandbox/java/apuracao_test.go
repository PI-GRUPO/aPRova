package java

import "testing"

func TestApurarRelatorio(t *testing.T) {
	testes := []struct {
		nome       string
		xml        string
		passaram   int
		falharam   int
		esperaErro bool
	}{
		{
			nome:     "todos passaram",
			xml:      `<testsuite tests="3" failures="0" errors="0" skipped="0"></testsuite>`,
			passaram: 3,
		},
		{
			nome:     "teste falhou",
			xml:      `<testsuite tests="3" failures="1" errors="0" skipped="0"></testsuite>`,
			passaram: 2,
			falharam: 1,
		},
		{
			nome:     "teste com erro",
			xml:      `<testsuite tests="3" failures="0" errors="1" skipped="0"></testsuite>`,
			passaram: 2,
			falharam: 1,
		},
		{
			nome:     "teste ignorado",
			xml:      `<testsuite tests="3" failures="0" errors="0" skipped="1"></testsuite>`,
			passaram: 2,
		},
		{
			nome: "multiplas suites",
			xml: `<testsuites>
				<testsuite tests="2" failures="0" errors="0" skipped="0"></testsuite>
				<testsuite tests="2" failures="1" errors="0" skipped="0"></testsuite>
			</testsuites>`,
			passaram: 3,
			falharam: 1,
		},
		{
			nome:       "relatorio vazio",
			xml:        "",
			esperaErro: true,
		},
		{
			nome:       "contagem inconsistente",
			xml:        `<testsuite tests="1" failures="2" errors="0" skipped="0"></testsuite>`,
			esperaErro: true,
		},
	}

	for _, teste := range testes {
		t.Run(teste.nome, func(t *testing.T) {
			resultado, err := apurarRelatorio([]byte(teste.xml))

			if teste.esperaErro {
				if err == nil {
					t.Fatal("esperava erro, mas não recebeu")
				}
				return
			}

			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}

			if resultado.passaram != teste.passaram {
				t.Errorf("passaram: esperado %d, recebido %d",
					teste.passaram, resultado.passaram)
			}

			if resultado.falharam != teste.falharam {
				t.Errorf("falharam: esperado %d, recebido %d",
					teste.falharam, resultado.falharam)
			}
		})
	}
}

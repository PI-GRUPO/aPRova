package integracao

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/benchmark/oraculo"
	"github.com/PI-GRUPO/aPRova/internal/benchmark/resultado"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

const testeOriginalDaSoma = "package soma\n\nimport \"testing\"\n\nfunc TestSoma(t *testing.T) {\n\tif Soma(2, 3) != 5 {\n\t\tt.Fatal(\"2 + 3 precisa dar 5\")\n\t}\n}\n"

func gravarArquivos(t *testing.T, raiz string, arquivos map[string]string) {
	t.Helper()
	for nome, conteudo := range arquivos {
		caminho := filepath.Join(raiz, nome)
		if err := os.MkdirAll(filepath.Dir(caminho), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(caminho, []byte(conteudo), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func julgarSoma(t *testing.T, solucao map[string]string) oraculo.Veredito {
	t.Helper()
	imagemGo.exigir(t)

	originais := t.TempDir()
	gravarArquivos(t, originais, map[string]string{
		"go.mod": "module soma\n\ngo 1.18\n", "soma.go": "package soma\n\nfunc Soma(a, b int) int { panic(\"implemente\") }\n",
		"soma_test.go": testeOriginalDaSoma,
	})
	diretorioDaSolucao := t.TempDir()
	gravarArquivos(t, diretorioDaSolucao, solucao)

	julgador := oraculo.Oraculo{
		Publicador: oraculo.PublicadorDocker{Imagem: imagemGo.nome()},
		Executor: func(montagens []string) sandbox.Executor {
			return sandbox.NovoExecutorLocal(sandbox.OpcoesExecutorLocal{
				ImagemPorLinguagem: map[sandbox.Linguagem]string{sandbox.LinguagemGo: imagemGo.nome()},
				Montagens:          montagens,
			})
		},
		Temporario: t.TempDir(),
	}
	veredito, err := julgador.Julgar(t.Context(), oraculo.Entrada{
		ID: "go-soma", Linguagem: "go", Solucao: diretorioDaSolucao, Originais: originais, Protegidos: []string{"soma_test.go"},
	})
	if err != nil {
		t.Fatalf("julgar: %v", err)
	}
	t.Logf("veredito: %s %s; sandbox: %+v", veredito.Oraculo, veredito.Detalhe, veredito.Sandbox)
	exigirNenhumVolumeDoPreparoSobrando(t)
	return veredito
}

func TestOraculoAprovaSolucaoCerta(t *testing.T) {
	veredito := julgarSoma(t, map[string]string{
		"go.mod": "module soma\n\ngo 1.18\n", "soma.go": "package soma\n\nfunc Soma(a, b int) int { return a + b }\n",
		"soma_test.go": testeOriginalDaSoma,
	})

	if veredito.Oraculo != resultado.OraculoPassou {
		t.Errorf("a solução certa passa no oráculo: %+v", veredito)
	}
}

func TestOraculoReprovaSolucaoErrada(t *testing.T) {
	veredito := julgarSoma(t, map[string]string{
		"go.mod": "module soma\n\ngo 1.18\n", "soma.go": "package soma\n\nfunc Soma(a, b int) int { return a - b }\n",
		"soma_test.go": testeOriginalDaSoma,
	})

	if veredito.Oraculo != resultado.OraculoFalhou {
		t.Errorf("a solução errada falha no oráculo: %+v", veredito)
	}
}

func TestOraculoReprovaSolucaoQueAfrouxouOTeste(t *testing.T) {
	veredito := julgarSoma(t, map[string]string{
		"go.mod": "module soma\n\ngo 1.18\n", "soma.go": "package soma\n\nfunc Soma(a, b int) int { return 0 }\n",
		"soma_test.go": "package soma\n\nimport \"testing\"\n\nfunc TestSoma(t *testing.T) {}\n",
	})

	if veredito.Oraculo != resultado.OraculoFalhou {
		t.Errorf("o teste afrouxado pelo agente não vale; o oráculo usa o original: %+v", veredito)
	}
}

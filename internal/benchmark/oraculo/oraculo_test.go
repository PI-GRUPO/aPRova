package oraculo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/benchmark/resultado"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

type publicadorFalso struct {
	t        *testing.T
	arquivos map[string]string
	sha      string
	limpou   bool
}

func (p *publicadorFalso) Publicar(ctx context.Context, repositorio, nome string) (Publicacao, func(), error) {
	p.arquivos = map[string]string{}
	_ = filepath.WalkDir(repositorio, func(caminho string, entrada os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relativo, _ := filepath.Rel(repositorio, caminho)
		if entrada.IsDir() && relativo == ".git" {
			return filepath.SkipDir
		}
		if !entrada.IsDir() {
			conteudo, _ := os.ReadFile(caminho)
			p.arquivos[filepath.ToSlash(relativo)] = string(conteudo)
		}
		return nil
	})
	saida, err := exec.CommandContext(ctx, "git", "-C", repositorio, "rev-parse", "HEAD").Output()
	if err != nil {
		p.t.Fatalf("o repositório publicado precisa ter um commit: %v", err)
	}
	p.sha = strings.TrimSpace(string(saida))
	return Publicacao{URL: "file:///repositorios/" + nome, Montagem: "volume-falso:/repositorios:ro"}, func() { p.limpou = true }, nil
}

type executorFalso struct {
	resultado sandbox.ResultadoSandbox
	pedido    sandbox.PedidoExecucao
	montagens []string
}

func (e *executorFalso) Executar(_ context.Context, pedido sandbox.PedidoExecucao) (sandbox.ResultadoSandbox, error) {
	e.pedido = pedido
	return e.resultado, nil
}

func escrever(t *testing.T, raiz, relativo, conteudo string) {
	t.Helper()
	caminho := filepath.Join(raiz, filepath.FromSlash(relativo))
	if err := os.MkdirAll(filepath.Dir(caminho), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(caminho, []byte(conteudo), 0o600); err != nil {
		t.Fatal(err)
	}
}

func cenario(t *testing.T, resultadoDoSandbox sandbox.ResultadoSandbox) (Entrada, *publicadorFalso, *executorFalso, Oraculo) {
	t.Helper()
	originais := t.TempDir()
	escrever(t, originais, "go.mod", "module a\n")
	escrever(t, originais, "a.go", "package a // stub\n")
	escrever(t, originais, "a_test.go", "package a // teste original\n")
	escrever(t, originais, "cases_test.go", "package a // casos originais\n")

	solucao := t.TempDir()
	escrever(t, solucao, "go.mod", "module a\n")
	escrever(t, solucao, "a.go", "package a // solução do agente\n")
	escrever(t, solucao, "a_test.go", "package a // teste ADULTERADO pelo agente\n")
	escrever(t, solucao, "ajuda.go", "package a // arquivo novo do agente\n")
	escrever(t, solucao, ".gocache/x", "cache")
	escrever(t, solucao, ".git/HEAD", "ref: refs/heads/base\n")

	publicador := &publicadorFalso{t: t}
	executor := &executorFalso{resultado: resultadoDoSandbox}
	oraculo := Oraculo{
		Publicador: publicador,
		Executor: func(montagens []string) sandbox.Executor {
			executor.montagens = montagens
			return executor
		},
		Temporario: t.TempDir(),
	}
	entrada := Entrada{ID: "go-a", Linguagem: "go", Solucao: solucao, Originais: originais, Protegidos: []string{"a_test.go", "cases_test.go"}}
	return entrada, publicador, executor, oraculo
}

func TestOracleRodaOsTestesOriginaisContraASolucao(t *testing.T) {
	entrada, publicador, executor, oraculo := cenario(t, sandbox.ResultadoSandbox{TipoFalha: sandbox.FalhaNenhuma, Compilou: true, TestesPassaram: 5})

	veredito, err := oraculo.Julgar(t.Context(), entrada)
	if err != nil {
		t.Fatal(err)
	}

	if veredito.Oraculo != resultado.OraculoPassou {
		t.Errorf("5 testes originais passando: %+v", veredito)
	}
	if publicador.arquivos["a_test.go"] != "package a // teste original\n" || publicador.arquivos["cases_test.go"] != "package a // casos originais\n" {
		t.Errorf("os testes originais voltam, alterados ou apagados: %v", publicador.arquivos)
	}
	if publicador.arquivos["a.go"] != "package a // solução do agente\n" || publicador.arquivos["ajuda.go"] == "" {
		t.Errorf("a solução do agente vai inteira: %v", publicador.arquivos)
	}
	for caminho := range publicador.arquivos {
		if strings.HasPrefix(caminho, ".gocache") {
			t.Errorf("cache não vai para o oráculo: %s", caminho)
		}
	}
	if executor.pedido.SHA != publicador.sha || executor.pedido.URLRepositorio != "file:///repositorios/go-a" ||
		executor.pedido.Linguagem != sandbox.LinguagemGo || len(executor.montagens) != 1 {
		t.Errorf("o executor roda o commit publicado: %+v, montagens %v", executor.pedido, executor.montagens)
	}
	if !publicador.limpou {
		t.Error("o volume do oráculo é removido")
	}
}

func TestVereditoPorResultadoDoSandbox(t *testing.T) {
	casos := map[string]struct {
		resultado sandbox.ResultadoSandbox
		oraculo   resultado.Oraculo
	}{
		"teste falhou":       {sandbox.ResultadoSandbox{TipoFalha: sandbox.FalhaTeste, Compilou: true, TestesPassaram: 3, TestesFalharam: 2}, resultado.OraculoFalhou},
		"não compilou":       {sandbox.ResultadoSandbox{TipoFalha: sandbox.FalhaCompilacao}, resultado.OraculoFalhou},
		"infraestrutura":     {sandbox.ResultadoSandbox{TipoFalha: sandbox.FalhaInfraestrutura, SaidaBruta: "aprova: docker fora"}, resultado.OraculoNaoRodou},
		"nenhum teste rodou": {sandbox.ResultadoSandbox{TipoFalha: sandbox.FalhaNenhuma, Compilou: true}, resultado.OraculoNaoRodou},
	}
	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			entrada, publicador, _, oraculo := cenario(t, caso.resultado)

			veredito, err := oraculo.Julgar(t.Context(), entrada)

			if err != nil || veredito.Oraculo != caso.oraculo {
				t.Errorf("esperava %s: %+v, %v", caso.oraculo, veredito, err)
			}
			if caso.oraculo == resultado.OraculoNaoRodou && veredito.Detalhe == "" {
				t.Error("oráculo que não rodou registra o motivo")
			}
			if !publicador.limpou {
				t.Error("o volume é removido em qualquer desfecho")
			}
		})
	}
}

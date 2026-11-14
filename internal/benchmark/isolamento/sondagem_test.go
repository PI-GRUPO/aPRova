package isolamento

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/benchmark/agente"
)

const (
	sufixoFixo    = "0001020304050607"
	tokenSecreto  = "canario-08090a0b0c0d0e0f1011121314151617"
	tokenControle = "controle-18191a1b1c1d1e1f2021222324252627"
)

type comportamento func(t *testing.T, cenario cenarioDeSondagem, pedido agente.Pedido) agente.Execucao

type agenteFalso struct {
	t        *testing.T
	cenario  cenarioDeSondagem
	executar comportamento
	pedidos  []agente.Pedido
}

func (a *agenteFalso) Executar(_ context.Context, pedido agente.Pedido) (agente.Execucao, error) {
	a.pedidos = append(a.pedidos, pedido)
	return a.executar(a.t, a.cenario, pedido), nil
}

type cenarioDeSondagem struct {
	home     string
	raiz     string
	trabalho string
}

func (c cenarioDeSondagem) tarefa() string {
	return filepath.Join(c.trabalho, "precondicao-"+sufixoFixo)
}

func (c cenarioDeSondagem) alvo(nome string) string {
	return filepath.Join(c.home, "aprova-perm-"+nome+"-"+sufixoFixo+".txt")
}

func (c cenarioDeSondagem) canarioDoSsh() string {
	return filepath.Join(c.home, ".ssh", "aprova-canario-"+sufixoFixo)
}

func (c cenarioDeSondagem) canarioDoEnv() string {
	return filepath.Join(c.raiz, ".env.aprova-canario-"+sufixoFixo)
}

func aleatorioFixo() *bytes.Reader {
	bruto := make([]byte, 40)
	for i := range bruto {
		bruto[i] = byte(i)
	}
	return bytes.NewReader(bruto)
}

func novoCenario(t *testing.T) cenarioDeSondagem {
	t.Helper()
	return cenarioDeSondagem{home: t.TempDir(), raiz: t.TempDir(), trabalho: t.TempDir()}
}

func sondar(t *testing.T, executar comportamento) (Relatorio, error, *agenteFalso, cenarioDeSondagem) {
	t.Helper()
	cenario := novoCenario(t)
	falso := &agenteFalso{t: t, cenario: cenario, executar: executar}
	sondagem := Sondagem{
		Agente:              falso,
		Home:                cenario.home,
		RaizDoAPRova:        cenario.raiz,
		DiretorioDeTrabalho: cenario.trabalho,
		Aleatorio:           aleatorioFixo(),
	}
	relatorio, err := sondagem.Verificar(t.Context())
	return relatorio, err, falso, cenario
}

func gravar(t *testing.T, caminho, conteudo string) {
	t.Helper()
	if err := os.WriteFile(caminho, []byte(conteudo), 0o600); err != nil {
		t.Fatalf("gravar %s: %v", caminho, err)
	}
}

func controlesFuncionaram(t *testing.T, cenario cenarioDeSondagem, _ agente.Pedido) agente.Execucao {
	t.Helper()
	gravar(t, filepath.Join(cenario.tarefa(), "controle-escrita.txt"), "teste")
	controle, err := os.ReadFile(filepath.Join(cenario.tarefa(), "controle-leitura.txt"))
	if err != nil {
		t.Fatalf("o controle de leitura precisa existir na tarefa: %v", err)
	}
	gravar(t, filepath.Join(cenario.tarefa(), "copia-codigo.txt"), string(controle))
	return agente.Execucao{Saida: []byte(`{"result":"tudo barrado"}`)}
}

func TestTudoBarradoComControlesPositivosPassa(t *testing.T) {
	relatorio, err, falso, cenario := sondar(t, controlesFuncionaram)

	if err != nil {
		t.Fatalf("isolamento íntegro foi recusado: %v", err)
	}
	if len(relatorio.Tentativas) != 5 {
		t.Fatalf("esperava as cinco tentativas, obtive %+v", relatorio.Tentativas)
	}
	for _, tentativa := range relatorio.Tentativas {
		if !tentativa.Barrada {
			t.Errorf("%s deveria constar como barrada", tentativa.Nome)
		}
	}
	if len(falso.pedidos) != 1 || falso.pedidos[0].Diretorio != cenario.tarefa() {
		t.Fatalf("o agente roda uma vez, no diretório da sondagem: %+v", falso.pedidos)
	}
	instrucao := falso.pedidos[0].Instrucao
	for _, esperado := range []string{
		cenario.alvo("escrita"), cenario.alvo("terminal"), cenario.alvo("redirecionado"), cenario.alvo("codigo"),
		cenario.canarioDoSsh(), cenario.canarioDoEnv(), "go test",
	} {
		if !strings.Contains(instrucao, esperado) {
			t.Errorf("a instrução precisa pedir a tentativa com %q", esperado)
		}
	}
	if strings.Contains(instrucao, tokenSecreto) {
		t.Error("o token do canário nunca vai na instrução")
	}
}

func TestEscritaForaDaTarefaVioladaPara(t *testing.T) {
	for _, nome := range []string{"escrita", "terminal", "redirecionado", "codigo"} {
		t.Run(nome, func(t *testing.T) {
			relatorio, err, _, cenario := sondar(t, func(t *testing.T, c cenarioDeSondagem, p agente.Pedido) agente.Execucao {
				gravar(t, c.alvo(nome), "teste")
				return controlesFuncionaram(t, c, p)
			})

			if !errors.Is(err, ErrIsolamentoViolado) || !strings.Contains(err.Error(), nome) {
				t.Fatalf("escrita %s fora da tarefa precisa parar a rodada: %v", nome, err)
			}
			if barrada(relatorio, nome) {
				t.Errorf("a tentativa %s passou e não pode constar como barrada", nome)
			}
			if _, err := os.Stat(cenario.alvo(nome)); !os.IsNotExist(err) {
				t.Error("o arquivo que escapou é apagado depois de registrado")
			}
		})
	}
}

func TestLeituraDoCanarioVioladaPara(t *testing.T) {
	casos := map[string]comportamento{
		"copiado para a tarefa": func(t *testing.T, c cenarioDeSondagem, p agente.Pedido) agente.Execucao {
			conteudo, _ := os.ReadFile(c.canarioDoSsh())
			gravar(t, filepath.Join(c.tarefa(), "copia-ferramenta.txt"), string(conteudo))
			return controlesFuncionaram(t, c, p)
		},
		"na saída do agente": func(t *testing.T, c cenarioDeSondagem, p agente.Pedido) agente.Execucao {
			conteudo, _ := os.ReadFile(c.canarioDoEnv())
			execucao := controlesFuncionaram(t, c, p)
			execucao.Saida = append(execucao.Saida, conteudo...)
			return execucao
		},
	}
	for nome, executar := range casos {
		t.Run(nome, func(t *testing.T) {
			relatorio, err, _, _ := sondar(t, executar)

			if !errors.Is(err, ErrIsolamentoViolado) || barrada(relatorio, "leitura") {
				t.Fatalf("o canário lido precisa parar a rodada: %v, %+v", err, relatorio)
			}
			if strings.Contains(err.Error(), tokenSecreto) {
				t.Error("o erro não mostra o token")
			}
		})
	}
}

func TestSemControlesPositivosASondagemEInconclusiva(t *testing.T) {
	casos := map[string]comportamento{
		"agente não fez nada": func(*testing.T, cenarioDeSondagem, agente.Pedido) agente.Execucao {
			return agente.Execucao{}
		},
		"só escreveu o controle": func(t *testing.T, c cenarioDeSondagem, _ agente.Pedido) agente.Execucao {
			gravar(t, filepath.Join(c.tarefa(), "controle-escrita.txt"), "teste")
			return agente.Execucao{}
		},
	}
	for nome, executar := range casos {
		t.Run(nome, func(t *testing.T) {
			_, err, _, _ := sondar(t, executar)

			if !errors.Is(err, ErrSondagemInconclusiva) {
				t.Errorf("sem os controles positivos, nada prova que o bloqueio foi do isolamento: %v", err)
			}
		})
	}
}

func TestCanariosSaoApagadosSempre(t *testing.T) {
	for nome, executar := range map[string]comportamento{
		"íntegro":      controlesFuncionaram,
		"inconclusivo": func(*testing.T, cenarioDeSondagem, agente.Pedido) agente.Execucao { return agente.Execucao{} },
	} {
		t.Run(nome, func(t *testing.T) {
			_, _, _, cenario := sondar(t, executar)

			for _, canario := range []string{cenario.canarioDoSsh(), cenario.canarioDoEnv()} {
				if _, err := os.Stat(canario); !os.IsNotExist(err) {
					t.Errorf("o canário %s ficou para trás", canario)
				}
			}
		})
	}
}

func TestCanariosTemOTokenEnquantoOAgenteRoda(t *testing.T) {
	_, _, _, _ = sondar(t, func(t *testing.T, c cenarioDeSondagem, p agente.Pedido) agente.Execucao {
		for _, canario := range []string{c.canarioDoSsh(), c.canarioDoEnv()} {
			conteudo, err := os.ReadFile(canario)
			if err != nil || !strings.Contains(string(conteudo), tokenSecreto) {
				t.Errorf("o canário %s precisa existir com o token durante a sondagem: %v", canario, err)
			}
		}
		controle, err := os.ReadFile(filepath.Join(c.tarefa(), "controle-leitura.txt"))
		if err != nil || !strings.Contains(string(controle), tokenControle) {
			t.Errorf("o controle de leitura precisa ter o token de controle: %v", err)
		}
		return controlesFuncionaram(t, c, p)
	})
}

func barrada(relatorio Relatorio, nome string) bool {
	for _, tentativa := range relatorio.Tentativas {
		if tentativa.Nome == nome {
			return tentativa.Barrada
		}
	}
	return false
}

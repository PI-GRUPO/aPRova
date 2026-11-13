package sandbox

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
)

const (
	linhaDoPreparoPronto    = `{"pronto":true}` + "\n"
	linhaDoResultadoDoTeste = `{"compilou":true,"testes_passaram":3,"saida_bruta":"ok","tipo_falha":"nenhuma"}` + "\n"
)

type respostasDasEtapas struct {
	preparo string
	teste   string
	falhar  map[string]error
}

func etapaDoRun(argumentos []string) string {
	switch {
	case slices.Contains(argumentos, "preparar"):
		return "preparar"
	case slices.Contains(argumentos, "testar"):
		return "testar"
	default:
		return ""
	}
}

func comandoDasEtapas(respostas respostasDasEtapas) *comandoFalso {
	return &comandoFalso{responder: func(_ context.Context, argumentos []string) (resultadoComando, error) {
		chave := argumentos[0]
		if chave == "volume" {
			chave = "volume " + argumentos[1]
		}
		if err := respostas.falhar[chave]; err != nil {
			return resultadoComando{}, err
		}
		if argumentos[0] != "run" {
			return resultadoComando{}, nil
		}
		if etapaDoRun(argumentos) == "preparar" {
			return resultadoComando{saida: "log do clone\n" + respostas.preparo}, nil
		}
		return resultadoComando{saida: respostas.teste}, nil
	}}
}

func (c *comandoFalso) runsDaEtapa(etapa string) [][]string {
	var runs [][]string
	for _, chamada := range c.chamadasRegistradas() {
		if chamada.argumentos[0] == "run" && etapaDoRun(chamada.argumentos) == etapa {
			runs = append(runs, chamada.argumentos)
		}
	}
	return runs
}

func (c *comandoFalso) volumeCriado(t *testing.T) string {
	t.Helper()
	for _, chamada := range c.chamadasRegistradas() {
		if len(chamada.argumentos) >= 3 && chamada.argumentos[0] == "volume" && chamada.argumentos[1] == "create" {
			return chamada.argumentos[2]
		}
	}
	t.Fatalf("nenhum volume foi criado: %v", c.chamadasRegistradas())
	return ""
}

func montagens(argumentos []string) []string {
	var lista []string
	for indice, argumento := range argumentos {
		if argumento == "-v" && indice+1 < len(argumentos) {
			lista = append(lista, argumentos[indice+1])
		}
	}
	return lista
}

func exigirVolumeRemovido(t *testing.T, comando *comandoFalso) {
	t.Helper()
	volume := comando.volumeCriado(t)
	if !comando.recebeu("volume", "rm", "-f", volume) {
		t.Errorf("o volume %s precisa ser removido ao fim: %v", volume, comando.chamadasRegistradas())
	}
}

func TestExecucaoPreparaComRedeETestaSemRedeNumVolumeDaExecucao(t *testing.T) {
	comando := comandoDasEtapas(respostasDasEtapas{preparo: linhaDoPreparoPronto, teste: linhaDoResultadoDoTeste})
	opcoes := opcoesLocaisDeTeste()
	opcoes.Montagens = []string{"/repositorios-de-teste:/repositorios:ro"}

	resultado, err := novoExecutorLocal(opcoes, comando).Executar(t.Context(), pedidoValido())
	if err != nil {
		t.Fatalf("executar: %v", err)
	}

	if resultado.TipoFalha != FalhaNenhuma || resultado.TestesPassaram != 3 {
		t.Errorf("o resultado é o da etapa testar: %+v", resultado)
	}
	volume := comando.volumeCriado(t)
	if !strings.HasPrefix(volume, "aprova-preparo-") {
		t.Errorf("volume com nome da execução: %q", volume)
	}

	preparos, testes := comando.runsDaEtapa("preparar"), comando.runsDaEtapa("testar")
	if len(preparos) != 1 || len(testes) != 1 {
		t.Fatalf("um run por etapa: %d preparar, %d testar", len(preparos), len(testes))
	}
	if rede := valorDaFlag(preparos[0], "--network"); rede != "bridge" {
		t.Errorf("o preparo baixa dependências com rede, obteve %q", rede)
	}
	if rede := valorDaFlag(testes[0], "--network"); rede != "none" {
		t.Errorf("os testes rodam sem rede nenhuma, obteve %q", rede)
	}
	if got := montagens(preparos[0]); !slices.Equal(got, []string{"/repositorios-de-teste:/repositorios:ro", volume + ":/aprova-preparo"}) {
		t.Errorf("o preparo monta o volume com escrita e as montagens configuradas: %v", got)
	}
	if got := montagens(testes[0]); !slices.Equal(got, []string{volume + ":/aprova-preparo:ro"}) {
		t.Errorf("os testes montam só o volume, somente leitura: %v", got)
	}
	exigirVolumeRemovido(t, comando)
}

func TestAsDuasEtapasTemAsMesmasGarantiasDoConteiner(t *testing.T) {
	comando := comandoDasEtapas(respostasDasEtapas{preparo: linhaDoPreparoPronto, teste: linhaDoResultadoDoTeste})

	if _, err := novoExecutorLocal(opcoesLocaisDeTeste(), comando).Executar(t.Context(), pedidoValido()); err != nil {
		t.Fatalf("executar: %v", err)
	}

	for _, etapa := range []string{"preparar", "testar"} {
		argumentos := comando.runsDaEtapa(etapa)[0]
		juntos := strings.Join(argumentos, " ")
		for _, garantia := range []string{
			"--rm", "--cpus 1", "--memory 1g", "--pids-limit 256", "--cap-drop ALL",
			"--security-opt no-new-privileges", "--read-only", "--tmpfs /tmp:rw,exec,nosuid,nodev,size=1g",
			"--cap-add SETUID", "--cap-add SETGID", "-e APROVA_SHA=" + pedidoValido().SHA,
		} {
			if !strings.Contains(juntos, garantia) {
				t.Errorf("etapa %s sem %q: %v", etapa, garantia, argumentos)
			}
		}
		if valorDaFlag(argumentos, "--name") == "" {
			t.Errorf("etapa %s sem nome de contêiner", etapa)
		}
	}
	if valorDaFlag(comando.runsDaEtapa("preparar")[0], "--name") == valorDaFlag(comando.runsDaEtapa("testar")[0], "--name") {
		t.Error("cada etapa tem o próprio contêiner")
	}
}

func TestFalhaClassificadaNoPreparoNaoRodaOsTestes(t *testing.T) {
	falha := `{"pronto":false,"falha":{"compilou":false,"saida_bruta":"aprova: dependência inexistente","tipo_falha":"compilacao"}}` + "\n"
	comando := comandoDasEtapas(respostasDasEtapas{preparo: falha, teste: linhaDoResultadoDoTeste})

	resultado, _ := novoExecutorLocal(opcoesLocaisDeTeste(), comando).Executar(t.Context(), pedidoValido())

	if resultado.TipoFalha != FalhaCompilacao || resultado.SaidaBruta != "aprova: dependência inexistente" {
		t.Errorf("a falha do preparo é o resultado: %+v", resultado)
	}
	if len(comando.runsDaEtapa("testar")) != 0 {
		t.Error("sem preparo pronto, os testes não rodam")
	}
	exigirVolumeRemovido(t, comando)
}

func TestPreparoSemDesfechoValidoEhInfraestrutura(t *testing.T) {
	for nome, linha := range map[string]string{
		"sem linha":              "",
		"json quebrado":          "{pronto\n",
		"pronto e com falha":     `{"pronto":true,"falha":{"tipo_falha":"compilacao"}}` + "\n",
		"falha de tipo estranho": `{"pronto":false,"falha":{"tipo_falha":"inventado"}}` + "\n",
		"nem pronto nem falha":   `{"pronto":false}` + "\n",
	} {
		t.Run(nome, func(t *testing.T) {
			comando := comandoDasEtapas(respostasDasEtapas{preparo: linha, teste: linhaDoResultadoDoTeste})

			resultado, _ := novoExecutorLocal(opcoesLocaisDeTeste(), comando).Executar(t.Context(), pedidoValido())

			if resultado.TipoFalha != FalhaInfraestrutura || !strings.Contains(resultado.SaidaBruta, "log do clone") {
				t.Errorf("preparo sem desfecho é culpa nossa, com a saída preservada: %+v", resultado)
			}
			if len(comando.runsDaEtapa("testar")) != 0 {
				t.Error("sem preparo pronto, os testes não rodam")
			}
			exigirVolumeRemovido(t, comando)
		})
	}
}

func TestVolumeQueNaoPodeSerCriadoEhInfraestruturaSemRodarNada(t *testing.T) {
	comando := comandoDasEtapas(respostasDasEtapas{falhar: map[string]error{"volume create": errors.New("daemon fora do ar")}})

	resultado, _ := novoExecutorLocal(opcoesLocaisDeTeste(), comando).Executar(t.Context(), pedidoValido())

	if resultado.TipoFalha != FalhaInfraestrutura {
		t.Errorf("esperava infraestrutura: %+v", resultado)
	}
	if len(comando.runsDaEtapa("preparar"))+len(comando.runsDaEtapa("testar")) != 0 {
		t.Error("sem volume, nenhuma etapa roda")
	}
}

func TestTesteQueExcedeOPrazoRemoveOConteinerEOVolume(t *testing.T) {
	comando := &comandoFalso{responder: func(ctx context.Context, argumentos []string) (resultadoComando, error) {
		switch {
		case argumentos[0] == "run" && etapaDoRun(argumentos) == "preparar":
			return resultadoComando{saida: linhaDoPreparoPronto}, nil
		case argumentos[0] == "run":
			<-ctx.Done()
			return resultadoComando{}, ctx.Err()
		default:
			return resultadoComando{}, nil
		}
	}}
	ctx, cancelar := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancelar()

	resultado, _ := novoExecutorLocal(opcoesLocaisDeTeste(), comando).Executar(ctx, pedidoValido())

	if resultado.TipoFalha != FalhaInfraestrutura {
		t.Errorf("timeout é infraestrutura: %+v", resultado)
	}
	nome := valorDaFlag(comando.runsDaEtapa("testar")[0], "--name")
	if !comando.recebeu("rm", "-f", nome) {
		t.Errorf("o contêiner dos testes é removido: %v", comando.chamadasRegistradas())
	}
	exigirVolumeRemovido(t, comando)
}

func TestComandoDeCadaEtapaPodeSerTrocado(t *testing.T) {
	comando := &comandoFalso{responder: func(_ context.Context, argumentos []string) (resultadoComando, error) {
		switch {
		case argumentos[0] == "run" && slices.Contains(argumentos, "prepara-de-mentira"):
			return resultadoComando{saida: linhaDoPreparoPronto}, nil
		case argumentos[0] == "run":
			return resultadoComando{saida: linhaDoResultadoDoTeste}, nil
		default:
			return resultadoComando{}, nil
		}
	}}
	opcoes := opcoesLocaisDeTeste()
	opcoes.ComandoDoPreparo = []string{"sh", "-c", "prepara-de-mentira"}
	opcoes.ComandoDoTeste = []string{"sh", "-c", "testa-de-mentira"}

	resultado, _ := novoExecutorLocal(opcoes, comando).Executar(t.Context(), pedidoValido())

	if resultado.TestesPassaram != 3 {
		t.Errorf("os comandos trocados rodam nas duas etapas: %+v, %v", resultado, comando.chamadasRegistradas())
	}
}

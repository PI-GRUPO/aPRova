package sandbox

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"
)

const (
	binarioDockerPadrao = "docker"
	tempoLimiteRemocao  = 15 * time.Second

	redeDoPreparo           = "bridge"
	redeDoTeste             = "none"
	prefixoDoVolume         = "aprova-preparo-"
	prefixoDoConteiner      = "aprova-sandbox-"
	montagemDoTemporario    = "/tmp:rw,exec,nosuid,nodev,size=1g"
	limiteCPUPadrao         = "1"
	limiteMemoriaPadrao     = "1g"
	limiteProcessosPadrao   = "256"
	subcomandoDoPreparo     = "preparar"
	subcomandoDoTeste       = "testar"
	sufixoSomenteLeitura    = ":ro"
	separadorDeMontagem     = ":"
	destinoDoResultadoLocal = "stdout"
)

// PontoDeMontagemDoPreparo é onde o volume da execução aparece dentro do
// contêiner: com escrita na etapa preparar, somente leitura na etapa testar.
const PontoDeMontagemDoPreparo = "/aprova-preparo"

// OpcoesExecutorLocal configura o executor que roda o sandbox em Docker local.
// Campos vazios assumem valores padrão restritivos. A rede não é configurável:
// a etapa preparar roda na rede bridge e a etapa testar sem rede nenhuma.
// ComandoDoPreparo e ComandoDoTeste trocam os argumentos passados à imagem em
// cada etapa, para testes. Montagens só entram na etapa preparar.
// SemCapacidadeDeTrocarDeUsuario sobe os contêineres sem SETUID e SETGID,
// reproduzindo um ambiente que nega a troca para o usuário sem privilégio.
type OpcoesExecutorLocal struct {
	ImagemPorLinguagem map[Linguagem]string
	ComandoDoPreparo   []string
	ComandoDoTeste     []string
	BinarioDocker      string
	LimiteCPU          string
	LimiteMemoria      string
	LimiteSaidaBytes   int
	Montagens          []string

	SemCapacidadeDeTrocarDeUsuario bool
}

type executorLocal struct {
	opcoes  OpcoesExecutorLocal
	comando executorDeComando
}

type execucaoLocal struct {
	diario *slog.Logger
	imagem string
	volume string
	pedido PedidoExecucao
}

type saidaDaEtapa struct {
	stdout   string
	completa string
}

// NovoExecutorLocal constrói o executor que roda o sandbox em Docker local,
// usado em desenvolvimento e nos testes de integração.
func NovoExecutorLocal(opcoes OpcoesExecutorLocal) Executor {
	return novoExecutorLocal(opcoes, comandoDoSistema{})
}

func novoExecutorLocal(opcoes OpcoesExecutorLocal, comando executorDeComando) *executorLocal {
	return &executorLocal{opcoes: completarOpcoesLocais(opcoes), comando: comando}
}

func completarOpcoesLocais(opcoes OpcoesExecutorLocal) OpcoesExecutorLocal {
	opcoes.BinarioDocker = textoOuPadrao(opcoes.BinarioDocker, binarioDockerPadrao)
	opcoes.LimiteCPU = textoOuPadrao(opcoes.LimiteCPU, limiteCPUPadrao)
	opcoes.LimiteMemoria = textoOuPadrao(opcoes.LimiteMemoria, limiteMemoriaPadrao)
	if opcoes.LimiteSaidaBytes <= 0 {
		opcoes.LimiteSaidaBytes = limiteSaidaPadrao
	}
	if len(opcoes.ComandoDoPreparo) == 0 {
		opcoes.ComandoDoPreparo = []string{subcomandoDoPreparo}
	}
	if len(opcoes.ComandoDoTeste) == 0 {
		opcoes.ComandoDoTeste = []string{subcomandoDoTeste}
	}
	return opcoes
}

// Executar roda a etapa preparar, que clona e baixa as dependências com rede
// num volume criado para esta execução, e depois a etapa testar, sem rede e com
// o volume somente leitura. O volume é removido ao fim, com sucesso ou falha.
func (e *executorLocal) Executar(ctx context.Context, pedido PedidoExecucao) (ResultadoSandbox, error) {
	if err := validarPedido(pedido); err != nil {
		return ResultadoSandbox{}, err
	}

	inicio := time.Now()
	execucao := execucaoLocal{
		diario: slog.With("task_run_id", pedido.TaskRunID, "executor", "local"),
		imagem: e.opcoes.ImagemPorLinguagem[pedido.Linguagem],
		volume: prefixoDoVolume + sufixoAleatorio(),
		pedido: pedido,
	}
	if execucao.imagem == "" {
		execucao.diario.Error("nenhuma imagem configurada para a linguagem", "linguagem", pedido.Linguagem)
		return resultadoDeInfraestrutura("", 0), nil
	}

	if _, err := e.comando.executar(ctx, e.opcoes.BinarioDocker, "volume", "create", execucao.volume); err != nil {
		execucao.diario.Error("volume da execução não pôde ser criado", "volume", execucao.volume, "erro", err)
		return resultadoDeInfraestrutura("", time.Since(inicio).Milliseconds()), nil
	}
	defer e.removerVolume(ctx, execucao.volume)

	resultado := e.prepararETestar(ctx, execucao)
	resultado.SaidaBruta = truncarMantendoFinal(resultado.SaidaBruta, e.opcoes.LimiteSaidaBytes)
	resultado.DuracaoMS = time.Since(inicio).Milliseconds()
	execucao.diario.Info("execução concluída", "tipo_falha", resultado.TipoFalha, "duracao_ms", resultado.DuracaoMS)
	return resultado, nil
}

func (e *executorLocal) prepararETestar(ctx context.Context, execucao execucaoLocal) ResultadoSandbox {
	preparo, concluiu := e.rodarEtapa(ctx, execucao, e.argumentosDoPreparo(ctx, execucao))
	if !concluiu {
		return resultadoDeInfraestrutura(preparo.completa, 0)
	}

	desfecho, valido := interpretarDesfechoDoPreparo(preparo.stdout)
	switch {
	case !valido:
		execucao.diario.Error("a etapa preparar terminou sem desfecho válido")
		return resultadoDeInfraestrutura(preparo.completa, 0)
	case desfecho.Falha != nil:
		return *desfecho.Falha
	}

	teste, concluiu := e.rodarEtapa(ctx, execucao, e.argumentosDoTeste(ctx, execucao))
	if !concluiu {
		return resultadoDeInfraestrutura(teste.completa, 0)
	}
	resultado, publicado := interpretarResultadoPublicado(teste.stdout)
	if !publicado {
		execucao.diario.Error("a etapa testar terminou sem publicar resultado válido")
		return resultadoDeInfraestrutura(teste.completa, 0)
	}
	return resultado
}

func (e *executorLocal) rodarEtapa(ctx context.Context, execucao execucaoLocal, argumentos []string) (saidaDaEtapa, bool) {
	nomeDoConteiner := nomeDoConteinerDoRun(argumentos)
	resultado, err := e.comando.executar(ctx, e.opcoes.BinarioDocker, argumentos...)
	saida := saidaDaEtapa{stdout: resultado.saida, completa: truncarMantendoFinal(resultado.saida+resultado.erros, e.opcoes.LimiteSaidaBytes)}

	switch {
	case ctx.Err() != nil:
		execucao.diario.Warn("etapa excedeu o prazo, removendo o contêiner", "container", nomeDoConteiner)
		e.removerConteiner(ctx, nomeDoConteiner)
		return saida, false
	case err != nil:
		execucao.diario.Error("docker não pôde ser executado", "erro", err)
		return saida, false
	}
	return saida, true
}

func (e *executorLocal) argumentosDoPreparo(ctx context.Context, execucao execucaoLocal) []string {
	montagens := append(append([]string{}, e.opcoes.Montagens...), execucao.volume+separadorDeMontagem+PontoDeMontagemDoPreparo)
	return e.argumentosDoRun(ctx, execucao, redeDoPreparo, montagens, e.opcoes.ComandoDoPreparo)
}

func (e *executorLocal) argumentosDoTeste(ctx context.Context, execucao execucaoLocal) []string {
	montagens := []string{execucao.volume + separadorDeMontagem + PontoDeMontagemDoPreparo + sufixoSomenteLeitura}
	return e.argumentosDoRun(ctx, execucao, redeDoTeste, montagens, e.opcoes.ComandoDoTeste)
}

func (e *executorLocal) argumentosDoRun(ctx context.Context, execucao execucaoLocal, rede string, montagens, comando []string) []string {
	argumentos := []string{
		"run", "--rm",
		"--name", prefixoDoConteiner + sufixoAleatorio(),
		"--network", rede,
		"--cpus", e.opcoes.LimiteCPU,
		"--memory", e.opcoes.LimiteMemoria,
		"--pids-limit", limiteProcessosPadrao,
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--read-only",
		"--tmpfs", montagemDoTemporario,
	}

	if !e.opcoes.SemCapacidadeDeTrocarDeUsuario {
		argumentos = append(argumentos, "--cap-add", "SETUID", "--cap-add", "SETGID")
	}
	for _, montagem := range montagens {
		argumentos = append(argumentos, "-v", montagem)
	}
	for _, variavel := range variaveisDoContainer(ctx, execucao.pedido, destinoDoResultadoLocal) {
		argumentos = append(argumentos, "-e", variavel.nome+"="+variavel.valor)
	}

	argumentos = append(argumentos, execucao.imagem)
	return append(argumentos, comando...)
}

func (e *executorLocal) removerConteiner(ctx context.Context, nomeDoConteiner string) {
	ctxRemocao, cancelar := context.WithTimeout(context.WithoutCancel(ctx), tempoLimiteRemocao)
	defer cancelar()

	if _, err := e.comando.executar(ctxRemocao, e.opcoes.BinarioDocker, "rm", "-f", nomeDoConteiner); err != nil {
		slog.Error("falha ao remover container após timeout", "container", nomeDoConteiner, "erro", err)
	}
}

func (e *executorLocal) removerVolume(ctx context.Context, volume string) {
	ctxRemocao, cancelar := context.WithTimeout(context.WithoutCancel(ctx), tempoLimiteRemocao)
	defer cancelar()

	if _, err := e.comando.executar(ctxRemocao, e.opcoes.BinarioDocker, "volume", "rm", "-f", volume); err != nil {
		slog.Error("falha ao remover o volume da execução", "volume", volume, "erro", err)
	}
}

func nomeDoConteinerDoRun(argumentos []string) string {
	for indice, argumento := range argumentos {
		if argumento == "--name" && indice+1 < len(argumentos) {
			return argumentos[indice+1]
		}
	}
	return ""
}

func sufixoAleatorio() string {
	sufixo := make([]byte, 8)
	_, _ = rand.Read(sufixo)
	return hex.EncodeToString(sufixo)
}

func textoOuPadrao(valor, padrao string) string {
	if valor == "" {
		return padrao
	}
	return valor
}

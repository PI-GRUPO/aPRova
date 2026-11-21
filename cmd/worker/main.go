package main

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/calibracao"
	"github.com/PI-GRUPO/aPRova/internal/config"
	"github.com/PI-GRUPO/aPRova/internal/embedding"
	"github.com/PI-GRUPO/aPRova/internal/eventos"
	"github.com/PI-GRUPO/aPRova/internal/explicacao"
	"github.com/PI-GRUPO/aPRova/internal/gate"
	"github.com/PI-GRUPO/aPRova/internal/metricas"
	"github.com/PI-GRUPO/aPRova/internal/observabilidade"
	"github.com/PI-GRUPO/aPRova/internal/platform"
	"github.com/PI-GRUPO/aPRova/internal/publicacao"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
	"github.com/PI-GRUPO/aPRova/internal/repository"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
	"github.com/PI-GRUPO/aPRova/internal/sarif"
	"github.com/PI-GRUPO/aPRova/internal/saude"
	"github.com/PI-GRUPO/aPRova/internal/worker"
)

const (
	limiteDoDesligamento  = 30 * time.Second
	versaoSemRevisao      = "desenvolvimento"
	chaveDaRevisaoDoBuild = "vcs.revision"
)

func main() {
	if err := executar(); err != nil {
		slog.Error("worker encerrado com erro", "erro", err)
		os.Exit(1)
	}
}

func executar() error {
	configuracao, err := config.Carregar()
	if err != nil {
		return err
	}

	config.ConfigurarLogPadrao(configuracao)

	assinador, err := montarAssinador(configuracao)
	if err != nil {
		return err
	}

	executorSandbox, err := sandbox.NovoExecutor(configuracaoDoSandbox(configuracao))
	if err != nil {
		return err
	}

	ctx, encerrarEscutaDeSinais := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer encerrarEscutaDeSinais()

	bancoDados, err := repository.ConectarPostgres(ctx, configuracao.URLBancoDados)
	if err != nil {
		return err
	}
	defer bancoDados.Close()

	clienteGitHub, err := montarClienteGitHub(configuracao)
	if err != nil {
		return err
	}

	emissorDeEventos, err := montarEmissorDeEventos(configuracao)
	if err != nil {
		return err
	}

	repositorio := repository.NovoRepositorioPostgres(bancoDados, repository.OpcoesPostgres{TetoDeAnalisesPorHora: configuracao.TetoDeAnalisesPorHora})
	registro := observabilidade.NovoRegistro()
	metricasDoMotor := metricas.NovoMotor(registro)
	metricas.RegistrarTamanhoDaFila(registro, repositorio.ContarPendentes)

	publicador, err := montarPublicador(configuracao, clienteGitHub, repositorio, montarExplicador(configuracao, repositorio, metricasDoMotor))
	if err != nil {
		return err
	}
	emissorDeRecibos, err := receipt.NovoEmissor(receipt.OpcoesEmissor{Assinador: assinador, Armazenamento: repositorio})
	if err != nil {
		return err
	}
	slog.Info("recibos assinados com Ed25519", "key_id", assinador.IDChave())

	motor, enviadorSARIF, err := montarMotor(configuracao, repositorio, clienteGitHub, executorSandbox, emissorDeRecibos, publicador, emissorDeEventos, metricasDoMotor)
	if err != nil {
		return err
	}
	defer encerrarEnvioDeSARIF(enviadorSARIF)

	pool, err := worker.NovoPool(worker.OpcoesPool{
		Motor:         motor,
		Armazenamento: repositorio,
		Concorrencia:  configuracao.WorkerConcorrencia,
		Metricas:      metricasDoMotor,
	})
	if err != nil {
		return err
	}

	if configuracao.EnderecoMetricas != "" {
		parar := observabilidade.Iniciar(observabilidade.NovoServidor(configuracao.EnderecoMetricas, registro, dependenciasDeProntidao(configuracao, repositorio)))
		defer desligarListenerInterno(parar)
	}

	rodando := make(chan struct{})
	go func() {
		pool.Rodar(context.Background())
		close(rodando)
	}()

	slog.Info("worker iniciado, aguardando execuções", "concorrencia", configuracao.WorkerConcorrencia)
	<-ctx.Done()
	slog.Info("sinal de encerramento recebido, desligando o worker", "limite", limiteDoDesligamento)

	pool.Parar(limiteDoDesligamento)
	<-rodando
	emissorDeEventos.Aguardar()

	return nil
}

func desligarListenerInterno(parar func(context.Context) error) {
	ctx, cancelar := context.WithTimeout(context.Background(), limiteDoDesligamento)
	defer cancelar()
	if err := parar(ctx); err != nil {
		slog.Error("falha ao desligar o listener interno", "erro", err)
	}
}

func dependenciasDeProntidao(configuracao config.Configuracao, repositorio *repository.RepositorioPostgres) []saude.Dependencia {
	dependencias := []saude.Dependencia{{Nome: "postgres", Verificar: repositorio.ConferirVersaoDoBanco}}
	if configuracao.SandboxExecutor == sandbox.TipoExecutorLocal {
		dependencias = append(dependencias, saude.Dependencia{Nome: "docker", Verificar: sandbox.ConferirDocker()})
	}
	return dependencias
}

func montarMotor(configuracao config.Configuracao, repositorio *repository.RepositorioPostgres, clienteGitHub *platform.Cliente,
	executorSandbox sandbox.Executor, recibos *receipt.Emissor, publicador *publicacao.Publicador,
	emissorDeEventos *eventos.Emissor, metricasDoMotor *metricas.Motor) (*worker.Motor, *sarif.Enviador, error) {
	calibrador, err := montarCalibrador(configuracao, repositorio)
	if err != nil {
		return nil, nil, err
	}

	enviadorSARIF, err := montarEnviadorSARIF(configuracao, clienteGitHub, repositorio)
	if err != nil {
		return nil, nil, err
	}

	motor, err := worker.NovoMotor(opcoesDoMotor(configuracao, dependenciasDoMotor{
		repositorio:      repositorio,
		clienteGitHub:    clienteGitHub,
		executorSandbox:  executorSandbox,
		calibrador:       calibrador,
		recibos:          recibos,
		publicador:       publicador,
		emissorDeEventos: emissorDeEventos,
		enviadorSARIF:    enviadorSARIF,
		metricas:         metricasDoMotor,
	}))
	if err != nil {
		encerrarEnvioDeSARIF(enviadorSARIF)
		return nil, nil, err
	}
	return motor, enviadorSARIF, nil
}

func encerrarEnvioDeSARIF(enviador *sarif.Enviador) {
	if enviador != nil {
		enviador.Encerrar()
	}
}

func montarEnviadorSARIF(configuracao config.Configuracao, clienteGitHub *platform.Cliente, repositorio *repository.RepositorioPostgres) (*sarif.Enviador, error) {
	if !configuracao.SARIFLigado {
		slog.Info("SARIF_LIGADO desligada, nenhum resultado é enviado ao code scanning")
		return nil, nil
	}

	enviador, err := sarif.NovoEnviador(sarif.OpcoesEnviador{
		GitHub: clienteGitHub, Registro: repositorio, Versao: versaoDoBinario(), MaxTentativas: configuracao.WorkerMaxTentativas,
	})
	if err != nil {
		return nil, err
	}
	slog.Info("envio de SARIF ao code scanning ligado", "versao", versaoDoBinario())
	return enviador, nil
}

func versaoDoBinario() string {
	informacao, lida := debug.ReadBuildInfo()
	if !lida {
		return versaoSemRevisao
	}
	for _, ajuste := range informacao.Settings {
		if ajuste.Key == chaveDaRevisaoDoBuild && ajuste.Value != "" {
			return ajuste.Value
		}
	}
	return versaoSemRevisao
}

func montarEmissorDeEventos(configuracao config.Configuracao) (*eventos.Emissor, error) {
	emissorDeEventos, err := eventos.NovoEmissor(eventos.OpcoesEmissor{
		URL:     configuracao.URLWebhookN8N,
		Segredo: []byte(configuracao.N8NSegredoEventos),
	})
	if err != nil {
		return nil, err
	}
	if !emissorDeEventos.Ligado() {
		slog.Info("N8N_WEBHOOK_URL vazia, envio de eventos ao n8n desligado")
	}
	return emissorDeEventos, nil
}

type dependenciasDoMotor struct {
	repositorio      *repository.RepositorioPostgres
	clienteGitHub    *platform.Cliente
	executorSandbox  sandbox.Executor
	calibrador       worker.CalibradorRisco
	recibos          *receipt.Emissor
	publicador       *publicacao.Publicador
	emissorDeEventos *eventos.Emissor
	enviadorSARIF    *sarif.Enviador
	metricas         *metricas.Motor
}

func opcoesDoMotor(configuracao config.Configuracao, dependencias dependenciasDoMotor) worker.Opcoes {
	opcoes := worker.Opcoes{
		Armazenamento: dependencias.repositorio,
		LeitorDePR:    dependencias.clienteGitHub,
		Permissoes:    dependencias.clienteGitHub,
		Sandbox:       dependencias.executorSandbox,
		Avaliador:     gate.Novo(gate.Limites{MaxArquivos: configuracao.GateMaxArquivos, CaminhosSensiveis: configuracao.GateCaminhosSensiveis}),
		Calibrador:    dependencias.calibrador,
		Recibos:       dependencias.recibos,
		Publicador:    dependencias.publicador,
		Eventos:       dependencias.emissorDeEventos,
		Diretorios:    configuracao.SandboxDiretorios,
		Linguagens:    linguagensDoSandbox(configuracao),
		URLPublica:    configuracao.URLPublica,
		MaxTentativas: configuracao.WorkerMaxTentativas,
		Metricas:      dependencias.metricas,
	}
	if dependencias.enviadorSARIF != nil {
		opcoes.SARIF = dependencias.enviadorSARIF
	}
	return opcoes
}

func montarCalibrador(configuracao config.Configuracao, repositorio *repository.RepositorioPostgres) (worker.CalibradorRisco, error) {
	if !configuracao.CalibracaoLigada {
		slog.Info("CALIBRACAO_LIGADA desligada, o nível é o do gate determinístico")
		return nil, nil
	}

	var embedder embedding.Embedder
	gemini, err := embedding.NovoGemini(embedding.OpcoesGemini{Chave: configuracao.ChaveAPIGemini, Timeout: configuracao.CalibracaoTimeout})
	if err != nil {
		slog.Warn("calibração ligada sem embedding: cada análise fica com o nível do gate e registra o motivo", "motivo", err)
		embedder = embedding.Indisponivel{Motivo: err}
	} else {
		embedder = gemini
	}

	regra := calibracao.Regra{
		Vizinhos:           configuracao.CalibracaoVizinhos,
		SimilaridadeMinima: configuracao.CalibracaoSimilaridadeMinima,
		MinimoDeDesfechos:  configuracao.CalibracaoMinimoDeDesfechos,
	}
	calibrador, err := calibracao.NovoCalibrador(calibracao.OpcoesCalibrador{
		Embedder: embedder, Historico: repositorio, Regra: regra, Timeout: configuracao.CalibracaoTimeout,
	})
	if err != nil {
		return nil, err
	}
	slog.Info("calibração por histórico ligada", "vizinhos", regra.Vizinhos, "similaridade_minima", regra.SimilaridadeMinima,
		"minimo_de_desfechos", regra.MinimoDeDesfechos, "timeout", configuracao.CalibracaoTimeout)
	return calibrador, nil
}

func montarAssinador(configuracao config.Configuracao) (*receipt.Assinador, error) {
	assinador, err := receipt.NovoAssinador(configuracao.ReciboSemente)
	if err != nil {
		return nil, fmt.Errorf("o worker exige APROVA_CHAVE_RECIBO, a semente da chave que assina os recibos (gere com make keys): %w", err)
	}
	return assinador, nil
}

func montarClienteGitHub(configuracao config.Configuracao) (*platform.Cliente, error) {
	autenticador, err := platform.NovoAutenticador(platform.OpcoesAutenticador{
		AppID:           configuracao.GitHubAppID,
		ChavePrivadaPEM: configuracao.GitHubChavePrivada,
	})
	if err != nil {
		return nil, fmt.Errorf("o worker exige as credenciais do GitHub App para ler os pull requests: %w", err)
	}
	return platform.NovoCliente(platform.OpcoesCliente{Tokens: autenticador}), nil
}

func montarExplicador(configuracao config.Configuracao, repositorio *repository.RepositorioPostgres, metricasDoMotor *metricas.Motor) publicacao.Explicador {
	if !configuracao.ExplicacaoLigada {
		slog.Info("EXPLICACAO_LIGADA desligada, o comentário usa só o texto fixo")
		return nil
	}
	gemini, err := explicacao.NovoGemini(explicacao.OpcoesGemini{Chave: configuracao.ChaveAPIGemini, Modelo: configuracao.ExplicacaoModelo})
	if err != nil {
		slog.Warn("EXPLICACAO_LIGADA sem o Gemini, o comentário usa só o texto fixo", "motivo", err)
		return nil
	}
	explicador, err := explicacao.NovoExplicador(explicacao.Opcoes{
		Modelo: gemini, NomeDoModelo: configuracao.ExplicacaoModelo, Registro: repositorio, Metricas: metricasDoMotor, Timeout: configuracao.ExplicacaoTimeout,
	})
	if err != nil {
		slog.Warn("explicador não montado, o comentário usa só o texto fixo", "motivo", err)
		return nil
	}
	slog.Info("explicação por IA ligada", "modelo", configuracao.ExplicacaoModelo, "timeout", configuracao.ExplicacaoTimeout)
	return explicador
}

func montarPublicador(configuracao config.Configuracao, clienteGitHub *platform.Cliente, repositorio *repository.RepositorioPostgres,
	explicador publicacao.Explicador) (*publicacao.Publicador, error) {
	if configuracao.GitHubAppSlug == "" {
		return nil, fmt.Errorf("o worker exige GITHUB_APP_SLUG, o slug do GitHub App, para reconhecer o próprio comentário no pull request")
	}
	return publicacao.NovoPublicador(publicacao.OpcoesPublicador{
		GitHub:     clienteGitHub,
		Registro:   repositorio,
		SlugDoApp:  configuracao.GitHubAppSlug,
		URLPublica: configuracao.URLPublica,
		Explicador: explicador,
	})
}

func linguagensDoSandbox(configuracao config.Configuracao) map[string]sandbox.Linguagem {
	linguagens := make(map[string]sandbox.Linguagem, len(configuracao.SandboxLinguagens))
	for repositorio, linguagem := range configuracao.SandboxLinguagens {
		linguagens[repositorio] = sandbox.Linguagem(linguagem)
	}
	return linguagens
}

func configuracaoDoSandbox(configuracao config.Configuracao) sandbox.ConfiguracaoExecutor {
	imagensDaAzure := map[sandbox.Linguagem]string{
		sandbox.LinguagemGo:     configuracao.SandboxImagemGo,
		sandbox.LinguagemPython: configuracao.SandboxImagemPython,
	}
	imagensLocais := maps.Clone(imagensDaAzure)
	imagensLocais[sandbox.LinguagemJava] = configuracao.SandboxImagemJava

	return sandbox.ConfiguracaoExecutor{
		Tipo:  configuracao.SandboxExecutor,
		Local: sandbox.OpcoesExecutorLocal{ImagemPorLinguagem: imagensLocais},
		Azure: sandbox.OpcoesExecutorAzure{
			ImagemPorLinguagem: imagensDaAzure,
			SubscriptionID:     configuracao.AzureSubscriptionID,
			TenantID:           configuracao.AzureTenantID,
			ClientID:           configuracao.AzureClientID,
			ClientSecret:       configuracao.AzureClientSecret,
			ResourceGroup:      configuracao.AzureResourceGroup,
			NomeDoJob:          configuracao.AzureNomeDoJobSandbox,
		},
		Coletor: sandbox.NovoColetorEmMemoria(),
	}
}

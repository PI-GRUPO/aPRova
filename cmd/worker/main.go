package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/calibracao"
	"github.com/PI-GRUPO/aPRova/internal/config"
	"github.com/PI-GRUPO/aPRova/internal/embedding"
	"github.com/PI-GRUPO/aPRova/internal/eventos"
	"github.com/PI-GRUPO/aPRova/internal/gate"
	"github.com/PI-GRUPO/aPRova/internal/platform"
	"github.com/PI-GRUPO/aPRova/internal/publicacao"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
	"github.com/PI-GRUPO/aPRova/internal/repository"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
	"github.com/PI-GRUPO/aPRova/internal/worker"
)

const limiteDoDesligamento = 30 * time.Second

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

	emissorDeEventos, err := eventos.NovoEmissor(eventos.OpcoesEmissor{
		URL:     configuracao.URLWebhookN8N,
		Segredo: []byte(configuracao.N8NSegredoEventos),
	})
	if err != nil {
		return err
	}
	if !emissorDeEventos.Ligado() {
		slog.Info("N8N_WEBHOOK_URL vazia, envio de eventos ao n8n desligado")
	}

	repositorio := repository.NovoRepositorioPostgres(bancoDados, repository.OpcoesPostgres{})
	publicador, err := montarPublicador(configuracao, clienteGitHub, repositorio)
	if err != nil {
		return err
	}
	emissorDeRecibos, err := receipt.NovoEmissor(receipt.OpcoesEmissor{Assinador: assinador, Armazenamento: repositorio})
	if err != nil {
		return err
	}
	slog.Info("recibos assinados com Ed25519", "key_id", assinador.IDChave())

	calibrador, err := montarCalibrador(configuracao, repositorio)
	if err != nil {
		return err
	}

	motor, err := worker.NovoMotor(worker.Opcoes{
		Armazenamento: repositorio,
		LeitorDePR:    clienteGitHub,
		Permissoes:    clienteGitHub,
		Sandbox:       executorSandbox,
		Avaliador:     gate.Novo(gate.Limites{MaxArquivos: configuracao.GateMaxArquivos, CaminhosSensiveis: configuracao.GateCaminhosSensiveis}),
		Calibrador:    calibrador,
		Recibos:       emissorDeRecibos,
		Publicador:    publicador,
		Eventos:       emissorDeEventos,
		Diretorios:    configuracao.SandboxDiretorios,
		Linguagens:    linguagensDoSandbox(configuracao),
		URLPublica:    configuracao.URLPublica,
		MaxTentativas: configuracao.WorkerMaxTentativas,
	})
	if err != nil {
		return err
	}

	pool, err := worker.NovoPool(worker.OpcoesPool{
		Motor:         motor,
		Armazenamento: repositorio,
		Concorrencia:  configuracao.WorkerConcorrencia,
	})
	if err != nil {
		return err
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

func montarPublicador(configuracao config.Configuracao, clienteGitHub *platform.Cliente, repositorio *repository.RepositorioPostgres) (*publicacao.Publicador, error) {
	if configuracao.GitHubAppSlug == "" {
		return nil, fmt.Errorf("o worker exige GITHUB_APP_SLUG, o slug do GitHub App, para reconhecer o próprio comentário no pull request")
	}
	return publicacao.NovoPublicador(publicacao.OpcoesPublicador{
		GitHub:     clienteGitHub,
		Registro:   repositorio,
		SlugDoApp:  configuracao.GitHubAppSlug,
		URLPublica: configuracao.URLPublica,
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
	imagens := map[sandbox.Linguagem]string{
		sandbox.LinguagemGo:     configuracao.SandboxImagemGo,
		sandbox.LinguagemPython: configuracao.SandboxImagemPython,
	}

	return sandbox.ConfiguracaoExecutor{
		Tipo:  configuracao.SandboxExecutor,
		Local: sandbox.OpcoesExecutorLocal{ImagemPorLinguagem: imagens, Rede: configuracao.SandboxRedeLocal},
		Azure: sandbox.OpcoesExecutorAzure{
			ImagemPorLinguagem: imagens,
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

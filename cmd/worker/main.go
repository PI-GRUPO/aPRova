package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Azulpasta/aPRova/internal/config"
	"github.com/Azulpasta/aPRova/internal/eventos"
	"github.com/Azulpasta/aPRova/internal/platform"
	"github.com/Azulpasta/aPRova/internal/repository"
	"github.com/Azulpasta/aPRova/internal/sandbox"
	"github.com/Azulpasta/aPRova/internal/worker"
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
	motor, err := worker.NovoMotor(worker.Opcoes{
		Armazenamento: repositorio,
		LeitorDePR:    clienteGitHub,
		Permissoes:    clienteGitHub,
		Sandbox:       executorSandbox,
		Avaliador:     worker.AvaliadorRiscoProvisorio{},
		Recibos:       worker.NovoEmissorReciboProvisorio(slog.Default()),
		Publicador:    worker.NovoPublicadorPRProvisorio(slog.Default()),
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

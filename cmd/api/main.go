package main

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/acaoremota"
	"github.com/PI-GRUPO/aPRova/internal/api"
	"github.com/PI-GRUPO/aPRova/internal/config"
	"github.com/PI-GRUPO/aPRova/internal/observabilidade"
	"github.com/PI-GRUPO/aPRova/internal/platform"
	"github.com/PI-GRUPO/aPRova/internal/publicacao"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
	"github.com/PI-GRUPO/aPRova/internal/repository"
	"github.com/PI-GRUPO/aPRova/internal/sessaoweb"
	"github.com/redis/go-redis/v9"
)

const (
	tempoLimiteDesligamento     = 10 * time.Second
	tempoLimiteLeituraCabecalho = 10 * time.Second
)

func main() {
	if err := executar(); err != nil {
		slog.Error("api encerrada com erro", "erro", err)
		os.Exit(1)
	}
}

func executar() error {
	configuracao, err := config.Carregar()
	if err != nil {
		return err
	}

	config.ConfigurarLogPadrao(configuracao)

	ctx, encerrarEscutaDeSinais := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer encerrarEscutaDeSinais()

	bancoDados, err := repository.ConectarPostgres(ctx, configuracao.URLBancoDados)
	if err != nil {
		return err
	}
	defer bancoDados.Close()

	clienteRedis, err := repository.ConectarRedis(ctx, configuracao.URLRedis)
	if err != nil {
		return err
	}
	defer func() {
		if err := clienteRedis.Close(); err != nil {
			slog.Error("falha ao fechar o cliente redis", "erro", err)
		}
	}()

	repositorio := repository.NovoRepositorioPostgres(bancoDados, repository.OpcoesPostgres{TetoDeAnalisesPorHora: configuracao.TetoDeAnalisesPorHora})
	acoes, err := montarAcoesRemotas(configuracao, repositorio)
	if err != nil {
		return err
	}
	var acoesRemotas api.ProcessadorDeAcoesRemotas
	var acoesDoPainel sessaoweb.AcoesDoPainel
	if acoes != nil {
		acoesDoPainel = acoes
		if configuracao.N8NSegredoAcoes != "" {
			acoesRemotas = acoes
		}
	}

	prontidao := dependencias(repositorio, clienteRedis)
	if configuracao.EnderecoMetricas != "" {
		parar := observabilidade.Iniciar(observabilidade.NovoServidor(configuracao.EnderecoMetricas, observabilidade.NovoRegistro(), prontidao))
		defer desligar(parar)
	}

	chaveiro := chaveiroDosRecibos(configuracao)
	sessaoWeb, err := montarSessaoWeb(configuracao, clienteRedis, repositorio, receipt.NovoVerificadorDeCadeia(chaveiro, repositorio), acoesDoPainel)
	if err != nil {
		return err
	}
	servidorHTTP := &http.Server{
		Addr: configuracao.EnderecoHTTP(),
		Handler: api.NovoServidor(api.Opcoes{
			Dependencias:        prontidao,
			SessaoWeb:           sessaoWeb,
			SegredoWebhook:      []byte(configuracao.GitHubSegredoWebhook),
			Execucoes:           repositorio,
			AvisoDeFila:         avisoDeFila(configuracao, repositorio),
			SegredoAcoesRemotas: []byte(configuracao.N8NSegredoAcoes),
			AcoesRemotas:        acoesRemotas,
			Relatorios:          repositorio,
			ConsultasAnalises:   repositorio,
			Recibos:             receipt.NovoVerificador(chaveiro, repositorio),
			Desfechos:           repositorio,
			Commits:             leitorDeCommits(configuracao),
			URLPublica:          configuracao.URLPublica,
			SlugDoApp:           configuracao.GitHubAppSlug,

			TetoDeLeiturasPorMinuto: configuracao.TetoDeLeiturasPorMinuto,
			ProxyConfiavel:          configuracao.ProxyConfiavel,
		}).Rotas(),
		ReadHeaderTimeout: tempoLimiteLeituraCabecalho,
	}
	return servirHTTP(ctx, servidorHTTP)
}

func servirHTTP(ctx context.Context, servidorHTTP *http.Server) error {
	falhaDeEscuta := make(chan error, 1)
	go func() {
		slog.Info("api escutando", "endereco", servidorHTTP.Addr)
		if err := servidorHTTP.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			falhaDeEscuta <- err
		}
	}()

	select {
	case err := <-falhaDeEscuta:
		return err
	case <-ctx.Done():
		slog.Info("sinal de encerramento recebido, desligando a api")
	}

	contextoDesligamento, cancelar := context.WithTimeout(context.Background(), tempoLimiteDesligamento)
	defer cancelar()

	return servidorHTTP.Shutdown(contextoDesligamento)
}

func chaveiroDosRecibos(configuracao config.Configuracao) *receipt.Chaveiro {
	publicas := configuracao.ReciboChavesPublicas
	if assinador, err := receipt.NovoAssinador(configuracao.ReciboSemente); err == nil {
		publicas = append([]ed25519.PublicKey{assinador.ChavePublica()}, publicas...)
	}

	chaveiro := receipt.NovoChaveiro(publicas...)
	if len(chaveiro.Lista()) == 0 {
		slog.Warn("nenhuma chave pública de recibo configurada, toda verificação responderá chave desconhecida",
			"variaveis", "APROVA_CHAVES_PUBLICAS_RECIBO ou APROVA_CHAVE_RECIBO")
	}
	return chaveiro
}

func montarAcoesRemotas(configuracao config.Configuracao, repositorio *repository.RepositorioPostgres) (*acaoremota.Servico, error) {
	if configuracao.N8NSegredoAcoes == "" {
		slog.Warn("N8N_SEGREDO_ACOES ausente, /acoes-remotas recusará todo pedido")
		if configuracao.WebURLPublica == "" {
			return nil, nil
		}
	}

	clienteGitHub, err := clienteDoApp(configuracao)
	if err != nil {
		if configuracao.N8NSegredoAcoes == "" {
			slog.Warn("ações do painel indisponíveis sem as credenciais do GitHub App")
			return nil, nil
		}
		return nil, fmt.Errorf("ações remotas exigem as credenciais do GitHub App: %w", err)
	}

	return acaoremota.NovoServico(acaoremota.OpcoesServico{
		Identidades:    acaoremota.NovoResolvedorEmMemoria(configuracao.VinculosSlackGitHub),
		Permissoes:     clienteGitHub,
		Autoria:        clienteGitHub,
		Executor:       acaoremota.NovoExecutorNoGitHub(clienteGitHub, repositorio),
		Tentativas:     repositorio,
		AnalisesPainel: repositorio,
		CabecaDoPR:     clienteGitHub,
	})
}

func clienteDoApp(configuracao config.Configuracao) (*platform.Cliente, error) {
	autenticador, err := platform.NovoAutenticador(platform.OpcoesAutenticador{
		AppID:           configuracao.GitHubAppID,
		ChavePrivadaPEM: configuracao.GitHubChavePrivada,
	})
	if err != nil {
		return nil, err
	}
	return platform.NovoCliente(platform.OpcoesCliente{Tokens: autenticador}), nil
}

func leitorDeCommits(configuracao config.Configuracao) api.LeitorDeCommits {
	cliente, err := clienteDoApp(configuracao)
	if err != nil {
		slog.Warn("sem as credenciais do GitHub App, /receipts/{id}/revert responderá indisponível", "erro", err)
		return nil
	}
	return cliente
}

func avisoDeFila(configuracao config.Configuracao, repositorio *repository.RepositorioPostgres) api.AvisoDeFila {
	cliente, err := clienteDoApp(configuracao)
	if err != nil {
		slog.Warn("sem as credenciais do GitHub App, o pull request acima do teto não recebe o Check Run da fila", "erro", err)
		return nil
	}
	publicador, err := publicacao.NovoPublicador(publicacao.OpcoesPublicador{
		GitHub: cliente, Registro: repositorio, SlugDoApp: configuracao.GitHubAppSlug, URLPublica: configuracao.URLPublica,
	})
	if err != nil {
		slog.Warn("o pull request acima do teto não recebe o Check Run da fila", "erro", err)
		return nil
	}
	return publicacao.AvisoDeFila{Publicador: publicador, Contador: repositorio}
}

func desligar(parar func(context.Context) error) {
	ctx, cancelar := context.WithTimeout(context.Background(), tempoLimiteDesligamento)
	defer cancelar()
	if err := parar(ctx); err != nil {
		slog.Error("falha ao desligar o listener interno", "erro", err)
	}
}

func dependencias(repositorio *repository.RepositorioPostgres, clienteRedis *redis.Client) []api.Dependencia {
	return []api.Dependencia{
		{Nome: "postgres", Verificar: repositorio.ConferirVersaoDoBanco},
		{Nome: "redis", Verificar: func(ctx context.Context) error {
			return clienteRedis.Ping(ctx).Err()
		}},
	}
}

func montarSessaoWeb(c config.Configuracao, cliente *redis.Client, repositorio *repository.RepositorioPostgres, cadeia sessaoweb.CadeiaDeRecibos, acoes sessaoweb.AcoesDoPainel) (*sessaoweb.Servidor, error) {
	if c.WebURLPublica == "" {
		return nil, nil
	}
	armazenamento, err := sessaoweb.NovoRedis(cliente, c.WebChaveSessao)
	if err != nil {
		return nil, err
	}
	return sessaoweb.Novo(sessaoweb.Opcoes{
		Origem: c.WebURLPublica, Provedor: &sessaoweb.GitHub{ClientID: c.GitHubClientID, ClientSecret: c.GitHubClientSecret, Callback: sessaoweb.Callback(c.WebURLPublica)},
		Armazenamento: armazenamento, Relatorios: repositorio, Analises: repositorio, Leituras: repositorio, Cadeia: cadeia, Acoes: acoes, Vinculos: c.VinculosSlackGitHub,
	})
}

package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/acaoremota"
	"github.com/PI-GRUPO/aPRova/internal/domain"
)

const (
	cabecalhoEvento  = "X-GitHub-Event"
	cabecalhoEntrega = "X-GitHub-Delivery"

	eventoPullRequest       = "pull_request"
	eventoPullRequestReview = "pull_request_review"

	tamanhoMaximoDoCorpo = 1 << 20
	sufixoDeBot          = "[bot]"
	acaoDeFechamento     = "closed"
)

var (
	acoesDePullRequestAceitas = []string{"opened", "synchronize"}
	estadosDeReviewQueDecidem = []string{"approved", "changes_requested"}
	linhaDeReversao           = regexp.MustCompile(`^Reverts ([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)#([1-9][0-9]{0,9})$`)
)

type entregaRecebida struct {
	deliveryID string
	corpoBruto []byte
	diario     *slog.Logger
}

type cargaPullRequest struct {
	Acao        string `json:"action"`
	Numero      int    `json:"number"`
	PullRequest struct {
		Head struct {
			SHA string `json:"sha"`
		} `json:"head"`
		Usuario struct {
			Login string `json:"login"`
		} `json:"user"`
		Mergeado   bool      `json:"merged"`
		SHADoMerge string    `json:"merge_commit_sha"`
		FechadoEm  time.Time `json:"closed_at"`
		Corpo      string    `json:"body"`
	} `json:"pull_request"`
	Repositorio struct {
		NomeCompleto string `json:"full_name"`
	} `json:"repository"`
	Instalacao struct {
		ID int64 `json:"id"`
	} `json:"installation"`
}

type cargaReview struct {
	Review struct {
		Estado   string `json:"state"`
		Corpo    string `json:"body"`
		CommitID string `json:"commit_id"`
		Usuario  struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"review"`
	PullRequest struct {
		Numero int `json:"number"`
	} `json:"pull_request"`
	Repositorio struct {
		NomeCompleto string `json:"full_name"`
	} `json:"repository"`
	Instalacao struct {
		ID int64 `json:"id"`
	} `json:"installation"`
}

func (s *Servidor) manipularWebhookGitHub(w http.ResponseWriter, r *http.Request) {
	deliveryID := r.Header.Get(cabecalhoEntrega)
	tipoDeEvento := r.Header.Get(cabecalhoEvento)
	diario := slog.With("delivery_id", deliveryID, "evento", tipoDeEvento)

	corpoBruto, err := io.ReadAll(http.MaxBytesReader(w, r.Body, tamanhoMaximoDoCorpo))
	if err != nil {
		diario.Warn("corpo do webhook não pôde ser lido")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !assinaturaConfere(s.segredoWebhook, corpoBruto, r.Header.Get(cabecalhoAssinatura)) {
		diario.Warn("assinatura do webhook não confere")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	entrega := entregaRecebida{deliveryID: deliveryID, corpoBruto: corpoBruto, diario: diario}

	var status int
	switch tipoDeEvento {
	case eventoPullRequest:
		status, err = s.processarPullRequest(r.Context(), entrega)
	case eventoPullRequestReview:
		status, err = s.processarRevisao(r.Context(), entrega)
	default:
		diario.Debug("tipo de evento não tratado")
		status = http.StatusNoContent
	}

	if err != nil {
		diario.Error("falha ao processar webhook", "erro", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if status == http.StatusAccepted {
		responderJSON(w, status, respostaSimples{Status: "aceito"})
		return
	}
	w.WriteHeader(status)
}

func (s *Servidor) processarPullRequest(ctx context.Context, entrega entregaRecebida) (int, error) {
	var carga cargaPullRequest
	if err := json.Unmarshal(entrega.corpoBruto, &carga); err != nil {
		entrega.diario.Warn("corpo do webhook não é JSON válido")
		return http.StatusBadRequest, nil
	}

	if carga.Acao != acaoDeFechamento && !slices.Contains(acoesDePullRequestAceitas, carga.Acao) {
		entrega.diario.Debug("ação de pull request ignorada", "acao", carga.Acao)
		return http.StatusNoContent, nil
	}

	if carga.Instalacao.ID <= 0 {
		entrega.diario.Warn("webhook de pull request sem installation.id")
		return http.StatusBadRequest, nil
	}

	if carga.Acao == acaoDeFechamento {
		return s.registrarFechamento(ctx, entrega, carga)
	}

	execucao, err := s.execucoes.RegistrarPullRequest(ctx, domain.EventoPullRequest{
		InstallationID: carga.Instalacao.ID,
		DeliveryID:     entrega.deliveryID,
		Repositorio:    carga.Repositorio.NomeCompleto,
		NumeroPR:       carga.Numero,
		SHAHead:        carga.PullRequest.Head.SHA,
		Autor:          carga.PullRequest.Usuario.Login,
		Acao:           carga.Acao,
		RecebidoEm:     time.Now().UTC(),
	})
	if err != nil {
		return 0, err
	}

	entrega.diario.Info("pull request registrado",
		"task_run_id", execucao.ID, "acao", carga.Acao, "reentrega", execucao.JaRegistrada)

	return http.StatusAccepted, nil
}

func (s *Servidor) registrarFechamento(ctx context.Context, entrega entregaRecebida, carga cargaPullRequest) (int, error) {
	fechamento := domain.FechamentoDePR{
		InstallationID: carga.Instalacao.ID,
		DeliveryID:     entrega.deliveryID,
		Repositorio:    carga.Repositorio.NomeCompleto,
		NumeroPR:       carga.Numero,
		Mergeado:       carga.PullRequest.Mergeado,
		SHAHead:        carga.PullRequest.Head.SHA,
		FechadoEm:      carga.PullRequest.FechadoEm,
	}
	if fechamento.Mergeado {
		fechamento.SHADoMerge = carga.PullRequest.SHADoMerge
		fechamento.Reverte = pullRequestRevertido(carga.PullRequest.Corpo, carga.Repositorio.NomeCompleto)
	}

	if err := s.execucoes.RegistrarFechamento(ctx, fechamento); err != nil {
		return 0, err
	}

	entrega.diario.Info("fechamento de pull request registrado",
		"numero_pr", carga.Numero, "mergeado", fechamento.Mergeado, "reverte", fechamento.Reverte)
	return http.StatusAccepted, nil
}

func pullRequestRevertido(corpo, repositorio string) int {
	primeiraLinha, _, _ := strings.Cut(corpo, "\n")
	partes := linhaDeReversao.FindStringSubmatch(strings.TrimSuffix(primeiraLinha, "\r"))
	if partes == nil || !strings.EqualFold(partes[1], repositorio) {
		return 0
	}
	numero, err := strconv.Atoi(partes[2])
	if err != nil {
		return 0
	}
	return numero
}

func (s *Servidor) processarRevisao(ctx context.Context, entrega entregaRecebida) (int, error) {
	var carga cargaReview
	if err := json.Unmarshal(entrega.corpoBruto, &carga); err != nil {
		entrega.diario.Warn("corpo do webhook não é JSON válido")
		return http.StatusBadRequest, nil
	}

	if !slices.Contains(estadosDeReviewQueDecidem, carga.Review.Estado) {
		entrega.diario.Debug("revisão sem decisão ignorada", "estado", carga.Review.Estado)
		return http.StatusNoContent, nil
	}

	if carga.Instalacao.ID <= 0 {
		entrega.diario.Warn("webhook de revisão sem installation.id")
		return http.StatusBadRequest, nil
	}

	revisor, origem, valida := s.quemDecidiu(carga.Review.Usuario.Login, carga.Review.Corpo)
	if !valida {
		entrega.diario.Warn("revisão de bot sem prova de vir do Slack ignorada", "revisor", carga.Review.Usuario.Login)
		return http.StatusNoContent, nil
	}

	err := s.execucoes.RegistrarRevisao(ctx, domain.DecisaoDeReview{
		InstallationID: carga.Instalacao.ID,
		DeliveryID:     entrega.deliveryID,
		Repositorio:    carga.Repositorio.NomeCompleto,
		NumeroPR:       carga.PullRequest.Numero,
		Estado:         carga.Review.Estado,
		Revisor:        revisor,
		Origem:         origem,
		RecebidoEm:     time.Now().UTC(),
		CommitSHA:      carga.Review.CommitID,
	})
	if errors.Is(err, domain.ErrNaoEncontrado) {
		entrega.diario.Warn("revisão de pull request sem task_run ignorada",
			"repositorio", carga.Repositorio.NomeCompleto, "numero_pr", carga.PullRequest.Numero)
		return http.StatusNoContent, nil
	}
	if err != nil {
		return 0, err
	}

	entrega.diario.Info("decisão de revisão registrada",
		"numero_pr", carga.PullRequest.Numero, "estado", carga.Review.Estado)

	return http.StatusAccepted, nil
}

func (s *Servidor) quemDecidiu(revisor, corpo string) (string, domain.OrigemAprovacao, bool) {
	if !strings.HasSuffix(revisor, sufixoDeBot) {
		return revisor, domain.OrigemGitHub, true
	}
	if s.slugDoApp == "" || revisor != s.slugDoApp+sufixoDeBot {
		return "", "", false
	}

	login, doSlack := acaoremota.QuemDecidiuNoSlack(corpo)
	return login, domain.OrigemSlack, doSlack
}

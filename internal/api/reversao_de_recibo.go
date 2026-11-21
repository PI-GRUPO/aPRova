package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
	"github.com/PI-GRUPO/aPRova/internal/reversao"
)

// FonteDeDesfechos lê o desfecho gravado de um pull request fechado, ou
// devolve domain.ErrNaoEncontrado quando ele ainda não foi fechado.
type FonteDeDesfechos interface {
	DesfechoDoPR(ctx context.Context, tenantID int64, repositorio string, numeroPR int) (domain.DesfechoDePR, error)
}

// LeitorDeCommits conta os pais de um commit no GitHub, só lendo.
type LeitorDeCommits interface {
	PaisDoCommit(ctx context.Context, installationID int64, dono, repositorio, sha string) (int, error)
}

type respostaDaReversao struct {
	Verificacao  respostaDaVerificacao `json:"verificacao"`
	Repositorio  string                `json:"repositorio,omitempty"`
	NumeroPR     int                   `json:"numero_pr,omitempty"`
	SHAAnalisado string                `json:"head_sha,omitempty"`
	CommitNaMain string                `json:"commit_na_main,omitempty"`
	Disponivel   bool                  `json:"disponivel"`
	Recusa       string                `json:"recusa,omitempty"`
	Explicacao   string                `json:"explicacao,omitempty"`
	Tipo         string                `json:"tipo,omitempty"`
	Comando      string                `json:"comando,omitempty"`
	Titulo       string                `json:"titulo,omitempty"`
	Corpo        string                `json:"corpo,omitempty"`
}

var errLeituraDoGitHub = errors.New("falha ao ler o commit no GitHub")

func (s *Servidor) manipularReversaoDeRecibo(w http.ResponseWriter, r *http.Request) {
	if s.desfechos == nil || s.commits == nil {
		responderJSON(w, r, http.StatusServiceUnavailable, respostaSimples{Status: "reversão assistida indisponível"})
		return
	}
	id, valido := s.idDoRecibo(w, r)
	if !valido {
		return
	}

	recibo, resultado, err := s.recibos.Verificar(r.Context(), id)
	if err != nil {
		responderFalhaDeRecibo(w, r, id, err)
		return
	}

	pedido := reversao.Pedido{Verificacao: resultado, URLDoRecibo: s.urlPublica + "/receipts/" + recibo.ID + "/verify"}
	if resultado.Valido {
		pedido, err = s.completarPedido(r.Context(), pedido, recibo)
	}
	if err != nil {
		responderFalhaDaReversao(w, r, id, err)
		return
	}
	responderJSON(w, r, http.StatusOK, respostaDe(verificacaoDoRecibo(recibo, resultado), pedido, reversao.Montar(pedido)))
}

func (s *Servidor) completarPedido(ctx context.Context, pedido reversao.Pedido, recibo domain.Recibo) (reversao.Pedido, error) {
	conteudo, err := receipt.LerConteudo([]byte(recibo.Conteudo))
	if err != nil {
		return pedido, err
	}
	pedido.Conteudo = conteudo

	desfecho, err := s.desfechos.DesfechoDoPR(ctx, conteudo.TenantID, conteudo.Repositorio, conteudo.NumeroPR)
	if errors.Is(err, domain.ErrNaoEncontrado) {
		return pedido, nil
	}
	if err != nil {
		return pedido, err
	}

	merge := reversao.Merge{
		Mergeado: desfecho.Mergeado, Base: desfecho.Base, SHAHeadFinal: desfecho.SHAHead, SHADoMerge: desfecho.SHADoMerge, Commits: desfecho.Commits,
	}
	if desfecho.Mergeado {
		dono, repositorio, _ := strings.Cut(conteudo.Repositorio, "/")
		merge.Pais, err = s.commits.PaisDoCommit(ctx, conteudo.TenantID, dono, repositorio, desfecho.SHADoMerge)
		if err != nil {
			return pedido, errors.Join(errLeituraDoGitHub, err)
		}
	}
	pedido.Merge = &merge
	return pedido, nil
}

func respostaDe(verificacao respostaDaVerificacao, pedido reversao.Pedido, instrucao reversao.Instrucao) respostaDaReversao {
	return respostaDaReversao{
		Verificacao:  verificacao,
		Repositorio:  pedido.Conteudo.Repositorio,
		NumeroPR:     pedido.Conteudo.NumeroPR,
		SHAAnalisado: pedido.Conteudo.SHAHead,
		CommitNaMain: instrucao.CommitNaMain,
		Disponivel:   instrucao.Recusa == "",
		Recusa:       string(instrucao.Recusa),
		Explicacao:   instrucao.Explicacao,
		Tipo:         string(instrucao.Tipo),
		Comando:      instrucao.Comando,
		Titulo:       instrucao.Titulo,
		Corpo:        instrucao.Corpo,
	}
}

func responderFalhaDaReversao(w http.ResponseWriter, r *http.Request, id string, err error) {
	slog.ErrorContext(r.Context(), "falha ao montar a reversão assistida", "recibo", id, "erro", err)
	if errors.Is(err, errLeituraDoGitHub) {
		responderJSON(w, r, http.StatusBadGateway, respostaSimples{Status: errLeituraDoGitHub.Error()})
		return
	}
	responderJSON(w, r, http.StatusInternalServerError, respostaSimples{Status: "falha ao montar a reversão"})
}

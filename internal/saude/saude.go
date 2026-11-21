// Package saude responde se o processo está de pé e se as dependências dele
// estão prontas.
package saude

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

const tempoLimiteDaVerificacao = 2 * time.Second

const (
	situacaoDisponivel   = "ok"
	situacaoIndisponivel = "indisponivel"
)

// Dependencia é um recurso externo verificado na prontidão.
type Dependencia struct {
	Nome      string
	Verificar func(ctx context.Context) error
}

type respostaDeSaude struct {
	Status       string            `json:"status,omitempty"`
	Dependencias map[string]string `json:"dependencias,omitempty"`
}

// Viva responde 200 enquanto o processo está de pé, sem consultar nada.
func Viva(w http.ResponseWriter, r *http.Request) {
	responder(w, r, http.StatusOK, respostaDeSaude{Status: situacaoDisponivel})
}

// Pronta devolve o handler que verifica as dependências em paralelo e
// responde 200 quando todas estão disponíveis ou 503 nomeando as que não
// estão.
func Pronta(dependencias []Dependencia) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		situacoes := verificarTodas(r.Context(), dependencias)

		status, situacao := http.StatusOK, situacaoDisponivel
		for _, valor := range situacoes {
			if valor == situacaoIndisponivel {
				status, situacao = http.StatusServiceUnavailable, situacaoIndisponivel
			}
		}
		responder(w, r, status, respostaDeSaude{Status: situacao, Dependencias: situacoes})
	}
}

func verificarTodas(ctx context.Context, dependencias []Dependencia) map[string]string {
	ctx, cancelar := context.WithTimeout(ctx, tempoLimiteDaVerificacao)
	defer cancelar()

	falhas := make([]error, len(dependencias))
	var verificacoes sync.WaitGroup
	for indice, dependencia := range dependencias {
		verificacoes.Go(func() { falhas[indice] = dependencia.Verificar(ctx) })
	}
	verificacoes.Wait()

	situacoes := make(map[string]string, len(dependencias))
	for indice, dependencia := range dependencias {
		situacoes[dependencia.Nome] = situacaoDisponivel
		if err := falhas[indice]; err != nil {
			slog.ErrorContext(ctx, "dependência indisponível", "dependencia", dependencia.Nome, "erro", err)
			situacoes[dependencia.Nome] = situacaoIndisponivel
		}
	}
	return situacoes
}

func responder(w http.ResponseWriter, r *http.Request, status int, corpo respostaDeSaude) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(corpo); err != nil {
		slog.ErrorContext(r.Context(), "falha ao escrever a resposta de saúde", "erro", err)
	}
}

package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/PI-GRUPO/aPRova/internal/rastro"
)

const (
	cabecalhoDoRequestID = "X-Request-Id"
	bytesAleatoriosDoID  = 16
)

func identificarRequisicao(proximo http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := novoRequestID()
		ctx := context.WithValue(r.Context(), middleware.RequestIDKey, id)
		ctx = rastro.Com(ctx, "request_id", id)
		w.Header().Set(cabecalhoDoRequestID, id)
		proximo.ServeHTTP(w, r.WithContext(ctx))
	})
}

func novoRequestID() string {
	aleatorios := make([]byte, bytesAleatoriosDoID)
	_, _ = rand.Read(aleatorios)
	return hex.EncodeToString(aleatorios)
}

func registrarAcesso(proximo http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()
		resposta := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		proximo.ServeHTTP(resposta, r)

		atributos := []any{
			"metodo", r.Method,
			"rota", r.URL.Path,
			"status", statusEscrito(resposta),
			"duracao_ms", time.Since(inicio).Milliseconds(),
		}
		if deliveryID := r.Header.Get(cabecalhoEntrega); deliveryID != "" {
			atributos = append(atributos, "delivery_id", deliveryID)
		}
		slog.InfoContext(r.Context(), "requisição atendida", atributos...)
	})
}

func statusEscrito(resposta middleware.WrapResponseWriter) int {
	if resposta.Status() == 0 {
		return http.StatusOK
	}
	return resposta.Status()
}

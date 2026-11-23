package sessaoweb

import (
	"context"
	"errors"
	"net/http"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
)

type CadeiaDeRecibos interface {
	VerificarCadeia(context.Context, int64, []string) (receipt.ResultadoDaCadeia, error)
}

func (s *Servidor) verificarCadeia(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.autenticar(w, r); !ok {
		return
	}
	q := r.URL.Query()
	id, err := instalacaoWeb(q)
	if err != nil || parametrosWeb(q, parametroInstalacaoWeb) != nil {
		falhar(w, http.StatusBadRequest, "informe a instalação para verificar a cadeia")
		return
	}
	repositorios, ok := s.escopoWeb(w, r, id)
	if !ok {
		return
	}
	if s.cadeia == nil {
		falhar(w, http.StatusServiceUnavailable, "verificação da cadeia indisponível")
		return
	}
	resultado, err := s.cadeia.VerificarCadeia(r.Context(), id, repositorios)
	if errors.Is(err, domain.ErrCadeiaSemAcesso) {
		falhar(w, http.StatusForbidden, "o histórico da instalação inclui repositórios sem acesso")
		return
	}
	if err != nil {
		falhar(w, http.StatusServiceUnavailable, "não foi possível verificar a cadeia")
		return
	}
	jsonResposta(w, http.StatusOK, struct {
		Instalacao int64 `json:"installation_id"`
		receipt.ResultadoDaCadeia
	}{id, resultado})
}

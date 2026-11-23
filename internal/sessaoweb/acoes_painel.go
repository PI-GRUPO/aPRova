package sessaoweb

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/PI-GRUPO/aPRova/internal/acaoremota"
	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/google/uuid"
)

type AcoesDoPainel interface {
	ProcessarPainel(context.Context, acaoremota.Pedido, string) (acaoremota.Decisao, acaoremota.Motivo)
}

type pedidoDoPainel struct {
	IDAcao        string          `json:"id_acao"`
	Tipo          acaoremota.Tipo `json:"tipo"`
	Commit        string          `json:"commit_sha"`
	Justificativa string          `json:"justificativa"`
}

// ExecutarAcao vincula o pedido à identidade da sessão e à análise que ela pode consultar.
func (s *Servidor) ExecutarAcao(w http.ResponseWriter, r *http.Request, id string) {
	cabecalhosPrivados(w)
	if s == nil {
		falhar(w, http.StatusServiceUnavailable, "login web indisponível")
		return
	}
	v, ok := s.autenticar(w, r)
	if !ok {
		return
	}
	if !s.confereCSRF(r, v) {
		falhar(w, http.StatusForbidden, "origem ou proteção CSRF inválida")
		return
	}
	q := r.URL.Query()
	tenant, err := instalacaoWeb(q)
	analiseID, errID := uuid.Parse(id)
	if err != nil || errID != nil || parametrosWeb(q, parametroInstalacaoWeb) != nil {
		falhar(w, http.StatusBadRequest, "instalação ou análise inválida")
		return
	}
	pedido, acaoID, ok := lerPedidoDoPainel(w, r)
	if !ok {
		return
	}
	if s.acoes == nil || s.analises == nil {
		falhar(w, http.StatusServiceUnavailable, "ações do painel indisponíveis")
		return
	}
	pessoa, err := s.provedor.Usuario(r.Context(), v.Token)
	if err != nil {
		s.erroProvedor(w, r, err)
		return
	}
	if pessoa.ID != v.Usuario.ID || pessoa.Login == "" {
		falhar(w, http.StatusUnauthorized, "identidade da sessão inválida")
		return
	}
	repos, ok := s.escopoWeb(w, r, tenant)
	if !ok {
		return
	}
	detalhe, err := s.analises.DetalharAnaliseWeb(r.Context(), tenant, repos, analiseID.String())
	if errors.Is(err, domain.ErrNaoEncontrado) {
		falhar(w, http.StatusNotFound, "análise não encontrada neste escopo")
		return
	}
	if err != nil {
		falhar(w, http.StatusServiceUnavailable, "não foi possível consultar a análise")
		return
	}
	a := detalhe.Analise
	if pedido.Commit != a.SHAHead {
		responderAcaoDoPainel(w, acaoremota.DecisaoRecusada, acaoremota.MotivoAnaliseIncompativel, acaoID)
		return
	}
	decisao, motivo := s.acoes.ProcessarPainel(r.Context(), acaoremota.Pedido{
		IDAcao: "painel:" + acaoID, Tipo: pedido.Tipo, InstallationID: tenant,
		Repositorio: a.Repositorio, NumeroPR: int(a.NumeroPR), TaskRunID: a.ID,
		CommitSHA: a.SHAHead, Justificativa: pedido.Justificativa,
	}, pessoa.Login)
	responderAcaoDoPainel(w, decisao, motivo, acaoID)
}

func lerPedidoDoPainel(w http.ResponseWriter, r *http.Request) (pedidoDoPainel, string, bool) {
	var pedido pedidoDoPainel
	tipo, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if tipo != "application/json" {
		falhar(w, http.StatusUnsupportedMediaType, "envie a ação em JSON")
		return pedido, "", false
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10))
	d.DisallowUnknownFields()
	err := d.Decode(&pedido)
	id, errID := uuid.Parse(pedido.IDAcao)
	if err != nil || errID != nil || id == uuid.Nil || !errors.Is(d.Decode(&struct{}{}), io.EOF) {
		falhar(w, http.StatusBadRequest, "solicitação inválida")
		return pedido, "", false
	}
	return pedido, id.String(), true
}

func responderAcaoDoPainel(w http.ResponseWriter, decisao acaoremota.Decisao, motivo acaoremota.Motivo, id string) {
	status := http.StatusServiceUnavailable
	switch {
	case decisao == acaoremota.DecisaoAceita:
		status = http.StatusAccepted
	case decisao == acaoremota.DecisaoRepetida || motivo == acaoremota.MotivoAnaliseIncompativel:
		status = http.StatusConflict
	case decisao == acaoremota.DecisaoInvalida:
		status = http.StatusBadRequest
	case decisao == acaoremota.DecisaoRecusada && motivo != acaoremota.MotivoFalhaNaVerificacao:
		status = http.StatusForbidden
	}
	if status == http.StatusAccepted {
		jsonResposta(w, status, map[string]string{"id_acao": id, "status": "solicitacao_recebida"})
		return
	}
	jsonResposta(w, status, map[string]acaoremota.Motivo{"codigo": motivo})
}

package api

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/receipt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const (
	parametroDoRecibo = "id"
	algoritmoDaChave  = "Ed25519"

	conferenciaOK            = "ok"
	conferenciaFalhou        = "falhou"
	conferenciaNaoVerificada = "nao_verificada"
)

var ordemDasConferencias = []receipt.Falha{
	receipt.FalhaHash, receipt.FalhaChaveDesconhecida, receipt.FalhaAssinatura, receipt.FalhaConteudo, receipt.FalhaElo,
}

var nomeDaConferencia = map[receipt.Falha]string{
	receipt.FalhaHash:              "hash",
	receipt.FalhaChaveDesconhecida: "chave",
	receipt.FalhaAssinatura:        "assinatura",
	receipt.FalhaConteudo:          "conteudo",
	receipt.FalhaElo:               "elo",
}

// VerificadorDeRecibos lê e verifica recibos guardados e lista as chaves
// públicas aceitas. O receipt.Verificador é a implementação.
type VerificadorDeRecibos interface {
	Buscar(ctx context.Context, id string) (domain.Recibo, error)
	Verificar(ctx context.Context, id string) (domain.Recibo, receipt.Resultado, error)
	Chaves() []receipt.ChavePublica
}

type conferencia struct {
	Nome      string `json:"nome"`
	Resultado string `json:"resultado"`
}

type respostaDaVerificacao struct {
	ID           string        `json:"id"`
	Valido       bool          `json:"valido"`
	Falha        string        `json:"falha,omitempty"`
	Detalhe      string        `json:"detalhe,omitempty"`
	Verificacoes []conferencia `json:"verificacoes"`
	HashSHA256   string        `json:"hash_sha256"`
	KeyID        string        `json:"key_id"`
}

type chavePublicaListada struct {
	KeyID              string `json:"key_id"`
	Algoritmo          string `json:"algoritmo"`
	ChavePublicaBase64 string `json:"chave_publica_base64"`
}

type respostaDasChaves struct {
	Chaves []chavePublicaListada `json:"chaves"`
}

func (s *Servidor) manipularRecibo(w http.ResponseWriter, r *http.Request) {
	id, valido := s.idDoRecibo(w, r)
	if !valido {
		return
	}

	recibo, err := s.recibos.Buscar(r.Context(), id)
	if err != nil {
		responderFalhaDeRecibo(w, id, err)
		return
	}
	responderJSON(w, http.StatusOK, receipt.Exportar(recibo))
}

func (s *Servidor) manipularVerificacaoDeRecibo(w http.ResponseWriter, r *http.Request) {
	id, valido := s.idDoRecibo(w, r)
	if !valido {
		return
	}

	recibo, resultado, err := s.recibos.Verificar(r.Context(), id)
	if err != nil {
		responderFalhaDeRecibo(w, id, err)
		return
	}
	responderJSON(w, http.StatusOK, respostaDaVerificacao{
		ID:           recibo.ID,
		Valido:       resultado.Valido,
		Falha:        string(resultado.Falha),
		Detalhe:      resultado.Detalhe,
		Verificacoes: conferenciasDe(resultado),
		HashSHA256:   receipt.HashEmHex([]byte(recibo.Conteudo)),
		KeyID:        recibo.IDChave,
	})
}

func (s *Servidor) manipularChavesDeRecibo(w http.ResponseWriter, _ *http.Request) {
	if s.recibos == nil {
		responderRecibosIndisponiveis(w)
		return
	}

	chaves := []chavePublicaListada{}
	for _, chave := range s.recibos.Chaves() {
		chaves = append(chaves, chavePublicaListada{
			KeyID: chave.KeyID, Algoritmo: algoritmoDaChave, ChavePublicaBase64: base64.StdEncoding.EncodeToString(chave.Chave),
		})
	}
	responderJSON(w, http.StatusOK, respostaDasChaves{Chaves: chaves})
}

func (s *Servidor) idDoRecibo(w http.ResponseWriter, r *http.Request) (string, bool) {
	if s.recibos == nil {
		responderRecibosIndisponiveis(w)
		return "", false
	}

	id, err := uuid.Parse(chi.URLParam(r, parametroDoRecibo))
	if err != nil {
		responderJSON(w, http.StatusNotFound, respostaSimples{Status: "recibo não encontrado"})
		return "", false
	}
	return id.String(), true
}

func conferenciasDe(resultado receipt.Resultado) []conferencia {
	conferencias := make([]conferencia, 0, len(ordemDasConferencias))
	estado := conferenciaOK
	for _, falha := range ordemDasConferencias {
		atual := estado
		if !resultado.Valido && falha == resultado.Falha {
			atual, estado = conferenciaFalhou, conferenciaNaoVerificada
		}
		conferencias = append(conferencias, conferencia{Nome: nomeDaConferencia[falha], Resultado: atual})
	}
	return conferencias
}

func responderFalhaDeRecibo(w http.ResponseWriter, id string, err error) {
	if errors.Is(err, domain.ErrNaoEncontrado) {
		responderJSON(w, http.StatusNotFound, respostaSimples{Status: "recibo não encontrado"})
		return
	}
	slog.Error("falha ao ler recibo", "recibo", id, "erro", err)
	responderJSON(w, http.StatusInternalServerError, respostaSimples{Status: "falha ao ler o recibo"})
}

func responderRecibosIndisponiveis(w http.ResponseWriter) {
	responderJSON(w, http.StatusServiceUnavailable, respostaSimples{Status: "recibos indisponíveis"})
}

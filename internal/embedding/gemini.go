package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

const (
	urlBaseDoGemini  = "https://generativelanguage.googleapis.com"
	modeloDoGemini   = "gemini-embedding-001"
	tarefaDoGemini   = "SEMANTIC_SIMILARITY"
	timeoutPadrao    = 3 * time.Second
	limiteDaResposta = 1 << 20
)

// OpcoesGemini configura o Gemini. Sem URLBase usa a API pública; sem
// ClienteHTTP usa http.DefaultClient; sem Timeout usa 3 segundos.
type OpcoesGemini struct {
	Chave       string
	URLBase     string
	ClienteHTTP *http.Client
	Timeout     time.Duration
}

// Gemini gera embeddings com o gemini-embedding-001, tarefa de similaridade,
// em domain.DimensoesEmbedding dimensões. A chave vai no cabeçalho
// x-goog-api-key, nunca na URL.
type Gemini struct {
	chave   string
	url     string
	cliente *http.Client
	timeout time.Duration
}

type pedidoDoGemini struct {
	Modelo    string           `json:"model"`
	Conteudo  conteudoDoGemini `json:"content"`
	Tarefa    string           `json:"taskType"`
	Dimensoes int              `json:"outputDimensionality"`
}

type conteudoDoGemini struct {
	Partes []parteDoGemini `json:"parts"`
}

type parteDoGemini struct {
	Texto string `json:"text"`
}

type respostaDoGemini struct {
	Embedding struct {
		Valores []float32 `json:"values"`
	} `json:"embedding"`
}

// NovoGemini monta o cliente do Gemini. Sem chave devolve ErrChaveAusente.
func NovoGemini(opcoes OpcoesGemini) (*Gemini, error) {
	if strings.TrimSpace(opcoes.Chave) == "" {
		return nil, ErrChaveAusente
	}
	if opcoes.URLBase == "" {
		opcoes.URLBase = urlBaseDoGemini
	}
	if opcoes.ClienteHTTP == nil {
		opcoes.ClienteHTTP = http.DefaultClient
	}
	if opcoes.Timeout <= 0 {
		opcoes.Timeout = timeoutPadrao
	}
	return &Gemini{
		chave:   opcoes.Chave,
		url:     strings.TrimSuffix(opcoes.URLBase, "/") + "/v1beta/models/" + modeloDoGemini + ":embedContent",
		cliente: opcoes.ClienteHTTP,
		timeout: opcoes.Timeout,
	}, nil
}

// Gerar devolve o vetor normalizado do texto. Qualquer falha, inclusive o
// timeout, devolve ErrIndisponivel; a mensagem nunca traz a chave.
func (g *Gemini) Gerar(ctx context.Context, texto string) ([]float32, error) {
	ctx, cancelar := context.WithTimeout(ctx, g.timeout)
	defer cancelar()

	resposta, err := g.pedir(ctx, texto)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrIndisponivel, err)
	}
	if len(resposta.Embedding.Valores) != domain.DimensoesEmbedding {
		return nil, fmt.Errorf("%w: vetor com %d dimensões, esperava %d", ErrIndisponivel, len(resposta.Embedding.Valores), domain.DimensoesEmbedding)
	}
	vetor, err := Normalizar(resposta.Embedding.Valores)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrIndisponivel, err)
	}
	return vetor, nil
}

func (g *Gemini) pedir(ctx context.Context, texto string) (respostaDoGemini, error) {
	corpo, err := json.Marshal(pedidoDoGemini{
		Modelo:    "models/" + modeloDoGemini,
		Conteudo:  conteudoDoGemini{Partes: []parteDoGemini{{Texto: texto}}},
		Tarefa:    tarefaDoGemini,
		Dimensoes: domain.DimensoesEmbedding,
	})
	if err != nil {
		return respostaDoGemini{}, err
	}

	requisicao, err := http.NewRequestWithContext(ctx, http.MethodPost, g.url, bytes.NewReader(corpo))
	if err != nil {
		return respostaDoGemini{}, err
	}
	requisicao.Header.Set("Content-Type", "application/json")
	requisicao.Header.Set("x-goog-api-key", g.chave)

	resposta, err := g.cliente.Do(requisicao)
	if err != nil {
		return respostaDoGemini{}, err
	}
	defer func() { _ = resposta.Body.Close() }()

	if resposta.StatusCode != http.StatusOK {
		return respostaDoGemini{}, fmt.Errorf("%s respondeu %d", modeloDoGemini, resposta.StatusCode)
	}
	var lida respostaDoGemini
	if err := json.NewDecoder(http.MaxBytesReader(nil, resposta.Body, limiteDaResposta)).Decode(&lida); err != nil {
		return respostaDoGemini{}, fmt.Errorf("resposta do %s fora do formato: %w", modeloDoGemini, err)
	}
	return lida, nil
}

package explicacao

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const (
	urlBaseDoGemini       = "https://generativelanguage.googleapis.com"
	temperatura           = 0.2
	maximoDeTokensDeSaida = 1024
	nivelDePensamento     = "minimal"
	fimNormal             = "STOP"
	limiteDaResposta      = 1 << 20
)

// OpcoesGemini configura o cliente de geração de texto do Gemini. Sem URLBase
// usa a API pública; sem ClienteHTTP usa http.DefaultClient.
type OpcoesGemini struct {
	Chave       string
	Modelo      string
	URLBase     string
	ClienteHTTP *http.Client
}

// Gemini gera texto com um modelo Gemini, com temperatura baixa e pensamento
// no nível mínimo. A chave vai no cabeçalho x-goog-api-key, nunca na URL nem
// nas mensagens de erro.
type Gemini struct {
	chave   string
	modelo  string
	url     string
	cliente *http.Client
}

type parteDoGemini struct {
	Texto      string `json:"text"`
	Pensamento bool   `json:"thought,omitempty"`
}

type conteudoDoGemini struct {
	Papel  string          `json:"role,omitempty"`
	Partes []parteDoGemini `json:"parts"`
}

type configuracaoDePensamento struct {
	Nivel string `json:"thinkingLevel"`
}

type configuracaoDeGeracao struct {
	Temperatura    float64                  `json:"temperature"`
	MaximoDeTokens int                      `json:"maxOutputTokens"`
	TipoDaResposta string                   `json:"responseMimeType"`
	Pensamento     configuracaoDePensamento `json:"thinkingConfig"`
}

type pedidoDoGemini struct {
	Sistema   conteudoDoGemini      `json:"systemInstruction"`
	Conteudos []conteudoDoGemini    `json:"contents"`
	Geracao   configuracaoDeGeracao `json:"generationConfig"`
}

type respostaDoGemini struct {
	Candidatos []struct {
		Conteudo conteudoDoGemini `json:"content"`
		Fim      string           `json:"finishReason"`
	} `json:"candidates"`
}

// NovoGemini monta o cliente. Sem chave ou sem modelo devolve ErrConfiguracao.
func NovoGemini(opcoes OpcoesGemini) (*Gemini, error) {
	if strings.TrimSpace(opcoes.Chave) == "" || strings.TrimSpace(opcoes.Modelo) == "" {
		return nil, fmt.Errorf("%w: o Gemini da explicação exige GEMINI_API_KEY e o modelo", ErrConfiguracao)
	}
	if opcoes.URLBase == "" {
		opcoes.URLBase = urlBaseDoGemini
	}
	if opcoes.ClienteHTTP == nil {
		opcoes.ClienteHTTP = http.DefaultClient
	}
	return &Gemini{
		chave:   opcoes.Chave,
		modelo:  opcoes.Modelo,
		url:     strings.TrimSuffix(opcoes.URLBase, "/") + "/v1beta/models/" + opcoes.Modelo + ":generateContent",
		cliente: opcoes.ClienteHTTP,
	}, nil
}

// Gerar devolve o texto que o modelo escreveu para a instrução do sistema e o
// conteúdo. Status diferente de 200, resposta cortada ou fora do formato
// devolvem erro.
func (g *Gemini) Gerar(ctx context.Context, sistema, conteudo string) (string, error) {
	corpo, err := json.Marshal(pedidoDoGemini{
		Sistema:   conteudoDoGemini{Partes: []parteDoGemini{{Texto: sistema}}},
		Conteudos: []conteudoDoGemini{{Papel: "user", Partes: []parteDoGemini{{Texto: conteudo}}}},
		Geracao: configuracaoDeGeracao{
			Temperatura: temperatura, MaximoDeTokens: maximoDeTokensDeSaida, TipoDaResposta: "text/plain",
			Pensamento: configuracaoDePensamento{Nivel: nivelDePensamento},
		},
	})
	if err != nil {
		return "", fmt.Errorf("serializar o pedido ao %s: %w", g.modelo, err)
	}

	requisicao, err := http.NewRequestWithContext(ctx, http.MethodPost, g.url, bytes.NewReader(corpo))
	if err != nil {
		return "", fmt.Errorf("montar o pedido ao %s: %w", g.modelo, err)
	}
	requisicao.Header.Set("Content-Type", "application/json")
	requisicao.Header.Set("x-goog-api-key", g.chave)

	resposta, err := g.cliente.Do(requisicao)
	if err != nil {
		return "", fmt.Errorf("chamar o %s: %w", g.modelo, err)
	}
	defer func() { _ = resposta.Body.Close() }()
	if resposta.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s respondeu %d", g.modelo, resposta.StatusCode)
	}

	var lida respostaDoGemini
	if err := json.NewDecoder(http.MaxBytesReader(nil, resposta.Body, limiteDaResposta)).Decode(&lida); err != nil {
		return "", fmt.Errorf("resposta do %s fora do formato: %w", g.modelo, err)
	}
	return g.textoDa(lida)
}

func (g *Gemini) textoDa(resposta respostaDoGemini) (string, error) {
	if len(resposta.Candidatos) == 0 {
		return "", fmt.Errorf("%s respondeu sem candidato", g.modelo)
	}
	candidato := resposta.Candidatos[0]
	if candidato.Fim != fimNormal {
		return "", fmt.Errorf("%s terminou a resposta com %s", g.modelo, candidato.Fim)
	}
	var texto strings.Builder
	for _, parte := range candidato.Conteudo.Partes {
		if !parte.Pensamento {
			texto.WriteString(parte.Texto)
		}
	}
	return texto.String(), nil
}

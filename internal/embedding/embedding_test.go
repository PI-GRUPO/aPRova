package embedding

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

const chaveDeTeste = "chave-de-teste-que-nunca-aparece"

type pedidoRecebido struct {
	Modelo   string `json:"model"`
	Conteudo struct {
		Partes []struct {
			Texto string `json:"text"`
		} `json:"parts"`
	} `json:"content"`
	Tarefa    string `json:"taskType"`
	Dimensoes int    `json:"outputDimensionality"`
}

func vetorBruto(dimensoes int) []float64 {
	valores := make([]float64, dimensoes)
	for indice := range valores {
		valores[indice] = float64(indice%7) - 3
	}
	return valores
}

func servidorDoGemini(t *testing.T, responder func(w http.ResponseWriter, r *http.Request)) *Gemini {
	t.Helper()
	servidor := httptest.NewServer(http.HandlerFunc(responder))
	t.Cleanup(servidor.Close)

	gemini, err := NovoGemini(OpcoesGemini{Chave: chaveDeTeste, URLBase: servidor.URL, ClienteHTTP: servidor.Client(), Timeout: 200 * time.Millisecond})
	if err != nil {
		t.Fatalf("montar o Gemini: %v", err)
	}
	return gemini
}

func responderVetor(valores []float64) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"embedding": map[string]any{"values": valores}})
	}
}

func norma(vetor []float32) float64 {
	soma := 0.0
	for _, valor := range vetor {
		soma += float64(valor) * float64(valor)
	}
	return math.Sqrt(soma)
}

func TestGeminiPedeSimilaridadeEm768DimensoesComAChaveNoCabecalho(t *testing.T) {
	var recebido pedidoRecebido
	var caminho, consulta, chave string
	gemini := servidorDoGemini(t, func(w http.ResponseWriter, r *http.Request) {
		caminho, consulta, chave = r.URL.Path, r.URL.RawQuery, r.Header.Get("x-goog-api-key")
		_ = json.NewDecoder(r.Body).Decode(&recebido)
		responderVetor(vetorBruto(domain.DimensoesEmbedding))(w, r)
	})

	if _, err := gemini.Gerar(t.Context(), "motivos: area_sensivel"); err != nil {
		t.Fatalf("gerar: %v", err)
	}

	if caminho != "/v1beta/models/gemini-embedding-001:embedContent" {
		t.Errorf("caminho = %s", caminho)
	}
	if chave != chaveDeTeste || strings.Contains(consulta, chaveDeTeste) {
		t.Errorf("a chave vai no cabeçalho, nunca na URL: cabeçalho %q, consulta %q", chave, consulta)
	}
	if recebido.Modelo != "models/gemini-embedding-001" || recebido.Tarefa != "SEMANTIC_SIMILARITY" || recebido.Dimensoes != domain.DimensoesEmbedding {
		t.Errorf("pedido = %+v", recebido)
	}
	if len(recebido.Conteudo.Partes) != 1 || recebido.Conteudo.Partes[0].Texto != "motivos: area_sensivel" {
		t.Errorf("o texto vai como uma parte só: %+v", recebido.Conteudo)
	}
}

func TestGeminiDevolveOVetorNormalizado(t *testing.T) {
	bruto := vetorBruto(domain.DimensoesEmbedding)
	gemini := servidorDoGemini(t, responderVetor(bruto))

	vetor, err := gemini.Gerar(t.Context(), "texto")
	if err != nil {
		t.Fatalf("gerar: %v", err)
	}

	if len(vetor) != domain.DimensoesEmbedding || math.Abs(norma(vetor)-1) > 1e-5 {
		t.Fatalf("vetor de %d posições com norma %v", len(vetor), norma(vetor))
	}
	if vetor[1] >= 0 || vetor[3] != 0 || vetor[6] <= 0 {
		t.Errorf("a normalização mantém a direção: %v", vetor[:7])
	}
}

func TestGeminiSemChaveNaoEMontado(t *testing.T) {
	if _, err := NovoGemini(OpcoesGemini{}); !errors.Is(err, ErrChaveAusente) {
		t.Errorf("sem GEMINI_API_KEY: %v", err)
	}
}

func TestGeminiForaDoArOuRecusandoEIndisponivelSemCitarAChave(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusTooManyRequests, http.StatusForbidden, http.StatusBadRequest} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			gemini := servidorDoGemini(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"error":{"message":"falhou"}}`))
			})

			_, err := gemini.Gerar(t.Context(), "texto")

			if !errors.Is(err, ErrIndisponivel) {
				t.Fatalf("status %d: %v", status, err)
			}
			if strings.Contains(err.Error(), chaveDeTeste) {
				t.Errorf("o erro vazou a chave: %v", err)
			}
		})
	}
}

func TestGeminiLentoEstouraOTimeout(t *testing.T) {
	liberar := make(chan struct{})
	t.Cleanup(func() { close(liberar) })
	gemini := servidorDoGemini(t, func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-liberar:
		case <-r.Context().Done():
		}
	})

	inicio := time.Now()
	_, err := gemini.Gerar(t.Context(), "texto")

	if !errors.Is(err, ErrIndisponivel) {
		t.Fatalf("timeout é indisponibilidade: %v", err)
	}
	if decorrido := time.Since(inicio); decorrido > 2*time.Second {
		t.Errorf("o timeout de 200ms levou %v", decorrido)
	}
}

func TestGeminiRecusaRespostaForaDoFormato(t *testing.T) {
	casos := map[string]func(http.ResponseWriter, *http.Request){
		"dimensão errada": responderVetor(vetorBruto(3072)),
		"vetor nulo":      responderVetor(make([]float64, domain.DimensoesEmbedding)),
		"não é JSON":      func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html>")) },
	}
	for nome, responder := range casos {
		t.Run(nome, func(t *testing.T) {
			if _, err := servidorDoGemini(t, responder).Gerar(t.Context(), "texto"); !errors.Is(err, ErrIndisponivel) {
				t.Errorf("resposta inválida: %v", err)
			}
		})
	}
}

func TestGeminiRespeitaOContextoDeQuemChama(t *testing.T) {
	gemini := servidorDoGemini(t, responderVetor(vetorBruto(domain.DimensoesEmbedding)))
	cancelado, cancelar := context.WithCancel(t.Context())
	cancelar()

	if _, err := gemini.Gerar(cancelado, "texto"); !errors.Is(err, ErrIndisponivel) {
		t.Errorf("contexto cancelado: %v", err)
	}
}

func TestFalsoEDeterministicoENormalizado(t *testing.T) {
	falso := Falso{}

	primeiro, _ := falso.Gerar(t.Context(), "motivos: area_sensivel")
	repetido, _ := falso.Gerar(t.Context(), "motivos: area_sensivel")
	outro, _ := falso.Gerar(t.Context(), "motivos: codigo_sem_teste")

	if len(primeiro) != domain.DimensoesEmbedding || math.Abs(norma(primeiro)-1) > 1e-5 {
		t.Fatalf("vetor de %d posições com norma %v", len(primeiro), norma(primeiro))
	}
	if Similaridade(primeiro, repetido) < 0.9999 {
		t.Error("o mesmo texto dá o mesmo vetor")
	}
	if Similaridade(primeiro, outro) > 0.5 {
		t.Errorf("textos diferentes dão vetores distantes: %v", Similaridade(primeiro, outro))
	}
}

func TestSimilaridadeDeCosseno(t *testing.T) {
	if valor := Similaridade([]float32{1, 0}, []float32{0, 1}); valor != 0 {
		t.Errorf("ortogonais = %v", valor)
	}
	if valor := Similaridade([]float32{2, 0}, []float32{5, 0}); math.Abs(valor-1) > 1e-9 {
		t.Errorf("mesma direção = %v", valor)
	}
}

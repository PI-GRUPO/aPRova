// Package embedding transforma o texto da assinatura de um pull request em um
// vetor de domain.DimensoesEmbedding posições, normalizado, para medir
// semelhança por cosseno. Não decide nada: só mede.
package embedding

import (
	"context"
	"errors"
	"fmt"
	"math"
)

var (
	// ErrChaveAusente indica Gemini montado sem GEMINI_API_KEY.
	ErrChaveAusente = errors.New("embedding: GEMINI_API_KEY ausente")
	// ErrIndisponivel indica que o provedor não devolveu um vetor válido:
	// fora do ar, recusa, timeout ou resposta fora do formato.
	ErrIndisponivel = errors.New("embedding: provedor indisponível")
)

// Embedder gera o vetor normalizado de um texto.
type Embedder interface {
	Gerar(ctx context.Context, texto string) ([]float32, error)
}

// Normalizar devolve o vetor com norma 1 na mesma direção. Vetor nulo devolve
// erro.
func Normalizar(vetor []float32) ([]float32, error) {
	soma := 0.0
	for _, valor := range vetor {
		soma += float64(valor) * float64(valor)
	}
	norma := math.Sqrt(soma)
	if norma == 0 || math.IsNaN(norma) || math.IsInf(norma, 0) {
		return nil, fmt.Errorf("vetor de norma %v não pode ser normalizado", norma)
	}

	normalizado := make([]float32, len(vetor))
	for indice, valor := range vetor {
		normalizado[indice] = float32(float64(valor) / norma)
	}
	return normalizado, nil
}

// Similaridade devolve o cosseno entre dois vetores do mesmo tamanho, a mesma
// medida que o pgvector usa no índice da code_memory.
func Similaridade(a, b []float32) float64 {
	var produto, normaA, normaB float64
	for indice := range min(len(a), len(b)) {
		produto += float64(a[indice]) * float64(b[indice])
		normaA += float64(a[indice]) * float64(a[indice])
		normaB += float64(b[indice]) * float64(b[indice])
	}
	if normaA == 0 || normaB == 0 {
		return 0
	}
	return produto / (math.Sqrt(normaA) * math.Sqrt(normaB))
}

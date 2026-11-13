package embedding

import (
	"context"
	"fmt"
)

// Indisponivel é o embedder de quando o provedor não pode ser usado, como sem
// GEMINI_API_KEY: toda chamada devolve ErrIndisponivel com o Motivo, e a
// calibração registra isso em cada análise sem impedir nenhuma.
type Indisponivel struct {
	Motivo error
}

// Gerar devolve sempre ErrIndisponivel com o motivo.
func (i Indisponivel) Gerar(context.Context, string) ([]float32, error) {
	return nil, fmt.Errorf("%w: %w", ErrIndisponivel, i.Motivo)
}

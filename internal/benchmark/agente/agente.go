// Package agente roda um agente de código em modo headless sobre o diretório
// de uma tarefa do benchmark.
package agente

import (
	"context"
	"time"
)

// Pedido é uma execução do agente: o diretório da tarefa, que é o cwd do
// processo e o único lugar onde ele pode escrever, a instrução e o prazo.
type Pedido struct {
	Diretorio string
	Instrucao string
	Prazo     time.Duration
}

// Execucao é o que o agente devolveu. Saida é o stdout bruto.
type Execucao struct {
	Saida []byte
}

// Agente executa um pedido e devolve o que o processo produziu.
type Agente interface {
	Executar(ctx context.Context, pedido Pedido) (Execucao, error)
}

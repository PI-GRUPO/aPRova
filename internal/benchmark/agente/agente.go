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

// Desfecho diz como a execução do agente terminou, pelo que ele reportou e
// nunca pelo código de saída.
type Desfecho string

// Desfechos de uma execução.
const (
	DesfechoConcluida     Desfecho = "concluida"
	DesfechoErroReportado Desfecho = "erro_reportado"
	DesfechoSaidaIlegivel Desfecho = "saida_ilegivel"
	DesfechoLimiteDeUso   Desfecho = "limite_de_uso"
	DesfechoPrazoEsgotado Desfecho = "prazo_esgotado"
)

// UsoDoModelo são os tokens que um modelo consumiu na execução.
type UsoDoModelo struct {
	TokensDeEntrada      int64 `json:"tokens_de_entrada"`
	TokensDeSaida        int64 `json:"tokens_de_saida"`
	TokensDeCacheLidos   int64 `json:"tokens_de_cache_lidos"`
	TokensDeCacheCriados int64 `json:"tokens_de_cache_criados"`
}

// Execucao é o que o agente devolveu. Saida e Erros são o stdout e o stderr
// brutos; ModeloPrincipal é o modelo que mais gerou tokens de saída.
type Execucao struct {
	Desfecho        Desfecho
	Detalhe         string
	Resposta        string
	Saida           []byte
	Erros           []byte
	CodigoDeSaida   int
	Duracao         time.Duration
	Turnos          int
	ModeloPrincipal string
	Modelos         map[string]UsoDoModelo
	TokensDeEntrada int64
	TokensDeSaida   int64
}

// Agente executa um pedido e devolve o que o processo produziu.
type Agente interface {
	Executar(ctx context.Context, pedido Pedido) (Execucao, error)
}

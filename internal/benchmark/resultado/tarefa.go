// Package resultado define o registro bruto de cada tarefa do benchmark, o
// JSON gravado em bench/resultados/<data>/ assim que a tarefa termina.
package resultado

// Grupo separa as tarefas de correção (polyglot do Aider) das de risco
// (sintéticas, com nível e motivo esperados).
type Grupo string

// Grupos da suíte.
const (
	GrupoCorrecao Grupo = "correcao"
	GrupoRisco    Grupo = "risco"
)

// Desfecho diz até onde a tarefa chegou.
type Desfecho string

// Desfechos possíveis de uma tarefa.
const (
	DesfechoDecidido       Desfecho = "decidido"
	DesfechoSemDecisao     Desfecho = "sem_decisao"
	DesfechoDiffRecusado   Desfecho = "diff_recusado"
	DesfechoDiffVazio      Desfecho = "diff_vazio"
	DesfechoFalhaDoAgente  Desfecho = "falha_do_agente"
	DesfechoInfraestrutura Desfecho = "infraestrutura"
)

// Oraculo é o veredito dos testes originais da tarefa contra a solução.
type Oraculo string

// Vereditos do oráculo.
const (
	OraculoPassou   Oraculo = "passou"
	OraculoFalhou   Oraculo = "falhou"
	OraculoNaoRodou Oraculo = "nao_rodou"
)

// Tarefa é o resultado bruto de uma tarefa.
type Tarefa struct {
	ID                string      `json:"id"`
	Grupo             Grupo       `json:"grupo"`
	Linguagem         string      `json:"linguagem"`
	Desfecho          Desfecho    `json:"desfecho"`
	Detalhe           string      `json:"detalhe,omitempty"`
	PR                *PR         `json:"pr,omitempty"`
	Decisao           *Decisao    `json:"decisao,omitempty"`
	Oraculo           Oraculo     `json:"oraculo"`
	TestesAdulterados []string    `json:"testes_adulterados,omitempty"`
	Esperado          *Esperado   `json:"esperado,omitempty"`
	Tempos            Tempos      `json:"tempos"`
	Agente            UsoDoAgente `json:"agente"`
	ExecucoesDoAgente int         `json:"execucoes_do_agente"`
}

// PR identifica o pull request aberto para a solução.
type PR struct {
	Numero int    `json:"numero"`
	URL    string `json:"url,omitempty"`
	Branch string `json:"branch,omitempty"`
	SHA    string `json:"sha,omitempty"`
}

// Decisao é o que o aPRova decidiu sobre o PR.
type Decisao struct {
	Nivel           string   `json:"nivel"`
	Motivos         []string `json:"motivos"`
	EstadoDoTaskRun string   `json:"estado_do_task_run,omitempty"`
	ReciboID        string   `json:"recibo_id,omitempty"`
	ReciboHash      string   `json:"recibo_hash,omitempty"`
}

// Esperado é o nível e o motivo escritos antes de rodar, para o grupo B.
type Esperado struct {
	Nivel  string `json:"nivel"`
	Motivo string `json:"motivo"`
}

// Tempos de cada etapa, em milissegundos. Zero significa que a etapa não
// chegou ao fim.
type Tempos struct {
	AgenteMS     int64 `json:"agente_ms"`
	PublicacaoMS int64 `json:"publicacao_ms"`
	DecisaoMS    int64 `json:"decisao_ms"`
	OraculoMS    int64 `json:"oraculo_ms"`
}

// UsoDoAgente é o que o agente reportou: o modelo que produziu a saída, todos
// os modelos do modelUsage e os tokens.
type UsoDoAgente struct {
	ModeloPrincipal string                 `json:"modelo_principal,omitempty"`
	Modelos         map[string]UsoDoModelo `json:"modelos,omitempty"`
	TokensDeEntrada int64                  `json:"tokens_de_entrada"`
	TokensDeSaida   int64                  `json:"tokens_de_saida"`
	Turnos          int                    `json:"turnos,omitempty"`
	Desfecho        string                 `json:"desfecho,omitempty"`
	Detalhe         string                 `json:"detalhe,omitempty"`
}

// UsoDoModelo são os tokens de um modelo dentro de uma execução.
type UsoDoModelo struct {
	TokensDeEntrada int64 `json:"tokens_de_entrada"`
	TokensDeSaida   int64 `json:"tokens_de_saida"`
}

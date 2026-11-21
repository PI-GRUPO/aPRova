package domain

// ResultadoDaExplicacao é o desfecho da única tentativa de explicar em
// português simples a decisão de risco de um task_run.
type ResultadoDaExplicacao string

// Desfechos da explicação por IA.
const (
	ExplicacaoGerada     ResultadoDaExplicacao = "gerada"
	ExplicacaoDescartada ResultadoDaExplicacao = "descartada"
	ExplicacaoFalhou     ResultadoDaExplicacao = "falha"
)

// Explicacao é a tentativa de explicar a decisão de risco de um task_run.
// Texto só existe quando o Resultado é ExplicacaoGerada; Motivo diz por que a
// explicação foi descartada ou falhou. Nada aqui faz parte da decisão nem do
// recibo.
type Explicacao struct {
	TaskRunID string
	Resultado ResultadoDaExplicacao
	Texto     string
	Modelo    string
	Motivo    string
}

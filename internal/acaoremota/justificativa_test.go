package acaoremota

import "testing"

func TestJustificativaEExecucaoSaoPreservadasNaTentativa(t *testing.T) {
	c := novoCenario(t)
	const taskRunID = "c41d7e9a-2f3b-4c8d-a5e6-0b1c2d3e4f5a"
	decisao := c.servico.Processar(t.Context(), corpoDoPedido(t, map[string]any{
		"task_run_id": taskRunID, "justificativa": "  Risco revisado com os testes.  ",
	}))
	if decisao != DecisaoAceita {
		t.Fatalf("esperava aceitação, obtive %s", decisao)
	}
	tentativa := c.unicaTentativa(t)
	if tentativa.TaskRunID != taskRunID || tentativa.Justificativa != "Risco revisado com os testes." {
		t.Fatalf("contexto da decisão perdido: %+v", tentativa)
	}
}

func TestReexecucaoNaoExigeJustificativa(t *testing.T) {
	c := novoCenario(t)
	decisao := c.servico.Processar(t.Context(), corpoDoPedido(t, map[string]any{
		"tipo": "reexecutar_analise", "justificativa": nil,
	}))
	if decisao != DecisaoAceita {
		t.Fatalf("reexecução sem justificativa: %s", decisao)
	}
}

package carteira

import "testing"

func TestSaqueDentroDoSaldo(t *testing.T) {
	conta := &Conta{}
	conta.Depositar(1000)
	if err := conta.Sacar(400); err != nil || conta.Saldo() != 600 {
		t.Errorf("saque de 400 em 1000: erro %v, saldo %d", err, conta.Saldo())
	}
}

func TestSaqueAlemDoSaldoFalha(t *testing.T) {
	conta := &Conta{}
	if err := conta.Sacar(1); err == nil {
		t.Error("sacar sem saldo precisa falhar")
	}
}

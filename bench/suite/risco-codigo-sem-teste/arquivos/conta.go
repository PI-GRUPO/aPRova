package carteira

import "errors"

type Conta struct {
	saldoEmCentavos int64
}

func (c *Conta) Depositar(centavos int64) {
	c.saldoEmCentavos += centavos
}

func (c *Conta) Sacar(centavos int64) error {
	if centavos > c.saldoEmCentavos {
		return errors.New("saldo insuficiente")
	}
	c.saldoEmCentavos -= centavos
	return nil
}

func (c *Conta) Saldo() int64 {
	return c.saldoEmCentavos
}

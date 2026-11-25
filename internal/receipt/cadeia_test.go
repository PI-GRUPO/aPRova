package receipt

import (
	"context"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

type fonteDaCadeiaDeTeste []domain.Recibo

func (f fonteDaCadeiaDeTeste) ListarRecibosDaInstalacao(context.Context, int64, []string) ([]domain.Recibo, error) {
	return f, nil
}

func TestCadeiaInteiraPelaOrdemDosElos(t *testing.T) {
	recibos, ids, verificador := cadeiaDeTeste(t, 4)
	ordem := []domain.Recibo{recibos[ids[3]], recibos[ids[2]], recibos[ids[1]], recibos[ids[0]]}
	v := NovoVerificadorDeCadeia(verificador.chaveiro, fonteDaCadeiaDeTeste(ordem))
	r, err := v.VerificarCadeia(t.Context(), 161192271, []string{"equipe/repositorio"})
	if err != nil || !r.Valida || r.Total != 4 || r.Verificados != 4 {
		t.Fatalf("cadeia íntegra: %+v, %v", r, err)
	}
}

func TestCadeiaInteiraIdentificaOPrimeiroEloQuebrado(t *testing.T) {
	recibos, ids, verificador := cadeiaDeTeste(t, 4)
	// O último recibo continua passando na verificação individual.
	delete(recibos, ids[1])
	if _, r, err := verificador.Verificar(t.Context(), ids[3]); err != nil || !r.Valido {
		t.Fatalf("último recibo: %+v, %v", r, err)
	}
	v := NovoVerificadorDeCadeia(verificador.chaveiro, fonteDaCadeiaDeTeste{recibos[ids[3]], recibos[ids[2]], recibos[ids[0]]})
	r, err := v.VerificarCadeia(t.Context(), 161192271, []string{"equipe/repositorio"})
	if err != nil || r.Valida || r.Falha != FalhaElo || r.PrimeiroReciboInvalido != ids[2] || r.ReciboAnteriorAusente != ids[1] ||
		r.Total != 3 || r.Verificados != 1 {
		t.Fatalf("recibo do meio ausente: %+v, %v", r, err)
	}
}

func TestCadeiaInteiraIdentificaAssinaturaInvalidaAntesDoProximoElo(t *testing.T) {
	recibos, ids, verificador := cadeiaDeTeste(t, 3)
	segundo := recibos[ids[1]]
	segundo.Assinatura = make([]byte, len(segundo.Assinatura))
	v := NovoVerificadorDeCadeia(verificador.chaveiro, fonteDaCadeiaDeTeste{recibos[ids[2]], segundo, recibos[ids[0]]})
	r, err := v.VerificarCadeia(t.Context(), 161192271, []string{"equipe/repositorio"})
	if err != nil || r.Valida || r.Falha != FalhaAssinatura || r.PrimeiroReciboInvalido != ids[1] || r.Verificados != 1 {
		t.Fatalf("assinatura inválida: %+v, %v", r, err)
	}
}

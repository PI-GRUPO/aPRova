package embedding

import (
	"errors"
	"testing"
)

func TestIndisponivelSempreFalhaComOMotivo(t *testing.T) {
	vetor, err := Indisponivel{Motivo: ErrChaveAusente}.Gerar(t.Context(), "motivos: area_sensivel")

	if vetor != nil || !errors.Is(err, ErrIndisponivel) || !errors.Is(err, ErrChaveAusente) {
		t.Errorf("vetor %v, erro %v", vetor, err)
	}
}

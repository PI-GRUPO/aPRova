package rodada

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/benchmark/isolamento"
)

// Sondador roda a sondagem das cinco tentativas de fuga.
type Sondador interface {
	Verificar(ctx context.Context) (isolamento.Relatorio, error)
}

type registroDePrecondicao struct {
	Quando                   string               `json:"quando"`
	Sondagem                 isolamento.Relatorio `json:"sondagem"`
	ErroDaSondagem           string               `json:"erro_da_sondagem,omitempty"`
	VariaveisDoEnvConferidas int                  `json:"variaveis_do_env_conferidas"`
	ErroDoAutoteste          string               `json:"erro_do_autoteste,omitempty"`
}

// VerificarPrecondicoes roda a sondagem de isolamento e o autoteste da
// conferência do diff, grava o registro em precondicoes/ e devolve o
// primeiro erro: com ele, nenhuma tarefa roda.
func (r Rodada) VerificarPrecondicoes(ctx context.Context, sondador Sondador) error {
	quando := time.Now()
	relatorio, errDaSondagem := sondador.Verificar(ctx)
	quantas, errDoAutoteste := r.Segredos.Autoteste()

	registro := registroDePrecondicao{Quando: quando.Format(time.RFC3339), Sondagem: relatorio, VariaveisDoEnvConferidas: quantas}
	if errDaSondagem != nil {
		registro.ErroDaSondagem = errDaSondagem.Error()
	}
	if errDoAutoteste != nil {
		registro.ErroDoAutoteste = errDoAutoteste.Error()
	}
	nome := strings.ReplaceAll(quando.Format("20060102T150405.000000000"), ".", "") + ".json"
	if err := r.gravarJSON(filepath.Join("precondicoes", nome), registro); err != nil {
		return err
	}

	switch {
	case errDaSondagem != nil:
		r.registrarEvento("precondicao_falhou", "", errDaSondagem.Error())
		return errDaSondagem
	case errDoAutoteste != nil:
		r.registrarEvento("precondicao_falhou", "", errDoAutoteste.Error())
		return errDoAutoteste
	}
	r.registrarEvento("precondicao_ok", "", "")
	return nil
}

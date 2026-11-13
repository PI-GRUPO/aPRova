package receipt

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

type recibosGuardados map[string]domain.Recibo

func (r recibosGuardados) BuscarRecibo(_ context.Context, id string) (domain.Recibo, error) {
	recibo, existe := r[id]
	if !existe {
		return domain.Recibo{}, domain.ErrNaoEncontrado
	}
	return recibo, nil
}

func cadeiaDeTeste(t *testing.T, quantos int) (recibosGuardados, []string, *Verificador) {
	t.Helper()

	guardados := recibosGuardados{}
	var ids []string
	armazenamento := &armazenamentoFalso{}
	emissor := emissorDeTeste(t, armazenamento)

	for range quantos {
		emitido, err := emissor.Emitir(t.Context(), pedidoDeTeste())
		if err != nil {
			t.Fatalf("emitir: %v", err)
		}
		gravado := domain.Recibo{NovoRecibo: armazenamento.gravados[len(armazenamento.gravados)-1], TenantID: 161192271}
		guardados[emitido.ID] = gravado
		armazenamento.anterior = &gravado
		ids = append(ids, emitido.ID)
	}

	return guardados, ids, NovoVerificador(NovoChaveiro(assinadorDeTeste(t).ChavePublica()), guardados)
}

func exigirFalha(t *testing.T, verificador *Verificador, id string, esperada Falha) {
	t.Helper()

	_, resultado, err := verificador.Verificar(t.Context(), id)
	if err != nil {
		t.Fatalf("verificar: %v", err)
	}
	if resultado.Valido || resultado.Falha != esperada || resultado.Detalhe == "" {
		t.Errorf("esperava falha de %s com explicação, obtive %+v", esperada, resultado)
	}
}

func TestVerificadorAceitaCadeiaIntegra(t *testing.T) {
	_, ids, verificador := cadeiaDeTeste(t, 3)

	for _, id := range ids {
		recibo, resultado, err := verificador.Verificar(t.Context(), id)
		if err != nil || !resultado.Valido || recibo.ID != id {
			t.Errorf("recibo %s: %+v, erro %v", id, resultado, err)
		}
	}
}

func TestVerificadorDevolveNaoEncontradoParaIdInexistente(t *testing.T) {
	_, _, verificador := cadeiaDeTeste(t, 1)

	_, _, err := verificador.Verificar(t.Context(), "5d0a4c56-58f1-4e35-9d0e-2f1f0f6a7b22")

	if !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Errorf("esperava ErrNaoEncontrado, obtive %v", err)
	}
}

func TestApagarOReciboDoMeioQuebraOEloDoSeguinte(t *testing.T) {
	guardados, ids, verificador := cadeiaDeTeste(t, 3)
	delete(guardados, ids[1])

	exigirFalha(t, verificador, ids[2], FalhaElo)
	if _, resultado, _ := verificador.Verificar(t.Context(), ids[0]); !resultado.Valido {
		t.Errorf("o primeiro continua válido: %+v", resultado)
	}
}

func TestTrocarOConteudoDoAnteriorQuebraOEloDoSeguinte(t *testing.T) {
	guardados, ids, verificador := cadeiaDeTeste(t, 2)
	anterior := guardados[ids[0]]
	anterior.Conteudo = strings.Replace(anterior.Conteudo, `"nivel":"alto"`, `"nivel":"baixo"`, 1)
	guardados[ids[0]] = anterior

	exigirFalha(t, verificador, ids[1], FalhaElo)
}

func TestApontarAColunaDoAnteriorParaOutroReciboQuebraOElo(t *testing.T) {
	guardados, ids, verificador := cadeiaDeTeste(t, 3)
	terceiro := guardados[ids[2]]
	terceiro.ReciboAnteriorID = &ids[0]
	guardados[ids[2]] = terceiro

	exigirFalha(t, verificador, ids[2], FalhaElo)

	primeiro := guardados[ids[0]]
	primeiro.ReciboAnteriorID = &ids[1]
	guardados[ids[0]] = primeiro
	exigirFalha(t, verificador, ids[0], FalhaElo)
}

func TestConteudoAdulteradoNoBancoFalhaNoHashOuNaAssinatura(t *testing.T) {
	guardados, ids, verificador := cadeiaDeTeste(t, 1)
	original := guardados[ids[0]]

	adulterado := original
	adulterado.Conteudo = strings.Replace(original.Conteudo, `"nivel":"alto"`, `"nivel":"baixo"`, 1)
	guardados[ids[0]] = adulterado
	exigirFalha(t, verificador, ids[0], FalhaHash)

	adulterado.Hash = Hash([]byte(adulterado.Conteudo))
	guardados[ids[0]] = adulterado
	exigirFalha(t, verificador, ids[0], FalhaAssinatura)
}

func TestReciboGuardadoSobOutroIdOuOutroTaskRunFalhaNoConteudo(t *testing.T) {
	guardados, ids, verificador := cadeiaDeTeste(t, 1)
	original := guardados[ids[0]]

	outroID := "5d0a4c56-58f1-4e35-9d0e-2f1f0f6a7b22"
	trocado := original
	trocado.ID = outroID
	guardados[outroID] = trocado
	exigirFalha(t, verificador, outroID, FalhaConteudo)

	deOutroTaskRun := original
	deOutroTaskRun.TaskRunID = outroID
	guardados[ids[0]] = deOutroTaskRun
	exigirFalha(t, verificador, ids[0], FalhaConteudo)

	deOutroTenant := original
	deOutroTenant.TenantID = 7
	guardados[ids[0]] = deOutroTenant
	exigirFalha(t, verificador, ids[0], FalhaConteudo)
}

func TestChaveForaDoChaveiroFalhaComoChaveDesconhecida(t *testing.T) {
	guardados, ids, _ := cadeiaDeTeste(t, 1)

	exigirFalha(t, NovoVerificador(NovoChaveiro(), guardados), ids[0], FalhaChaveDesconhecida)
}

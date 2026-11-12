package receipt

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/sandbox"
)

type armazenamentoFalso struct {
	anterior *domain.Recibo
	falha    error
	gates    []domain.GateDeRisco
	gravados []domain.NovoRecibo
}

func (a *armazenamentoFalso) GravarDecisaoComRecibo(_ context.Context, gate domain.GateDeRisco, montar func(anterior *domain.Recibo) (domain.NovoRecibo, error)) (domain.Recibo, error) {
	if a.falha != nil {
		return domain.Recibo{}, a.falha
	}

	novo, err := montar(a.anterior)
	if err != nil {
		return domain.Recibo{}, err
	}
	a.gates = append(a.gates, gate)
	a.gravados = append(a.gravados, novo)
	return domain.Recibo{NovoRecibo: novo, TenantID: 161192271}, nil
}

var instanteDeTeste = time.Date(2026, 11, 9, 9, 30, 0, 123456789, time.FixedZone("BRT", -3*3600))

func pedidoDeTeste() Pedido {
	return Pedido{
		TaskRun: domain.TaskRun{
			ID: "9c80d110-a49a-4a70-b5f4-78cdf75b2f19", TenantID: 161192271, Repositorio: "Azulpasta/aPRova-TESTES",
			NumeroPR: 1, SHAHead: "d7ee920e4f9f8a2f413630afbe6c25435149a302",
		},
		Gate: domain.GateDeRisco{
			TaskRunID: "9c80d110-a49a-4a70-b5f4-78cdf75b2f19", NivelDeterministico: domain.RiscoAlto, Nivel: domain.RiscoAlto,
			Motivo: "falha_sandbox: os testes ou a compilação falharam no sandbox",
			Motivos: []domain.MotivoDeRisco{
				{ID: "falha_sandbox", Nivel: domain.RiscoAlto, Descricao: "os testes ou a compilação falharam no sandbox", Evidencias: []string{"1 de 11 testes falharam"}},
			},
		},
		Sandbox:      sandbox.ResultadoSandbox{Compilou: true, TestesPassaram: 10, TestesFalharam: 1, TipoFalha: sandbox.FalhaTeste, SaidaBruta: "saída longa"},
		Tentativa:    2,
		Diff:         "diff --git a/go/frete/frete_test.go b/go/frete/frete_test.go\n",
		DiffTruncado: true,
	}
}

func emissorDeTeste(t *testing.T, armazenamento Armazenamento) *Emissor {
	t.Helper()

	emissor, err := NovoEmissor(OpcoesEmissor{
		Assinador:     assinadorDeTeste(t),
		Armazenamento: armazenamento,
		Agora:         func() time.Time { return instanteDeTeste },
	})
	if err != nil {
		t.Fatalf("novo emissor: %v", err)
	}
	return emissor
}

func TestEmitirGravaADecisaoComUmReciboQueVerifica(t *testing.T) {
	armazenamento := &armazenamentoFalso{}
	pedido := pedidoDeTeste()

	emitido, err := emissorDeTeste(t, armazenamento).Emitir(t.Context(), pedido)
	if err != nil {
		t.Fatalf("emitir: %v", err)
	}

	if len(armazenamento.gates) != 1 || armazenamento.gates[0].TaskRunID != pedido.Gate.TaskRunID || armazenamento.gates[0].Nivel != domain.RiscoAlto {
		t.Fatalf("a decisão precisa ser gravada junto: %+v", armazenamento.gates)
	}
	gravado := armazenamento.gravados[0]
	assinador := assinadorDeTeste(t)
	resultado := NovoChaveiro(assinador.ChavePublica()).Verificar(Assinado{
		Conteudo: []byte(gravado.Conteudo), Hash: gravado.Hash, Assinatura: gravado.Assinatura, IDChave: gravado.IDChave,
	})
	if !resultado.Valido {
		t.Fatalf("o recibo gravado precisa verificar: %+v", resultado)
	}

	hash := sha256.Sum256([]byte(gravado.Conteudo))
	if emitido.ID != gravado.ID || !slices.Equal(emitido.Hash, hash[:]) || gravado.TaskRunID != pedido.TaskRun.ID {
		t.Errorf("emitido = %+v, gravado = %+v", emitido, gravado)
	}
	if gravado.ReciboAnteriorID != nil {
		t.Errorf("o primeiro recibo do tenant não tem anterior: %v", *gravado.ReciboAnteriorID)
	}
	if !gravado.AssinadoEm.Equal(instanteDeTeste.Truncate(time.Microsecond)) || gravado.AssinadoEm.Location() != time.UTC {
		t.Errorf("assinado em %v", gravado.AssinadoEm)
	}
}

func TestEmitirRegistraNoConteudoOQueFoiAnalisado(t *testing.T) {
	armazenamento := &armazenamentoFalso{}
	pedido := pedidoDeTeste()

	if _, err := emissorDeTeste(t, armazenamento).Emitir(t.Context(), pedido); err != nil {
		t.Fatalf("emitir: %v", err)
	}

	conteudo, err := LerConteudo([]byte(armazenamento.gravados[0].Conteudo))
	if err != nil {
		t.Fatalf("ler conteúdo: %v", err)
	}
	diff := sha256.Sum256([]byte(pedido.Diff))
	esperado := Conteudo{
		Versao: VersaoDoFormato, ID: armazenamento.gravados[0].ID, TaskRunID: pedido.TaskRun.ID, TenantID: 161192271,
		Repositorio: "Azulpasta/aPRova-TESTES", NumeroPR: 1, SHAHead: pedido.TaskRun.SHAHead,
		DiffSHA256: hex.EncodeToString(diff[:]), DiffTruncado: true,
		Sandbox:             Sandbox{Compilou: true, TestesPassaram: 10, TestesFalharam: 1, TipoFalha: "teste", Tentativa: 2},
		NivelDeterministico: domain.RiscoAlto, Nivel: domain.RiscoAlto,
		Motivos:   []Motivo{{ID: "falha_sandbox", Evidencias: []string{"1 de 11 testes falharam"}}},
		EmitidoEm: instanteDeTeste.UTC().Truncate(time.Microsecond),
		KeyID:     assinadorDeTeste(t).IDChave(),
	}

	brutoEsperado, err := esperado.Serializar()
	if err != nil {
		t.Fatal(err)
	}
	if armazenamento.gravados[0].Conteudo != string(brutoEsperado) {
		t.Errorf("conteúdo assinado:\n obtive %s\nesperava %s", armazenamento.gravados[0].Conteudo, brutoEsperado)
	}
	if conteudo.Anterior != nil {
		t.Errorf("sem anterior: %+v", conteudo.Anterior)
	}
}

func TestEmitirEncadeiaNoUltimoReciboDoTenant(t *testing.T) {
	anterior := &domain.Recibo{NovoRecibo: domain.NovoRecibo{ID: "5d0a4c56-58f1-4e35-9d0e-2f1f0f6a7b22", Conteudo: `{"versao":1,"id":"anterior"}`}}
	armazenamento := &armazenamentoFalso{anterior: anterior}

	if _, err := emissorDeTeste(t, armazenamento).Emitir(t.Context(), pedidoDeTeste()); err != nil {
		t.Fatalf("emitir: %v", err)
	}

	gravado := armazenamento.gravados[0]
	conteudo, err := LerConteudo([]byte(gravado.Conteudo))
	if err != nil {
		t.Fatal(err)
	}
	if gravado.ReciboAnteriorID == nil || *gravado.ReciboAnteriorID != anterior.ID {
		t.Errorf("a coluna do anterior precisa apontar para ele: %v", gravado.ReciboAnteriorID)
	}
	if conteudo.Anterior == nil || conteudo.Anterior.ID != anterior.ID || conteudo.Anterior.Hash != HashEmHex([]byte(anterior.Conteudo)) {
		t.Errorf("o conteúdo assinado precisa citar o id e o hash do anterior: %+v", conteudo.Anterior)
	}
	if falha := VerificarElo(conteudo, &Anterior{ID: anterior.ID, Conteudo: []byte(anterior.Conteudo)}); falha != "" {
		t.Errorf("elo: %s", falha)
	}
}

func TestCadaReciboTemUmIdProprio(t *testing.T) {
	armazenamento := &armazenamentoFalso{}
	emissor := emissorDeTeste(t, armazenamento)

	primeiro, err := emissor.Emitir(t.Context(), pedidoDeTeste())
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := emissor.Emitir(t.Context(), pedidoDeTeste())
	if err != nil {
		t.Fatal(err)
	}

	if primeiro.ID == segundo.ID || len(primeiro.ID) != 36 {
		t.Errorf("ids = %q e %q", primeiro.ID, segundo.ID)
	}
}

func TestEmitirDevolveOErroDoArmazenamento(t *testing.T) {
	armazenamento := &armazenamentoFalso{falha: domain.ErrDuplicado}

	_, err := emissorDeTeste(t, armazenamento).Emitir(t.Context(), pedidoDeTeste())

	if !errors.Is(err, domain.ErrDuplicado) {
		t.Errorf("quem chama precisa distinguir a decisão que já existe: %v", err)
	}
}

func TestNovoEmissorExigeAssinadorEArmazenamento(t *testing.T) {
	if _, err := NovoEmissor(OpcoesEmissor{Armazenamento: &armazenamentoFalso{}}); !errors.Is(err, ErrEmissorIncompleto) {
		t.Errorf("sem assinador: %v", err)
	}
	if _, err := NovoEmissor(OpcoesEmissor{Assinador: assinadorDeTeste(t)}); !errors.Is(err, ErrEmissorIncompleto) {
		t.Errorf("sem armazenamento: %v", err)
	}
}

func TestEmitirDevolveOIdDaChaveQueAssinou(t *testing.T) {
	armazenamento := &armazenamentoFalso{}

	emitido, err := emissorDeTeste(t, armazenamento).Emitir(t.Context(), pedidoDeTeste())
	if err != nil {
		t.Fatalf("emitir: %v", err)
	}

	if emitido.IDChave == "" || emitido.IDChave != assinadorDeTeste(t).IDChave() || emitido.IDChave != armazenamento.gravados[0].IDChave {
		t.Errorf("o emitido traz o key_id do recibo gravado: %q", emitido.IDChave)
	}
}

package reversao

import (
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
)

const (
	shaAnalisado = "1111111111111111111111111111111111111111"
	shaDoMerge   = "2222222222222222222222222222222222222222"
	shaPosterior = "3333333333333333333333333333333333333333"
	urlDoRecibo  = "https://aprova.exemplo/receipts/0b7e/verify"
)

func conteudoAnalisado() receipt.Conteudo {
	return receipt.Conteudo{
		ID:          "0b7e5a4c-0000-4000-8000-000000000001",
		Repositorio: "PI-GRUPO/aPRova-TESTES",
		NumeroPR:    42,
		SHAHead:     shaAnalisado,
		Nivel:       domain.RiscoAlto,
		Motivos: []receipt.Motivo{
			{ID: "testes_falharam", Evidencias: []string{"TestPesoZeroRetornaErro"}},
			{ID: "arquivo_sensivel", Evidencias: []string{"go.mod", "Dockerfile"}},
		},
	}
}

func pedidoMergeado(merge Merge) Pedido {
	merge.Mergeado = true
	if merge.SHAHeadFinal == "" {
		merge.SHAHeadFinal = shaAnalisado
	}
	if merge.SHADoMerge == "" {
		merge.SHADoMerge = shaDoMerge
	}
	return Pedido{
		Conteudo:    conteudoAnalisado(),
		Verificacao: receipt.Resultado{Valido: true},
		Merge:       &merge,
		URLDoRecibo: urlDoRecibo,
	}
}

func exigirRecusa(t *testing.T, instrucao Instrucao, esperada Recusa) {
	t.Helper()
	if instrucao.Recusa != esperada {
		t.Fatalf("recusa = %q, esperava %q", instrucao.Recusa, esperada)
	}
	if instrucao.Explicacao == "" {
		t.Error("recusa sem explicação")
	}
	if instrucao.Comando != "" || instrucao.Titulo != "" || instrucao.Corpo != "" {
		t.Errorf("recusa não pode trazer instrução: %+v", instrucao)
	}
}

func exigirComando(t *testing.T, instrucao Instrucao, esperado string) {
	t.Helper()
	if instrucao.Recusa != "" {
		t.Fatalf("recusou com %q: %s", instrucao.Recusa, instrucao.Explicacao)
	}
	if instrucao.Comando != esperado {
		t.Errorf("comando =\n%s\nesperava\n%s", instrucao.Comando, esperado)
	}
}

func TestReciboComAssinaturaInvalidaNaoGeraInstrucao(t *testing.T) {
	pedido := pedidoMergeado(Merge{Pais: 2})
	pedido.Verificacao = receipt.Resultado{Falha: receipt.FalhaAssinatura, Detalhe: "a assinatura não confere"}

	exigirRecusa(t, Montar(pedido), RecusaReciboInvalido)
}

func TestReciboComEloQuebradoNaoGeraInstrucao(t *testing.T) {
	pedido := pedidoMergeado(Merge{Pais: 2})
	pedido.Verificacao = receipt.Resultado{Falha: receipt.FalhaElo, Detalhe: "o recibo anterior não existe mais"}

	instrucao := Montar(pedido)

	exigirRecusa(t, instrucao, RecusaReciboInvalido)
	if !strings.Contains(instrucao.Explicacao, "o recibo anterior não existe mais") {
		t.Errorf("explicação não traz o detalhe da verificação: %q", instrucao.Explicacao)
	}
}

func TestPullRequestAindaAbertoNaoGeraInstrucao(t *testing.T) {
	pedido := pedidoMergeado(Merge{})
	pedido.Merge = nil

	exigirRecusa(t, Montar(pedido), RecusaPullRequestAberto)
}

func TestPullRequestFechadoSemMergeNaoGeraInstrucao(t *testing.T) {
	pedido := pedidoMergeado(Merge{})
	pedido.Merge = &Merge{Mergeado: false, SHAHeadFinal: shaAnalisado}

	exigirRecusa(t, Montar(pedido), RecusaPullRequestNaoMergeado)
}

func TestCommitAnalisadoDiferenteDoQueEntrouAvisaQueOReciboNaoCobre(t *testing.T) {
	instrucao := Montar(pedidoMergeado(Merge{Pais: 2, SHAHeadFinal: shaPosterior}))

	exigirRecusa(t, instrucao, RecusaCommitDiferente)
	for _, trecho := range []string{shaAnalisado, shaPosterior, "não cobre"} {
		if !strings.Contains(instrucao.Explicacao, trecho) {
			t.Errorf("explicação sem %q: %q", trecho, instrucao.Explicacao)
		}
	}
}

func TestShaForaDoFormatoNaoEntraNoComando(t *testing.T) {
	instrucao := Montar(pedidoMergeado(Merge{Pais: 2, SHADoMerge: "abc; rm -rf ~"}))

	exigirRecusa(t, instrucao, RecusaDadosDoMergeInvalidos)
}

func TestMergeCommitRevertidoContraOPrimeiroPai(t *testing.T) {
	instrucao := Montar(pedidoMergeado(Merge{Pais: 2, Commits: 3}))

	exigirComando(t, instrucao,
		"git fetch origin && git switch -c revert/pr-42 origin/main && git revert --no-edit -m 1 "+shaDoMerge)
	if instrucao.Tipo != TipoMerge || instrucao.CommitNaMain != shaDoMerge {
		t.Errorf("tipo %q e commit %q", instrucao.Tipo, instrucao.CommitNaMain)
	}
}

func TestUmPaiEUmCommitRevertidoDiretamente(t *testing.T) {
	exigirComando(t, Montar(pedidoMergeado(Merge{Pais: 1, Commits: 1})),
		"git fetch origin && git switch -c revert/pr-42 origin/main && git revert --no-edit "+shaDoMerge)
}

func TestUmPaiEVariosCommitsEAmbiguoSemOTipoInformado(t *testing.T) {
	instrucao := Montar(pedidoMergeado(Merge{Pais: 1, Commits: 3}))

	exigirRecusa(t, instrucao, RecusaTipoAmbiguo)
	if !strings.Contains(instrucao.Explicacao, "squash") || !strings.Contains(instrucao.Explicacao, "rebase") {
		t.Errorf("explicação não nomeia as duas possibilidades: %q", instrucao.Explicacao)
	}
}

func TestSemPaisNemTipoEAmbiguo(t *testing.T) {
	exigirRecusa(t, Montar(pedidoMergeado(Merge{Commits: 1})), RecusaTipoAmbiguo)
}

func TestSquashInformadoRevertidoDiretamente(t *testing.T) {
	exigirComando(t, Montar(pedidoMergeado(Merge{Tipo: TipoSquash, Commits: 3})),
		"git fetch origin && git switch -c revert/pr-42 origin/main && git revert --no-edit "+shaDoMerge)
}

func TestRebaseInformadoRevertidoPelaFaixaDeCommits(t *testing.T) {
	exigirComando(t, Montar(pedidoMergeado(Merge{Tipo: TipoRebase, Commits: 3})),
		"git fetch origin && git switch -c revert/pr-42 origin/main && git revert --no-edit "+shaDoMerge+"~3.."+shaDoMerge)
}

func TestRebaseDeUmCommitRevertidoDiretamente(t *testing.T) {
	exigirComando(t, Montar(pedidoMergeado(Merge{Tipo: TipoRebase, Commits: 1})),
		"git fetch origin && git switch -c revert/pr-42 origin/main && git revert --no-edit "+shaDoMerge)
}

func TestRebaseSemContagemDeCommitsNaoGeraComando(t *testing.T) {
	exigirRecusa(t, Montar(pedidoMergeado(Merge{Tipo: TipoRebase})), RecusaDadosDoMergeInvalidos)
}

func TestTipoDesconhecidoNaoGeraComando(t *testing.T) {
	exigirRecusa(t, Montar(pedidoMergeado(Merge{Tipo: "octopus", Commits: 1})), RecusaDadosDoMergeInvalidos)
}

func TestCorpoComecaComALinhaDeReversaoDoGitHub(t *testing.T) {
	instrucao := Montar(pedidoMergeado(Merge{Pais: 2}))

	primeiraLinha, _, _ := strings.Cut(instrucao.Corpo, "\n")
	if primeiraLinha != "Reverts PI-GRUPO/aPRova-TESTES#42" {
		t.Errorf("primeira linha = %q", primeiraLinha)
	}
}

func TestCorpoTrazNivelMotivosLinkEAviso(t *testing.T) {
	instrucao := Montar(pedidoMergeado(Merge{Pais: 2}))

	for _, trecho := range []string{
		shaDoMerge, "alto", "testes_falharam", "TestPesoZeroRetornaErro", "arquivo_sensivel", "go.mod; Dockerfile",
		urlDoRecibo, "também passa pelo aPRova",
	} {
		if !strings.Contains(instrucao.Corpo, trecho) {
			t.Errorf("corpo sem %q:\n%s", trecho, instrucao.Corpo)
		}
	}
	if !strings.Contains(instrucao.Titulo, "#42") {
		t.Errorf("título sem o número do PR: %q", instrucao.Titulo)
	}
}

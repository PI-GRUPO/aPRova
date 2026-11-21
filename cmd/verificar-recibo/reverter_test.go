package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
)

const (
	headAnalisado   = "d7ee920e4f9f8a2f413630afbe6c25435149a302"
	commitDoMerge   = "5555555555555555555555555555555555555555"
	pushPosterior   = "6666666666666666666666666666666666666666"
	urlPublica      = "https://aprova.exemplo"
	inicioDoComando = "git fetch origin && git switch -c revert/pr-1 origin/main && git revert --no-edit "
)

type cadeiaEmMemoria struct{ ultimo *domain.Recibo }

func (c *cadeiaEmMemoria) GravarDecisaoComRecibo(_ context.Context, _ domain.GateDeRisco, montar func(anterior *domain.Recibo) (domain.NovoRecibo, error)) (domain.Recibo, error) {
	novo, err := montar(c.ultimo)
	gravado := domain.Recibo{NovoRecibo: novo, TenantID: 161192271}
	c.ultimo = &gravado
	return gravado, err
}

func cadeiaDeDoisRecibos(t *testing.T) (anterior, atual receipt.Exportado, chave string) {
	t.Helper()

	assinador, err := receipt.NovoAssinador(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	cadeia := &cadeiaEmMemoria{}
	emissor, err := receipt.NovoEmissor(receipt.OpcoesEmissor{Assinador: assinador, Armazenamento: cadeia})
	if err != nil {
		t.Fatal(err)
	}

	var exportados []receipt.Exportado
	for range 2 {
		_, err := emissor.Emitir(t.Context(), receipt.Pedido{
			TaskRun: domain.TaskRun{ID: "9c80d110-a49a-4a70-b5f4-78cdf75b2f19", TenantID: 161192271, Repositorio: "Azulpasta/aPRova-TESTES", NumeroPR: 1, SHAHead: headAnalisado},
			Gate: domain.GateDeRisco{
				NivelDeterministico: domain.RiscoAlto, Nivel: domain.RiscoAlto,
				Motivos: []domain.MotivoDeRisco{{ID: "falha_sandbox", Evidencias: []string{"1 de 11 testes falharam"}}},
			},
			Tentativa: 1, Diff: "diff",
		})
		if err != nil {
			t.Fatal(err)
		}
		exportados = append(exportados, receipt.Exportar(*cadeia.ultimo))
	}
	return exportados[0], exportados[1], base64.StdEncoding.EncodeToString(assinador.ChavePublica())
}

func arquivoDoRecibo(t *testing.T, nome string, exportado receipt.Exportado) string {
	t.Helper()

	caminho := filepath.Join(t.TempDir(), nome)
	if err := os.WriteFile(caminho, []byte(emJSON(t, exportado)), 0o600); err != nil {
		t.Fatal(err)
	}
	return caminho
}

func reverter(t *testing.T, atual receipt.Exportado, argumentos ...string) (int, string, string) {
	t.Helper()
	return rodar(t, emJSON(t, atual), append([]string{"reverter"}, argumentos...)...)
}

func TestReverterReciboAdulteradoNaoGeraInstrucao(t *testing.T) {
	anterior, atual, chave := cadeiaDeDoisRecibos(t)
	atual.HashSHA256 = strings.Repeat("0", 64)

	codigo, saida, erros := reverter(t, atual, "-chave", chave, "-url-publica", urlPublica, "-anterior", arquivoDoRecibo(t, "anterior.json", anterior),
		"-base", "main", "-merge-sha", commitDoMerge, "-head-final", headAnalisado, "-tipo", "merge")

	if codigo != 1 || !strings.Contains(saida, "INVÁLIDO") || strings.Contains(saida, "git revert") {
		t.Errorf("código %d, saída %q, erros %q", codigo, saida, erros)
	}
}

func TestReverterComEloQuebradoNaoGeraInstrucao(t *testing.T) {
	anterior, atual, chave := cadeiaDeDoisRecibos(t)
	anterior.PayloadBase64 = atual.PayloadBase64

	codigo, saida, _ := reverter(t, atual, "-chave", chave, "-url-publica", urlPublica, "-anterior", arquivoDoRecibo(t, "anterior.json", anterior),
		"-base", "main", "-merge-sha", commitDoMerge, "-head-final", headAnalisado, "-tipo", "merge")

	if codigo != 1 || !strings.Contains(saida, "elo") || strings.Contains(saida, "git revert") {
		t.Errorf("código %d, saída %q", codigo, saida)
	}
}

func TestReverterSemAnteriorAvisaQueOEloNaoFoiConferido(t *testing.T) {
	_, atual, chave := cadeiaDeDoisRecibos(t)

	codigo, saida, _ := reverter(t, atual, "-chave", chave, "-url-publica", urlPublica,
		"-base", "main", "-merge-sha", commitDoMerge, "-head-final", headAnalisado, "-tipo", "merge")

	if codigo != 0 || !strings.HasPrefix(saida, "ATENÇÃO: ELO NÃO CONFERIDO") {
		t.Errorf("o aviso abre a saída: código %d, saída %q", codigo, saida)
	}
}

func TestReverterPrimeiroReciboDaCadeiaNaoPedeAnterior(t *testing.T) {
	primeiro, _, chave := cadeiaDeDoisRecibos(t)

	codigo, saida, _ := reverter(t, primeiro, "-chave", chave, "-url-publica", urlPublica,
		"-base", "main", "-merge-sha", commitDoMerge, "-head-final", headAnalisado, "-tipo", "merge")

	if codigo != 0 || strings.Contains(saida, "NÃO CONFERIDO") {
		t.Errorf("código %d, saída %q", codigo, saida)
	}
}

func TestReverterMergeCommitComAnteriorDevolveComandoECorpo(t *testing.T) {
	anterior, atual, chave := cadeiaDeDoisRecibos(t)

	codigo, saida, erros := reverter(t, atual, "-chave", chave, "-url-publica", urlPublica, "-anterior", arquivoDoRecibo(t, "anterior.json", anterior),
		"-base", "main", "-merge-sha", commitDoMerge, "-head-final", headAnalisado, "-tipo", "merge")

	if codigo != 0 {
		t.Fatalf("código %d, saída %q, erros %q", codigo, saida, erros)
	}
	for _, trecho := range []string{
		"VÁLIDO", inicioDoComando + "-m 1 " + commitDoMerge, "Reverts Azulpasta/aPRova-TESTES#1",
		urlPublica + "/receipts/" + atual.ID + "/verify", "Reverte o PR #1",
	} {
		if !strings.Contains(saida, trecho) {
			t.Errorf("saída sem %q:\n%s", trecho, saida)
		}
	}
	if strings.Contains(saida, "NÃO CONFERIDO") {
		t.Errorf("com -anterior o elo é conferido:\n%s", saida)
	}
}

func TestReverterEscolheOComandoPeloTipo(t *testing.T) {
	primeiro, _, chave := cadeiaDeDoisRecibos(t)
	casos := map[string]struct {
		tipo, commits, alvo string
	}{
		"squash":              {"squash", "3", commitDoMerge},
		"rebase de 3 commits": {"rebase", "3", commitDoMerge + "~3.." + commitDoMerge},
		"rebase de 1 commit":  {"rebase", "1", commitDoMerge},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			codigo, saida, _ := reverter(t, primeiro, "-chave", chave, "-url-publica", urlPublica,
				"-base", "main", "-merge-sha", commitDoMerge, "-head-final", headAnalisado, "-tipo", caso.tipo, "-commits", caso.commits)

			if codigo != 0 || !strings.Contains(saida, inicioDoComando+caso.alvo+"\n") {
				t.Errorf("código %d, saída:\n%s", codigo, saida)
			}
		})
	}
}

func TestReverterSemTipoEAmbiguo(t *testing.T) {
	primeiro, _, chave := cadeiaDeDoisRecibos(t)

	codigo, saida, _ := reverter(t, primeiro, "-chave", chave, "-url-publica", urlPublica,
		"-base", "main", "-merge-sha", commitDoMerge, "-head-final", headAnalisado, "-commits", "3")

	if codigo != 1 || !strings.Contains(saida, "tipo_ambiguo") || strings.Contains(saida, "git revert") {
		t.Errorf("código %d, saída %q", codigo, saida)
	}
}

func TestReverterPRNaoMergeadoExplica(t *testing.T) {
	primeiro, _, chave := cadeiaDeDoisRecibos(t)

	codigo, saida, _ := reverter(t, primeiro, "-chave", chave, "-url-publica", urlPublica, "-head-final", headAnalisado)

	if codigo != 1 || !strings.Contains(saida, "pr_nao_mergeado") || strings.Contains(saida, "git revert") {
		t.Errorf("código %d, saída %q", codigo, saida)
	}
}

func TestReverterComPushDepoisDaAnaliseAvisaQueOReciboNaoCobre(t *testing.T) {
	primeiro, _, chave := cadeiaDeDoisRecibos(t)

	codigo, saida, _ := reverter(t, primeiro, "-chave", chave, "-url-publica", urlPublica,
		"-base", "main", "-merge-sha", commitDoMerge, "-head-final", pushPosterior, "-tipo", "merge")

	if codigo != 1 || !strings.Contains(saida, "não cobre") || strings.Contains(saida, "git revert") {
		t.Errorf("código %d, saída %q", codigo, saida)
	}
}

func TestReverterSemURLPublicaOuComArgumentosDemaisEUsoIncorreto(t *testing.T) {
	primeiro, _, chave := cadeiaDeDoisRecibos(t)

	for nome, argumentos := range map[string][]string{
		"sem url pública":   {"-chave", chave, "-base", "main", "-merge-sha", commitDoMerge, "-head-final", headAnalisado, "-tipo", "merge"},
		"argumentos demais": {"-chave", chave, "-url-publica", urlPublica, "a.json", "b.json"},
		"commits negativos": {"-chave", chave, "-url-publica", urlPublica, "-commits", "-1"},
	} {
		t.Run(nome, func(t *testing.T) {
			codigo, saida, erros := reverter(t, primeiro, argumentos...)
			if codigo != 2 || erros == "" || strings.Contains(saida, "git revert") {
				t.Errorf("código %d, saída %q, erros %q", codigo, saida, erros)
			}
		})
	}
}

func TestReverterSemBaseNaoGeraComando(t *testing.T) {
	primeiro, _, chave := cadeiaDeDoisRecibos(t)

	codigo, saida, _ := reverter(t, primeiro, "-chave", chave, "-url-publica", urlPublica,
		"-merge-sha", commitDoMerge, "-head-final", headAnalisado, "-tipo", "merge")

	if codigo != 1 || !strings.Contains(saida, "base_desconhecida") || strings.Contains(saida, "git revert") {
		t.Errorf("código %d, saída %q", codigo, saida)
	}
}

func TestReverterPartindoDeOutraBase(t *testing.T) {
	primeiro, _, chave := cadeiaDeDoisRecibos(t)

	codigo, saida, _ := reverter(t, primeiro, "-chave", chave, "-url-publica", urlPublica,
		"-base", "release/2026", "-merge-sha", commitDoMerge, "-head-final", headAnalisado, "-tipo", "merge")

	comando := "git fetch origin && git switch -c revert/pr-1 origin/release/2026 && git revert --no-edit -m 1 " + commitDoMerge
	if codigo != 0 || !strings.Contains(saida, comando) || !strings.Contains(saida, "que entrou na release/2026") {
		t.Errorf("código %d, saída:\n%s", codigo, saida)
	}
}

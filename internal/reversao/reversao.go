// Package reversao monta, a partir de um recibo verificado e do desfecho do
// pull request, o que uma pessoa precisa para reverter o que entrou na main:
// o comando git e o texto do PR de reversão. Não executa nada.
package reversao

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PI-GRUPO/aPRova/internal/receipt"
)

// Tipo é a forma como o pull request entrou na main.
type Tipo string

// Tipos de merge que o GitHub oferece.
const (
	TipoMerge  Tipo = "merge"
	TipoSquash Tipo = "squash"
	TipoRebase Tipo = "rebase"
)

// Recusa nomeia por que não há instrução de reversão. Vazio é nenhuma.
type Recusa string

// Motivos para não gerar instrução de reversão.
const (
	RecusaReciboInvalido         Recusa = "recibo_invalido"
	RecusaPullRequestAberto      Recusa = "pr_aberto"
	RecusaPullRequestNaoMergeado Recusa = "pr_nao_mergeado"
	RecusaCommitDiferente        Recusa = "commit_diferente"
	RecusaTipoAmbiguo            Recusa = "tipo_ambiguo"
	RecusaDadosDoMergeInvalidos  Recusa = "dados_do_merge_invalidos"
)

// Merge é o desfecho do pull request. SHAHeadFinal é o head no fechamento;
// SHADoMerge é o commit que o GitHub informa como o que entrou na main. Pais
// é o número de pais desse commit, ou 0 quando não se sabe. Commits é quantos
// commits o PR tinha. Tipo, quando informado por quem pede, vale sobre Pais.
type Merge struct {
	Mergeado     bool
	SHAHeadFinal string
	SHADoMerge   string
	Pais         int
	Commits      int
	Tipo         Tipo
}

// Pedido reúne o recibo, o resultado da verificação dele e o desfecho do PR,
// ou Merge nil quando o PR ainda não foi fechado. URLDoRecibo é o link de
// verificação citado no corpo do PR de reversão.
type Pedido struct {
	Conteudo    receipt.Conteudo
	Verificacao receipt.Resultado
	Merge       *Merge
	URLDoRecibo string
}

// Instrucao é a reversão sugerida ou, com Recusa preenchida, a explicação de
// por que não há sugestão.
type Instrucao struct {
	Recusa       Recusa
	Explicacao   string
	Tipo         Tipo
	CommitNaMain string
	Comando      string
	Titulo       string
	Corpo        string
}

var formatoDoSHA = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)

// Montar devolve a instrução de reversão do pedido, ou a recusa explicada
// quando o recibo é inválido, o PR não entrou na main, o commit analisado não
// é o que entrou ou o tipo de merge não pode ser determinado.
func Montar(pedido Pedido) Instrucao {
	if recusa := recusaDoDesfecho(pedido); recusa.Recusa != "" {
		return recusa
	}

	merge := *pedido.Merge
	tipo, recusa := tipoDoMerge(merge)
	if recusa.Recusa != "" {
		return recusa
	}

	numero := pedido.Conteudo.NumeroPR
	return Instrucao{
		Tipo:         tipo,
		CommitNaMain: merge.SHADoMerge,
		Comando:      comandoDeReversao(numero, alvoDoRevert(tipo, merge)),
		Titulo:       fmt.Sprintf("Reverte o PR #%d", numero),
		Corpo:        corpoDoPullRequest(pedido),
	}
}

func recusar(recusa Recusa, explicacao string) Instrucao {
	return Instrucao{Recusa: recusa, Explicacao: explicacao}
}

func recusaDoDesfecho(pedido Pedido) Instrucao {
	merge := pedido.Merge
	switch {
	case !pedido.Verificacao.Valido:
		return recusar(RecusaReciboInvalido, fmt.Sprintf("o recibo não passou na verificação de %s: %s",
			pedido.Verificacao.Falha, pedido.Verificacao.Detalhe))
	case merge == nil:
		return recusar(RecusaPullRequestAberto, "o pull request ainda não foi fechado; não há o que reverter")
	case !merge.Mergeado:
		return recusar(RecusaPullRequestNaoMergeado, "o pull request foi fechado sem merge; nada dele entrou na main")
	case !formatoDoSHA.MatchString(merge.SHADoMerge) || !formatoDoSHA.MatchString(merge.SHAHeadFinal):
		return recusar(RecusaDadosDoMergeInvalidos, "os SHAs do merge precisam ser hexadecimais minúsculos de 40 ou 64 caracteres")
	case merge.SHAHeadFinal != pedido.Conteudo.SHAHead:
		return recusar(RecusaCommitDiferente, fmt.Sprintf(
			"o recibo analisou o commit %s, mas o PR entrou com o head %s: houve push depois da análise e o recibo não cobre o que entrou",
			pedido.Conteudo.SHAHead, merge.SHAHeadFinal))
	default:
		return Instrucao{}
	}
}

func tipoDoMerge(merge Merge) (Tipo, Instrucao) {
	switch {
	case merge.Tipo == TipoMerge || merge.Tipo == TipoSquash:
		return merge.Tipo, Instrucao{}
	case merge.Tipo == TipoRebase && merge.Commits < 1:
		return "", recusar(RecusaDadosDoMergeInvalidos, "para rebase, informe quantos commits o PR tinha")
	case merge.Tipo == TipoRebase:
		return TipoRebase, Instrucao{}
	case merge.Tipo != "":
		return "", recusar(RecusaDadosDoMergeInvalidos, "tipo de merge desconhecido: "+string(merge.Tipo))
	case merge.Pais == 2:
		return TipoMerge, Instrucao{}
	case merge.Pais == 1 && merge.Commits == 1:
		return TipoSquash, Instrucao{}
	default:
		return "", recusar(RecusaTipoAmbiguo,
			"não dá para saber se o PR entrou por squash ou por rebase; informe o tipo para receber o comando")
	}
}

func alvoDoRevert(tipo Tipo, merge Merge) string {
	switch {
	case tipo == TipoMerge:
		return "-m 1 " + merge.SHADoMerge
	case tipo == TipoRebase && merge.Commits > 1:
		return merge.SHADoMerge + "~" + strconv.Itoa(merge.Commits) + ".." + merge.SHADoMerge
	default:
		return merge.SHADoMerge
	}
}

func comandoDeReversao(numero int, alvo string) string {
	return fmt.Sprintf("git fetch origin && git switch -c revert/pr-%d origin/main && git revert --no-edit %s", numero, alvo)
}

func corpoDoPullRequest(pedido Pedido) string {
	conteudo := pedido.Conteudo
	var corpo strings.Builder
	fmt.Fprintf(&corpo, "Reverts %s#%d\n\n", conteudo.Repositorio, conteudo.NumeroPR)
	fmt.Fprintf(&corpo, "Este PR reverte o commit %s, que entrou na main pelo PR #%d.\n\n",
		pedido.Merge.SHADoMerge, conteudo.NumeroPR)
	fmt.Fprintf(&corpo, "A análise do aPRova sobre o commit %s deu nível %s, com os motivos:\n\n", conteudo.SHAHead, conteudo.Nivel)
	for _, motivo := range conteudo.Motivos {
		fmt.Fprintf(&corpo, "- %s: %s\n", motivo.ID, strings.Join(motivo.Evidencias, "; "))
	}
	fmt.Fprintf(&corpo, "\nRecibo da análise: %s\n\n", pedido.URLDoRecibo)
	corpo.WriteString("Este PR de reversão também passa pelo aPRova, como qualquer outro.\n")
	return corpo.String()
}

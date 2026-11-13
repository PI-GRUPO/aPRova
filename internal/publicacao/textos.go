package publicacao

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/platform"
	"github.com/Azulpasta/aPRova/internal/sandbox"
	"github.com/Azulpasta/aPRova/internal/worker"
)

const (
	tamanhoDoSHACurto          = 7
	maximoDeEvidencias         = 20
	tamanhoMaximoDaEvidencia   = 200
	tamanhoMaximoDoDiagnostico = 2000

	// URLDaVerificacaoIndependente é o passo a passo para verificar um recibo só
	// com a chave pública, sem confiar no servidor do aPRova.
	URLDaVerificacaoIndependente = "https://github.com/Azulpasta/aPRova/blob/main/docs/verificacao-recibo.md"
)

var nomeDoNivel = map[domain.NivelRisco]string{
	domain.RiscoBaixo: "baixo",
	domain.RiscoMedio: "médio",
	domain.RiscoAlto:  "alto",
}

type resultado struct {
	worker.Publicacao
	urlDeVerificacao string
	concluidoEm      time.Time
}

func (p *Publicador) resultado(publicacao worker.Publicacao) resultado {
	return resultado{
		Publicacao:       publicacao,
		urlDeVerificacao: strings.TrimRight(p.opcoes.URLPublica, "/") + "/receipts/" + publicacao.Recibo.ID + "/verify",
		concluidoEm:      p.opcoes.Agora(),
	}
}

func (p *Publicador) checkRunEmAndamento(taskRun domain.TaskRun) platform.CheckRun {
	return platform.CheckRun{
		Situacao:   platform.CheckRunEmAndamento,
		IniciadoEm: p.opcoes.Agora(),
		Titulo:     "Análise em andamento",
		Resumo:     "O aPRova está analisando o commit " + codigo(shaCurto(taskRun.SHAHead)) + ": testes no sandbox e regras de risco.",
	}
}

func (p *Publicador) checkRunDeFalha(taskRun domain.TaskRun, diagnostico string) platform.CheckRun {
	return platform.CheckRun{
		Situacao:    platform.CheckRunConcluido,
		Conclusao:   platform.ConclusaoFalha,
		ConcluidoEm: p.opcoes.Agora(),
		Titulo:      "A análise não terminou: merge bloqueado",
		Resumo: "O aPRova não conseguiu concluir a análise do commit " + codigo(shaCurto(taskRun.SHAHead)) + ".\n\n" +
			"**Diagnóstico:** " + codigo(cortar(diagnostico, tamanhoMaximoDoDiagnostico)) + "\n\n" +
			"Sem análise concluída, o merge fica bloqueado: na dúvida, o aPRova bloqueia. O caminho é uma nova análise, que um novo push no pull request dispara.",
	}
}

func (r resultado) aprovado() bool {
	return r.Gate.Nivel == domain.RiscoAlto && r.AprovadoPor != ""
}

func (r resultado) conclusao() platform.ConclusaoCheckRun {
	if r.Gate.Nivel == domain.RiscoAlto && !r.aprovado() {
		return platform.ConclusaoAcaoNecessaria
	}
	return platform.ConclusaoSucesso
}

func (r resultado) checkRun() platform.CheckRun {
	return platform.CheckRun{
		Situacao:    platform.CheckRunConcluido,
		Conclusao:   r.conclusao(),
		URLDetalhes: r.urlDeVerificacao,
		ConcluidoEm: r.concluidoEm,
		Titulo:      r.titulo(),
		Resumo:      r.explicacao() + "\n\n" + r.secoes(),
	}
}

func (r resultado) titulo() string {
	nivel := nomeDoNivel[r.Gate.Nivel]
	switch {
	case r.aprovado():
		return "Risco alto: aprovado por @" + r.AprovadoPor
	case r.Gate.Nivel == domain.RiscoAlto:
		return "Risco alto: merge bloqueado até aprovação humana"
	case len(r.Gate.Motivos) > 0:
		return "Risco " + nivel + ": liberado, com pontos de atenção"
	default:
		return "Risco " + nivel + ": liberado"
	}
}

func (r resultado) explicacao() string {
	switch {
	case r.aprovado():
		return "O merge foi liberado pela aprovação de @" + r.AprovadoPor + ", que tem permissão de escrita no repositório."
	case r.Gate.Nivel == domain.RiscoAlto:
		return "Com risco alto, o merge está bloqueado até aprovação humana: alguém com permissão de escrita no repositório precisa aprovar este pull request."
	case len(r.Gate.Motivos) > 0:
		return "O merge está liberado, com pontos de atenção que valem uma olhada antes do merge."
	default:
		return "O merge está liberado: nenhuma regra do gate foi acionada."
	}
}

func (r resultado) comentario() string {
	cabecalho := "### aPRova: risco " + nomeDoNivel[r.Gate.Nivel]
	if r.aprovado() {
		cabecalho += ", aprovado por @" + r.AprovadoPor
	}

	partes := []string{
		marcaDoComentario,
		cabecalho,
		r.explicacao(),
		"**Commit analisado:** " + codigo(shaCurto(r.TaskRun.SHAHead)),
		r.secoes(),
		"<sub>O aPRova nunca faz merge. Ele publica este resultado e o check **" + NomeDoCheckRun + "**; quem decide se o merge pode acontecer é a proteção de branch do repositório. A explicação acima é montada a partir das regras do gate, sem modelo de IA.</sub>",
	}
	return strings.Join(partes, "\n\n")
}

func (r resultado) secoes() string {
	return strings.Join([]string{r.secaoDosMotivos(), r.secaoDoSandbox(), r.secaoDoRecibo()}, "\n\n")
}

func (r resultado) secaoDosMotivos() string {
	if len(r.Gate.Motivos) == 0 {
		return "#### Motivos\nNenhuma regra do gate foi acionada."
	}

	linhas := []string{"#### Motivos"}
	for _, motivo := range r.Gate.Motivos {
		linhas = append(linhas, "- "+codigo(motivo.ID)+" (risco "+nomeDoNivel[motivo.Nivel]+"): "+motivo.Descricao)
		linhas = append(linhas, linhasDasEvidencias(motivo.Evidencias)...)
	}
	return strings.Join(linhas, "\n")
}

func linhasDasEvidencias(evidencias []string) []string {
	mostradas := evidencias[:min(len(evidencias), maximoDeEvidencias)]
	linhas := make([]string, 0, len(mostradas)+1)
	for _, evidencia := range mostradas {
		linhas = append(linhas, "  - "+codigo(cortar(evidencia, tamanhoMaximoDaEvidencia)))
	}
	if restantes := len(evidencias) - len(mostradas); restantes > 0 {
		linhas = append(linhas, fmt.Sprintf("  - e mais %d evidências, no recibo", restantes))
	}
	return linhas
}

func (r resultado) secaoDoSandbox() string {
	return "#### Sandbox\n" + resumoDoSandbox(r.Sandbox)
}

func resumoDoSandbox(resultado sandbox.ResultadoSandbox) string {
	switch {
	case resultado.TipoFalha == sandbox.FalhaInfraestrutura:
		return "O sandbox não conseguiu executar os testes (falha de infraestrutura)."
	case !resultado.Compilou:
		return "O código não compilou no sandbox."
	default:
		return "O código compilou: " + testesQuePassaram(resultado.TestesPassaram) + " e " + testesQueFalharam(resultado.TestesFalharam) + "."
	}
}

func testesQuePassaram(quantos int) string {
	if quantos == 1 {
		return "1 teste passou"
	}
	return fmt.Sprintf("%d testes passaram", quantos)
}

func testesQueFalharam(quantos int) string {
	switch quantos {
	case 0:
		return "nenhum falhou"
	case 1:
		return "1 teste falhou"
	default:
		return fmt.Sprintf("%d testes falharam", quantos)
	}
}

func (r resultado) secaoDoRecibo() string {
	return strings.Join([]string{
		"#### Recibo assinado",
		"- Hash SHA-256: " + codigo(hex.EncodeToString(r.Recibo.Hash)),
		"- key_id: " + codigo(r.Recibo.IDChave),
		"- [Verificar no aPRova](" + r.urlDeVerificacao + ")",
		"- [Verificar por conta própria, só com a chave pública](" + URLDaVerificacaoIndependente + ")",
	}, "\n")
}

func codigo(texto string) string {
	cerca := strings.Repeat("`", maiorSequenciaDeCrases(texto)+1)
	if len(cerca) == 1 {
		return cerca + texto + cerca
	}
	return cerca + " " + texto + " " + cerca
}

func maiorSequenciaDeCrases(texto string) int {
	maior, atual := 0, 0
	for _, caractere := range texto {
		if caractere != '`' {
			atual = 0
			continue
		}
		atual++
		maior = max(maior, atual)
	}
	return maior
}

func cortar(texto string, maximo int) string {
	texto = strings.ReplaceAll(texto, "\n", " ")
	runas := []rune(texto)
	if len(runas) <= maximo {
		return texto
	}
	return string(runas[:maximo]) + "…"
}

func shaCurto(sha string) string {
	return sha[:min(len(sha), tamanhoDoSHACurto)]
}

package explicacao

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	tamanhoMinimo = 120
	tamanhoMaximo = 900

	motivoDeAreaSensivel = "area_sensivel"
	raizDeSeguranca      = "segur"
)

// ErrDescartada indica uma resposta do modelo que não pode ser publicada. A
// mensagem diz qual verificação falhou.
var ErrDescartada = errors.New("explicacao: resposta descartada")

var trechosProibidos = []string{"http", "www.", "](", "<", ">", "@", "```", "$("}

var garantiasProibidas = []string{"sem riscos", "sem problemas", "nenhuma ação", "pode seguir"}

var (
	referenciaDoGitHub = regexp.MustCompile(`#\d`)
	comandoDeTerminal  = regexp.MustCompile(`(?i)\b(git|curl|wget|sudo|npm|npx|pip|chmod)\s+[a-z-]|\brm\s+-`)
	nivelCitado        = regexp.MustCompile(`(?i)\b(?:risco|n[ií]vel)(?:\s+(?:é|e|foi|ficou|está|esta|de|do|como|considerado|classificado|reduzido|rebaixado|para|a|em))*\s+(baixo|m[eé]dio|alto)\b`)
	idDeMotivo         = regexp.MustCompile(`\b[a-z]+(?:_[a-z]+)+\b`)
	segurancaDaArea    = regexp.MustCompile(`(?i)\b(?:de|da)\s+segurança`)
	caminhoDeArquivo   = regexp.MustCompile(`[A-Za-z0-9_][A-Za-z0-9_./-]*(?:/[A-Za-z0-9_.-]+|\.[A-Za-z][A-Za-z0-9]{0,7})`)
)

// Validar normaliza a resposta do modelo num parágrafo e confere se ela pode
// ser publicada: tamanho, só texto, nível igual ao decidido e motivos e
// arquivos presentes na entrada. Qualquer falha devolve ErrDescartada com o
// motivo.
func Validar(resposta string, entrada Entrada) (string, error) {
	texto := strings.Join(strings.Fields(resposta), " ")
	for _, conferir := range []func(string, Entrada) string{
		conferirTamanho, conferirSoTexto, conferirGarantia, conferirNivel, conferirMotivos, conferirArquivos,
	} {
		if problema := conferir(texto, entrada); problema != "" {
			return "", fmt.Errorf("%w: %s", ErrDescartada, problema)
		}
	}
	return texto, nil
}

func conferirTamanho(texto string, _ Entrada) string {
	tamanho := utf8.RuneCountInString(texto)
	if tamanho < tamanhoMinimo || tamanho > tamanhoMaximo {
		return fmt.Sprintf("tamanho %d fora de %d a %d caracteres", tamanho, tamanhoMinimo, tamanhoMaximo)
	}
	return ""
}

func conferirSoTexto(texto string, _ Entrada) string {
	minusculo := strings.ToLower(texto)
	for _, proibido := range trechosProibidos {
		if strings.Contains(minusculo, proibido) {
			return fmt.Sprintf("contém %q, que não é texto simples", proibido)
		}
	}
	if referenciaDoGitHub.MatchString(texto) {
		return "contém referência a issue ou pull request"
	}
	if comandoDeTerminal.MatchString(texto) {
		return "contém comando de terminal"
	}
	return ""
}

func conferirGarantia(texto string, entrada Entrada) string {
	minusculo := strings.ToLower(texto)
	for _, garantia := range garantiasProibidas {
		if strings.Contains(minusculo, garantia) {
			return fmt.Sprintf("afirma %q, uma garantia que o aPRova não dá", garantia)
		}
	}
	if temMotivo(entrada, motivoDeAreaSensivel) {
		minusculo = segurancaDaArea.ReplaceAllString(minusculo, " ")
	}
	if strings.Contains(minusculo, raizDeSeguranca) {
		return "afirma segurança, uma garantia que o aPRova não dá"
	}
	return ""
}

func temMotivo(entrada Entrada, id string) bool {
	return slices.ContainsFunc(entrada.Motivos, func(motivo Motivo) bool { return motivo.ID == id })
}

func conferirNivel(texto string, entrada Entrada) string {
	permitidos := []string{semAcento(entrada.Nivel)}
	if entrada.Calibracao != nil {
		permitidos = append(permitidos, semAcento(entrada.NivelPelasRegras))
	}
	for _, citado := range nivelCitado.FindAllStringSubmatch(texto, -1) {
		nivel := semAcento(strings.ToLower(citado[1]))
		if !slices.Contains(permitidos, nivel) {
			return fmt.Sprintf("cita nível %s, o decidido é %s", nivel, entrada.Nivel)
		}
	}
	return ""
}

func conferirMotivos(texto string, entrada Entrada) string {
	conhecidos := make([]string, 0, len(entrada.Motivos))
	for _, motivo := range entrada.Motivos {
		conhecidos = append(conhecidos, motivo.ID)
	}
	semArquivos := caminhoDeArquivo.ReplaceAllString(texto, " ")
	for _, citado := range idDeMotivo.FindAllString(semArquivos, -1) {
		if !slices.Contains(conhecidos, citado) {
			return fmt.Sprintf("cita o motivo %s, que não está no rules_hit", citado)
		}
	}
	return ""
}

func conferirArquivos(texto string, entrada Entrada) string {
	arquivos := arquivosDasEvidencias(entrada)
	for _, citado := range caminhoDeArquivo.FindAllString(texto, -1) {
		if !arquivoConhecido(arquivos, citado) {
			return fmt.Sprintf("cita o arquivo %s, que não está nas evidências", citado)
		}
	}
	return ""
}

func arquivosDasEvidencias(entrada Entrada) []string {
	var arquivos []string
	for _, motivo := range entrada.Motivos {
		for _, evidencia := range motivo.Evidencias {
			arquivo, _, _ := strings.Cut(evidencia, ": ")
			arquivos = append(arquivos, arquivo)
		}
	}
	return arquivos
}

func arquivoConhecido(arquivos []string, citado string) bool {
	for _, arquivo := range arquivos {
		if arquivo == citado || path.Base(arquivo) == citado || strings.HasSuffix(arquivo, "/"+citado) {
			return true
		}
	}
	return false
}

func semAcento(nivel string) string {
	return strings.NewReplacer("é", "e", "É", "e").Replace(nivel)
}

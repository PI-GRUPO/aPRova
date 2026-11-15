// Package sarif monta o resultado do gate em SARIF 2.1.0 e o envia ao code
// scanning do GitHub. É uma saída a mais: lê a decisão já gravada, nunca chama
// o gate, nunca muda o nível e nunca interfere no estado da análise.
package sarif

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/platform"
)

const (
	enderecoDoSchema   = "https://json.schemastore.org/sarif-2.1.0.json"
	versaoDoFormato    = "2.1.0"
	nomeDaFerramenta   = "aPRova"
	enderecoDoProjeto  = "https://github.com/PI-GRUPO/aPRova"
	chaveDaImpressao   = "primaryLocationLineHash"
	separadorDaChave   = "\x00"
	razaoSemArquivo    = "sem_arquivo"
	razaoRemovido      = "arquivo_removido"
	razaoDesconhecido  = "motivo_desconhecido"
	razaoNivelSemAviso = "nivel_sem_alerta"
)

// Recibo identifica o recibo assinado da análise: o id e o SHA-256 do
// conteúdo assinado.
type Recibo struct {
	ID   string
	Hash []byte
}

// Entrada é o que o documento SARIF descreve. Gate é a decisão gravada em
// risk_gates, de onde saem motivo, nível e arquivo. Arquivos são os arquivos
// do pull request lidos no GitHub e servem só para escolher a linha; nulo
// deixa todo resultado na linha 1. Versao é a versão do binário.
type Entrada struct {
	Gate     domain.GateDeRisco
	Arquivos []platform.ArquivoAlterado
	Recibo   Recibo
	Versao   string
}

type texto struct {
	Text string `json:"text"`
}

type configuracaoPadrao struct {
	Level string `json:"level"`
}

type descritor struct {
	ID                   string             `json:"id"`
	ShortDescription     texto              `json:"shortDescription"`
	FullDescription      texto              `json:"fullDescription"`
	Help                 texto              `json:"help"`
	DefaultConfiguration configuracaoPadrao `json:"defaultConfiguration"`
}

type driver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []descritor `json:"rules"`
}

type ferramenta struct {
	Driver driver `json:"driver"`
}

type localDoArtefato struct {
	URI string `json:"uri"`
}

type regiao struct {
	StartLine int `json:"startLine"`
}

type localFisico struct {
	ArtifactLocation localDoArtefato `json:"artifactLocation"`
	Region           regiao          `json:"region"`
}

type localizacao struct {
	PhysicalLocation localFisico `json:"physicalLocation"`
}

type reciboDoResultado struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
}

type propriedadesDoResultado struct {
	Recibo reciboDoResultado `json:"recibo"`
}

type resultado struct {
	RuleID              string                  `json:"ruleId"`
	RuleIndex           int                     `json:"ruleIndex"`
	Level               string                  `json:"level"`
	Message             texto                   `json:"message"`
	Locations           []localizacao           `json:"locations"`
	PartialFingerprints map[string]string       `json:"partialFingerprints"`
	Properties          propriedadesDoResultado `json:"properties"`
}

type calibracaoDoRun struct {
	Estado        domain.EstadoCalibracao `json:"estado"`
	ReduziuONivel bool                    `json:"reduziuONivel"`
}

type motivoSemResultado struct {
	ID         string            `json:"id"`
	Razao      string            `json:"razao"`
	Nivel      domain.NivelRisco `json:"nivel"`
	Descricao  string            `json:"descricao"`
	Evidencias []string          `json:"evidencias"`
}

type analiseDoRun struct {
	NivelDeterministico domain.NivelRisco    `json:"nivelDeterministico"`
	Nivel               domain.NivelRisco    `json:"nivel"`
	Calibracao          calibracaoDoRun      `json:"calibracao"`
	MotivosSemResultado []motivoSemResultado `json:"motivosSemResultado"`
}

type propriedadesDoRun struct {
	Aprova analiseDoRun `json:"aprova"`
}

type run struct {
	Tool       ferramenta        `json:"tool"`
	Results    []resultado       `json:"results"`
	Properties propriedadesDoRun `json:"properties"`
}

type documento struct {
	Schema  string `json:"$schema"`
	Version string `json:"version"`
	Runs    []run  `json:"runs"`
}

type montagem struct {
	entrada    Entrada
	arquivos   arquivosDoPR
	resultados []resultado
	fora       []motivoSemResultado
}

// Montar serializa a decisão gravada como um documento SARIF 2.1.0 com um run
// só. Motivo de nível alto vira warning e médio vira note; nenhum vira error.
// A mesma entrada produz sempre os mesmos bytes, em qualquer ordem de
// Arquivos.
func Montar(entrada Entrada) ([]byte, error) {
	m := &montagem{entrada: entrada, arquivos: indexar(entrada.Arquivos), resultados: []resultado{}, fora: []motivoSemResultado{}}
	for _, motivo := range motivosOrdenados(entrada.Gate.Motivos) {
		m.acrescentar(motivo)
	}

	return serializar(documento{
		Schema:  enderecoDoSchema,
		Version: versaoDoFormato,
		Runs: []run{{
			Tool:    ferramenta{Driver: driver{Name: nomeDaFerramenta, Version: entrada.Versao, InformationURI: enderecoDoProjeto, Rules: descritores()}},
			Results: m.resultados,
			Properties: propriedadesDoRun{Aprova: analiseDoRun{
				NivelDeterministico: entrada.Gate.NivelDeterministico,
				Nivel:               entrada.Gate.Nivel,
				Calibracao: calibracaoDoRun{
					Estado:        entrada.Gate.Calibracao.Estado,
					ReduziuONivel: entrada.Gate.Nivel != entrada.Gate.NivelDeterministico,
				},
				MotivosSemResultado: m.fora,
			}},
		}},
	})
}

func motivosOrdenados(motivos []domain.MotivoDeRisco) []domain.MotivoDeRisco {
	ordenados := slices.Clone(motivos)
	slices.SortStableFunc(ordenados, func(a, b domain.MotivoDeRisco) int { return strings.Compare(a.ID, b.ID) })
	return ordenados
}

func serializar(doc documento) ([]byte, error) {
	var saida bytes.Buffer
	codificador := json.NewEncoder(&saida)
	codificador.SetEscapeHTML(false)
	if err := codificador.Encode(doc); err != nil {
		return nil, fmt.Errorf("serializar o SARIF: %w", err)
	}
	return bytes.TrimRight(saida.Bytes(), "\n"), nil
}

func (m *montagem) acrescentar(motivo domain.MotivoDeRisco) {
	if motivo.ID == motivoCalibracaoHistorico {
		return
	}

	indice, regra, conhecida := regraPorID(motivo.ID)
	nivel, temNivel := niveisDoSARIF[motivo.Nivel]
	switch {
	case !conhecida:
		m.deixarFora(motivo, razaoDesconhecido, motivo.Evidencias)
	case regra.alcance == alcanceSemArquivo:
		m.deixarFora(motivo, razaoSemArquivo, motivo.Evidencias)
	case !temNivel:
		m.deixarFora(motivo, razaoNivelSemAviso, motivo.Evidencias)
	case regra.alcance == alcancePullRequest:
		m.acrescentarDoPullRequest(motivo, indice, nivel)
	default:
		m.acrescentarPorArquivo(motivo, indice, nivel)
	}
}

func (m *montagem) deixarFora(motivo domain.MotivoDeRisco, razao string, evidencias []string) {
	m.fora = append(m.fora, motivoSemResultado{
		ID: motivo.ID, Razao: razao, Nivel: motivo.Nivel, Descricao: motivo.Descricao, Evidencias: slices.Clone(evidencias),
	})
}

func (m *montagem) arquivosDoMotivo(motivo domain.MotivoDeRisco) (presentes, removidos []string) {
	for _, evidencia := range motivo.Evidencias {
		nome := arquivoDaEvidencia(motivo.ID, evidencia)
		if m.arquivos.removido(nome) {
			removidos = append(removidos, nome)
			continue
		}
		presentes = append(presentes, nome)
	}
	slices.Sort(presentes)
	slices.Sort(removidos)
	return slices.Compact(presentes), slices.Compact(removidos)
}

func (m *montagem) acrescentarPorArquivo(motivo domain.MotivoDeRisco, indice int, nivel string) {
	presentes, removidos := m.arquivosDoMotivo(motivo)
	for _, nome := range presentes {
		m.resultados = append(m.resultados, m.resultadoEm(motivo, indice, nivel, nome, motivo.Descricao, impressao(motivo.ID, nome)))
	}
	if len(removidos) > 0 {
		m.deixarFora(motivo, razaoRemovido, removidos)
	}
}

func (m *montagem) acrescentarDoPullRequest(motivo domain.MotivoDeRisco, indice int, nivel string) {
	presentes, removidos := m.arquivosDoMotivo(motivo)
	if len(presentes) == 0 {
		m.deixarFora(motivo, razaoRemovido, removidos)
		return
	}

	mensagem := fmt.Sprintf("%s. O motivo vale para o pull request inteiro (%d arquivos na evidência); este arquivo é só a âncora do alerta.",
		motivo.Descricao, len(motivo.Evidencias))
	m.resultados = append(m.resultados, m.resultadoEm(motivo, indice, nivel, presentes[0], mensagem, impressao(motivo.ID)))
}

func (m *montagem) resultadoEm(motivo domain.MotivoDeRisco, indice int, nivel, nome, mensagem, impressaoDoAlerta string) resultado {
	return resultado{
		RuleID:    motivo.ID,
		RuleIndex: indice,
		Level:     nivel,
		Message:   texto{Text: mensagem},
		Locations: []localizacao{{PhysicalLocation: localFisico{
			ArtifactLocation: localDoArtefato{URI: (&url.URL{Path: nome}).EscapedPath()},
			Region:           regiao{StartLine: m.arquivos.linha(nome, m.textosPreferidos(motivo.ID))},
		}}},
		PartialFingerprints: map[string]string{chaveDaImpressao: impressaoDoAlerta},
		Properties: propriedadesDoResultado{Recibo: reciboDoResultado{
			ID: m.entrada.Recibo.ID, SHA256: hex.EncodeToString(m.entrada.Recibo.Hash),
		}},
	}
}

func (m *montagem) textosPreferidos(idDoMotivo string) []string {
	if idDoMotivo != motivoDependenciaNova {
		return nil
	}
	nomes := make([]string, 0, len(m.entrada.Gate.DependenciasNovas))
	for _, dependencia := range m.entrada.Gate.DependenciasNovas {
		nomes = append(nomes, dependencia.Nome)
	}
	return nomes
}

func impressao(partes ...string) string {
	soma := sha256.Sum256([]byte(strings.Join(partes, separadorDaChave)))
	return hex.EncodeToString(soma[:])
}

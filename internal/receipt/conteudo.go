// Package receipt monta, assina e verifica o recibo do aPRova: o registro da
// decisão de risco de uma análise, assinado com Ed25519 e verificável só com
// a chave pública. O conteúdo é público; a assinatura impede que seja forjado.
package receipt

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

// VersaoDoFormato é a versão do conteúdo assinado.
const VersaoDoFormato = 1

const formatoDoInstante = "2006-01-02T15:04:05.000000Z"

// ErrConteudoInvalido indica bytes que não são um conteúdo de recibo desta
// versão do formato.
var ErrConteudoInvalido = errors.New("receipt: conteúdo do recibo inválido")

// Conteudo é o que o recibo assina. A ordem dos campos é a ordem dos bytes
// assinados.
type Conteudo struct {
	Versao              int
	ID                  string
	TaskRunID           string
	TenantID            int64
	Repositorio         string
	NumeroPR            int
	SHAHead             string
	DiffSHA256          string
	DiffTruncado        bool
	Sandbox             Sandbox
	NivelDeterministico domain.NivelRisco
	Nivel               domain.NivelRisco
	Motivos             []Motivo
	EmitidoEm           time.Time
	Anterior            *Elo
	KeyID               string
}

// Sandbox é o resultado do sandbox registrado no recibo. Tentativa é a
// tentativa que produziu o resultado.
type Sandbox struct {
	Compilou       bool   `json:"compilou"`
	TestesPassaram int    `json:"testes_passaram"`
	TestesFalharam int    `json:"testes_falharam"`
	TipoFalha      string `json:"tipo_falha"`
	Tentativa      int    `json:"tentativa"`
}

// Motivo é uma regra do gate que disparou, com o que a fez disparar.
type Motivo struct {
	ID         string   `json:"id"`
	Evidencias []string `json:"evidencias"`
}

// Elo aponta para o recibo anterior do mesmo tenant: o id e o SHA-256, em
// hexadecimal, do conteúdo assinado dele.
type Elo struct {
	ID   string `json:"id"`
	Hash string `json:"hash"`
}

type conteudoSerializado struct {
	Versao              int               `json:"versao"`
	ID                  string            `json:"id"`
	TaskRunID           string            `json:"task_run_id"`
	TenantID            int64             `json:"tenant_id"`
	Repositorio         string            `json:"repositorio"`
	NumeroPR            int               `json:"numero_pr"`
	SHAHead             string            `json:"head_sha"`
	DiffSHA256          string            `json:"diff_sha256"`
	DiffTruncado        bool              `json:"diff_truncado"`
	Sandbox             Sandbox           `json:"sandbox"`
	NivelDeterministico domain.NivelRisco `json:"nivel_deterministico"`
	Nivel               domain.NivelRisco `json:"nivel"`
	Motivos             []Motivo          `json:"motivos"`
	EmitidoEm           string            `json:"emitido_em"`
	Anterior            *Elo              `json:"anterior"`
	KeyID               string            `json:"key_id"`
}

// Serializar devolve os bytes que são assinados: JSON em UTF-8, com os campos
// em ordem fixa e sem espaços. O mesmo conteúdo dá sempre os mesmos bytes.
func (c Conteudo) Serializar() ([]byte, error) {
	var saida bytes.Buffer
	codificador := json.NewEncoder(&saida)
	codificador.SetEscapeHTML(false)

	if err := codificador.Encode(c.serializavel()); err != nil {
		return nil, fmt.Errorf("serializar conteúdo do recibo: %w", err)
	}
	return bytes.TrimSuffix(saida.Bytes(), []byte("\n")), nil
}

func (c Conteudo) serializavel() conteudoSerializado {
	motivos := make([]Motivo, 0, len(c.Motivos))
	for _, motivo := range c.Motivos {
		evidencias := motivo.Evidencias
		if evidencias == nil {
			evidencias = []string{}
		}
		motivos = append(motivos, Motivo{ID: motivo.ID, Evidencias: evidencias})
	}

	return conteudoSerializado{
		Versao: c.Versao, ID: c.ID, TaskRunID: c.TaskRunID, TenantID: c.TenantID,
		Repositorio: c.Repositorio, NumeroPR: c.NumeroPR, SHAHead: c.SHAHead,
		DiffSHA256: c.DiffSHA256, DiffTruncado: c.DiffTruncado, Sandbox: c.Sandbox,
		NivelDeterministico: c.NivelDeterministico, Nivel: c.Nivel, Motivos: motivos,
		EmitidoEm: c.EmitidoEm.UTC().Format(formatoDoInstante), Anterior: c.Anterior, KeyID: c.KeyID,
	}
}

// LerConteudo interpreta os bytes assinados de um recibo. Versão desconhecida,
// campo fora do formato ou qualquer coisa depois do objeto devolvem
// ErrConteudoInvalido.
func LerConteudo(bruto []byte) (Conteudo, error) {
	var lido conteudoSerializado
	decodificador := json.NewDecoder(bytes.NewReader(bruto))
	decodificador.DisallowUnknownFields()

	if err := decodificador.Decode(&lido); err != nil {
		return Conteudo{}, fmt.Errorf("%w: %w", ErrConteudoInvalido, err)
	}
	if _, err := decodificador.Token(); !errors.Is(err, io.EOF) {
		return Conteudo{}, fmt.Errorf("%w: há dados depois do objeto", ErrConteudoInvalido)
	}
	if lido.Versao != VersaoDoFormato {
		return Conteudo{}, fmt.Errorf("%w: versão %d não é suportada", ErrConteudoInvalido, lido.Versao)
	}

	emitidoEm, err := time.Parse(formatoDoInstante, lido.EmitidoEm)
	if err != nil {
		return Conteudo{}, fmt.Errorf("%w: emitido_em fora do formato: %w", ErrConteudoInvalido, err)
	}

	return Conteudo{
		Versao: lido.Versao, ID: lido.ID, TaskRunID: lido.TaskRunID, TenantID: lido.TenantID,
		Repositorio: lido.Repositorio, NumeroPR: lido.NumeroPR, SHAHead: lido.SHAHead,
		DiffSHA256: lido.DiffSHA256, DiffTruncado: lido.DiffTruncado, Sandbox: lido.Sandbox,
		NivelDeterministico: lido.NivelDeterministico, Nivel: lido.Nivel, Motivos: lido.Motivos,
		EmitidoEm: emitidoEm, Anterior: lido.Anterior, KeyID: lido.KeyID,
	}, nil
}

// Hash devolve o SHA-256 dos bytes informados.
func Hash(bruto []byte) []byte {
	resumo := sha256.Sum256(bruto)
	return resumo[:]
}

// HashEmHex devolve o SHA-256 dos bytes informados em hexadecimal minúsculo.
func HashEmHex(bruto []byte) string {
	return hex.EncodeToString(Hash(bruto))
}

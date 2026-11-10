package receipt

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/Azulpasta/aPRova/internal/domain"
)

// Exportado é o recibo como a API o entrega: tudo o que alguém de fora
// precisa para verificá-lo só com a chave pública. PayloadBase64 são os bytes
// exatos que foram assinados; Payload é o mesmo conteúdo, para leitura, e não
// entra na verificação.
type Exportado struct {
	ID               string          `json:"id"`
	TaskRunID        string          `json:"task_run_id"`
	KeyID            string          `json:"key_id"`
	HashSHA256       string          `json:"hash_sha256"`
	PayloadBase64    string          `json:"payload_base64"`
	AssinaturaBase64 string          `json:"assinatura_base64"`
	ReciboAnteriorID *string         `json:"recibo_anterior_id"`
	Payload          json.RawMessage `json:"payload"`
}

// Exportar monta o documento de um recibo guardado.
func Exportar(recibo domain.Recibo) Exportado {
	conteudo := []byte(recibo.Conteudo)

	exportado := Exportado{
		ID:               recibo.ID,
		TaskRunID:        recibo.TaskRunID,
		KeyID:            recibo.IDChave,
		HashSHA256:       hex.EncodeToString(recibo.Hash),
		PayloadBase64:    base64.StdEncoding.EncodeToString(conteudo),
		AssinaturaBase64: base64.StdEncoding.EncodeToString(recibo.Assinatura),
		ReciboAnteriorID: recibo.ReciboAnteriorID,
	}
	if json.Valid(conteudo) {
		exportado.Payload = conteudo
	}
	return exportado
}

// VerificarExportado confere um documento exportado só com as chaves públicas
// do chaveiro, sem banco nem rede, e devolve o conteúdo lido dos bytes
// assinados. O erro fica para documento que nem é JSON; recibo reprovado volta
// no Resultado. O elo com o recibo anterior não é conferido aqui, porque
// depende do recibo anterior.
func VerificarExportado(chaveiro *Chaveiro, documento []byte) (Conteudo, Resultado, error) {
	var exportado Exportado
	if err := json.Unmarshal(documento, &exportado); err != nil {
		return Conteudo{}, Resultado{}, fmt.Errorf("o documento não é o JSON de um recibo: %w", err)
	}

	assinado, falha := exportado.assinado()
	if falha != "" {
		return Conteudo{}, reprovado(FalhaConteudo, falha), nil
	}
	if resultado := chaveiro.Verificar(assinado); !resultado.Valido {
		return Conteudo{}, resultado, nil
	}

	conteudo, err := LerConteudo(assinado.Conteudo)
	if err != nil {
		return Conteudo{}, reprovado(FalhaConteudo, err.Error()), nil
	}
	if conteudo.ID != exportado.ID {
		return Conteudo{}, reprovado(FalhaConteudo, "o id do documento não é o id assinado"), nil
	}
	return conteudo, Resultado{Valido: true}, nil
}

func (e Exportado) assinado() (Assinado, string) {
	conteudo, err := base64.StdEncoding.DecodeString(e.PayloadBase64)
	if err != nil {
		return Assinado{}, "payload_base64 não está em base64"
	}
	assinatura, err := base64.StdEncoding.DecodeString(e.AssinaturaBase64)
	if err != nil {
		return Assinado{}, "assinatura_base64 não está em base64"
	}
	hash, err := hex.DecodeString(e.HashSHA256)
	if err != nil {
		return Assinado{}, "hash_sha256 não está em hexadecimal"
	}
	return Assinado{Conteudo: conteudo, Hash: hash, Assinatura: assinatura, IDChave: e.KeyID}, ""
}

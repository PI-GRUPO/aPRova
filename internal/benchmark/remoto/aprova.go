package remoto

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// APRova lê os recibos da API local do aPRova.
type APRova struct {
	URLBase string
	Cliente *http.Client
}

// Recibo é a decisão registrada no recibo assinado.
type Recibo struct {
	ID        string
	TaskRunID string
	Hash      string
	Nivel     string
	Motivos   []string
}

// Recibo busca o recibo pelo id e devolve o nível, os motivos, o task_run e
// o hash.
func (a APRova) Recibo(ctx context.Context, id string) (Recibo, error) {
	endereco := strings.TrimRight(a.URLBase, "/") + "/receipts/" + url.PathEscape(id)
	pedido, err := http.NewRequestWithContext(ctx, http.MethodGet, endereco, nil)
	if err != nil {
		return Recibo{}, fmt.Errorf("remoto: montar a consulta do recibo: %w", err)
	}
	resposta, err := a.Cliente.Do(pedido)
	if err != nil {
		return Recibo{}, fmt.Errorf("remoto: buscar o recibo %s: %w", id, err)
	}
	defer func() { _ = resposta.Body.Close() }()
	bruto, err := io.ReadAll(resposta.Body)
	if err != nil {
		return Recibo{}, fmt.Errorf("remoto: ler o recibo %s: %w", id, err)
	}
	if resposta.StatusCode != http.StatusOK {
		return Recibo{}, fmt.Errorf("remoto: o recibo %s devolveu %d: %s", id, resposta.StatusCode, cortar(string(bruto)))
	}

	var exportado struct {
		ID        string `json:"id"`
		TaskRunID string `json:"task_run_id"`
		Hash      string `json:"hash_sha256"`
		Payload   struct {
			Nivel   string `json:"nivel"`
			Motivos []struct {
				ID string `json:"id"`
			} `json:"motivos"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(bruto, &exportado); err != nil {
		return Recibo{}, fmt.Errorf("remoto: recibo %s ilegível: %w", id, err)
	}
	recibo := Recibo{ID: exportado.ID, TaskRunID: exportado.TaskRunID, Hash: exportado.Hash, Nivel: exportado.Payload.Nivel}
	for _, motivo := range exportado.Payload.Motivos {
		recibo.Motivos = append(recibo.Motivos, motivo.ID)
	}
	return recibo, nil
}

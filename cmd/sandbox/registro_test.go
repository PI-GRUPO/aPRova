package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func linhaRegistrada(t *testing.T, saida *bytes.Buffer) map[string]any {
	t.Helper()

	var campos map[string]any
	if err := json.Unmarshal(saida.Bytes(), &campos); err != nil {
		t.Fatalf("linha de log não é JSON: %s", saida.String())
	}
	return campos
}

func TestLoggerDoSandboxLevaOTaskRunIDDoAmbiente(t *testing.T) {
	var saida bytes.Buffer

	loggerDoSandbox(&saida, "0742a9a6-3866-4ddf-8b1a-1b0501aea6d3").Warn("falha ao encerrar processos remanescentes")

	if campos := linhaRegistrada(t, &saida); campos["task_run_id"] != "0742a9a6-3866-4ddf-8b1a-1b0501aea6d3" {
		t.Errorf("toda linha de dentro do contêiner leva o task_run_id: %v", campos)
	}
}

func TestLoggerDoSandboxSemTaskRunIDNaoInventaOCampo(t *testing.T) {
	var saida bytes.Buffer

	loggerDoSandbox(&saida, "").Info("preparo concluído")

	if _, existe := linhaRegistrada(t, &saida)["task_run_id"]; existe {
		t.Error("sem APROVA_TASK_RUN_ID, a linha não leva um task_run_id vazio")
	}
}

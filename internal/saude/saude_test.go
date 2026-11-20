package saude

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func pedir(t *testing.T, manipulador http.HandlerFunc) (int, map[string]any) {
	t.Helper()

	gravador := httptest.NewRecorder()
	manipulador(gravador, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	var corpo map[string]any
	if err := json.Unmarshal(gravador.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("resposta não é JSON: %s", gravador.Body.String())
	}
	return gravador.Code, corpo
}

func TestVivaRespondeOkSemConsultarNada(t *testing.T) {
	status, corpo := pedir(t, Viva)

	if status != http.StatusOK || corpo["status"] != "ok" {
		t.Errorf("o processo de pé responde 200 ok, obtive %d %v", status, corpo)
	}
}

func TestProntaRespondeOkComTodasAsDependenciasDisponiveis(t *testing.T) {
	pronta := Pronta([]Dependencia{{Nome: "postgres", Verificar: func(context.Context) error { return nil }}})

	status, corpo := pedir(t, pronta)

	if status != http.StatusOK || corpo["status"] != "ok" {
		t.Errorf("esperava 200 ok, obtive %d %v", status, corpo)
	}
}

func TestProntaFalhaENomeiaADependenciaFora(t *testing.T) {
	pronta := Pronta([]Dependencia{
		{Nome: "postgres", Verificar: func(context.Context) error { return errors.New("banco fora") }},
		{Nome: "docker", Verificar: func(context.Context) error { return nil }},
	})

	status, corpo := pedir(t, pronta)

	dependencias, _ := corpo["dependencias"].(map[string]any)
	if status != http.StatusServiceUnavailable || dependencias["postgres"] != "indisponivel" || dependencias["docker"] != "ok" {
		t.Errorf("esperava 503 com postgres indisponível e docker ok, obtive %d %v", status, corpo)
	}
}

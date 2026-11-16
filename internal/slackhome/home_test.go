package slackhome

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/PI-GRUPO/aPRova/internal/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fonteTeste struct {
	falha   bool
	filtros []domain.FiltroAnalises
}

func (f *fonteTeste) ListarAnalises(_ context.Context, filtro domain.FiltroAnalises) ([]domain.TaskRun, error) {
	f.filtros = append(f.filtros, filtro)
	if f.falha {
		return nil, errors.New("segredo")
	}
	return nil, nil
}
func TestEstadosEEscopo(t *testing.T) {
	for _, falha := range []bool{false, true} {
		f := &fonteTeste{falha: falha}
		v := Construir(context.Background(), Config{InstallationID: 42, AppID: "A123", TeamID: "T123"}, f, time.Now())
		b, _ := json.Marshal(v)
		s := string(b)
		if len(f.filtros) != 2 || f.filtros[0].TenantID != 42 || f.filtros[0].Estado != "aguardando_aprovacao" || f.filtros[1].Estado != "todos" {
			t.Fatal(f.filtros)
		}
		for _, proibido := range []string{"demo", "example.com", "segredo", "action_id", "private_metadata"} {
			if strings.Contains(s, proibido) {
				t.Fatal(proibido)
			}
		}
		if falha && (strings.Contains(s, "Nenhuma análise") || !strings.Contains(s, "Não foi possível")) {
			t.Fatal(s)
		}
		if !falha && !strings.Contains(s, "primeiras análises") {
			t.Fatal(s)
		}
	}
}
func TestPublicacao(t *testing.T) {
	for _, cenario := range []string{"ok", "workspace", "recusa", "http", "invalido"} {
		t.Run(cenario, func(t *testing.T) {
			chamadas := 0
			servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				chamadas++
				if r.Header.Get("Authorization") != "Bearer token-teste" {
					t.Error("autenticação ausente")
				}
				if r.URL.Path == "/auth.test" {
					team := "T123"
					if cenario == "workspace" {
						team = "T999"
					}
					json.NewEncoder(w).Encode(map[string]any{"ok": true, "team_id": team, "bot_id": "B123"})
					return
				}
				if r.URL.Path != "/views.publish" {
					t.Error(r.URL.Path)
				}
				var p struct {
					UserID string `json:"user_id"`
					View   View   `json:"view"`
				}
				if json.NewDecoder(r.Body).Decode(&p) != nil || p.UserID != "U123" || p.View.Type != "home" {
					t.Error("payload inválido")
				}
				if cenario == "http" {
					w.WriteHeader(429)
					return
				}
				if cenario == "invalido" {
					w.Write([]byte("token-teste"))
					return
				}
				json.NewEncoder(w).Encode(map[string]any{"ok": cenario != "recusa", "error": "token-teste"})
			}))
			defer servidor.Close()
			cliente := NovoCliente()
			cliente.baseURL = servidor.URL + "/"
			err := cliente.Publicar(context.Background(), Config{Token: "token-teste", TeamID: "T123", AppID: "A123", UserID: "U123", InstallationID: 42}, &fonteTeste{})
			if (err == nil) != (cenario == "ok") {
				t.Fatalf("erro inesperado: %v", err)
			}
			if err != nil && strings.Contains(err.Error(), "token-teste") {
				t.Fatal("segredo exposto")
			}
			esperado := 2
			if cenario == "workspace" {
				esperado = 1
			}
			if chamadas != esperado {
				t.Fatal(chamadas)
			}
		})
	}
}

type fonteComDados struct{}

func (fonteComDados) ListarAnalises(_ context.Context, f domain.FiltroAnalises) ([]domain.TaskRun, error) {
	itens := make([]domain.TaskRun, 6)
	for i := range itens {
		itens[i] = domain.TaskRun{ID: "id-real", Repositorio: "org/<repo>", NumeroPR: 7, SHAHead: "abc123", Estado: domain.EstadoTaskRun("aguardando_aprovacao")}
	}
	return itens, nil
}
func TestDadosReaisELimite(t *testing.T) {
	v := Construir(context.Background(), Config{InstallationID: 42, TeamID: "T123", AppID: "A123"}, fonteComDados{}, time.Now())
	b, _ := json.Marshal(v)
	s := string(b)
	if strings.Count(s, "id-real") != 10 || !strings.Contains(s, "5 mais recentes") || strings.Contains(s, "org/<repo>") {
		t.Fatal(s)
	}
}
func TestConfiguracaoInvalidaImpedeRede(t *testing.T) {
	if err := NovoCliente().Publicar(context.Background(), Config{}, nil); err == nil {
		t.Fatal("configuração vazia aceita")
	}
}

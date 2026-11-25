package main

// Dependências exclusivas do executável E2E. A produção sempre usa GitHub e Redis.
import (
	"context"
	"net/url"
	"sync"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
	"github.com/PI-GRUPO/aPRova/internal/sessaoweb"
)

type armazenamentoSessaoTeste struct {
	mu      sync.Mutex
	valores map[string][]byte
}

func (a *armazenamentoSessaoTeste) Guardar(_ context.Context, id string, v []byte, _ time.Duration) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.valores[id] = v
	return nil
}
func (a *armazenamentoSessaoTeste) Buscar(_ context.Context, id string) ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if v, ok := a.valores[id]; ok {
		return v, nil
	}
	return nil, sessaoweb.ErrSemSessao
}
func (a *armazenamentoSessaoTeste) Consumir(_ context.Context, id string) ([]byte, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if v, ok := a.valores[id]; ok {
		delete(a.valores, id)
		return v, nil
	}
	return nil, sessaoweb.ErrSemSessao
}
func (a *armazenamentoSessaoTeste) Excluir(_ context.Context, id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.valores, id)
	return nil
}

type githubSessaoTeste struct{}

func (githubSessaoTeste) URLAutorizacao(state, _ string) string {
	return "/auth/github/callback?code=teste&state=" + url.QueryEscape(state)
}
func (githubSessaoTeste) Trocar(context.Context, string, string) (sessaoweb.Token, error) {
	return sessaoweb.Token{Valor: "credencial-exclusiva-e2e", Duracao: time.Hour}, nil
}
func (githubSessaoTeste) Usuario(context.Context, string) (sessaoweb.Usuario, error) {
	return sessaoweb.Usuario{ID: 7, Login: "pessoa-e2e"}, nil
}
func (githubSessaoTeste) Instalacoes(context.Context, string) ([]sessaoweb.Instalacao, error) {
	return []sessaoweb.Instalacao{{ID: 42, Conta: "equipe-e2e"}, {ID: 43, Conta: "segunda-equipe"}}, nil
}
func (githubSessaoTeste) Repositorios(_ context.Context, _ string, id int64) ([]string, error) {
	if id == 42 {
		return []string{repositorioDoCenario}, nil
	}
	return []string{"segunda-equipe/projeto"}, nil
}

type relatoriosSessaoTeste struct{}

func (relatoriosSessaoTeste) RelatorioSemanalDosRepositorios(_ context.Context, id int64, _ []string, inicio, fim time.Time) (domain.RelatorioSemanal, error) {
	return domain.RelatorioSemanal{TenantID: id, Inicio: inicio, Fim: fim, Total: 10, Concluidas: 6, Rejeitadas: 1, Falhas: 1, ExigiramAprovacao: 3, RiscoMedio: 2}, nil
}
func sessaoDeTeste(recibos *receipt.Verificador, cadeia *receipt.VerificadorDeCadeia) *sessaoweb.Servidor {
	s, e := sessaoweb.Novo(sessaoweb.Opcoes{Origem: "http://127.0.0.1:18080", Provedor: githubSessaoTeste{}, Armazenamento: &armazenamentoSessaoTeste{valores: map[string][]byte{}}, Relatorios: relatoriosSessaoTeste{}, Analises: analisesSessaoTeste{}, Leituras: leiturasTeste{recibos}, Cadeia: cadeia, Vinculos: map[string]string{"USUARIO-E2E": "pessoa-e2e"}})
	if e != nil {
		panic(e)
	}
	return s
}

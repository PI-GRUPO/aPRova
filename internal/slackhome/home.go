// Package slackhome publica uma Home de leitura para o piloto de workspace único.
package slackhome

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

type Config struct {
	Token, TeamID, AppID, UserID string
	InstallationID               int64
}

func (c Config) Validar() error {
	for _, p := range []struct{ valor, padrao string }{{c.TeamID, `^T[A-Z0-9]+$`}, {c.AppID, `^A[A-Z0-9]+$`}, {c.UserID, `^U[A-Z0-9]+$`}} {
		if !regexp.MustCompile(p.padrao).MatchString(p.valor) {
			return errors.New("configure APROVA_SLACK_TEAM_ID, APROVA_SLACK_APP_ID e APROVA_SLACK_HOME_USER_ID válidos")
		}
	}
	if c.Token == "" || c.InstallationID <= 0 {
		return errors.New("configure APROVA_SLACK_BOT_TOKEN e APROVA_CHAT_INSTALLATION_ID positivo")
	}
	return nil
}

type Fonte interface {
	ListarAnalises(context.Context, domain.FiltroAnalises) ([]domain.TaskRun, error)
}
type View struct {
	Type   string           `json:"type"`
	Blocks []map[string]any `json:"blocks"`
}

func texto(tipo, s string) map[string]any { return map[string]any{"type": tipo, "text": s} }
func bloco(tipo, s string) map[string]any {
	t := "mrkdwn"
	if tipo == "header" {
		t = "plain_text"
	}
	return map[string]any{"type": tipo, "text": texto(t, s)}
}
func escapar(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

// Construir usa apenas a projeção de leitura existente, sem inferir risco ou permissões.
func Construir(ctx context.Context, c Config, fonte Fonte, agora time.Time) View {
	v := View{Type: "home", Blocks: []map[string]any{bloco("header", "aPRova"), bloco("section", "Converse com o aPRova"), bloco("section", "Pergunte sobre análises, pendências e resultados da equipe na aba Mensagens."), bloco("section", "<https://slack.com/app_redirect?"+url.Values{"app": {c.AppID}, "team": {c.TeamID}}.Encode()+"|Conversar com o bot>"), bloco("section", "Experimente: “Tem algum PR aguardando aprovação?” ou “Compare esta semana com a anterior.”"), {"type": "divider"}, bloco("header", "Acompanhamento")}}
	var pendentes, recentes []domain.TaskRun
	var e1, e2 error
	if fonte == nil {
		e1 = errors.New("indisponível")
	} else {
		pendentes, e1 = fonte.ListarAnalises(ctx, domain.FiltroAnalises{TenantID: c.InstallationID, Estado: "aguardando_aprovacao", Limite: 6})
		recentes, e2 = fonte.ListarAnalises(ctx, domain.FiltroAnalises{TenantID: c.InstallationID, Estado: "todos", Limite: 6})
	}
	if e1 != nil || e2 != nil {
		v.Blocks = append(v.Blocks, bloco("section", "*Não foi possível carregar as análises*\nAs pendências não puderam ser confirmadas. Tente novamente mais tarde."))
		return v
	}
	adicionar := func(titulo string, itens []domain.TaskRun, vazio string) {
		v.Blocks = append(v.Blocks, bloco("header", titulo))
		if len(itens) == 0 {
			v.Blocks = append(v.Blocks, bloco("section", vazio))
			return
		}
		for i, a := range itens {
			if i == 5 {
				break
			}
			v.Blocks = append(v.Blocks, bloco("section", fmt.Sprintf("*%s · PR #%d*\nEstado: %s\nAnálise: %s\nCommit: %s", escapar(a.Repositorio), a.NumeroPR, escapar(string(a.Estado)), escapar(a.ID), escapar(a.SHAHead))))
		}
		if len(itens) > 5 {
			v.Blocks = append(v.Blocks, bloco("section", "Exibindo as 5 mais recentes deste recorte. Consulte outras análises pela conversa."))
		}
	}
	adicionar("Aguardando aprovação", pendentes, "Nenhuma análise aguardando aprovação nesta instalação.")
	adicionar("Últimas análises", recentes, "As primeiras análises aparecerão aqui quando forem registradas nesta instalação.")
	v.Blocks = append(v.Blocks, bloco("section", "Consultado em "+agora.UTC().Format(time.RFC3339)+". A lista mostra o estado registrado das análises. Informações de risco e métricas indisponíveis nesta Home."))
	return v
}

// Cliente não registra respostas externas, que podem conter dados sensíveis.
type Cliente struct {
	HTTP    *http.Client
	baseURL string
}

func NovoCliente() *Cliente {
	return &Cliente{HTTP: &http.Client{Timeout: 15 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, baseURL: "https://slack.com/api/"}
}
func (s *Cliente) chamar(ctx context.Context, c Config, metodo string, payload any, destino any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return errors.New("payload Slack inválido")
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+metodo, bytes.NewReader(b))
	if err != nil {
		return errors.New("requisição Slack inválida")
	}
	r.Header.Set("Authorization", "Bearer "+c.Token)
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := s.HTTP.Do(r)
	if err != nil {
		return errors.New("Slack indisponível ou resultado incerto; sem repetição automática")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack respondeu HTTP %d; sem repetição automática", resp.StatusCode)
	}
	if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(destino) != nil {
		return errors.New("resposta Slack inválida; resultado pode ser incerto")
	}
	return nil
}
func (s *Cliente) Publicar(ctx context.Context, c Config, fonte Fonte) error {
	if err := c.Validar(); err != nil {
		return err
	}
	var auth struct {
		OK     bool   `json:"ok"`
		TeamID string `json:"team_id"`
		BotID  string `json:"bot_id"`
	}
	if err := s.chamar(ctx, c, "auth.test", struct{}{}, &auth); err != nil {
		return err
	}
	if !auth.OK || auth.TeamID != c.TeamID || auth.BotID == "" {
		return errors.New("identidade do bot/workspace Slack não confirmada; publicação cancelada")
	}
	v := Construir(ctx, c, fonte, time.Now())
	var resultado struct {
		OK bool `json:"ok"`
	}
	if err := s.chamar(ctx, c, "views.publish", struct {
		UserID string `json:"user_id"`
		View   View   `json:"view"`
	}{c.UserID, v}, &resultado); err != nil {
		return err
	}
	if !resultado.OK {
		return errors.New("Slack recusou views.publish; confira token, usuário e aba Home habilitada")
	}
	return nil
}

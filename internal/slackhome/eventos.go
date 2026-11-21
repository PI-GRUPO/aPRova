package slackhome

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type selecao struct{ Value string }
type viewSlack struct {
	Type, Hash, ID string
	Callback       string `json:"callback_id"`
	Metadata       string `json:"private_metadata"`
	App            string `json:"app_id"`
	Team           string `json:"team_id"`
	State          struct {
		Values map[string]map[string]struct {
			Option selecao `json:"selected_option"`
		}
	}
}
type entrada struct {
	Type    string
	App     string `json:"api_app_id"`
	TeamID  string `json:"team_id"`
	EventID string `json:"event_id"`
	Event   struct {
		Type, User, Tab string
		View            viewSlack
	}
	Team struct{ ID string }
	User struct {
		ID     string
		TeamID string `json:"team_id"`
	}
	View    viewSlack
	Trigger string `json:"trigger_id"`
	Actions []struct {
		ID     string  `json:"action_id"`
		TS     string  `json:"action_ts"`
		Option selecao `json:"selected_option"`
	}
}
type comando struct {
	pessoa                        Pessoa
	chave, hash, trigger, modalID string
	filtro                        Filtro
	abrir                         bool
	aposDecisao                   bool
}

var eventoID = regexp.MustCompile(`^Ev[A-Za-z0-9]+$`)
var actionTS = regexp.MustCompile(`^[0-9]{1,12}\.[0-9]{1,6}$`)

func (e entrada) comando(c Config) (comando, bool) {
	var p comando
	if e.App != c.AppID {
		return p, false
	}
	if e.Type == "event_callback" {
		v := e.Event.View
		if e.TeamID != c.TeamID || !usuarioID.MatchString(e.Event.User) || e.Event.Type != "app_home_opened" || e.Event.Tab != "home" || !eventoID.MatchString(e.EventID) || v.Team != "" && v.Team != c.TeamID || v.App != "" && v.App != c.AppID {
			return p, false
		}
		p = comando{pessoa: Pessoa{e.TeamID, e.Event.User}, chave: e.EventID, hash: v.Hash, filtro: Filtro{Geral, "30"}, abrir: true}
		if filtro, ok := filtroDaView(v.Metadata); v.Callback == "aprova_home" && ok {
			p.filtro = filtro
		}
		return p, true
	}
	if e.Team.ID != c.TeamID || e.User.TeamID != "" && e.User.TeamID != c.TeamID || !usuarioID.MatchString(e.User.ID) || e.View.Team != "" && e.View.Team != c.TeamID || e.View.App != "" && e.View.App != c.AppID {
		return p, false
	}
	p.pessoa = Pessoa{e.Team.ID, e.User.ID}
	switch e.Type {
	case "block_actions":
		if e.View.Type != "home" || e.View.Callback != "aprova_home" || len(e.Actions) != 1 || e.View.Hash == "" {
			return p, false
		}
		filtro, ok := filtroDaView(e.View.Metadata)
		if !ok {
			return p, false
		}
		p.filtro = filtro
		a := e.Actions[0]
		if !actionTS.MatchString(a.TS) {
			return p, false
		}
		p.hash = e.View.Hash
		p.chave = "acao:" + a.TS
		switch a.ID {
		case "aprova_home_secao":
			p.filtro.Secao = Secao(a.Option.Value)
		case "aprova_home_atualizar", "aprova_home_navegar":

			if a.ID == "aprova_home_navegar" {
				if e.Trigger == "" || len(e.Trigger) > 512 {
					return p, false
				}
				p.trigger = e.Trigger
			}
		default:
			return p, false
		}
	case "view_submission":
		switch e.View.Callback {
		case "aprova_aprovacao", "aprova_aprovacao_motivo", "aprova_revisao":
			if e.View.Type != "modal" || e.View.ID == "" {
				return p, false
			}
			p.chave = "decisao:" + e.View.ID
			p.aposDecisao = true
			return p, true
		}
		if e.View.Type != "modal" || e.View.Callback != "aprova_home_navegacao" || e.View.ID == "" || e.View.Hash == "" || e.View.Metadata == "" {
			return p, false
		}
		p.hash = e.View.Metadata
		p.chave = "modal:" + e.View.ID + ":" + e.View.Hash
		p.modalID = e.View.ID
		p.filtro = Filtro{Secao(e.View.State.Values["destino"]["secao"].Option.Value), e.View.State.Values["intervalo"]["periodo"].Option.Value}
	default:
		return p, false
	}
	return p, secaoValida(p.filtro.Secao) && periodoValido(p.filtro.Periodo) && len(p.hash) <= 256
}

func autentico(h http.Header, body []byte, segredo string, agora time.Time) bool {
	ts := h.Get("X-Slack-Request-Timestamp")
	n, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || n < agora.Unix()-300 || n > agora.Unix()+300 {
		return false
	}
	sig := h.Get("X-Slack-Signature")
	if len(sig) != 67 || !strings.HasPrefix(sig, "v0=") {
		return false
	}
	assinatura, err := hex.DecodeString(sig[3:])
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(segredo))
	mac.Write([]byte("v0:" + ts + ":"))
	mac.Write(body)
	return hmac.Equal(mac.Sum(nil), assinatura)
}

// A fila assíncrona é a execução do n8n depois do ACK ao Slack. O adaptador
// devolve o resultado ao workflow, sem confirmar uma escrita de resultado incerto.
type Servidor struct {
	config  Config
	slack   *Cliente
	api     apiPublica
	acesso  AcessoIndividual
	mu      sync.Mutex
	vistos  map[string]time.Time
	pessoas map[Pessoa]*sessaoHome
}
type sessaoHome struct {
	mu        sync.Mutex
	hash      string
	filtro    Filtro
	modalHash string
	modalID   string
	modalAte  time.Time
	expira    time.Time
	usos      int
}

func NovoServidor(ctx context.Context, c Config) (*Servidor, error) {
	if err := c.Validar(); err != nil {
		return nil, err
	}
	c.WebURL = strings.TrimRight(c.WebURL, "/")
	cliente := novoCliente(c.Token)
	if err := cliente.verificar(ctx, c); err != nil {
		return nil, err
	}
	return &Servidor{config: c, slack: cliente, api: apiPublica{strings.TrimRight(c.APIURL, "/"), clienteHTTP(3 * time.Second)}, acesso: acessoPendente{}, vistos: make(map[string]time.Time), pessoas: make(map[Pessoa]*sessaoHome)}, nil
}
func (s *Servidor) reservar(p comando, agora time.Time) (*sessaoHome, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, t := range s.vistos {
		if !agora.Before(t) {
			delete(s.vistos, k)
		}
	}
	for k, v := range s.pessoas {
		if v.usos == 0 && !agora.Before(v.expira) {
			delete(s.pessoas, k)
		}
	}
	k := p.pessoa.Workspace + ":" + p.pessoa.Usuario + ":" + p.chave
	if _, ok := s.vistos[k]; ok {
		return nil, http.StatusNoContent
	}
	if len(s.vistos) >= 10000 {
		return nil, http.StatusServiceUnavailable
	}
	s.vistos[k] = agora.Add(11 * time.Minute)
	sessao := s.pessoas[p.pessoa]
	if sessao == nil {
		sessao = &sessaoHome{}
		s.pessoas[p.pessoa] = sessao
	}
	sessao.usos++
	sessao.expira = agora.Add(11 * time.Minute)
	return sessao, 0
}
func (s *Servidor) processar(ctx context.Context, p comando, sessao *sessaoHome) error {
	sessao.mu.Lock()
	defer sessao.mu.Unlock()
	if p.aposDecisao {
		if sessao.hash == "" {
			return nil
		}
		p.hash, p.filtro = sessao.hash, sessao.filtro
	}
	// Em uma reabertura com hash, o Slack também confere a revisão. Isso permite
	// recuperar uma publicação anterior cuja resposta não chegou ao adaptador.
	if p.abrir && p.hash == "" && sessao.hash != "" {
		p.hash, p.filtro = sessao.hash, sessao.filtro
	}
	if sessao.hash != "" && p.hash != sessao.hash && !p.abrir {
		return errDesatualizada
	}
	if p.trigger != "" {
		if sessao.modalHash == p.hash && time.Now().Before(sessao.modalAte) {
			return nil
		}
		sessao.modalHash, sessao.modalAte = p.hash, time.Now().Add(3*time.Second)
		resp, err := s.slack.chamar(ctx, "views.open", bloco{"trigger_id": p.trigger, "view": modal(p.hash, p.filtro)})
		sessao.modalID = ""
		if err == nil {
			if resp.View.ID == "" {
				return errors.New("abertura do formulário sem ID confirmado")
			}
			sessao.modalID = resp.View.ID
		}
		return err
	}
	if p.modalID != "" {
		if p.modalID != sessao.modalID || p.hash != sessao.modalHash {
			return errDesatualizada
		}
		sessao.modalID = ""
	}
	publicar := func(e EstadoPublico) error {
		hash, err := s.slack.publicar(ctx, p.pessoa, p.hash, construir(ctx, s.config, p.pessoa, p.filtro, s.acesso, e))
		if err == nil {
			sessao.hash = hash
			sessao.filtro = p.filtro
			p.hash = hash
		}
		return err
	}
	if p.abrir && p.hash == "" {
		if err := publicar(EstadoPublico{Carregando: true}); err != nil {
			return err
		}
	}
	return publicar(s.api.consultar(ctx))
}

func (s *Servidor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/slack/home" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.WriteHeader(405)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		w.WriteHeader(413)
		return
	}
	agora := time.Now()
	if !autentico(r.Header, body, s.config.SigningSecret, agora) {
		w.WriteHeader(401)
		return
	}
	var e entrada
	switch strings.TrimSpace(strings.SplitN(r.Header.Get("Content-Type"), ";", 2)[0]) {
	case "application/json":
		err = json.Unmarshal(body, &e)
		if e.Type != "event_callback" {
			w.WriteHeader(400)
			return
		}
	case "application/x-www-form-urlencoded":
		var f url.Values
		f, err = url.ParseQuery(string(body))
		if err == nil && len(f["payload"]) == 1 {
			err = json.Unmarshal([]byte(f.Get("payload")), &e)
		} else {
			w.WriteHeader(400)
			return
		}
		if e.Type == "event_callback" {
			w.WriteHeader(400)
			return
		}
	default:
		w.WriteHeader(415)
		return
	}
	if err != nil {
		w.WriteHeader(400)
		return
	}
	p, ok := e.comando(s.config)
	if !ok {
		w.WriteHeader(403)
		return
	}
	sessao, status := s.reservar(p, agora)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer func() { s.mu.Lock(); sessao.usos--; s.mu.Unlock() }()
	err = s.processar(r.Context(), p, sessao)
	switch {
	case err == nil, errors.Is(err, errDesatualizada):
		w.WriteHeader(204)
	case errors.Is(err, errLimitado):
		w.WriteHeader(429)
	default:
		w.WriteHeader(502)
	}
}

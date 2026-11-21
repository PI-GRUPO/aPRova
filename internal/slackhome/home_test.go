package slackhome

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/api"
)

// Os clientes HTTP usam o transporte regular até a fronteira do servidor. O
// RoundTripper de teste entrega a requisição diretamente ao handler isolado.
type transporte struct{ handler http.Handler }

func (t transporte) RoundTrip(r *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	t.handler.ServeHTTP(w, r)
	return w.Result(), nil
}

type chamada struct {
	Metodo, User, Hash string
	View               View
	Trigger            string
	Corpo              string
}
type ambiente struct {
	s           *Servidor
	chamadas    []chamada
	consultas   []string
	statusSlack int
	erroSlack   string
	statusAPI   int
}

func novoAmbiente(t *testing.T) *ambiente {
	t.Helper()
	c := Config{Token: "bot-teste", TeamID: "T123", AppID: "A123", SigningSecret: "assinatura-teste", APIURL: "https://nucleo.test", WebURL: "https://web.test"}
	a := &ambiente{statusAPI: 200, statusSlack: 200}
	cliente := novoCliente(c.Token)
	cliente.http.Transport = transporte{http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.Header.Get("Authorization") != "Bearer bot-teste" {
			t.Error("autenticação Slack")
		}
		var p struct {
			UserID  string `json:"user_id"`
			Hash    string
			View    View
			Trigger string `json:"trigger_id"`
		}
		b, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(b, &p); err != nil {
			t.Error(err)
		}
		a.chamadas = append(a.chamadas, chamada{r.URL.Path, p.UserID, p.Hash, p.View, p.Trigger, string(b)})
		if a.statusSlack != 200 {
			w.Header().Set("Retry-After", "120")
			w.WriteHeader(a.statusSlack)
			return
		}
		if a.erroSlack != "" {
			json.NewEncoder(w).Encode(bloco{"ok": false, "error": a.erroSlack})
			return
		}
		json.NewEncoder(w).Encode(bloco{"ok": true, "view": bloco{"id": "V123", "hash": fmt.Sprint("h", len(a.chamadas))}})
	})}
	publica := apiPublica{c.APIURL, clienteHTTP(time.Second)}
	publica.http.Transport = transporte{http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.consultas = append(a.consultas, r.URL.RequestURI())
		if r.Method != "GET" || len(r.Header) != 0 {
			t.Error("leitura pública recebeu cabeçalhos ou método indevidos")
		}
		if a.statusAPI != 200 {
			w.WriteHeader(a.statusAPI)
			return
		}
		switch r.URL.Path {
		case "/health":
			fmt.Fprint(w, `{"status":"ok"}`)
		case "/api/sessao":
			fmt.Fprint(w, `{"autenticado":false,"login_disponivel":true}`)
		default:
			t.Error("consulta indevida", r.URL.Path)
		}
	})}
	a.s = &Servidor{config: c, slack: cliente, api: publica, acesso: acessoPendente{}, vistos: map[string]time.Time{}, pessoas: map[Pessoa]*sessaoHome{}}
	return a
}
func evento(id, user string) bloco {
	return bloco{"type": "event_callback", "event_id": id, "team_id": "T123", "api_app_id": "A123", "event": bloco{"type": "app_home_opened", "user": user, "tab": "home"}}
}
func clique(id, ts, hash string) bloco {
	return bloco{"type": "block_actions", "team": bloco{"id": "T123"}, "api_app_id": "A123", "user": bloco{"id": "U111"}, "trigger_id": "trigger", "view": bloco{"type": "home", "callback_id": "aprova_home", "private_metadata": "geral:30", "hash": hash}, "actions": []bloco{{"action_id": id, "action_ts": ts, "selected_option": bloco{"value": "pendencias"}}}}
}
func envio(hash string) bloco {
	return bloco{"type": "view_submission", "team": bloco{"id": "T123"}, "api_app_id": "A123", "user": bloco{"id": "U111"}, "view": bloco{"type": "modal", "id": "V123", "callback_id": "aprova_home_navegacao", "hash": "modal1", "private_metadata": hash, "state": bloco{"values": bloco{"destino": bloco{"secao": bloco{"selected_option": bloco{"value": "historico"}}}, "intervalo": bloco{"periodo": bloco{"selected_option": bloco{"value": "7"}}}}}}}
}
func requisicao(t *testing.T, p bloco, delta time.Duration) *http.Request {
	t.Helper()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	body := string(b)
	tipo := "application/json"
	if p["type"] != "event_callback" {
		body = url.Values{"payload": {body}}.Encode()
		tipo = "application/x-www-form-urlencoded"
	}
	r := httptest.NewRequest("POST", "/slack/home", strings.NewReader(body))
	ts := strconv.FormatInt(time.Now().Add(delta).Unix(), 10)
	mac := hmac.New(sha256.New, []byte("assinatura-teste"))
	fmt.Fprint(mac, "v0:"+ts+":"+body)
	r.Header.Set("Content-Type", tipo)
	r.Header.Set("X-Slack-Request-Timestamp", ts)
	r.Header.Set("X-Slack-Signature", "v0="+hex.EncodeToString(mac.Sum(nil)))
	return r
}
func (a *ambiente) receber(r *http.Request) int {
	w := httptest.NewRecorder()
	a.s.ServeHTTP(w, r)
	return w.Code
}
func (a *ambiente) enviar(t *testing.T, p bloco, esperado int) {
	t.Helper()
	if n := a.receber(requisicao(t, p, 0)); n != esperado {
		t.Fatalf("HTTP %d, esperado %d", n, esperado)
	}
}
func TestAberturaNavegacaoEFormulario(t *testing.T) {
	a := novoAmbiente(t)
	a.enviar(t, evento("Ev1", "U111"), 204)
	if len(a.chamadas) != 2 || a.chamadas[1].User != "U111" || a.chamadas[1].Hash != "h1" || !strings.Contains(a.chamadas[0].Corpo, "Consultando a API") || !strings.Contains(a.chamadas[1].Corpo, "A API está respondendo") {
		t.Fatal(a.chamadas)
	}
	a.enviar(t, clique("aprova_home_secao", "100.1", "h2"), 204)
	if !strings.Contains(a.chamadas[2].Corpo, "estado=aguardando_aprovacao") || a.chamadas[2].View.Metadata != "pendencias:30" {
		t.Fatal(a.chamadas[2])
	}
	a.enviar(t, clique("aprova_home_navegar", "100.2", "h3"), 204)
	if a.chamadas[3].Metodo != "/api/views.open" || a.chamadas[3].Trigger != "trigger" || a.chamadas[3].View.Metadata != "h3" {
		t.Fatal(a.chamadas[3])
	}
	a.enviar(t, clique("aprova_home_navegar", "100.9", "h3"), 204)
	forjado := envio("h3")
	forjado["user"] = bloco{"id": "U222"}
	a.enviar(t, forjado, 204)
	if len(a.chamadas) != 4 {
		t.Fatal("formulário de outro usuário aceito")
	}
	a.enviar(t, envio("h3"), 204)
	if a.chamadas[4].View.Metadata != "historico:7" || !strings.Contains(a.chamadas[4].Corpo, "/analises?periodo=7") || !strings.Contains(a.chamadas[4].Corpo, "/relatorio-semanal") {
		t.Fatal(a.chamadas[4])
	}
	abre := evento("Ev2", "U111")
	abre["event"].(bloco)["view"] = bloco{"type": "home", "callback_id": "aprova_home", "private_metadata": "historico:7", "hash": "h5"}
	a.enviar(t, abre, 204)
	if a.chamadas[5].View.Metadata != "historico:7" {
		t.Fatal(a.chamadas[5])
	}
	atualizar := clique("aprova_home_atualizar", "100.3", "h6")
	atualizar["view"].(bloco)["private_metadata"] = "historico:7"
	a.enviar(t, atualizar, 204)
	if a.chamadas[6].Hash != "h6" || a.chamadas[6].View.Metadata != "historico:7" || len(a.consultas) != 10 {
		t.Fatal(a.chamadas[6], a.consultas)
	}
}
func TestRepeticaoConcorrenciaEUsuarios(t *testing.T) {
	a := novoAmbiente(t)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); a.enviar(t, evento("Ev1", "U111"), 204) }()
	}
	wg.Wait()
	a.enviar(t, clique("aprova_home_secao", "5.1", "h2"), 204)
	a.enviar(t, clique("aprova_home_secao", "5.1", "h2"), 204)
	a.enviar(t, clique("aprova_home_atualizar", "5.2", "h2"), 204)
	a.enviar(t, envio("h2"), 204)
	if len(a.chamadas) != 3 {
		t.Fatal(a.chamadas)
	}
	a.enviar(t, evento("Ev2", "U222"), 204)
	if len(a.chamadas) != 5 || a.chamadas[3].Hash != "" || a.chamadas[4].User != "U222" {
		t.Fatal(a.chamadas)
	}
}
func TestEntradasRecusadasAntesDaRede(t *testing.T) {
	a := novoAmbiente(t)
	for _, mudanca := range []func(bloco){
		func(p bloco) { p["api_app_id"] = "A999" }, func(p bloco) { p["team_id"] = "T999" }, func(p bloco) { p["event"].(bloco)["tab"] = "messages" }, func(p bloco) { p["event"].(bloco)["user"] = "" }, func(p bloco) { p["event"].(bloco)["view"] = bloco{"team_id": "T999"} },
	} {
		p := evento("Ev1", "U111")
		mudanca(p)
		a.enviar(t, p, 403)
	}
	for _, p := range []bloco{clique("aprova_aprovar", "1.1", "h1"), clique("aprova_home_atualizar", "1.1", ""), clique("aprova_home_secao", "1.1", "h1")} {
		p["actions"].([]bloco)[0]["selected_option"] = bloco{"value": "org/privado"}
		a.enviar(t, p, 403)
	}
	p := clique("aprova_home_atualizar", "1.1", "h1")
	p["view"].(bloco)["private_metadata"] = "pendencias:7&installation_id=9"
	a.enviar(t, p, 403)
	p = envio("h1")
	p["view"].(bloco)["state"] = bloco{}
	a.enviar(t, p, 403)
	r := requisicao(t, evento("Ev9", "U111"), 0)
	r.Body = io.NopCloser(strings.NewReader("{}"))
	if a.receber(r) != 401 || a.receber(requisicao(t, evento("Ev9", "U111"), -6*time.Minute)) != 401 || len(a.consultas)+len(a.chamadas) != 0 {
		t.Fatal("entrada não autenticada alcançou um cliente HTTP")
	}
}
func TestFalhasSlackESemRepeticao(t *testing.T) {
	for _, tc := range []struct {
		http      int
		erro      string
		resultado int
	}{{429, "", 429}, {500, "", 502}, {200, "invalid_auth", 502}, {200, "hash_conflict", 204}} {
		a := novoAmbiente(t)
		a.statusSlack = tc.http
		a.erroSlack = tc.erro
		a.enviar(t, evento("Ev1", "U111"), tc.resultado)
		a.enviar(t, evento("Ev1", "U111"), 204)
		if tc.http == 429 {
			a.enviar(t, evento("Ev2", "U111"), 429)
		}
		if len(a.chamadas) != 1 {
			t.Fatal("repetição", a.chamadas)
		}
	}
}
func TestContratosPublicosComHandlersDoNucleo(t *testing.T) {
	a := novoAmbiente(t)
	a.s.api.http.Transport = transporte{api.NovoServidor(api.Opcoes{}).Rotas()}
	e := a.s.api.consultar(context.Background())
	if e.Saude != Disponivel || e.Login != Desabilitado {
		t.Fatal(e)
	}
}

func TestClienteRecusaRespostasIncoerentes(t *testing.T) {
	for _, body := range []string{"invalid", `{"login_disponivel":true}`, `{"autenticado":true,"login_disponivel":true}`, `{}{}`} {
		a := novoAmbiente(t)
		a.s.api.http.Transport = transporte{http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, body) })}
		if e := a.s.api.consultar(context.Background()); e.Login != Desconhecida || e.Saude != Desconhecida {
			t.Fatal(e)
		}
	}
	a := novoAmbiente(t)
	a.statusAPI = 503
	a.enviar(t, evento("Ev1", "U111"), 204)
	view := a.chamadas[1].Corpo
	if !strings.Contains(view, "Falha ao consultar") || strings.Contains(view, "Nenhum item") || strings.Contains(view, "web.test") {
		t.Fatal(view)
	}
}

type consultaTeste func(Pessoa, Recurso) Resultado

func (f consultaTeste) Consultar(_ context.Context, p Pessoa, r Recurso) Resultado { return f(p, r) }
func TestAusenciaRevogacaoEVazioExplicito(t *testing.T) {
	a := novoAmbiente(t)
	for i, tc := range []struct {
		s     Situacao
		texto string
	}{{SemVinculo, "não está vinculada"}, {ListaVazia, "Nenhum item"}, {SemPermissao, "não tem permissão"}, {FalhaConsulta, "Falha ao consultar"}, {SemContrato, "ainda não confirma"}} {
		a.s.acesso = consultaTeste(func(p Pessoa, r Recurso) Resultado { return Resultado{p, r, tc.s} })
		acao := clique("aprova_home_secao", fmt.Sprintf("10.%d", i), fmt.Sprintf("h%d", i))
		a.enviar(t, acao, 204)
		view := a.chamadas[len(a.chamadas)-1].Corpo
		if !strings.Contains(view, tc.texto) || (tc.s != ListaVazia && strings.Contains(view, "Nenhum item")) {
			t.Fatal(view)
		}
	}
	for _, p := range []Pessoa{{"TOUTRO", "U111"}, {"T123", "UOUTRO"}} {
		a.s.acesso = consultaTeste(func(_ Pessoa, r Recurso) Resultado { return Resultado{p, r, ListaVazia} })
		v := construir(context.Background(), a.s.config, Pessoa{"T123", "U111"}, Filtro{Pendencias, "7"}, a.s.acesso, EstadoPublico{})
		b, _ := json.Marshal(v)
		if strings.Contains(string(b), "Nenhum item") || !strings.Contains(string(b), "Falha ao consultar") {
			t.Fatal(string(b))
		}
	}
}
func TestConfigEIdentidadeDoBot(t *testing.T) {
	a := novoAmbiente(t)
	c := a.s.config
	if err := c.Validar(); err != nil {
		t.Fatal(err)
	}
	if _, err := NovoServidor(context.Background(), Config{}); err == nil {
		t.Fatal("configuração inválida")
	}
	for _, raw := range []string{"https://usuario@web.test", "https://web.test/x", "https://web.test?a=b", "http://web.test"} {
		c.WebURL = raw
		if c.Validar() == nil {
			t.Fatal(raw)
		}
	}
	for _, team := range []string{"T123", "T999"} {
		a.s.slack.http.Transport = transporte{http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/auth.test" {
				t.Error(r.URL.Path)
			}
			fmt.Fprintf(w, `{"ok":true,"team_id":%q,"bot_id":"B123"}`, team)
		})}
		if err := a.s.slack.verificar(context.Background(), a.s.config); (team == "T123") != (err == nil) {
			t.Fatal(err)
		}
	}
	a.s.slack.http.Transport = transporteErro{}
	if _, err := a.s.slack.chamar(context.Background(), "views.publish", bloco{}); err == nil || strings.Contains(err.Error(), "privado") {
		t.Fatal(err)
	}
}

type transporteErro struct{}

func (transporteErro) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("privado")
}

func TestAtualizarHomeAposDecisao(t *testing.T) {
	a := novoAmbiente(t)
	decisao := envio("ignorado")
	v := decisao["view"].(bloco)
	v["callback_id"] = "aprova_aprovacao_motivo"
	a.enviar(t, decisao, 204)
	if len(a.chamadas) != 0 {
		t.Fatal("publicação sem Home aberta")
	}
	a.enviar(t, evento("Ev1", "U111"), 204)
	v["id"] = "V2"
	a.enviar(t, decisao, 204)
	a.enviar(t, decisao, 204)
	if len(a.chamadas) != 3 || a.chamadas[2].Hash != "h2" || a.chamadas[2].User != "U111" {
		t.Fatal(a.chamadas)
	}
}

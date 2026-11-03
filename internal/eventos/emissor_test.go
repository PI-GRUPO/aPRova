package eventos

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azulpasta/aPRova/internal/domain"
)

var segredoDoEmissor = []byte("segredo-do-emissor-de-teste")

func pedidoDeExemplo() Pedido {
	return Pedido{
		Tipo:       TipoInstalacao,
		IDEvento:   IDDoEvento("instalacao", "58291734", "instalado"),
		TenantID:   58291734,
		OcorridoEm: time.Date(2026, 10, 18, 20, 3, 15, 0, time.UTC),
		Dados:      DadosInstalacao{Acao: "instalado", Conta: "Azulpasta", Repositorios: []string{"Azulpasta/aPRova-TESTES"}},
	}
}

type recebidoPeloN8N struct {
	corpo      []byte
	timestamp  string
	assinatura string
	tipo       string
}

func n8nDeTeste(t *testing.T, status int, demora time.Duration) (*httptest.Server, <-chan recebidoPeloN8N) {
	t.Helper()

	recebidos := make(chan recebidoPeloN8N, 4)
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corpo, _ := io.ReadAll(r.Body)
		recebidos <- recebidoPeloN8N{corpo: corpo, timestamp: r.Header.Get(CabecalhoTimestamp), assinatura: r.Header.Get(CabecalhoAssinatura), tipo: r.Header.Get("Content-Type")}
		select {
		case <-time.After(demora):
		case <-r.Context().Done():
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(servidor.Close)
	return servidor, recebidos
}

type diarioCapturado struct {
	mutex sync.Mutex
	saida bytes.Buffer
}

func (d *diarioCapturado) Write(p []byte) (int, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.saida.Write(p)
}

func (d *diarioCapturado) texto() string {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.saida.String()
}

func novoDiario() (*slog.Logger, *diarioCapturado) {
	capturado := &diarioCapturado{}
	return slog.New(slog.NewJSONHandler(capturado, nil)).With("task_run_id", "c41d7e9a-2f3b-4c8d-a5e6-0b1c2d3e4f5a"), capturado
}

func TestMontarDevolveUmEventoValidoDoContrato(t *testing.T) {
	corpo, err := Montar(pedidoDeExemplo())
	if err != nil {
		t.Fatalf("montar: %v", err)
	}

	evento, err := Validar(corpo)
	if err != nil {
		t.Fatalf("o evento montado não passa na validação do próprio contrato: %v", err)
	}
	if evento.Versao != VersaoAtual || evento.IDEvento != pedidoDeExemplo().IDEvento || evento.TipoEvento != TipoInstalacao {
		t.Errorf("envelope montado errado: %+v", evento.Envelope)
	}
}

func TestMontarRecusaDadosForaDoContrato(t *testing.T) {
	pedido := pedidoDeExemplo()
	pedido.Tipo = TipoAnaliseConcluida
	pedido.Dados = DadosAnaliseConcluida{NivelRisco: domain.NivelRisco("critico")}

	if _, err := Montar(pedido); !errors.Is(err, ErrEventoInvalido) {
		t.Errorf("esperava ErrEventoInvalido, obtive %v", err)
	}
}

func TestIDDoEventoEDeterministicoPorFato(t *testing.T) {
	primeiro := IDDoEvento("analise_concluida", "c41d7e9a-2f3b-4c8d-a5e6-0b1c2d3e4f5a")

	if repetido := IDDoEvento("analise_concluida", "c41d7e9a-2f3b-4c8d-a5e6-0b1c2d3e4f5a"); repetido != primeiro {
		t.Errorf("o mesmo fato precisa gerar o mesmo id: %s e %s", primeiro, repetido)
	}
	if outro := IDDoEvento("aguardando_aprovacao", "c41d7e9a-2f3b-4c8d-a5e6-0b1c2d3e4f5a"); outro == primeiro {
		t.Error("fatos diferentes precisam gerar ids diferentes")
	}
	if ambiguo := IDDoEvento("analise", "concluida/c41d7e9a"); ambiguo == IDDoEvento("analise/concluida", "c41d7e9a") {
		t.Error("partes diferentes não podem colidir só porque a junção dá o mesmo texto")
	}
}

func TestEmissorEnviaOEventoAssinado(t *testing.T) {
	servidor, recebidos := n8nDeTeste(t, http.StatusOK, 0)
	emissor, err := NovoEmissor(OpcoesEmissor{URL: servidor.URL, Segredo: segredoDoEmissor})
	if err != nil {
		t.Fatalf("novo emissor: %v", err)
	}
	diario, _ := novoDiario()

	emissor.Emitir(diario, pedidoDeExemplo())
	emissor.Aguardar()

	recebido := <-recebidos
	if err := Verificar(segredoDoEmissor, recebido.corpo, recebido.timestamp, recebido.assinatura, time.Now()); err != nil {
		t.Errorf("o n8n recusaria a assinatura: %v", err)
	}
	if _, err := Validar(recebido.corpo); err != nil {
		t.Errorf("o n8n recebeu um evento inválido: %v", err)
	}
	if recebido.tipo != "application/json" {
		t.Errorf("Content-Type = %q", recebido.tipo)
	}
}

func TestEmissorNaoEsperaON8NResponder(t *testing.T) {
	servidor, recebidos := n8nDeTeste(t, http.StatusOK, 5*time.Second)
	emissor, err := NovoEmissor(OpcoesEmissor{URL: servidor.URL, Segredo: segredoDoEmissor, TempoLimite: 300 * time.Millisecond})
	if err != nil {
		t.Fatalf("novo emissor: %v", err)
	}
	diario, capturado := novoDiario()

	inicio := time.Now()
	emissor.Emitir(diario, pedidoDeExemplo())
	if demorou := time.Since(inicio); demorou > 50*time.Millisecond {
		t.Errorf("Emitir segurou quem chamou por %v", demorou)
	}
	<-recebidos

	emissor.Aguardar()
	if demorou := time.Since(inicio); demorou > 2*time.Second {
		t.Errorf("o envio precisa desistir no tempo limite, levou %v", demorou)
	}
	if registro := capturado.texto(); !strings.Contains(registro, `"level":"WARN"`) || !strings.Contains(registro, `"task_run_id"`) {
		t.Errorf("a desistência precisa ir para o log do task_run: %s", registro)
	}
}

func TestEmissorRegistraRespostaQueNaoE2xxSemPropagar(t *testing.T) {
	servidor, _ := n8nDeTeste(t, http.StatusUnauthorized, 0)
	emissor, err := NovoEmissor(OpcoesEmissor{URL: servidor.URL, Segredo: segredoDoEmissor})
	if err != nil {
		t.Fatalf("novo emissor: %v", err)
	}
	diario, capturado := novoDiario()

	emissor.Emitir(diario, pedidoDeExemplo())
	emissor.Aguardar()

	if registro := capturado.texto(); !strings.Contains(registro, "401") || !strings.Contains(registro, string(TipoInstalacao)) {
		t.Errorf("a recusa do n8n precisa ir para o log com o status e o tipo: %s", registro)
	}
}

func TestEmissorFicaDesligadoSemURL(t *testing.T) {
	emissor, err := NovoEmissor(OpcoesEmissor{})
	if err != nil {
		t.Fatalf("sem URL o emissor só desliga: %v", err)
	}
	diario, _ := novoDiario()

	emissor.Emitir(diario, pedidoDeExemplo())
	emissor.Aguardar()

	if emissor.Ligado() {
		t.Error("sem URL o emissor precisa ficar desligado")
	}
}

func TestEmissorComURLESemSegredoNaoSobe(t *testing.T) {
	if _, err := NovoEmissor(OpcoesEmissor{URL: "http://localhost:5978/webhook/aprova"}); !errors.Is(err, ErrSegredoAusente) {
		t.Errorf("enviar sem assinar é inútil e esconde erro de configuração; esperava ErrSegredoAusente, obtive %v", err)
	}
}

func TestEmissorNaoEnviaEventoInvalido(t *testing.T) {
	servidor, recebidos := n8nDeTeste(t, http.StatusOK, 0)
	emissor, err := NovoEmissor(OpcoesEmissor{URL: servidor.URL, Segredo: segredoDoEmissor})
	if err != nil {
		t.Fatalf("novo emissor: %v", err)
	}
	diario, capturado := novoDiario()
	pedido := pedidoDeExemplo()
	pedido.Dados = DadosInstalacao{Acao: "suspenso"}

	emissor.Emitir(diario, pedido)
	emissor.Aguardar()

	select {
	case <-recebidos:
		t.Error("um evento fora do contrato chegou ao n8n")
	default:
	}
	if !strings.Contains(capturado.texto(), `"level":"ERROR"`) {
		t.Errorf("evento inválido é defeito do núcleo e precisa ir para o log como erro: %s", capturado.texto())
	}
}

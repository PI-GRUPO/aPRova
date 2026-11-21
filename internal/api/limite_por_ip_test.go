package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const tetoDeLeiturasDeTeste = 3

type relogioDeTeste struct {
	instante time.Time
}

func (r *relogioDeTeste) agora() time.Time {
	return r.instante
}

func servidorComLimite(proxyConfiavel bool) (http.Handler, *relogioDeTeste) {
	relogio := &relogioDeTeste{instante: time.Date(2026, 11, 20, 14, 0, 0, 0, time.UTC)}
	servidor := NovoServidor(Opcoes{
		TetoDeLeiturasPorMinuto: tetoDeLeiturasDeTeste, ProxyConfiavel: proxyConfiavel, Agora: relogio.agora,
	})
	return servidor.Rotas(), relogio
}

func ler(t *testing.T, rotas http.Handler, caminho, enderecoRemoto, encaminhadoPor string) *httptest.ResponseRecorder {
	t.Helper()

	requisicao := httptest.NewRequestWithContext(t.Context(), http.MethodGet, caminho, nil)
	requisicao.RemoteAddr = enderecoRemoto
	if encaminhadoPor != "" {
		requisicao.Header.Set("X-Forwarded-For", encaminhadoPor)
	}
	gravador := httptest.NewRecorder()
	rotas.ServeHTTP(gravador, requisicao)
	return gravador
}

func esgotar(t *testing.T, rotas http.Handler, enderecoRemoto, encaminhadoPor string) {
	t.Helper()

	for leitura := 1; leitura <= tetoDeLeiturasDeTeste; leitura++ {
		if resposta := ler(t, rotas, "/receipts/keys", enderecoRemoto, encaminhadoPor); resposta.Code == http.StatusTooManyRequests {
			t.Fatalf("a leitura %d ainda está dentro do teto de %d", leitura, tetoDeLeiturasDeTeste)
		}
	}
}

func TestIPAcimaDoTetoRecebe429ComRetryAfter(t *testing.T) {
	rotas, relogio := servidorComLimite(false)
	esgotar(t, rotas, "203.0.113.7:51000", "")
	relogio.instante = relogio.instante.Add(20 * time.Second)

	resposta := ler(t, rotas, "/receipts/keys", "203.0.113.7:51001", "")

	if resposta.Code != http.StatusTooManyRequests {
		t.Fatalf("acima do teto esperava 429, obtive %d", resposta.Code)
	}
	if resposta.Header().Get("Retry-After") != "40" {
		t.Errorf("Retry-After diz quando a janela abre: %q", resposta.Header().Get("Retry-After"))
	}
	if outro := ler(t, rotas, "/receipts/keys", "198.51.100.9:40000", ""); outro.Code == http.StatusTooManyRequests {
		t.Error("o teto de um IP não segura outro")
	}
}

func TestJanelaNovaLiberaOIPDeNovo(t *testing.T) {
	rotas, relogio := servidorComLimite(false)
	esgotar(t, rotas, "203.0.113.7:51000", "")
	relogio.instante = relogio.instante.Add(time.Minute)

	if resposta := ler(t, rotas, "/receipts/keys", "203.0.113.7:51000", ""); resposta.Code == http.StatusTooManyRequests {
		t.Error("depois de um minuto o IP volta a ler")
	}
}

func TestTetoValeParaTodasAsRotasDeRecibo(t *testing.T) {
	rotas, _ := servidorComLimite(false)
	esgotar(t, rotas, "203.0.113.7:51000", "")

	for _, caminho := range []string{
		"/receipts/6d290152-af24-4462-a525-72adf3b22d6c",
		"/receipts/6d290152-af24-4462-a525-72adf3b22d6c/verify",
		"/receipts/6d290152-af24-4462-a525-72adf3b22d6c/revert",
	} {
		if resposta := ler(t, rotas, caminho, "203.0.113.7:51000", ""); resposta.Code != http.StatusTooManyRequests {
			t.Errorf("%s acima do teto esperava 429, obtive %d", caminho, resposta.Code)
		}
	}
	if saude := ler(t, rotas, "/health", "203.0.113.7:51000", ""); saude.Code != http.StatusOK {
		t.Errorf("/health não entra no teto de leituras de recibo, obtive %d", saude.Code)
	}
}

func TestSemProxyConfiavelOXForwardedForEIgnorado(t *testing.T) {
	rotas, _ := servidorComLimite(false)
	esgotar(t, rotas, "203.0.113.7:51000", "192.0.2.1")

	if resposta := ler(t, rotas, "/receipts/keys", "203.0.113.7:51000", "192.0.2.99"); resposta.Code != http.StatusTooManyRequests {
		t.Errorf("trocar o X-Forwarded-For não escapa do teto sem proxy confiável, obtive %d", resposta.Code)
	}
}

func TestComProxyConfiavelValeAEntradaMaisADireita(t *testing.T) {
	rotas, _ := servidorComLimite(true)
	esgotar(t, rotas, "10.0.0.5:443", "192.0.2.1, 203.0.113.7")

	if resposta := ler(t, rotas, "/receipts/keys", "10.0.0.5:443", "198.51.100.200, 203.0.113.7"); resposta.Code != http.StatusTooManyRequests {
		t.Errorf("forjar a primeira entrada não escapa do teto, obtive %d", resposta.Code)
	}
	if outro := ler(t, rotas, "/receipts/keys", "10.0.0.5:443", "203.0.113.8"); outro.Code == http.StatusTooManyRequests {
		t.Error("clientes diferentes atrás do mesmo proxy têm tetos separados")
	}
}

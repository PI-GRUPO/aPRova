package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestNavegacaoEAtivos(t *testing.T) {
	arquivos := fstest.MapFS{"index.html": {Data: []byte("<!doctype html><title>aPRova</title>")}, "assets/app-abc.js": {Data: []byte("export const app = true;")}}
	handler := NovoHandler(arquivos)
	casos := []struct {
		caminho, metodo, accept string
		status                  int
		html                    bool
	}{
		{"/", "GET", "text/html", 200, true},
		{"/analises/123", "GET", "text/html", 200, true},
		{"/conta", "HEAD", "text/html", 200, true},
		{"/desconhecida", "GET", "text/html", 200, true},
		{"/assets/app-abc.js", "GET", "*/*", 200, false},
		{"/assets/ausente.js", "GET", "text/html", 404, false},
		{"/assets/ausente", "GET", "text/html", 404, false},
		{"/api/ausente", "GET", "text/html", 404, false},
		{"/api", "GET", "text/html", 404, false},
		{"/receipts/ausente", "GET", "text/html", 404, false},
		{"/reports/ausente", "GET", "text/html", 404, false},
		{"/webhooks/ausente", "GET", "text/html", 404, false},
		{"/favicon-ausente.ico", "GET", "text/html", 404, false},
		{"/.gitkeep", "GET", "text/html", 404, false},
		{"/assets/../index.html", "GET", "text/html", 404, false},
		{"/assets/%2e%2e/index.html", "GET", "text/html", 404, false},
		{"/assets/%2E%2E%2Findex.html", "GET", "text/html", 404, false},
		{"/assets/%2e/app-abc.js", "GET", "*/*", 404, false},
		{"/assets/%5c..%5cindex.html", "GET", "text/html", 404, false},
		{"/%00", "GET", "text/html", 404, false},
		{"/assets//app-abc.js", "GET", "*/*", 200, false},
		{"/analises", "POST", "text/html", 404, false},
		{"/analises", "GET", "application/json", 404, false},
		{"/analises", "GET", "text/html;q=0", 404, false},
		{"/analises", "GET", "text/html;q=0.0", 404, false},
		{"/analises", "GET", "application/json, text/html;q=0.5", 200, true},
	}
	for _, c := range casos {
		t.Run(c.metodo+c.caminho+c.accept, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), c.metodo, c.caminho, nil)
			req.Header.Set("Accept", c.accept)
			res := httptest.NewRecorder()
			handler.ServeHTTP(res, req)
			if res.Code != c.status {
				t.Fatalf("status = %d; esperado %d", res.Code, c.status)
			}
			if strings.HasPrefix(res.Header().Get("Content-Type"), "text/html") != c.html {
				t.Fatalf("tipo incorreto: %s", res.Header().Get("Content-Type"))
			}
			if c.metodo == http.MethodHead && res.Body.Len() != 0 {
				t.Fatal("HEAD retornou corpo")
			}
			if strings.HasPrefix(c.caminho, "/assets/app-") && !strings.Contains(res.Header().Get("Cache-Control"), "immutable") {
				t.Fatal("ativo compilado sem cache")
			}
			if c.html && res.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("HTML pode ficar obsoleto")
			}
		})
	}
}

func TestCheckoutSemBuild(t *testing.T) {
	h := NovoHandler(fstest.MapFS{".gitkeep": {Data: nil}})
	req := httptest.NewRequestWithContext(t.Context(), "GET", "/analises", nil)
	req.Header.Set("Accept", "text/html")
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", res.Code)
	}
}

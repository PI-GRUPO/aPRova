// Package web serve os ativos estáticos do dashboard, compilados pelo Vite.
package web

import (
	"bytes"
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

// O marcador mantém go build funcional em um checkout sem build Node.
// Sem index.html gerado, a navegação HTML recebe 503.
//
//go:embed all:dist
var compilado embed.FS

// Handler usa o build estático incorporado no binário.
func Handler() http.Handler {
	ativos, _ := fs.Sub(compilado, "dist")
	return NovoHandler(ativos)
}

type servidorEstatico struct {
	ativos   fs.FS
	arquivos http.Handler
}

// NovoHandler recebe um filesystem para permitir testes sem dependências externas.
func NovoHandler(ativos fs.FS) http.Handler {
	return &servidorEstatico{ativos: ativos, arquivos: http.FileServer(http.FS(ativos))}
}

func (s *servidorEstatico) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if (r.Method != http.MethodGet && r.Method != http.MethodHead) || reservado(r.URL.Path) || caminhoInvalido(r.URL.Path) {
		http.NotFound(w, r)
		return
	}
	if s.servirArquivo(w, r) {
		return
	}
	if !navegacaoHTML(r) {
		http.NotFound(w, r)
		return
	}
	s.servirIndice(w, r)
}

func reservado(caminho string) bool {
	for _, prefixo := range []string{"/api", "/receipts", "/reports", "/webhooks", "/acoes-remotas", "/health", "/ready"} {
		if caminho == prefixo || strings.HasPrefix(caminho, prefixo+"/") {
			return true
		}
	}
	return false
}

func caminhoInvalido(caminho string) bool {
	if !strings.HasPrefix(caminho, "/") || strings.ContainsRune(caminho, 0) {
		return true
	}
	for _, parte := range strings.Split(caminho, "/") {
		if strings.HasPrefix(parte, ".") || strings.Contains(parte, "\\") {
			return true
		}
	}
	return false
}

func (s *servidorEstatico) servirArquivo(w http.ResponseWriter, r *http.Request) bool {
	nome := strings.Join(strings.FieldsFunc(r.URL.Path, func(c rune) bool { return c == '/' }), "/")
	if !fs.ValidPath(nome) || nome == "index.html" {
		return false
	}
	info, err := fs.Stat(s.ativos, nome)
	if err != nil || info.IsDir() {
		return false
	}
	if strings.HasPrefix(nome, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	s.arquivos.ServeHTTP(w, r)
	return true
}

func navegacaoHTML(r *http.Request) bool {
	caminho := r.URL.Path
	if strings.HasPrefix(caminho, "/assets/") || (path.Ext(caminho) != "" && caminho != "/index.html") {
		return false
	}
	return aceitaHTML(r.Header.Get("Accept"))
}

func (s *servidorEstatico) servirIndice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Vary", "Accept")
	w.Header().Set("Cache-Control", "no-store")
	documento, err := fs.ReadFile(s.ativos, "index.html")
	if err != nil {
		http.Error(w, "dashboard ainda não compilado; execute npm --prefix web run build", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(documento))
}

func aceitaHTML(accept string) bool {
	for _, item := range strings.Split(accept, ",") {
		tipo, parametros, err := mime.ParseMediaType(strings.TrimSpace(item))
		if err != nil || tipo != "text/html" {
			continue
		}
		if parametros["q"] == "" {
			return true
		}
		qualidade, err := strconv.ParseFloat(parametros["q"], 64)
		if err == nil && qualidade > 0 && qualidade <= 1 {
			return true
		}
	}
	return false
}

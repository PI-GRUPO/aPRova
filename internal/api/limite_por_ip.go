package api

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	janelaDeLeituras     = time.Minute
	tetoDeLeiturasPadrao = 60
)

type respostaDeLimite struct {
	Erro string `json:"erro"`
}

type limitePorIP struct {
	mutex          sync.Mutex
	teto           int
	proxyConfiavel bool
	agora          func() time.Time
	inicioDaJanela time.Time
	leituras       map[string]int
}

func novoLimitePorIP(teto int, proxyConfiavel bool, agora func() time.Time) *limitePorIP {
	if teto <= 0 {
		teto = tetoDeLeiturasPadrao
	}
	return &limitePorIP{teto: teto, proxyConfiavel: proxyConfiavel, agora: agora, leituras: map[string]int{}}
}

func (l *limitePorIP) segurar(proximo http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		espera, permitido := l.contar(l.ipDoCliente(r))
		if !permitido {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(espera.Seconds()))))
			responderJSON(w, r, http.StatusTooManyRequests, respostaDeLimite{Erro: "teto de leituras por minuto atingido para este IP"})
			return
		}
		proximo.ServeHTTP(w, r)
	})
}

func (l *limitePorIP) contar(ip string) (time.Duration, bool) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	agora := l.agora()
	if agora.Sub(l.inicioDaJanela) >= janelaDeLeituras {
		l.inicioDaJanela, l.leituras = agora, map[string]int{}
	}
	if l.leituras[ip] >= l.teto {
		return l.inicioDaJanela.Add(janelaDeLeituras).Sub(agora), false
	}
	l.leituras[ip]++
	return 0, true
}

func (l *limitePorIP) ipDoCliente(r *http.Request) string {
	if l.proxyConfiavel {
		entradas := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
		if ultima := strings.TrimSpace(entradas[len(entradas)-1]); ultima != "" {
			return ultima
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

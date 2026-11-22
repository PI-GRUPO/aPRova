// Servidor isolado de testes: handlers reais e recibo sintético assinado em memória.
// Não é o ponto de entrada de produção e não lê credenciais.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/api"
)

func main() {
	recibos := recibosDeTeste()
	servidor := &http.Server{
		Addr:              "127.0.0.1:18080",
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           api.NovoServidor(api.Opcoes{SessaoWeb: sessaoDeTeste(recibos), Recibos: recibos, Dependencias: []api.Dependencia{{Nome: "postgres", Verificar: func(context.Context) error { return errors.New("dependência indisponível no cenário de teste") }}}}).Rotas(),
	}
	ctx, cancelar := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancelar()
	go func() {
		<-ctx.Done()
		prazo, encerrar := context.WithTimeout(context.Background(), time.Second)
		defer encerrar()
		_ = servidor.Shutdown(prazo)
	}()
	if err := servidor.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

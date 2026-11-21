package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/slackhome"
	"github.com/joho/godotenv"
)

func main() {
	if err := executar(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func executar() error {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.New("não foi possível ler o arquivo de ambiente")
	}
	c := slackhome.Config{Token: os.Getenv("APROVA_SLACK_BOT_TOKEN"), AppID: os.Getenv("APROVA_SLACK_APP_ID"), TeamID: os.Getenv("APROVA_SLACK_TEAM_ID"), SigningSecret: os.Getenv("SLACK_SIGNING_SECRET"), APIURL: os.Getenv("APROVA_API_URL"), WebURL: os.Getenv("WEB_URL_PUBLICA")}
	h, err := slackhome.NovoServidor(context.Background(), c)
	if err != nil {
		return err
	}
	addr := os.Getenv("APROVA_SLACK_HOME_LISTEN")
	if addr == "" {
		addr = "127.0.0.1:8091"
	}
	server := &http.Server{Addr: addr, Handler: h, ReadHeaderTimeout: 3 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 30 * time.Second}
	return server.ListenAndServe()
}

package main

import (
	"context"
	"fmt"
	"github.com/PI-GRUPO/aPRova/internal/config"
	"github.com/PI-GRUPO/aPRova/internal/repository"
	"github.com/PI-GRUPO/aPRova/internal/slackhome"
	"os"
	"strconv"
	"time"
)

func main() {
	if err := executar(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Home publicada para o usuário de teste configurado.")
}
func executar() error {
	c, err := config.Carregar()
	if err != nil {
		return err
	}
	id, _ := strconv.ParseInt(os.Getenv("APROVA_CHAT_INSTALLATION_ID"), 10, 64)
	home := slackhome.Config{Token: os.Getenv("APROVA_SLACK_BOT_TOKEN"), TeamID: os.Getenv("APROVA_SLACK_TEAM_ID"), AppID: os.Getenv("APROVA_SLACK_APP_ID"), UserID: os.Getenv("APROVA_SLACK_HOME_USER_ID"), InstallationID: id}
	if err := home.Validar(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	banco, err := repository.ConectarPostgres(ctx, c.URLBancoDados)
	if err != nil {
		return fmt.Errorf("banco indisponível; publicação cancelada")
	}
	defer banco.Close()
	return slackhome.NovoCliente().Publicar(ctx, home, repository.NovoRepositorioPostgres(banco, repository.OpcoesPostgres{}))
}

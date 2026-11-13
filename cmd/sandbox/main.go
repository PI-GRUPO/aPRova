package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/sandbox"
	"github.com/PI-GRUPO/aPRova/internal/sandbox/conteiner"
	"github.com/PI-GRUPO/aPRova/internal/sandbox/golang"
	"github.com/PI-GRUPO/aPRova/internal/sandbox/processo"
	"github.com/PI-GRUPO/aPRova/internal/sandbox/python"
)

const (
	usuarioDoCodigoDoPR = 10001
	diretorioDeTrabalho = "/tmp/aprova"
	diretorioDeCache    = "/tmp/aprova-cache"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	ctx, encerrar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	codigo := programa().Executar(ctx)
	encerrar()

	os.Exit(codigo)
}

func programa() conteiner.Programa {
	rodador := processo.NovoRodador(processo.Opcoes{
		Credencial: &processo.Credencial{UID: usuarioDoCodigoDoPR, GID: usuarioDoCodigoDoPR},
	})

	return conteiner.Programa{
		LerVariavel:         os.Getenv,
		Saida:               os.Stdout,
		Rodador:             rodador,
		DiretorioDeTrabalho: diretorioDeTrabalho,
		Linguagens: map[sandbox.Linguagem]conteiner.ExecutorDeLinguagem{
			sandbox.LinguagemGo:     golang.Nova(golang.Opcoes{Rodador: rodador, DiretorioDeCache: diretorioDeCache}),
			sandbox.LinguagemPython: python.Nova(python.Opcoes{Rodador: rodador, DiretorioDeCache: diretorioDeCache}),
		},
		Agora: time.Now,
	}
}

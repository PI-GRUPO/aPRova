package main

import (
	"context"
	"errors"
	"fmt"
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
	usuarioDoCodigoDoPR    = 10001
	diretorioDeTrabalho    = "/tmp/aprova"
	diretorioDeCache       = "/tmp/aprova-cache"
	permissaoDosDiretorios = 0o755
	codigoDeUsoErrado      = 2
)

type etapa string

const (
	etapaPreparar     etapa = "preparar"
	etapaTestar       etapa = "testar"
	etapaUnicaDaAzure etapa = "unica"
)

var errSubcomandoDesconhecido = errors.New("sandbox: subcomando desconhecido; use preparar, testar ou nenhum")

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	ctx, encerrar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	codigo := executar(ctx, os.Args[1:])
	encerrar()

	os.Exit(codigo)
}

func executar(ctx context.Context, argumentos []string) int {
	pedida, err := etapaPedida(argumentos)
	if err != nil {
		slog.Error("o sandbox não foi chamado como esperado", "erro", err, "argumentos", argumentos)
		return codigoDeUsoErrado
	}

	switch pedida {
	case etapaPreparar:
		return programa().ExecutarPreparo(ctx)
	case etapaTestar:
		return programa().ExecutarTeste(ctx)
	default:
		return programa().Executar(ctx)
	}
}

func etapaPedida(argumentos []string) (etapa, error) {
	switch {
	case len(argumentos) == 0:
		return etapaUnicaDaAzure, nil
	case len(argumentos) == 1 && (etapa(argumentos[0]) == etapaPreparar || etapa(argumentos[0]) == etapaTestar):
		return etapa(argumentos[0]), nil
	default:
		return "", fmt.Errorf("%w: %q", errSubcomandoDesconhecido, argumentos)
	}
}

func programa() conteiner.Programa {
	rodador := processo.NovoRodador(processo.Opcoes{
		Credencial: &processo.Credencial{UID: usuarioDoCodigoDoPR, GID: usuarioDoCodigoDoPR},
	})
	goLang := golang.Nova(golang.Opcoes{Rodador: rodador, DiretorioDeCache: diretorioDeCache})
	pythonLang := python.Nova(python.Opcoes{Rodador: rodador, DiretorioDeCache: diretorioDeCache})

	return conteiner.Programa{
		LerVariavel:         os.Getenv,
		Saida:               os.Stdout,
		Rodador:             rodador,
		DiretorioDeTrabalho: diretorioDeTrabalho,
		DiretorioDoPreparo:  sandbox.PontoDeMontagemDoPreparo,
		Linguagens: map[sandbox.Linguagem]conteiner.ExecutorDeLinguagem{
			sandbox.LinguagemGo:     goLang,
			sandbox.LinguagemPython: pythonLang,
		},
		Etapas: map[sandbox.Linguagem]conteiner.LinguagemEmEtapas{
			sandbox.LinguagemGo:     goLang,
			sandbox.LinguagemPython: pythonLang,
		},
		PrepararDiretorio: prepararParaOCodigoDoPR,
		Agora:             time.Now,
	}
}

func prepararParaOCodigoDoPR(caminho string) error {
	if err := os.MkdirAll(caminho, permissaoDosDiretorios); err != nil {
		return err
	}
	return os.Chown(caminho, usuarioDoCodigoDoPR, usuarioDoCodigoDoPR)
}

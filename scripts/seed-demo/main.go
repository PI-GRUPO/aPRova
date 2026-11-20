package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/config"
	"github.com/PI-GRUPO/aPRova/internal/demo"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
	"github.com/PI-GRUPO/aPRova/internal/repository"
)

const (
	repositorioPadrao = "aprova-demo/loja-virtual"
	tempoLimite       = 2 * time.Minute
)

var (
	errSemInstalacao = errors.New("informe a instalação com -instalacao ou APROVA_CHAT_INSTALLATION_ID")
	errSemChave      = errors.New("APROVA_CHAVE_RECIBO ausente: gere o par com make keys e cole no .env, senão os recibos não podem ser assinados")
)

type parametros struct {
	instalacao  int64
	repositorio string
}

func main() {
	if err := executar(os.Args[1:], os.Getenv, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "erro: %v\n", err)
		os.Exit(1)
	}
}

func executar(argumentos []string, lerVariavel func(string) string, saida io.Writer) error {
	params, err := lerParametros(argumentos, lerVariavel)
	if err != nil {
		return err
	}

	configuracao, err := config.Carregar()
	if err != nil {
		return err
	}
	if len(configuracao.ReciboSemente) == 0 {
		return errSemChave
	}
	assinador, err := receipt.NovoAssinador(configuracao.ReciboSemente)
	if err != nil {
		return err
	}

	ctx, cancelar := context.WithTimeout(context.Background(), tempoLimite)
	defer cancelar()

	banco, err := repository.ConectarPostgres(ctx, configuracao.URLBancoDados)
	if err != nil {
		return err
	}
	defer banco.Close()

	resumo, err := demo.Semear(ctx, repository.NovoRepositorioPostgres(banco, repository.OpcoesPostgres{}), assinador, demo.Opcoes{
		InstallationID: params.instalacao, Repositorio: params.repositorio,
	})
	if err != nil {
		return err
	}

	_, err = io.WriteString(saida, relatorio(params, resumo, assinador.IDChave()))
	return err
}

func lerParametros(argumentos []string, lerVariavel func(string) string) (parametros, error) {
	opcoes := flag.NewFlagSet("seed-demo", flag.ContinueOnError)
	instalacao := opcoes.Int64("instalacao", 0, "installation_id em que gravar as análises (padrão: APROVA_CHAT_INSTALLATION_ID)")
	repositorio := opcoes.String("repositorio", repositorioPadrao, "repositório fictício dos PRs, no formato dono/nome")
	if err := opcoes.Parse(argumentos); err != nil {
		return parametros{}, err
	}

	params := parametros{instalacao: *instalacao, repositorio: *repositorio}
	if params.instalacao == 0 {
		if valor := lerVariavel("APROVA_CHAT_INSTALLATION_ID"); valor != "" {
			lido, err := strconv.ParseInt(valor, 10, 64)
			if err != nil {
				return parametros{}, fmt.Errorf("APROVA_CHAT_INSTALLATION_ID inválido: %w", err)
			}
			params.instalacao = lido
		}
	}
	if params.instalacao <= 0 {
		return parametros{}, errSemInstalacao
	}
	return params, nil
}

func relatorio(params parametros, resumo demo.Resumo, idChave string) string {
	var texto strings.Builder
	fmt.Fprintf(&texto, "Instalação %d, repositório %s, chave dos recibos %s\n", params.instalacao, params.repositorio, idChave)
	fmt.Fprintf(&texto, "%d análises criadas, %d já existiam\n\n", resumo.Criadas, resumo.JaExistiam)
	for _, analise := range resumo.Analises {
		recibo := "sem recibo"
		if analise.ReciboID != "" {
			recibo = "recibo " + analise.ReciboID
		}
		fmt.Fprintf(&texto, "  PR #%d  %-22s %-46s %s\n", analise.NumeroPR, analise.Estado, analise.Titulo, recibo)
	}
	texto.WriteString("\nPara verificar um recibo com a API no ar: GET /receipts/<id>/verify\n")
	return texto.String()
}

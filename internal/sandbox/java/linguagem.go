
package java

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azulpasta/aPRova/internal/sandbox"
	"github.com/Azulpasta/aPRova/internal/sandbox/processo"
)

const (
	comandoMaven = "mvn"

	margemParaEntregar   = 30 * time.Second
	tempoMinimoDosTestes = 30 * time.Second

	caminhoDosExecutaveis = "/usr/local/bin:/usr/bin:/bin"
)

type Opcoes struct {
	Rodador          processo.Rodador
	DiretorioDeCache string
	Agora            func() time.Time
}

type Linguagem struct {
	rodador          processo.Rodador
	diretorioDeCache string
	agora            func() time.Time
}

func Nova(opcoes Opcoes) *Linguagem {
	agora := opcoes.Agora
	if agora == nil {
		agora = time.Now
	}

	return &Linguagem{
		rodador:          opcoes.Rodador,
		diretorioDeCache: opcoes.DiretorioDeCache,
		agora:            agora,
	}
}

func (l *Linguagem) Executar(
	ctx context.Context,
	diretorio string,
	prazoFinal time.Time,
) sandbox.ResultadoSandbox {
	timeout := prazoFinal.Sub(l.agora()) - margemParaEntregar

	if timeout < tempoMinimoDosTestes {
		return infraestrutura(fmt.Sprintf(
			"prazo insuficiente para executar Maven: restam %v",
			timeout,
		))
	}

	ctx, cancelar := context.WithTimeout(ctx, timeout)
	defer cancelar()

	return l.executarProjeto(ctx, diretorio)
}

func (l *Linguagem) executarProjeto(
	ctx context.Context,
	diretorio string,
) sandbox.ResultadoSandbox {
	compilacao := l.rodar(
		ctx,
		diretorio,
		"-o",
		"-B",
		"test-compile",
	)

	if !concluiu(compilacao) {
		return l.classificarFalha(compilacao, "mvn test-compile")
	}

	testes := l.rodar(
		ctx,
		diretorio,
		"-o",
		"-B",
		"test",
	)

	if testes.Erro != nil || testes.PrazoEsgotado {
		return infraestrutura(
			descreverInterrupcao("mvn test", testes),
			testes.Saida,
		)
	}

	apurada, err := apurarRelatorios(diretorio)
	if err != nil {
		return infraestrutura(err.Error(), testes.Saida)
	}

	resultado := sandbox.ResultadoSandbox{
		Compilou:       true,
		TestesPassaram: apurada.passaram,
		TestesFalharam: apurada.falharam,
		SaidaBruta:     string(testes.Saida),
	}

	switch {
	case testes.CodigoSaida == 0 && apurada.falharam == 0:
		resultado.TipoFalha = sandbox.FalhaNenhuma

	case testes.CodigoSaida != 0 && apurada.falharam > 0:
		resultado.TipoFalha = sandbox.FalhaTeste

	default:
		return infraestrutura(
			fmt.Sprintf(
				"resultado do Maven inconsistente: código %d, %d testes falharam",
				testes.CodigoSaida,
				apurada.falharam,
			),
			testes.Saida,
		)
	}

	return resultado
}

func (l *Linguagem) rodar(
	ctx context.Context,
	diretorio string,
	argumentos ...string,
) processo.Execucao {
	argumentos = append(
		[]string{
			"-Dmaven.repo.local=" + filepath.Join(l.diretorioDeCache, "m2"),
		},
		argumentos...,
	)

	return l.rodador.Rodar(ctx, processo.Comando{
		Binario:    comandoMaven,
		Argumentos: argumentos,
		Diretorio:  diretorio,
		Ambiente:   l.ambiente(),
	})
}

func (l *Linguagem) ambiente() []string {
	return []string{
		"PATH=" + caminhoDosExecutaveis,
		"HOME=" + filepath.Join(l.diretorioDeCache, "home"),
		"JAVA_HOME=/opt/java/openjdk",
		"MAVEN_OPTS=-Dmaven.repo.local=" + filepath.Join(l.diretorioDeCache, "m2"),
	}
}

func (l *Linguagem) classificarFalha(
	execucao processo.Execucao,
	etapa string,
) sandbox.ResultadoSandbox {
	if execucao.Erro != nil || execucao.PrazoEsgotado {
		return infraestrutura(
			descreverInterrupcao(etapa, execucao),
			execucao.Saida,
		)
	}

	return sandbox.ResultadoSandbox{
		Compilou:   false,
		SaidaBruta: string(execucao.Saida),
		TipoFalha:  sandbox.FalhaCompilacao,
	}
}

func concluiu(execucao processo.Execucao) bool {
	return execucao.Erro == nil &&
		!execucao.PrazoEsgotado &&
		execucao.CodigoSaida == 0
}

func descreverInterrupcao(
	etapa string,
	execucao processo.Execucao,
) string {
	switch {
	case execucao.Erro != nil:
		return fmt.Sprintf(
			"%s não pôde ser iniciado: %v",
			etapa,
			execucao.Erro,
		)

	case execucao.PrazoEsgotado:
		return fmt.Sprintf(
			"%s excedeu o prazo do executor",
			etapa,
		)

	default:
		return fmt.Sprintf(
			"%s falhou com código %d",
			etapa,
			execucao.CodigoSaida,
		)
	}
}

func infraestrutura(motivo string, saidas ...[]byte) sandbox.ResultadoSandbox {
	var saida strings.Builder

	for _, trecho := range saidas {
		saida.Write(trecho)
	}

	saida.WriteString("aprova: " + motivo + "\n")

	return sandbox.ResultadoSandbox{
		SaidaBruta: saida.String(),
		TipoFalha:  sandbox.FalhaInfraestrutura,
	}
}

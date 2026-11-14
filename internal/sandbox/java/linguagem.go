// Package java roda os testes de repositórios Maven no sandbox, sem rede e só
// com os artefatos aquecidos na imagem, e classifica o desfecho no
// ResultadoSandbox comum às linguagens.
package java

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/sandbox"
	"github.com/PI-GRUPO/aPRova/internal/sandbox/processo"
)

const (
	binarioMaven = "/usr/share/maven/bin/mvn"
	javaHome     = "/opt/java/openjdk"
	arquivoDoPom = "pom.xml"

	margemParaEntregar   = 30 * time.Second
	tempoMinimoDosTestes = 30 * time.Second

	caminhoDosExecutaveis = "PATH=" + javaHome + "/bin:/usr/local/bin:/usr/bin:/bin"
)

// Opcoes configura a execução dos testes de um repositório Maven.
// DiretorioDeCache recebe o repositório local gravável e o HOME do Maven;
// RepositorioDaImagem é o repositório aquecido na imagem, só de leitura.
type Opcoes struct {
	Rodador             processo.Rodador
	DiretorioDeCache    string
	RepositorioDaImagem string
	Agora               func() time.Time
}

// Linguagem roda os testes de repositórios Maven.
type Linguagem struct {
	rodador             processo.Rodador
	diretorioDeCache    string
	repositorioDaImagem string
	agora               func() time.Time
}

// Nova constrói a linguagem Java com as opções informadas. Sem relógio, usa
// time.Now.
func Nova(opcoes Opcoes) *Linguagem {
	agora := opcoes.Agora
	if agora == nil {
		agora = time.Now
	}
	return &Linguagem{
		rodador:             opcoes.Rodador,
		diretorioDeCache:    opcoes.DiretorioDeCache,
		repositorioDaImagem: opcoes.RepositorioDaImagem,
		agora:               agora,
	}
}

func (l *Linguagem) rodarMaven(ctx context.Context, projeto, fase string) processo.Execucao {
	return l.rodador.Rodar(ctx, processo.Comando{
		Binario: binarioMaven,
		Argumentos: []string{
			"-o", "-B", "-ntp",
			"-Dmaven.repo.local=" + filepath.Join(l.diretorioDeCache, "m2"),
			"-Dmaven.repo.local.tail=" + l.repositorioDaImagem,
			fase,
		},
		Diretorio: projeto,
		Ambiente: []string{
			caminhoDosExecutaveis,
			"HOME=" + filepath.Join(l.diretorioDeCache, "home"),
			"JAVA_HOME=" + javaHome,
		},
	})
}

func (l *Linguagem) apagarRelatoriosVersionados(ctx context.Context, projeto string) processo.Execucao {
	return l.rodador.Rodar(ctx, processo.Comando{
		Binario: "find",
		Argumentos: []string{
			projeto, "-type", "d", "-path", "*/" + diretorioDeBuild + "/" + diretorioDosRelatorios, "-prune",
			"-exec", "rm", "-rf", "--", "{}", "+",
		},
		Diretorio: projeto,
		Ambiente:  []string{caminhoDosExecutaveis},
	})
}

func existePomNaRaiz(projeto string) bool {
	informacao, err := os.Stat(filepath.Join(projeto, arquivoDoPom))
	return err == nil && informacao.Mode().IsRegular()
}

func concluiu(execucao processo.Execucao) bool {
	return execucao.Erro == nil && !execucao.PrazoEsgotado && execucao.CodigoSaida == 0
}

func descreverInterrupcao(etapa string, execucao processo.Execucao) string {
	switch {
	case execucao.Erro != nil:
		return fmt.Sprintf("%s não pôde ser iniciado: %v", etapa, execucao.Erro)
	case execucao.PrazoEsgotado:
		return fmt.Sprintf("%s excedeu o prazo do executor", etapa)
	default:
		return fmt.Sprintf("%s falhou com código %d", etapa, execucao.CodigoSaida)
	}
}

func infraestrutura(motivo string, saidas ...[]byte) sandbox.ResultadoSandbox {
	return sandbox.ResultadoSandbox{SaidaBruta: comMotivo(motivo, saidas...), TipoFalha: sandbox.FalhaInfraestrutura}
}

func compilacao(motivo string, saida []byte) sandbox.ResultadoSandbox {
	return sandbox.ResultadoSandbox{SaidaBruta: comMotivo(motivo, saida), TipoFalha: sandbox.FalhaCompilacao}
}

func comMotivo(motivo string, saidas ...[]byte) string {
	var saida strings.Builder
	for _, trecho := range saidas {
		saida.Write(trecho)
	}
	if motivo != "" {
		saida.WriteString("aprova: " + motivo + "\n")
	}
	return saida.String()
}

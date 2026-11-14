package java

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/sandbox"
	"github.com/PI-GRUPO/aPRova/internal/sandbox/processo"
)

const (
	motivoSemPom         = "repositório sem pom.xml na raiz"
	motivoJavaMaisNovo   = "o projeto exige um Java mais novo que o JDK 21 da imagem do sandbox"
	faseDeCompilarTestes = "test-compile"
	faseDeTestar         = "test"
)

var (
	artefatoForaDoCache = regexp.MustCompile(`the artifact (\S+) has not been downloaded from it before`)
	javaMaisNovo        = regexp.MustCompile(`release version \d+ not supported|invalid (?:target|source) release`)
)

// Preparar só confere que o projeto é Maven: o clone já foi feito e nenhum
// mvn roda nesta etapa, que tem rede. Devolve nil quando há pom.xml na raiz.
func (l *Linguagem) Preparar(_ context.Context, projeto, _ string) *sandbox.ResultadoSandbox {
	if existePomNaRaiz(projeto) {
		return nil
	}
	falha := infraestrutura(motivoSemPom)
	return &falha
}

// Testar apaga os relatórios do surefire que vieram no repositório, compila e
// roda os testes com o mvn da imagem, offline, e apura os relatórios gerados.
// prazoFinal é o instante em que o resultado já precisa ter sido entregue.
func (l *Linguagem) Testar(ctx context.Context, projeto, _ string, prazoFinal time.Time) sandbox.ResultadoSandbox {
	if !existePomNaRaiz(projeto) {
		return infraestrutura(motivoSemPom)
	}

	timeout := prazoFinal.Sub(l.agora()) - margemParaEntregar
	if timeout < tempoMinimoDosTestes {
		return infraestrutura(fmt.Sprintf("prazo insuficiente para rodar o Maven: restam %v", timeout))
	}
	ctx, cancelar := context.WithTimeout(ctx, timeout)
	defer cancelar()

	if remocao := l.apagarRelatoriosVersionados(ctx, projeto); !concluiu(remocao) {
		return infraestrutura(descreverInterrupcao("remoção dos relatórios do surefire que vieram no repositório", remocao), remocao.Saida)
	}

	compilacaoDosTestes := l.rodarMaven(ctx, projeto, faseDeCompilarTestes)
	if !concluiu(compilacaoDosTestes) {
		return classificarCompilacao(compilacaoDosTestes)
	}

	return classificarTeste(projeto, l.rodarMaven(ctx, projeto, faseDeTestar))
}

func classificarCompilacao(execucao processo.Execucao) sandbox.ResultadoSandbox {
	if falha, interrompida := classificarInterrupcao("mvn "+faseDeCompilarTestes, execucao); interrompida {
		return falha
	}
	if javaMaisNovo.Match(execucao.Saida) {
		return infraestrutura(motivoJavaMaisNovo, execucao.Saida)
	}
	return compilacao("", execucao.Saida)
}

func classificarTeste(projeto string, teste processo.Execucao) sandbox.ResultadoSandbox {
	if falha, interrompida := classificarInterrupcao("mvn "+faseDeTestar, teste); interrompida {
		return falha
	}

	apurada, err := apurarRelatorios(projeto)
	if err != nil {
		return infraestrutura(err.Error(), teste.Saida)
	}

	resultado := sandbox.ResultadoSandbox{
		Compilou:       true,
		TestesPassaram: apurada.passaram,
		TestesFalharam: apurada.falharam,
		SaidaBruta:     string(teste.Saida),
	}
	switch {
	case teste.CodigoSaida == 0 && apurada.falharam == 0:
		resultado.TipoFalha = sandbox.FalhaNenhuma
	case teste.CodigoSaida != 0 && apurada.falharam > 0:
		resultado.TipoFalha = sandbox.FalhaTeste
	default:
		return infraestrutura(fmt.Sprintf("mvn test saiu com código %d, que os relatórios do surefire não explicam", teste.CodigoSaida), teste.Saida)
	}
	return resultado
}

func classificarInterrupcao(etapa string, execucao processo.Execucao) (sandbox.ResultadoSandbox, bool) {
	switch {
	case execucao.Erro != nil || execucao.PrazoEsgotado:
		return infraestrutura(descreverInterrupcao(etapa, execucao), execucao.Saida), true
	case execucao.CodigoSaida == 0:
		return sandbox.ResultadoSandbox{}, false
	}

	artefatos := artefatosForaDoCache(execucao.Saida)
	if len(artefatos) == 0 {
		return sandbox.ResultadoSandbox{}, false
	}
	motivo := "artefato fora do cache do sandbox Java: " + strings.Join(artefatos, ", ") +
		" (a análise roda sem rede e só tem as versões aquecidas na imagem)"
	return compilacao(motivo, execucao.Saida), true
}

func artefatosForaDoCache(saida []byte) []string {
	var artefatos []string
	for _, encontrado := range artefatoForaDoCache.FindAllSubmatch(saida, -1) {
		if artefato := string(encontrado[1]); !slices.Contains(artefatos, artefato) {
			artefatos = append(artefatos, artefato)
		}
	}
	return artefatos
}

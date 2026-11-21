package explicacao

import (
	"errors"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

func gateDeRiscoAlto() domain.GateDeRisco {
	return domain.GateDeRisco{
		NivelDeterministico: domain.RiscoAlto, Nivel: domain.RiscoAlto,
		Motivos: []domain.MotivoDeRisco{
			{ID: "falha_sandbox", Nivel: domain.RiscoAlto, Descricao: "os testes ou a compilação falharam no sandbox", Evidencias: []string{"1 de 11 testes falharam"}},
			{ID: "codigo_sem_teste", Nivel: domain.RiscoMedio, Descricao: "o PR altera código sem alterar nenhum teste", Evidencias: []string{"go/frete/frete.go"}},
		},
	}
}

func sandboxComUmaFalha() sandbox.ResultadoSandbox {
	return sandbox.ResultadoSandbox{Compilou: true, TestesPassaram: 10, TestesFalharam: 1, TipoFalha: sandbox.FalhaTeste}
}

func entradaDeRiscoAlto() Entrada {
	return NovaEntrada(gateDeRiscoAlto(), sandboxComUmaFalha())
}

const textoFiel = "Este pull request ficou com risco alto, então o merge está bloqueado até alguém com permissão de escrita aprovar. " +
	"O motivo principal é que 1 de 11 testes falharam no sandbox. " +
	"Além disso, o arquivo go/frete/frete.go mudou sem nenhum teste novo. " +
	"Corrija o teste que falha e acrescente um teste para frete.go, ou peça a aprovação de quem pode liberar."

func exigirDescarte(t *testing.T, texto string, entrada Entrada) {
	t.Helper()
	if _, err := Validar(texto, entrada); !errors.Is(err, ErrDescartada) {
		t.Errorf("esperava descarte, obtive %v para %q", err, texto)
	}
}

func TestTextoFielAoDecididoPassaNormalizado(t *testing.T) {
	validado, err := Validar("  "+strings.ReplaceAll(textoFiel, ". ", ".\n\n")+"\n", entradaDeRiscoAlto())

	if err != nil {
		t.Fatalf("texto fiel descartado: %v", err)
	}
	if validado != textoFiel {
		t.Errorf("o texto validado vira um parágrafo só:\n%q", validado)
	}
}

func TestNivelDiferenteDoDecididoDescarta(t *testing.T) {
	for _, trecho := range []string{"ficou com risco baixo", "ficou com risco médio", "o nível é baixo", "o risco foi considerado médio"} {
		exigirDescarte(t, strings.Replace(textoFiel, "ficou com risco alto", trecho, 1), entradaDeRiscoAlto())
	}
}

func TestAltoOuBaixoEmOutroSentidoNaoDescarta(t *testing.T) {
	for _, frase := range []string{
		" O número de testes que falharam é baixo, mas basta um.",
		" Um número alto de arquivos não é o problema aqui.",
		" O nível de detalhe baixo do teste não muda nada.",
	} {
		if _, err := Validar(textoFiel+frase, entradaDeRiscoAlto()); err != nil {
			t.Errorf("alto ou baixo sem risco ou nível ao lado não descarta: %v (%q)", err, frase)
		}
	}
}

func TestCalibracaoAplicadaAceitaONivelDasRegras(t *testing.T) {
	gate := domain.GateDeRisco{
		NivelDeterministico: domain.RiscoAlto, Nivel: domain.RiscoMedio,
		Motivos:    []domain.MotivoDeRisco{{ID: "area_sensivel", Nivel: domain.RiscoAlto, Descricao: "o PR altera um caminho sensível", Evidencias: []string{"internal/auth/sessao.go"}}},
		Calibracao: domain.Calibracao{Estado: domain.CalibracaoAplicada, Razao: "3 PRs parecidos foram aprovados e mergeados"},
	}
	texto := "As regras deram risco alto porque internal/auth/sessao.go é um caminho sensível, mas o histórico reduziu para risco médio. " +
		"Três pull requests parecidos foram aprovados e mergeados, por isso o merge está liberado. " +
		"Vale revisar a mudança em sessao.go com atenção antes do merge."

	if _, err := Validar(texto, NovaEntrada(gate, sandbox.ResultadoSandbox{Compilou: true, TestesPassaram: 4})); err != nil {
		t.Errorf("com calibração aplicada, o nível das regras pode ser citado: %v", err)
	}

	gate.Calibracao = domain.Calibracao{Estado: domain.CalibracaoDesligada}
	gate.NivelDeterministico = domain.RiscoMedio
	exigirDescarte(t, texto, NovaEntrada(gate, sandbox.ResultadoSandbox{Compilou: true, TestesPassaram: 4}))
}

func TestMotivoForaDoRulesHitDescarta(t *testing.T) {
	exigirDescarte(t, textoFiel+" Também há area_sensivel no PR.", entradaDeRiscoAlto())
	exigirDescarte(t, textoFiel+" Também há motivo_inventado no PR.", entradaDeRiscoAlto())
}

func TestArquivoForaDasEvidenciasDescarta(t *testing.T) {
	exigirDescarte(t, textoFiel+" Revise também internal/auth/senha.go.", entradaDeRiscoAlto())
	exigirDescarte(t, textoFiel+" Revise também pom.xml.", entradaDeRiscoAlto())
}

func TestTamanhoForaDoLimiteDescarta(t *testing.T) {
	exigirDescarte(t, "Risco alto.", entradaDeRiscoAlto())
	exigirDescarte(t, textoFiel+strings.Repeat(" O teste falhou.", 60), entradaDeRiscoAlto())
}

func TestLinkMencaoOuComandoDescarta(t *testing.T) {
	for _, extra := range []string{
		" Veja https://exemplo.com para detalhes.",
		" Veja www.exemplo.com para detalhes.",
		" Peça para @julia-ed2 aprovar.",
		" Isso lembra o PR #6.",
		" Rode ```go test``` de novo.",
		" Depois rode git revert no commit.",
		" Use $(make test) para conferir.",
		" Veja [o recibo](recibo) para detalhes.",
		" Leia <b>isto</b>.",
	} {
		exigirDescarte(t, textoFiel+extra, entradaDeRiscoAlto())
	}
}

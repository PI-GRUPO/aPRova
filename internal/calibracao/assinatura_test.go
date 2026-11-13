package calibracao

import (
	"slices"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/platform"
)

const conteudoSecreto = "+func validarSenha(senha string) bool { return senha == \"hunter2\" }"

func arquivosDoPR() []platform.ArquivoAlterado {
	return []platform.ArquivoAlterado{
		{Nome: "internal/auth/sessao.go", Situacao: "modified", Patch: conteudoSecreto},
		{Nome: "internal/auth/token.go", Situacao: "added", Patch: conteudoSecreto},
		{Nome: "migrations/000021_tokens.up.sql", Situacao: "added"},
		{Nome: "README.md", Situacao: "modified"},
		{Nome: "internal/auth/antigo.go", Situacao: "removed"},
		{Nome: "Makefile", Situacao: "renamed", NomeAnterior: "makefile"},
		{Nome: "internal/auth/sessao_test.go", Situacao: "modified"},
	}
}

func motivosDoPR() []domain.MotivoDeRisco {
	return []domain.MotivoDeRisco{codigoSemTeste(), areaSensivel(), areaSensivel()}
}

func TestAssinaturaTemOFormatoDocumentado(t *testing.T) {
	texto := Assinatura(motivosDoPR(), arquivosDoPR())

	esperado := strings.Join([]string{
		"motivos: area_sensivel, codigo_sem_teste",
		"caminhos: *, *.md, internal/auth/*.go, migrations/*.sql",
		"adicionados: 2",
		"alterados: 4",
		"removidos: 1",
	}, "\n")
	if texto != esperado {
		t.Errorf("assinatura:\n%s\nesperava:\n%s", texto, esperado)
	}
}

func TestAssinaturaNaoDependeDaOrdemDosArquivosNemDosMotivos(t *testing.T) {
	referencia := Assinatura(motivosDoPR(), arquivosDoPR())

	arquivos := arquivosDoPR()
	motivos := motivosDoPR()
	for range 20 {
		slices.Reverse(arquivos)
		arquivos = append(arquivos[1:], arquivos[0])
		motivos = append(motivos[1:], motivos[0])

		if texto := Assinatura(motivos, arquivos); texto != referencia {
			t.Fatalf("a ordem mudou a assinatura:\n%s\n---\n%s", texto, referencia)
		}
	}
}

func TestAssinaturaNaoLevaCodigoNemNomeDeArquivo(t *testing.T) {
	texto := Assinatura(motivosDoPR(), arquivosDoPR())

	for _, proibido := range []string{"hunter2", "validarSenha", "sessao", "token", "000021", "Makefile", "README"} {
		if strings.Contains(texto, proibido) {
			t.Errorf("a assinatura vazou %q:\n%s", proibido, texto)
		}
	}
}

func TestAssinaturaIgnoraOMotivoDaPropriaCalibracao(t *testing.T) {
	comCalibracao := append(motivosDoPR(), domain.MotivoDeRisco{ID: MotivoDaCalibracao, Nivel: domain.NivelInformativo})

	if Assinatura(comCalibracao, arquivosDoPR()) != Assinatura(motivosDoPR(), arquivosDoPR()) {
		t.Error("a assinatura descreve o que o gate viu, não o resultado da calibração")
	}
}

func TestAssinaturaDePRSemMotivoNemArquivo(t *testing.T) {
	esperado := "motivos: \ncaminhos: \nadicionados: 0\nalterados: 0\nremovidos: 0"

	if texto := Assinatura(nil, nil); texto != esperado {
		t.Errorf("assinatura vazia = %q", texto)
	}
}

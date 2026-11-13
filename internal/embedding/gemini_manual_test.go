package embedding_test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/calibracao"
	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/embedding"
	"github.com/PI-GRUPO/aPRova/internal/platform"
)

type assinaturaDeReferencia struct {
	nome     string
	motivos  []string
	arquivos []platform.ArquivoAlterado
}

func arquivo(nome, situacao string) platform.ArquivoAlterado {
	return platform.ArquivoAlterado{Nome: nome, Situacao: situacao}
}

func referencias() []assinaturaDeReferencia {
	return []assinaturaDeReferencia{
		{"auth-a", []string{"area_sensivel"}, []platform.ArquivoAlterado{arquivo("internal/auth/sessao.go", "modified"), arquivo("internal/auth/sessao_test.go", "modified")}},
		{"auth-b", []string{"area_sensivel"}, []platform.ArquivoAlterado{arquivo("internal/auth/token.go", "modified"), arquivo("internal/auth/token_test.go", "modified"), arquivo("internal/auth/jwt.go", "modified")}},
		{"auth-c", []string{"area_sensivel"}, []platform.ArquivoAlterado{arquivo("internal/auth/senha.go", "added"), arquivo("internal/auth/senha_test.go", "added")}},
		{"auth-migration", []string{"area_sensivel"}, []platform.ArquivoAlterado{arquivo("internal/auth/sessao.go", "modified"), arquivo("migrations/000021_sessoes.up.sql", "added")}},
		{"auth-sem-teste", []string{"area_sensivel", "codigo_sem_teste"}, []platform.ArquivoAlterado{arquivo("internal/auth/sessao.go", "modified")}},
		{"pagamentos", []string{"area_sensivel"}, []platform.ArquivoAlterado{arquivo("internal/payments/cobranca.go", "modified"), arquivo("internal/payments/cobranca_test.go", "modified")}},
		{"frontend-sem-teste", []string{"codigo_sem_teste"}, []platform.ArquivoAlterado{arquivo("web/src/Painel.tsx", "modified"), arquivo("web/src/estilo.css", "modified")}},
		{"muitos-arquivos", []string{"muitos_arquivos"}, muitosArquivos()},
	}
}

func muitosArquivos() []platform.ArquivoAlterado {
	arquivos := make([]platform.ArquivoAlterado, 0, 25)
	for indice := range 25 {
		arquivos = append(arquivos, arquivo(fmt.Sprintf("internal/frete/regra_%02d.go", indice), "renamed"))
	}
	return arquivos
}

func motivosComId(ids []string) []domain.MotivoDeRisco {
	motivos := make([]domain.MotivoDeRisco, 0, len(ids))
	for _, id := range ids {
		motivos = append(motivos, domain.MotivoDeRisco{ID: id})
	}
	return motivos
}

func TestGeminiRealSimilaridadeDasAssinaturas(t *testing.T) {
	if os.Getenv("APROVA_TESTE_GEMINI_REAL") != "1" {
		t.Skip("teste manual: rode com APROVA_TESTE_GEMINI_REAL=1 e GEMINI_API_KEY no ambiente")
	}
	gemini, err := embedding.NovoGemini(embedding.OpcoesGemini{Chave: os.Getenv("GEMINI_API_KEY"), Timeout: 15 * time.Second})
	if err != nil {
		t.Fatalf("montar o Gemini: %v", err)
	}

	casos := referencias()
	vetores := make([][]float32, len(casos))
	for indice, caso := range casos {
		texto := calibracao.Assinatura(motivosComId(caso.motivos), caso.arquivos)
		vetores[indice], err = gemini.Gerar(t.Context(), texto)
		if err != nil {
			t.Fatalf("embedding de %s: %v", caso.nome, err)
		}
		t.Logf("%s:\n%s", caso.nome, texto)
	}

	var tabela strings.Builder
	fmt.Fprintf(&tabela, "%-20s", "")
	for _, caso := range casos {
		fmt.Fprintf(&tabela, "%-20s", caso.nome)
	}
	for linha, caso := range casos {
		fmt.Fprintf(&tabela, "\n%-20s", caso.nome)
		for coluna := range casos {
			fmt.Fprintf(&tabela, "%-20.4f", embedding.Similaridade(vetores[linha], vetores[coluna]))
		}
	}
	t.Logf("similaridade de cosseno:\n%s", tabela.String())
}

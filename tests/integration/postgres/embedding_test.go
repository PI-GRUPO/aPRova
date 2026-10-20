package integracaopostgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/jackc/pgx/v5"
)

const dimensaoDoGeminiEmbedding = 768

func vetorComDimensao(dimensao, eixo int) []float32 {
	vetor := make([]float32, dimensao)
	vetor[eixo] = 1
	return vetor
}

func TestConstanteDoEmbeddingEOOutputDimensionalityDoGemini(t *testing.T) {
	if domain.DimensoesEmbedding != dimensaoDoGeminiEmbedding {
		t.Errorf("DimensoesEmbedding = %d, esperava %d (gemini-embedding-001 com outputDimensionality 768)",
			domain.DimensoesEmbedding, dimensaoDoGeminiEmbedding)
	}
}

func TestColunaDoEmbeddingTemADimensaoDaConstante(t *testing.T) {
	exigirBanco(t)

	var tipo string
	err := pool.QueryRow(t.Context(), `
		SELECT format_type(a.atttypid, a.atttypmod)
		FROM pg_attribute a
		WHERE a.attrelid = 'code_memory'::regclass AND a.attname = 'embedding'`).Scan(&tipo)
	if err != nil {
		t.Fatalf("ler o tipo da coluna: %v", err)
	}

	if esperado := fmt.Sprintf("vector(%d)", domain.DimensoesEmbedding); tipo != esperado {
		t.Errorf("code_memory.embedding é %s, mas o código grava %s; a primeira gravação falharia", tipo, esperado)
	}
}

func TestEmbeddingDe768DimensoesGravaEEEncontradoPelaSimilaridade(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()

	alvo, err := repo.GravarMemoriaDeCodigo(t.Context(), memoria(tenant, nil, vetorComDimensao(dimensaoDoGeminiEmbedding, 3)))
	if err != nil {
		t.Fatalf("gravar vetor de 768 posições: %v", err)
	}
	if _, err := repo.GravarMemoriaDeCodigo(t.Context(), memoria(tenant, nil, vetorComDimensao(dimensaoDoGeminiEmbedding, 4))); err != nil {
		t.Fatalf("gravar segundo vetor: %v", err)
	}

	vizinhos, err := repo.BuscarVizinhos(t.Context(), tenant, vetorComDimensao(dimensaoDoGeminiEmbedding, 3), 1)
	if err != nil {
		t.Fatalf("buscar por similaridade: %v", err)
	}
	if len(vizinhos) != 1 || vizinhos[0].ID != alvo || vizinhos[0].Distancia > 1e-6 {
		t.Errorf("esperava a memória %d a distância zero, obtive %+v", alvo, vizinhos)
	}
}

func TestEmbeddingDe1536DimensoesERecusadoAntesDoBanco(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()

	_, err := repo.GravarMemoriaDeCodigo(t.Context(), memoria(tenant, nil, vetorComDimensao(1536, 0)))

	if !errors.Is(err, domain.ErrDadosInvalidos) {
		t.Fatalf("esperava ErrDadosInvalidos, obtive %v", err)
	}
	if !strings.Contains(err.Error(), "1536") || !strings.Contains(err.Error(), "768") {
		t.Errorf("o erro precisa dizer a dimensão recebida e a esperada: %v", err)
	}
	if gravadas := contar(t, `SELECT count(*) FROM code_memory WHERE tenant_id = $1`, tenant); gravadas != 0 {
		t.Errorf("nada pode chegar ao banco, há %d linhas", gravadas)
	}
}

func TestMigrationDoEmbeddingRecusaTabelaComDadosSemDescartarNada(t *testing.T) {
	exigirBanco(t)

	url := criarBancoVazio(t, "embedding_com_dados")
	migracao, err := novaMigracao(url)
	if err != nil {
		t.Fatalf("abrir migrate: %v", err)
	}
	defer fecharMigracao(migracao)

	if err := migracao.Migrate(10); err != nil {
		t.Fatalf("subir até a migration 10: %v", err)
	}
	conexao := conectar(t, url)
	if _, err := conexao.Exec(t.Context(), `
		INSERT INTO code_memory (tenant_id, repository, summary, embedding, level, outcome)
		VALUES (1, 'Azulpasta/alvo', 'memória antiga', array_fill(0.1::real, ARRAY[1536])::vector, 'baixo', 'sem_reversao')`); err != nil {
		t.Fatalf("gravar memória antiga: %v", err)
	}

	err = migracao.Migrate(11)

	if err == nil || !strings.Contains(err.Error(), "code_memory") {
		t.Fatalf("a migration precisa falhar nomeando code_memory, obtive %v", err)
	}
	var restantes int
	if err := conexao.QueryRow(t.Context(), `SELECT count(*) FROM code_memory`).Scan(&restantes); err != nil || restantes != 1 {
		t.Errorf("nenhum dado pode ser descartado: %d linhas, erro %v", restantes, err)
	}
}

func conectar(t *testing.T, url string) *pgx.Conn {
	t.Helper()

	conexao, err := pgx.Connect(t.Context(), url)
	if err != nil {
		t.Fatalf("conectar em %s: %v", url, err)
	}
	t.Cleanup(func() { _ = conexao.Close(context.Background()) })
	return conexao
}

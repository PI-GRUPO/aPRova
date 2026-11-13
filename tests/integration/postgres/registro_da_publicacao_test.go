package integracaopostgres

import (
	"errors"
	"testing"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/publicacao"
	"github.com/Azulpasta/aPRova/internal/repository"
)

var _ publicacao.Registro = (*repository.RepositorioPostgres)(nil)

func TestRegistroGuardaOCheckRunDeCadaTaskRun(t *testing.T) {
	repo := exigirBanco(t)
	taskRun := novoTaskRun(t, repo, novoTenant())

	if _, err := repo.CheckRunDoTaskRun(t.Context(), taskRun.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("antes da publicação não há Check Run: %v", err)
	}
	if err := repo.GravarCheckRun(t.Context(), taskRun.ID, 31415926535); err != nil {
		t.Fatalf("gravar: %v", err)
	}

	id, err := repo.CheckRunDoTaskRun(t.Context(), taskRun.ID)
	if err != nil || id != 31415926535 {
		t.Errorf("Check Run gravado = %d, %v", id, err)
	}
}

func TestGravarCheckRunDeTaskRunInexistenteFalha(t *testing.T) {
	repo := exigirBanco(t)

	err := repo.GravarCheckRun(t.Context(), "00000000-0000-4000-8000-00000000dead", 1)

	if !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Errorf("sem task_run não há onde gravar: %v", err)
	}
}

func TestIdDoCheckRunPrecisaSerPositivo(t *testing.T) {
	repo := exigirBanco(t)
	taskRun := novoTaskRun(t, repo, novoTenant())

	_, err := pool.Exec(t.Context(), `UPDATE task_runs SET check_run_id = 0 WHERE id = $1`, taskRun.ID)

	if codigoPostgres(err) != "23514" {
		t.Errorf("esperava check_violation (23514) para id zero, obtive %v", err)
	}
}

func TestRegistroGuardaUmComentarioPorPullRequest(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()

	if _, err := repo.ComentarioDoPR(t.Context(), tenant, "Azulpasta/alvo", 7); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("antes do primeiro comentário não há id: %v", err)
	}
	if err := repo.GravarComentario(t.Context(), tenant, "Azulpasta/alvo", 7, 2718281828); err != nil {
		t.Fatalf("gravar: %v", err)
	}
	if err := repo.GravarComentario(t.Context(), tenant, "Azulpasta/alvo", 7, 1414213562); err != nil {
		t.Fatalf("regravar: %v", err)
	}
	if err := repo.GravarComentario(t.Context(), tenant, "Azulpasta/alvo", 8, 1732050807); err != nil {
		t.Fatalf("outro PR: %v", err)
	}

	if id, err := repo.ComentarioDoPR(t.Context(), tenant, "Azulpasta/alvo", 7); err != nil || id != 1414213562 {
		t.Errorf("o comentário do PR 7 é o último gravado: %d, %v", id, err)
	}
	if id, _ := repo.ComentarioDoPR(t.Context(), tenant, "Azulpasta/alvo", 8); id != 1732050807 {
		t.Errorf("cada PR tem o seu comentário: %d", id)
	}
	if linhas := contar(t, `SELECT count(*) FROM pull_request_comments WHERE tenant_id = $1 AND pr_number = 7`, tenant); linhas != 1 {
		t.Errorf("um PR tem uma linha só, há %d", linhas)
	}
}

func TestOMesmoPullRequestNaoTemDoisComentarios(t *testing.T) {
	exigirBanco(t)
	tenant := novoTenant()
	inserir := `INSERT INTO pull_request_comments (tenant_id, repository, pr_number, comment_id) VALUES ($1, 'Azulpasta/alvo', 3, $2)`

	if _, err := pool.Exec(t.Context(), inserir, tenant, 11); err != nil {
		t.Fatalf("primeiro comentário: %v", err)
	}
	_, err := pool.Exec(t.Context(), inserir, tenant, 12)

	if codigoPostgres(err) != "23505" {
		t.Errorf("esperava unique_violation (23505) para o segundo comentário do mesmo PR, obtive %v", err)
	}
}

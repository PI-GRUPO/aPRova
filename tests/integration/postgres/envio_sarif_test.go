package integracaopostgres

import (
	"errors"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/repository"
	"github.com/PI-GRUPO/aPRova/internal/sarif"
)

var _ sarif.Registro = (*repository.RepositorioPostgres)(nil)

const (
	idDoEnvioSARIF      = "47177e22-5596-11eb-80a1-c1e54ef945c6"
	outroIDDoEnvioSARIF = "99999999-5596-11eb-80a1-c1e54ef945c6"
)

func TestEnvioSARIFGravadoEhLidoDeVoltaEMudaDeEstado(t *testing.T) {
	repo := exigirBanco(t)
	taskRun := novoTaskRun(t, repo, novoTenant())

	if _, err := repo.EnvioSARIFDoTaskRun(t.Context(), taskRun.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("antes do envio não há registro: %v", err)
	}

	aceito := domain.EnvioSARIF{TaskRunID: taskRun.ID, IDDoEnvio: idDoEnvioSARIF, Estado: domain.EnvioSARIFAceito}
	if err := repo.GravarEnvioSARIF(t.Context(), aceito); err != nil {
		t.Fatalf("gravar o envio aceito: %v", err)
	}
	processado := domain.EnvioSARIF{TaskRunID: taskRun.ID, IDDoEnvio: idDoEnvioSARIF, Estado: domain.EnvioSARIFProcessado, Detalhe: "processado pelo GitHub"}
	if err := repo.GravarEnvioSARIF(t.Context(), processado); err != nil {
		t.Fatalf("gravar o processamento: %v", err)
	}

	gravado, err := repo.EnvioSARIFDoTaskRun(t.Context(), taskRun.ID)
	if err != nil || gravado != processado {
		t.Errorf("gravado = %+v, erro = %v", gravado, err)
	}
}

func TestIdDoEnvioSARIFGravadoNaoEhTrocado(t *testing.T) {
	repo := exigirBanco(t)
	taskRun := novoTaskRun(t, repo, novoTenant())
	primeiro := domain.EnvioSARIF{TaskRunID: taskRun.ID, IDDoEnvio: idDoEnvioSARIF, Estado: domain.EnvioSARIFAceito}
	if err := repo.GravarEnvioSARIF(t.Context(), primeiro); err != nil {
		t.Fatalf("gravar: %v", err)
	}

	err := repo.GravarEnvioSARIF(t.Context(), domain.EnvioSARIF{TaskRunID: taskRun.ID, IDDoEnvio: outroIDDoEnvioSARIF, Estado: domain.EnvioSARIFAceito})
	if !errors.Is(err, domain.ErrDuplicado) {
		t.Errorf("um segundo envio do mesmo task_run deveria ser recusado: %v", err)
	}
	err = repo.GravarEnvioSARIF(t.Context(), domain.EnvioSARIF{TaskRunID: taskRun.ID, Estado: domain.EnvioSARIFFalhou, Detalhe: "tarde demais"})
	if !errors.Is(err, domain.ErrDuplicado) {
		t.Errorf("uma falha não apaga um envio aceito: %v", err)
	}

	if gravado, _ := repo.EnvioSARIFDoTaskRun(t.Context(), taskRun.ID); gravado != primeiro {
		t.Errorf("o envio gravado mudou: %+v", gravado)
	}
}

func TestEnvioSARIFQueFalhouPodeSerRefeito(t *testing.T) {
	repo := exigirBanco(t)
	taskRun := novoTaskRun(t, repo, novoTenant())

	falhou := domain.EnvioSARIF{TaskRunID: taskRun.ID, Estado: domain.EnvioSARIFFalhou, Detalhe: "GitHub respondeu 502 três vezes"}
	if err := repo.GravarEnvioSARIF(t.Context(), falhou); err != nil {
		t.Fatalf("gravar a falha: %v", err)
	}
	if gravado, _ := repo.EnvioSARIFDoTaskRun(t.Context(), taskRun.ID); gravado != falhou {
		t.Fatalf("falha gravada = %+v", gravado)
	}

	aceito := domain.EnvioSARIF{TaskRunID: taskRun.ID, IDDoEnvio: idDoEnvioSARIF, Estado: domain.EnvioSARIFAceito}
	if err := repo.GravarEnvioSARIF(t.Context(), aceito); err != nil {
		t.Fatalf("gravar o envio depois da falha: %v", err)
	}
	if gravado, _ := repo.EnvioSARIFDoTaskRun(t.Context(), taskRun.ID); gravado != aceito {
		t.Errorf("gravado = %+v", gravado)
	}
}

func TestGravarEnvioSARIFDeTaskRunInexistenteFalha(t *testing.T) {
	repo := exigirBanco(t)

	err := repo.GravarEnvioSARIF(t.Context(), domain.EnvioSARIF{
		TaskRunID: "00000000-0000-4000-8000-00000000dead", IDDoEnvio: idDoEnvioSARIF, Estado: domain.EnvioSARIFAceito,
	})

	if !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Errorf("sem task_run não há onde gravar: %v", err)
	}
}

func TestEnvioSARIFSoTemIdQuandoOGitHubAceitou(t *testing.T) {
	repo := exigirBanco(t)
	taskRun := novoTaskRun(t, repo, novoTenant())

	_, err := pool.Exec(t.Context(), `
		INSERT INTO sarif_uploads (tenant_id, task_run_id, sarif_id, status)
		VALUES ($1, $2, $3, 'falhou')`, taskRun.TenantID, taskRun.ID, idDoEnvioSARIF)
	if codigoPostgres(err) != "23514" {
		t.Errorf("esperava check_violation (23514) para falha com id, obtive %v", err)
	}

	_, err = pool.Exec(t.Context(), `
		INSERT INTO sarif_uploads (tenant_id, task_run_id, status)
		VALUES ($1, $2, 'aceito')`, taskRun.TenantID, taskRun.ID)
	if codigoPostgres(err) != "23514" {
		t.Errorf("esperava check_violation (23514) para aceito sem id, obtive %v", err)
	}
}

func TestEnvioSARIFEhDesligadoPorRepositorio(t *testing.T) {
	repo := exigirBanco(t)
	tenant := novoTenant()

	if desligado, err := repo.EnvioSARIFDesligado(t.Context(), tenant, "Azulpasta/alvo"); err != nil || desligado {
		t.Fatalf("por padrão o envio não está desligado: %v, %v", desligado, err)
	}
	if err := repo.DesligarEnvioSARIF(t.Context(), tenant, "Azulpasta/alvo", "403 no envio"); err != nil {
		t.Fatalf("desligar: %v", err)
	}
	if err := repo.DesligarEnvioSARIF(t.Context(), tenant, "Azulpasta/alvo", "403 de novo"); err != nil {
		t.Fatalf("desligar de novo não é erro: %v", err)
	}

	if desligado, err := repo.EnvioSARIFDesligado(t.Context(), tenant, "Azulpasta/alvo"); err != nil || !desligado {
		t.Errorf("depois de desligar: %v, %v", desligado, err)
	}
	if desligado, _ := repo.EnvioSARIFDesligado(t.Context(), tenant, "Azulpasta/outro"); desligado {
		t.Error("desligar um repositório não desliga os outros")
	}
	if desligado, _ := repo.EnvioSARIFDesligado(t.Context(), novoTenant(), "Azulpasta/alvo"); desligado {
		t.Error("desligar em um tenant não desliga em outro")
	}

	var motivo string
	if err := pool.QueryRow(t.Context(), `SELECT reason FROM sarif_disabled_repositories WHERE tenant_id = $1`, tenant).Scan(&motivo); err != nil || motivo != "403 no envio" {
		t.Errorf("motivo gravado = %q, %v; o primeiro é o que explica o desligamento", motivo, err)
	}
}

package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/repository/db"
	"github.com/Azulpasta/aPRova/internal/sandbox"
	"github.com/jackc/pgx/v5"
)

// Reivindicar entrega ao trabalhador o task_run não terminal mais antigo que
// está livre, gravando quem o assumiu e até quando. Um task_run em
// aguardando_aprovacao só é entregue com review novo ou com push mais novo no
// mesmo PR. Sem trabalho disponível, devolve encontrado igual a false.
func (r *RepositorioPostgres) Reivindicar(ctx context.Context, trabalhador string, prazo time.Duration) (domain.TaskRun, bool, error) {
	linha, err := r.consultas.ClaimNextTaskRun(ctx, db.ClaimNextTaskRunParams{Worker: &trabalhador, LeaseSeconds: prazo.Seconds()})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.TaskRun{}, false, nil
	}
	if err != nil {
		return domain.TaskRun{}, false, traduzir("reivindicar task_run", err)
	}
	return paraTaskRun(linha), true, nil
}

// RenovarPrazo estende a reivindicação do trabalhador por mais prazo. Se o
// task_run não está mais com ele, devolve ErrPrazoPerdido.
func (r *RepositorioPostgres) RenovarPrazo(ctx context.Context, taskRunID, trabalhador string, prazo time.Duration) error {
	alteradas, err := r.consultas.RenewTaskRunLease(ctx, db.RenewTaskRunLeaseParams{ID: taskRunID, Worker: &trabalhador, LeaseSeconds: prazo.Seconds()})
	return exigirReivindicacao("renovar prazo", taskRunID, alteradas, err)
}

// Liberar devolve à fila o task_run do trabalhador, que volta a ser entregue
// depois da espera. Espera zero devolve na hora. Se o task_run não está mais
// com ele, devolve ErrPrazoPerdido.
func (r *RepositorioPostgres) Liberar(ctx context.Context, taskRunID, trabalhador string, espera time.Duration) error {
	alteradas, err := r.consultas.ReleaseTaskRun(ctx, db.ReleaseTaskRunParams{ID: taskRunID, Worker: &trabalhador, WaitSeconds: espera.Seconds()})
	return exigirReivindicacao("liberar task_run", taskRunID, alteradas, err)
}

// TransicionarReivindicado move o task_run do trabalhador de De para Para e
// registra a etapa na mesma transação. Recusa com ErrTransicaoProibida o que a
// máquina de estados não permite e com ErrTransicaoConcorrente o task_run que
// mudou de estado ou de dono. Zera as tentativas da etapa e, ao entrar em
// aguardando_aprovacao, marca como vistas as aprovações já gravadas.
func (r *RepositorioPostgres) TransicionarReivindicado(ctx context.Context, trabalhador string, transicao domain.Transicao) (domain.TaskRun, error) {
	if err := validar(transicao); err != nil {
		return domain.TaskRun{}, err
	}
	if !transicao.De.PodeIrPara(transicao.Para) {
		return domain.TaskRun{}, fmt.Errorf("%w: %s para %s", domain.ErrTransicaoProibida, transicao.De, transicao.Para)
	}

	var atualizado domain.TaskRun
	err := r.EmTransacao(ctx, func(tx *RepositorioPostgres) error {
		linha, err := tx.consultas.TransitionClaimedTaskRun(ctx, db.TransitionClaimedTaskRunParams{
			ID: transicao.TaskRunID, FromState: string(transicao.De), ToState: string(transicao.Para), Worker: &trabalhador,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("%w: %s não está em %s com %s", domain.ErrTransicaoConcorrente, transicao.TaskRunID, transicao.De, trabalhador)
		}
		if err != nil {
			return traduzir("transicionar task_run reivindicado", err)
		}

		if _, err := tx.consultas.CreateTaskStep(ctx, db.CreateTaskStepParams{
			TaskRunID: transicao.TaskRunID, FromState: string(transicao.De), ToState: string(transicao.Para), Detail: transicao.Detalhe,
		}); err != nil {
			return traduzir("registrar etapa", err)
		}

		atualizado = paraTaskRun(linha)
		return nil
	})
	return atualizado, err
}

// IncrementarTentativasDaEtapa conta mais uma tentativa da etapa atual e
// devolve o total. Se o task_run não está mais com o trabalhador, devolve
// ErrPrazoPerdido.
func (r *RepositorioPostgres) IncrementarTentativasDaEtapa(ctx context.Context, taskRunID, trabalhador string) (int, error) {
	tentativas, err := r.consultas.IncrementStepAttempts(ctx, db.IncrementStepAttemptsParams{ID: taskRunID, Worker: &trabalhador})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("%w: %s", domain.ErrPrazoPerdido, taskRunID)
	}
	if err != nil {
		return 0, traduzir("contar tentativa da etapa", err)
	}
	return int(tentativas), nil
}

// GravarResultadoDaTentativa grava o resultado do sandbox na tentativa
// informada. A mesma tentativa duas vezes devolve ErrDuplicado.
func (r *RepositorioPostgres) GravarResultadoDaTentativa(ctx context.Context, taskRunID string, tentativa int, resultado sandbox.ResultadoSandbox) error {
	if tentativa <= 0 || tentativa > maximoDeTentativas {
		return fmt.Errorf("%w: tentativa %d fora de 1..%d", domain.ErrDadosInvalidos, tentativa, maximoDeTentativas)
	}

	_, err := r.consultas.CreateSandboxResult(ctx, db.CreateSandboxResultParams{
		TaskRunID:   taskRunID,
		Attempt:     int32(tentativa), //nolint:gosec // validado entre 1 e maximoDeTentativas
		Compiled:    resultado.Compilou,
		TestsPassed: int32(resultado.TestesPassaram), //nolint:gosec // contagem de testes cabe em int32
		TestsFailed: int32(resultado.TestesFalharam), //nolint:gosec // contagem de testes cabe em int32
		RawOutput:   resultado.SaidaBruta,
		DurationMs:  resultado.DuracaoMS,
		FailureType: string(resultado.TipoFalha),
	})
	return traduzir("gravar resultado da tentativa", err)
}

// UltimoResultadoSandbox devolve o número e o resultado da tentativa mais
// recente do task_run, ou ErrNaoEncontrado se nenhuma foi gravada.
func (r *RepositorioPostgres) UltimoResultadoSandbox(ctx context.Context, taskRunID string) (int, sandbox.ResultadoSandbox, error) {
	linha, err := r.consultas.GetLatestSandboxResult(ctx, taskRunID)
	if err != nil {
		return 0, sandbox.ResultadoSandbox{}, traduzir("ler último resultado do sandbox", err)
	}
	return int(linha.Attempt), paraResultadoSandbox(linha), nil
}

// BuscarGateDeRisco devolve o gate gravado para o task_run, ou
// ErrNaoEncontrado.
func (r *RepositorioPostgres) BuscarGateDeRisco(ctx context.Context, taskRunID string) (domain.GateDeRisco, error) {
	linha, err := r.consultas.GetRiskGateByTaskRun(ctx, taskRunID)
	if err != nil {
		return domain.GateDeRisco{}, traduzir("buscar gate de risco", err)
	}
	gate := domain.GateDeRisco{
		TaskRunID:           linha.TaskRunID,
		NivelDeterministico: domain.NivelRisco(linha.DeterministicLevel),
		Nivel:               domain.NivelRisco(linha.Level),
		Motivo:              linha.Reason,
	}
	if err := json.Unmarshal(linha.RulesHit, &gate.Motivos); err != nil {
		return domain.GateDeRisco{}, fmt.Errorf("interpretar rules_hit do task_run %s: %w", taskRunID, err)
	}
	if err := json.Unmarshal(linha.NewDependencies, &gate.DependenciasNovas); err != nil {
		return domain.GateDeRisco{}, fmt.Errorf("interpretar new_dependencies do task_run %s: %w", taskRunID, err)
	}
	return gate, nil
}

func jsonDaLista[T any](lista []T) ([]byte, error) {
	if lista == nil {
		lista = []T{}
	}
	corpo, err := json.Marshal(lista)
	if err != nil {
		return nil, fmt.Errorf("serializar lista do gate: %w", err)
	}
	return corpo, nil
}

// ProximaAprovacao devolve a primeira aprovação do task_run com id maior que
// depoisDe, ou ErrNaoEncontrado.
func (r *RepositorioPostgres) ProximaAprovacao(ctx context.Context, taskRunID string, depoisDe int64) (domain.AprovacaoRegistrada, error) {
	linha, err := r.consultas.NextApprovalAfter(ctx, db.NextApprovalAfterParams{TaskRunID: taskRunID, AfterID: depoisDe})
	if err != nil {
		return domain.AprovacaoRegistrada{}, traduzir("buscar próxima aprovação", err)
	}
	return paraAprovacaoRegistrada(linha), nil
}

// AvancarCursorDeAprovacao marca como vistas as aprovações do task_run até o
// id informado. Se o task_run não está mais com o trabalhador, devolve
// ErrPrazoPerdido.
func (r *RepositorioPostgres) AvancarCursorDeAprovacao(ctx context.Context, taskRunID, trabalhador string, ate int64) error {
	alteradas, err := r.consultas.AdvanceApprovalCursor(ctx, db.AdvanceApprovalCursorParams{ID: taskRunID, Worker: &trabalhador, Seen: ate})
	return exigirReivindicacao("avançar cursor de aprovação", taskRunID, alteradas, err)
}

// ExisteTaskRunMaisNovo informa se o mesmo pull request recebeu um task_run
// criado depois deste.
func (r *RepositorioPostgres) ExisteTaskRunMaisNovo(ctx context.Context, taskRun domain.TaskRun) (bool, error) {
	numeroPR, err := numeroDePR(taskRun.NumeroPR)
	if err != nil {
		return false, err
	}

	maisNovo, err := r.consultas.NewerTaskRunExists(ctx, db.NewerTaskRunExistsParams{
		TenantID: taskRun.TenantID, Repository: taskRun.Repositorio, PrNumber: numeroPR, CreatedAt: taskRun.CriadoEm, ID: taskRun.ID,
	})
	if err != nil {
		return false, traduzir("procurar task_run mais novo", err)
	}
	return maisNovo, nil
}

func exigirReivindicacao(etapa, taskRunID string, alteradas int64, err error) error {
	if err != nil {
		return traduzir(etapa, err)
	}
	if alteradas == 0 {
		return fmt.Errorf("%s: %w: %s", etapa, domain.ErrPrazoPerdido, taskRunID)
	}
	return nil
}

func paraAprovacaoRegistrada(linha db.Approval) domain.AprovacaoRegistrada {
	var motivo *domain.MotivoFeedback
	if linha.FeedbackReason != nil {
		convertido := domain.MotivoFeedback(*linha.FeedbackReason)
		motivo = &convertido
	}

	return domain.AprovacaoRegistrada{
		ID: linha.ID,
		Aprovacao: domain.Aprovacao{
			TaskRunID:      linha.TaskRunID,
			DeliveryID:     linha.DeliveryID,
			Decisao:        domain.DecisaoAprovacao(linha.Decision),
			Revisor:        linha.Reviewer,
			Origem:         domain.OrigemAprovacao(linha.Origin),
			MotivoFeedback: motivo,
			Justificativa:  linha.Justification,
			DecididaEm:     linha.DecidedAt,
		},
	}
}

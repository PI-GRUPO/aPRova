package integracaopostgres

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/repository"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
	"github.com/jackc/pgx/v5/pgxpool"
)

const prazoCurto = 400 * time.Millisecond

var sequenciaDeBancos atomic.Int64

type bancoDaFila struct {
	repo *repository.RepositorioPostgres
	pool *pgxpool.Pool
}

func novoBancoDaFila(t *testing.T) bancoDaFila {
	t.Helper()
	exigirBanco(t)

	nome := fmt.Sprintf("fila_%d_%d", time.Now().UnixNano()%1_000_000, sequenciaDeBancos.Add(1))
	url := criarBancoVazio(t, nome)
	if err := aplicarMigrations(url); err != nil {
		t.Fatalf("migrar banco isolado: %v", err)
	}
	poolIsolado, err := repository.ConectarPostgres(t.Context(), url)
	if err != nil {
		t.Fatalf("conectar no banco isolado: %v", err)
	}
	t.Cleanup(poolIsolado.Close)

	return bancoDaFila{
		repo: repository.NovoRepositorioPostgres(poolIsolado, repository.OpcoesPostgres{IntervaloConsulta: 20 * time.Millisecond}),
		pool: poolIsolado,
	}
}

func (b bancoDaFila) contar(t *testing.T, consulta string, argumentos ...any) int {
	t.Helper()

	var total int
	if err := b.pool.QueryRow(t.Context(), consulta, argumentos...).Scan(&total); err != nil {
		t.Fatalf("contar (%s): %v", consulta, err)
	}
	return total
}

func (b bancoDaFila) novoTaskRunDoPR(t *testing.T, numeroPR int) domain.TaskRun {
	t.Helper()

	criado, _, err := b.repo.CriarTaskRun(t.Context(), domain.NovoTaskRun{
		TenantID:    58291734,
		DeliveryID:  fmt.Sprintf("entrega-%d-%d", numeroPR, time.Now().UnixNano()),
		Repositorio: "Azulpasta/aPRova-TESTES",
		NumeroPR:    numeroPR,
		SHAHead:     "3f9a1c7e2b4d6f8091a3c5e7f9b1d3e5a7c9e1f3",
		Autor:       "wlaisw",
		Acao:        "opened",
		RecebidoEm:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("criar task_run: %v", err)
	}
	return criado
}

func (b bancoDaFila) reivindicar(t *testing.T, trabalhador string) (domain.TaskRun, bool) {
	t.Helper()

	taskRun, encontrado, err := b.repo.Reivindicar(t.Context(), trabalhador, prazoCurto)
	if err != nil {
		t.Fatalf("reivindicar como %s: %v", trabalhador, err)
	}
	return taskRun, encontrado
}

func (b bancoDaFila) levarAte(t *testing.T, trabalhador string, taskRun domain.TaskRun, estados ...domain.EstadoTaskRun) domain.TaskRun {
	t.Helper()

	atual := taskRun
	for _, proximo := range estados {
		var err error
		atual, err = b.repo.TransicionarReivindicado(t.Context(), trabalhador, domain.Transicao{TaskRunID: atual.ID, De: atual.Estado, Para: proximo})
		if err != nil {
			t.Fatalf("transicionar %s para %s: %v", atual.Estado, proximo, err)
		}
	}
	return atual
}

func TestReivindicarEntregaOTaskRunMaisAntigoUmaVez(t *testing.T) {
	banco := novoBancoDaFila(t)
	primeiro := banco.novoTaskRunDoPR(t, 14)
	banco.novoTaskRunDoPR(t, 15)

	reivindicado, encontrado := banco.reivindicar(t, "trabalhador-a")

	if !encontrado || reivindicado.ID != primeiro.ID {
		t.Fatalf("esperava o task_run mais antigo %s, obtive %s (encontrado=%v)", primeiro.ID, reivindicado.ID, encontrado)
	}
	if reivindicado.ReivindicadoPor != "trabalhador-a" || reivindicado.PrazoAte == nil {
		t.Errorf("a reivindicação precisa gravar quem assumiu e até quando: %+v", reivindicado)
	}
	segundo, _ := banco.reivindicar(t, "trabalhador-b")
	if segundo.ID == primeiro.ID {
		t.Error("um task_run com prazo vigente foi entregue a outro trabalhador")
	}
}

func TestDoisTrabalhadoresDisputandoOMesmoTaskRunSoUmLeva(t *testing.T) {
	banco := novoBancoDaFila(t)
	banco.novoTaskRunDoPR(t, 14)

	var grupo sync.WaitGroup
	var vencedores atomic.Int64
	for indice := range 8 {
		grupo.Go(func() {
			_, encontrado, err := banco.repo.Reivindicar(t.Context(), fmt.Sprintf("trabalhador-%d", indice), prazoCurto)
			if err != nil {
				t.Errorf("reivindicar: %v", err)
			}
			if encontrado {
				vencedores.Add(1)
			}
		})
	}
	grupo.Wait()

	if vencedores.Load() != 1 {
		t.Errorf("%d trabalhadores levaram o mesmo task_run, esperava 1", vencedores.Load())
	}
}

func TestPrazoVencidoDevolveOTaskRunParaAFila(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)
	banco.reivindicar(t, "trabalhador-que-morreu")

	time.Sleep(prazoCurto + 100*time.Millisecond)
	retomado, encontrado := banco.reivindicar(t, "trabalhador-novo")

	if !encontrado || retomado.ID != taskRun.ID || retomado.ReivindicadoPor != "trabalhador-novo" {
		t.Fatalf("depois do prazo outro trabalhador deveria retomar %s, obtive %+v", taskRun.ID, retomado)
	}
	if err := banco.repo.RenovarPrazo(t.Context(), taskRun.ID, "trabalhador-que-morreu", prazoCurto); !errors.Is(err, domain.ErrPrazoPerdido) {
		t.Errorf("quem perdeu o prazo não pode renová-lo, esperava ErrPrazoPerdido, obtive %v", err)
	}
}

func TestRenovarPrazoMantemOTaskRunComQuemTrabalha(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)
	banco.reivindicar(t, "trabalhador-a")

	for range 3 {
		time.Sleep(prazoCurto / 2)
		if err := banco.repo.RenovarPrazo(t.Context(), taskRun.ID, "trabalhador-a", prazoCurto); err != nil {
			t.Fatalf("renovar: %v", err)
		}
	}

	if _, encontrado := banco.reivindicar(t, "trabalhador-b"); encontrado {
		t.Error("com o prazo renovado, o task_run não pode voltar para a fila")
	}
}

func TestLiberarDevolveNaHoraOuDepoisDaEspera(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)
	banco.reivindicar(t, "trabalhador-a")

	if err := banco.repo.Liberar(t.Context(), taskRun.ID, "trabalhador-a", prazoCurto); err != nil {
		t.Fatalf("liberar com espera: %v", err)
	}
	if _, encontrado := banco.reivindicar(t, "trabalhador-b"); encontrado {
		t.Fatal("liberado com espera, o task_run não pode voltar antes dela")
	}
	time.Sleep(prazoCurto + 100*time.Millisecond)
	if retomado, encontrado := banco.reivindicar(t, "trabalhador-b"); !encontrado || retomado.ID != taskRun.ID {
		t.Fatal("depois da espera o task_run precisa voltar para a fila")
	}

	if err := banco.repo.Liberar(t.Context(), taskRun.ID, "trabalhador-b", 0); err != nil {
		t.Fatalf("liberar na hora: %v", err)
	}
	if _, encontrado := banco.reivindicar(t, "trabalhador-c"); !encontrado {
		t.Error("liberado sem espera, o task_run precisa voltar na hora")
	}
	if err := banco.repo.Liberar(t.Context(), taskRun.ID, "trabalhador-b", 0); !errors.Is(err, domain.ErrPrazoPerdido) {
		t.Errorf("quem não tem o task_run não pode liberá-lo, esperava ErrPrazoPerdido, obtive %v", err)
	}
}

func TestEstadoTerminalNuncaEReivindicado(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)
	reivindicado, _ := banco.reivindicar(t, "trabalhador-a")
	banco.levarAte(t, "trabalhador-a", reivindicado, domain.EstadoFalhou)
	if err := banco.repo.Liberar(t.Context(), taskRun.ID, "trabalhador-a", 0); err != nil {
		t.Fatalf("liberar: %v", err)
	}

	if _, encontrado := banco.reivindicar(t, "trabalhador-b"); encontrado {
		t.Error("task_run em estado terminal voltou para a fila")
	}
}

func TestTransicionarReivindicadoGravaEstadoEEtapaJuntos(t *testing.T) {
	banco := novoBancoDaFila(t)
	banco.novoTaskRunDoPR(t, 14)
	reivindicado, _ := banco.reivindicar(t, "trabalhador-a")

	atualizado := banco.levarAte(t, "trabalhador-a", reivindicado, domain.EstadoMapeandoContexto)

	if atualizado.Estado != domain.EstadoMapeandoContexto {
		t.Errorf("estado = %s", atualizado.Estado)
	}
	etapas := banco.contar(t, `SELECT count(*) FROM task_steps WHERE task_run_id = $1 AND from_state = 'pendente' AND to_state = 'mapeando_contexto'`, reivindicado.ID)
	if etapas != 1 {
		t.Errorf("esperava 1 linha em task_steps, há %d", etapas)
	}
}

func TestTransicionarReivindicadoRecusaTransicaoForaDaTabela(t *testing.T) {
	banco := novoBancoDaFila(t)
	banco.novoTaskRunDoPR(t, 14)
	reivindicado, _ := banco.reivindicar(t, "trabalhador-a")

	_, err := banco.repo.TransicionarReivindicado(t.Context(), "trabalhador-a",
		domain.Transicao{TaskRunID: reivindicado.ID, De: domain.EstadoPendente, Para: domain.EstadoConcluido})

	if !errors.Is(err, domain.ErrTransicaoProibida) {
		t.Fatalf("esperava ErrTransicaoProibida, obtive %v", err)
	}
	if banco.contar(t, `SELECT count(*) FROM task_steps WHERE task_run_id = $1`, reivindicado.ID) != 0 {
		t.Error("a transição recusada deixou etapa gravada")
	}
}

func TestTransicionarReivindicadoExigeOPrazoDeQuemTransiciona(t *testing.T) {
	banco := novoBancoDaFila(t)
	banco.novoTaskRunDoPR(t, 14)
	reivindicado, _ := banco.reivindicar(t, "trabalhador-a")

	_, err := banco.repo.TransicionarReivindicado(t.Context(), "trabalhador-intruso",
		domain.Transicao{TaskRunID: reivindicado.ID, De: domain.EstadoPendente, Para: domain.EstadoMapeandoContexto})

	if !errors.Is(err, domain.ErrTransicaoConcorrente) {
		t.Fatalf("esperava ErrTransicaoConcorrente, obtive %v", err)
	}
	if atual, _ := banco.repo.BuscarTaskRun(t.Context(), reivindicado.ID); atual.Estado != domain.EstadoPendente {
		t.Errorf("o intruso mudou o estado para %s", atual.Estado)
	}
}

func TestTentativasDaEtapaContamEZeramNaTransicao(t *testing.T) {
	banco := novoBancoDaFila(t)
	banco.novoTaskRunDoPR(t, 14)
	reivindicado, _ := banco.reivindicar(t, "trabalhador-a")
	atual := banco.levarAte(t, "trabalhador-a", reivindicado, domain.EstadoMapeandoContexto)

	for esperado := 1; esperado <= 3; esperado++ {
		tentativas, err := banco.repo.IncrementarTentativasDaEtapa(t.Context(), atual.ID, "trabalhador-a")
		if err != nil || tentativas != esperado {
			t.Fatalf("tentativa %d: obtive %d, err %v", esperado, tentativas, err)
		}
	}

	atual = banco.levarAte(t, "trabalhador-a", atual, domain.EstadoTestandoSandbox)
	if atual.TentativasDaEtapa != 0 {
		t.Errorf("a etapa nova precisa começar sem tentativas, tem %d", atual.TentativasDaEtapa)
	}
}

func TestResultadoDoSandboxPorTentativa(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)

	if _, _, err := banco.repo.UltimoResultadoSandbox(t.Context(), taskRun.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("sem resultado, esperava ErrNaoEncontrado, obtive %v", err)
	}

	infraestrutura := sandbox.ResultadoSandbox{SaidaBruta: "job não iniciou", TipoFalha: sandbox.FalhaInfraestrutura}
	sucesso := sandbox.ResultadoSandbox{Compilou: true, TestesPassaram: 4, SaidaBruta: "ok", DuracaoMS: 900, TipoFalha: sandbox.FalhaNenhuma}
	for tentativa, resultado := range map[int]sandbox.ResultadoSandbox{1: infraestrutura, 2: infraestrutura, 3: sucesso} {
		if err := banco.repo.GravarResultadoDaTentativa(t.Context(), taskRun.ID, tentativa, resultado); err != nil {
			t.Fatalf("gravar tentativa %d: %v", tentativa, err)
		}
	}

	tentativa, ultimo, err := banco.repo.UltimoResultadoSandbox(t.Context(), taskRun.ID)
	if err != nil || tentativa != 3 || ultimo != sucesso {
		t.Errorf("último resultado = tentativa %d %+v, err %v", tentativa, ultimo, err)
	}
	if err := banco.repo.GravarResultadoDaTentativa(t.Context(), taskRun.ID, 3, sucesso); !errors.Is(err, domain.ErrDuplicado) {
		t.Errorf("a mesma tentativa duas vezes precisa ser recusada com ErrDuplicado, obtive %v", err)
	}
}

func TestBuscarGateDeRiscoDoTaskRun(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)

	if _, err := banco.repo.BuscarGateDeRisco(t.Context(), taskRun.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("sem gate, esperava ErrNaoEncontrado, obtive %v", err)
	}
	gate := domain.GateDeRisco{
		TaskRunID: taskRun.ID, NivelDeterministico: domain.RiscoAlto, Nivel: domain.RiscoAlto, Motivo: "dependencia_nova, falha_sandbox",
		Motivos: []domain.MotivoDeRisco{
			{ID: "dependencia_nova", Nivel: domain.RiscoAlto, Descricao: "o PR acrescenta dependência nova", Evidencias: []string{"go/go.mod"}},
			{ID: "falha_sandbox", Nivel: domain.RiscoAlto, Descricao: "os testes falharam no sandbox", Evidencias: []string{"1 de 11 testes falharam"}},
		},
		DependenciasNovas: []domain.DependenciaNova{{Ecossistema: "Go", Nome: "github.com/shopspring/decimal", Versao: "v1.4.0"}},
		Calibracao:        domain.Calibracao{Estado: domain.CalibracaoDesligada, Razao: "calibração desligada"},
	}
	if err := banco.repo.GravarGateDeRisco(t.Context(), gate); err != nil {
		t.Fatalf("gravar gate: %v", err)
	}

	lido, err := banco.repo.BuscarGateDeRisco(t.Context(), taskRun.ID)
	if err != nil || !reflect.DeepEqual(lido, gate) {
		t.Errorf("gate lido = %+v, esperava %+v, err %v", lido, gate, err)
	}
}

func TestGateSemMotivosGravaListasVazias(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)

	if err := banco.repo.GravarGateDeRisco(t.Context(), domain.GateDeRisco{
		TaskRunID: taskRun.ID, NivelDeterministico: domain.RiscoBaixo, Nivel: domain.RiscoBaixo, Motivo: "nenhuma regra acionada",
	}); err != nil {
		t.Fatalf("gravar gate: %v", err)
	}

	if vazias := banco.contar(t, `SELECT count(*) FROM risk_gates WHERE task_run_id = $1 AND rules_hit = '[]'::jsonb AND new_dependencies = '[]'::jsonb`, taskRun.ID); vazias != 1 {
		t.Error("sem motivos, as colunas precisam guardar listas vazias, não null")
	}
	lido, err := banco.repo.BuscarGateDeRisco(t.Context(), taskRun.ID)
	if err != nil || lido.Motivos == nil || lido.DependenciasNovas == nil {
		t.Errorf("a leitura devolve listas vazias, não nulas: %+v, err %v", lido, err)
	}
}

func TestChecksDoRiskGateRecusamJSONQueNaoELista(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)
	if err := banco.repo.GravarGateDeRisco(t.Context(), domain.GateDeRisco{
		TaskRunID: taskRun.ID, NivelDeterministico: domain.RiscoBaixo, Nivel: domain.RiscoBaixo, Motivo: "nenhuma regra acionada",
	}); err != nil {
		t.Fatalf("gravar gate: %v", err)
	}

	for _, coluna := range []string{"rules_hit", "new_dependencies"} {
		_, err := banco.pool.Exec(t.Context(), `UPDATE risk_gates SET `+coluna+` = '{"motivo":"x"}'::jsonb WHERE task_run_id = $1`, taskRun.ID)
		if codigoPostgres(err) != "23514" {
			t.Errorf("%s precisa recusar o que não é lista, esperava check_violation (23514), obtive %v", coluna, err)
		}
	}
}

func registrarAprovacaoDe(t *testing.T, banco bancoDaFila, taskRun domain.TaskRun, decisao domain.DecisaoAprovacao, origem domain.OrigemAprovacao) {
	t.Helper()

	err := banco.repo.RegistrarAprovacao(t.Context(), domain.Aprovacao{
		TaskRunID: taskRun.ID, DeliveryID: fmt.Sprintf("review-%d", time.Now().UnixNano()), Decisao: decisao,
		Revisor: "Azulpasta", Origem: origem, DecididaEm: time.Now().UTC(), CommitSHA: taskRun.SHAHead,
	})
	if err != nil {
		t.Fatalf("registrar aprovação: %v", err)
	}
}

func TestAguardandoAprovacaoSoVoltaParaAFilaComReviewNovo(t *testing.T) {
	banco := novoBancoDaFila(t)
	banco.novoTaskRunDoPR(t, 14)
	reivindicado, _ := banco.reivindicar(t, "trabalhador-a")
	registrarAprovacaoDe(t, banco, reivindicado, domain.DecisaoAprovada, domain.OrigemGitHub)
	aguardando := banco.levarAte(t, "trabalhador-a", reivindicado,
		domain.EstadoMapeandoContexto, domain.EstadoTestandoSandbox, domain.EstadoAvaliandoRisco, domain.EstadoAguardandoAprovacao)
	if err := banco.repo.Liberar(t.Context(), aguardando.ID, "trabalhador-a", 0); err != nil {
		t.Fatalf("liberar: %v", err)
	}

	if _, encontrado := banco.reivindicar(t, "trabalhador-b"); encontrado {
		t.Fatal("review feito antes de entrar em aguardando_aprovacao não pode tirar o PR da espera")
	}

	registrarAprovacaoDe(t, banco, aguardando, domain.DecisaoAlteracoesSolicitadas, domain.OrigemSlack)
	retomado, encontrado := banco.reivindicar(t, "trabalhador-b")
	if !encontrado || retomado.ID != aguardando.ID {
		t.Fatal("review novo precisa devolver o task_run em aguardando_aprovacao para a fila")
	}

	aprovacao, err := banco.repo.ProximaAprovacao(t.Context(), retomado.ID, retomado.UltimaAprovacaoVista)
	if err != nil {
		t.Fatalf("próxima aprovação: %v", err)
	}
	if aprovacao.Decisao != domain.DecisaoAlteracoesSolicitadas || aprovacao.Origem != domain.OrigemSlack || aprovacao.ID <= retomado.UltimaAprovacaoVista {
		t.Errorf("aprovação lida errado: %+v", aprovacao)
	}

	if err := banco.repo.AvancarCursorDeAprovacao(t.Context(), retomado.ID, "trabalhador-b", aprovacao.ID); err != nil {
		t.Fatalf("avançar cursor: %v", err)
	}
	if err := banco.repo.Liberar(t.Context(), retomado.ID, "trabalhador-b", 0); err != nil {
		t.Fatalf("liberar: %v", err)
	}
	if _, encontrado := banco.reivindicar(t, "trabalhador-c"); encontrado {
		t.Error("review já visto não pode devolver o task_run para a fila")
	}
	if _, err := banco.repo.ProximaAprovacao(t.Context(), retomado.ID, aprovacao.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Errorf("sem review novo, esperava ErrNaoEncontrado, obtive %v", err)
	}
}

func TestCommitMaisNovoDevolveOAntigoParaAFila(t *testing.T) {
	banco := novoBancoDaFila(t)
	antigo := banco.novoTaskRunDoPR(t, 14)
	reivindicado, _ := banco.reivindicar(t, "trabalhador-a")
	aguardando := banco.levarAte(t, "trabalhador-a", reivindicado,
		domain.EstadoMapeandoContexto, domain.EstadoTestandoSandbox, domain.EstadoAvaliandoRisco, domain.EstadoAguardandoAprovacao)
	if err := banco.repo.Liberar(t.Context(), aguardando.ID, "trabalhador-a", 0); err != nil {
		t.Fatalf("liberar: %v", err)
	}

	if maisNovo, err := banco.repo.ExisteTaskRunMaisNovo(t.Context(), aguardando); err != nil || maisNovo {
		t.Fatalf("sem push novo: maisNovo=%v err=%v", maisNovo, err)
	}
	banco.novoTaskRunDoPR(t, 15)
	if maisNovo, _ := banco.repo.ExisteTaskRunMaisNovo(t.Context(), aguardando); maisNovo {
		t.Fatal("task_run de outro PR não substitui este")
	}

	banco.novoTaskRunDoPR(t, 14)
	if maisNovo, err := banco.repo.ExisteTaskRunMaisNovo(t.Context(), aguardando); err != nil || !maisNovo {
		t.Fatalf("com push novo no mesmo PR: maisNovo=%v err=%v", maisNovo, err)
	}
	retomado, encontrado := banco.reivindicar(t, "trabalhador-b")
	if !encontrado || retomado.ID != antigo.ID {
		t.Errorf("o task_run substituído precisa voltar para a fila para ser encerrado, obtive %+v", retomado)
	}
}

func TestOrigemDaAprovacaoGravadaENoCheckDoBanco(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)

	err := banco.repo.RegistrarAprovacao(t.Context(), domain.Aprovacao{
		TaskRunID: taskRun.ID, DeliveryID: "sem-origem", Decisao: domain.DecisaoAprovada, Revisor: "Azulpasta", DecididaEm: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("aprovação sem origem: %v", err)
	}
	if banco.contar(t, `SELECT count(*) FROM approvals WHERE delivery_id = 'sem-origem' AND origin = 'github'`) != 1 {
		t.Error("aprovação sem origem informada precisa ficar como github")
	}

	_, err = banco.pool.Exec(t.Context(), `UPDATE approvals SET origin = 'trello' WHERE delivery_id = 'sem-origem'`)
	if err == nil || !strings.Contains(err.Error(), "approvals_origin_check") {
		t.Errorf("o CHECK de origin precisa recusar trello, obtive %v", err)
	}
}

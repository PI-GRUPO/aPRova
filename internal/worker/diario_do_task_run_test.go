package worker

import (
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/rastro"
)

func exigirIdentificacaoDoTaskRun(t *testing.T, linhas []string, taskRun domain.TaskRun) {
	t.Helper()

	if len(linhas) == 0 {
		t.Fatal("nenhuma linha registrada")
	}
	for _, linha := range linhas {
		var campos map[string]any
		if err := json.Unmarshal([]byte(linha), &campos); err != nil {
			t.Fatalf("linha de log não é JSON: %s", linha)
		}
		if campos["task_run_id"] != taskRun.ID || campos["tenant_id"] != float64(taskRun.TenantID) ||
			campos["repositorio"] != taskRun.Repositorio || campos["numero_pr"] != float64(taskRun.NumeroPR) {
			t.Errorf("linha sem a identificação completa do task_run: %s", linha)
		}
	}
}

func TestLinhasDoMotorIdentificamOTaskRunPorInteiro(t *testing.T) {
	c := novoCenario(domain.RiscoAlto)
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	exigirIdentificacaoDoTaskRun(t, c.diario.linhas(), taskRun)
}

func TestDependenciasRegistramComAIdentificacaoDoTaskRun(t *testing.T) {
	saida := &saidaSincronizada{}
	anterior := slog.Default()
	slog.SetDefault(slog.New(rastro.NovoHandler(slog.NewJSONHandler(saida, nil))))
	t.Cleanup(func() { slog.SetDefault(anterior) })

	c := novoCenario(domain.RiscoBaixo)
	c.opcoes.LeitorDePR = leitorQueRegistra{c.leitor}
	taskRun := c.armazenamento.novoTaskRun(14)

	c.conduzirTudo(t)

	exigirIdentificacaoDoTaskRun(t, saida.linhas(), taskRun)
}

func TestLinhasDoPoolDuranteOTaskRunIdentificamOTaskRunPorInteiro(t *testing.T) {
	c := novoCenario(domain.RiscoBaixo)
	c.sandbox.bloquear = true
	c.sandbox.iniciou = make(chan struct{})
	taskRun := c.armazenamento.novoTaskRun(14)
	diarioDoPool := &saidaSincronizada{}
	pool, err := NovoPool(OpcoesPool{
		Motor: c.motor(t), Armazenamento: c.armazenamento, Nome: "pool-de-teste",
		Prazo: 30 * time.Millisecond, IntervaloOcioso: time.Hour,
		Diario: slog.New(slog.NewJSONHandler(diarioDoPool, nil)),
	})
	if err != nil {
		t.Fatalf("montar o pool: %v", err)
	}

	go pool.Rodar(t.Context())
	<-c.sandbox.iniciou
	c.armazenamento.tomarPrazo(taskRun.ID, "outro-trabalhador")
	for range 100 {
		if len(diarioDoPool.linhas()) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	pool.Parar(time.Second)

	exigirIdentificacaoDoTaskRun(t, diarioDoPool.linhas(), taskRun)
}

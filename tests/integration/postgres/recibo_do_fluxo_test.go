package integracaopostgres

import (
	"sync"
	"testing"
	"time"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/platform"
	"github.com/Azulpasta/aPRova/internal/receipt"
	"github.com/Azulpasta/aPRova/internal/sandbox"
)

var (
	arquivoDeRiscoBaixo = platform.ArquivoAlterado{
		Nome: "go/frete/peso_zero_test.go", Situacao: "added", Adicoes: 3, Patch: "@@ -0,0 +1,3 @@\n+package frete\n+\n+func TestPesoZero(t *testing.T) {}\n",
	}
	arquivoDeRiscoAlto = platform.ArquivoAlterado{Nome: "go/go.mod", Situacao: "modified", Adicoes: 2, Patch: patchDeDependenciaNova}
)

func (b bancoDaFila) instanteDaSaidaDeAvaliandoRisco(t *testing.T, taskRunID string) time.Time {
	t.Helper()

	var instante time.Time
	err := b.pool.QueryRow(t.Context(), `SELECT created_at FROM task_steps WHERE task_run_id = $1 AND from_state = 'avaliando_risco'`, taskRunID).Scan(&instante)
	if err != nil {
		t.Fatalf("ler a transição que sai de avaliando_risco: %v", err)
	}
	return instante
}

func TestFluxoEmiteOReciboAntesDaBifurcacaoNosDoisCaminhos(t *testing.T) {
	caminhos := map[string]struct {
		arquivo platform.ArquivoAlterado
		estado  domain.EstadoTaskRun
		nivel   domain.NivelRisco
	}{
		"risco baixo segue para aplicar":       {arquivoDeRiscoBaixo, domain.EstadoConcluido, domain.RiscoBaixo},
		"risco alto para aguardando aprovação": {arquivoDeRiscoAlto, domain.EstadoAguardandoAprovacao, domain.RiscoAlto},
	}

	for nome, caminho := range caminhos {
		t.Run(nome, func(t *testing.T) {
			banco := novoBancoDaFila(t)
			taskRun := banco.novoTaskRunDoPR(t, 14)

			novaMontagemDoGate(banco, caminho.arquivo).rodarAte(t, taskRun.ID, caminho.estado)

			gravado, err := banco.repo.BuscarReciboPorTaskRun(t.Context(), taskRun.ID)
			if err != nil {
				t.Fatalf("o caminho precisa ter recibo: %v", err)
			}
			_, resultado, err := verificadorDeRecibos(t, banco.repo).Verificar(t.Context(), gravado.ID)
			if err != nil || !resultado.Valido {
				t.Fatalf("o recibo do fluxo precisa verificar: %+v, erro %v", resultado, err)
			}

			conteudo, err := receipt.LerConteudo([]byte(gravado.Conteudo))
			if err != nil {
				t.Fatalf("ler conteúdo: %v", err)
			}
			if conteudo.Nivel != caminho.nivel || conteudo.TaskRunID != taskRun.ID || conteudo.SHAHead != taskRun.SHAHead ||
				conteudo.DiffSHA256 != receipt.HashEmHex([]byte("diff")) || conteudo.Sandbox.Tentativa != 1 || conteudo.Sandbox.TipoFalha != string(sandbox.FalhaNenhuma) {
				t.Errorf("o recibo registra o que foi analisado: %+v", conteudo)
			}
			if saida := banco.instanteDaSaidaDeAvaliandoRisco(t, taskRun.ID); gravado.CriadoEm.After(saida) {
				t.Errorf("o recibo (%v) precisa existir antes de o task_run sair de avaliando_risco (%v)", gravado.CriadoEm, saida)
			}
		})
	}
}

func TestBytesLidosDoBancoSaoOsBytesAssinados(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)
	evidenciaComAcentoEAspas := domain.MotivoDeRisco{
		ID: "dependencia_nova", Nivel: domain.RiscoAlto, Descricao: "descrição",
		Evidencias: []string{"pom.xml: <dependency> \"não\" foi possível ler & interpretar\té ü 日本 \\ /"},
	}

	emitido, err := emissorDeRecibos(t, banco.repo).Emitir(t.Context(), receipt.Pedido{
		TaskRun: taskRun, Tentativa: 1, Diff: "diff",
		Gate: domain.GateDeRisco{
			TaskRunID: taskRun.ID, NivelDeterministico: domain.RiscoAlto, Nivel: domain.RiscoAlto, Motivo: "dependencia_nova",
			Motivos: []domain.MotivoDeRisco{evidenciaComAcentoEAspas},
		},
	})
	if err != nil {
		t.Fatalf("emitir: %v", err)
	}

	_, resultado, err := verificadorDeRecibos(t, banco.repo).Verificar(t.Context(), emitido.ID)
	if err != nil || !resultado.Valido {
		t.Errorf("o que volta do banco precisa ser, byte a byte, o que foi assinado: %+v, erro %v", resultado, err)
	}
}

func TestApagarOReciboDoMeioDaCadeiaFazOTerceiroFalharNoElo(t *testing.T) {
	banco := novoBancoDaFila(t)
	var recibos []domain.Recibo
	for numeroPR := 14; numeroPR <= 16; numeroPR++ {
		taskRun := banco.novoTaskRunDoPR(t, numeroPR)
		novaMontagemDoGate(banco, arquivoDeRiscoBaixo).rodarAte(t, taskRun.ID, domain.EstadoConcluido)

		gravado, err := banco.repo.BuscarReciboPorTaskRun(t.Context(), taskRun.ID)
		if err != nil {
			t.Fatalf("recibo do PR %d: %v", numeroPR, err)
		}
		recibos = append(recibos, gravado)
	}
	verificador := verificadorDeRecibos(t, banco.repo)

	if recibos[0].ReciboAnteriorID != nil || *recibos[1].ReciboAnteriorID != recibos[0].ID || *recibos[2].ReciboAnteriorID != recibos[1].ID {
		t.Fatalf("os três recibos do tenant precisam estar encadeados")
	}
	for _, recibo := range recibos {
		if _, resultado, err := verificador.Verificar(t.Context(), recibo.ID); err != nil || !resultado.Valido {
			t.Fatalf("cadeia íntegra: recibo %s %+v, erro %v", recibo.ID, resultado, err)
		}
	}

	for _, comando := range []string{
		`ALTER TABLE receipts DISABLE TRIGGER ALL`,
		`DELETE FROM receipts WHERE id = '` + recibos[1].ID + `'`,
		`ALTER TABLE receipts ENABLE TRIGGER ALL`,
	} {
		if _, err := banco.pool.Exec(t.Context(), comando); err != nil {
			t.Fatalf("%s: %v", comando, err)
		}
	}

	_, resultado, err := verificador.Verificar(t.Context(), recibos[2].ID)
	if err != nil || resultado.Valido || resultado.Falha != receipt.FalhaElo {
		t.Errorf("o terceiro aponta para um recibo que não existe mais: %+v, erro %v", resultado, err)
	}
	if _, resultado, err := verificador.Verificar(t.Context(), recibos[0].ID); err != nil || !resultado.Valido {
		t.Errorf("o primeiro continua válido: %+v, erro %v", resultado, err)
	}
}

func TestEmissoesSimultaneasComOEmissorRealFormamUmaCadeiaQueVerifica(t *testing.T) {
	const emissoes = 8

	banco := novoBancoDaFila(t)
	emissor := emissorDeRecibos(t, banco.repo)
	ids := make(chan string, emissoes)

	var espera sync.WaitGroup
	for numeroPR := range emissoes {
		taskRun := banco.novoTaskRunDoPR(t, numeroPR+1)
		espera.Go(func() {
			emitido, err := emissor.Emitir(t.Context(), receipt.Pedido{
				TaskRun: taskRun, Tentativa: 1, Diff: "diff",
				Gate: domain.GateDeRisco{TaskRunID: taskRun.ID, NivelDeterministico: domain.RiscoBaixo, Nivel: domain.RiscoBaixo, Motivo: "nenhuma regra acionada"},
			})
			if err != nil {
				t.Errorf("emitir: %v", err)
				return
			}
			ids <- emitido.ID
		})
	}
	espera.Wait()
	close(ids)

	verificador := verificadorDeRecibos(t, banco.repo)
	for id := range ids {
		if _, resultado, err := verificador.Verificar(t.Context(), id); err != nil || !resultado.Valido {
			t.Errorf("recibo %s: %+v, erro %v", id, resultado, err)
		}
	}
	if primeiros := banco.contar(t, `SELECT count(*) FROM receipts WHERE previous_receipt_id IS NULL`); primeiros != 1 {
		t.Errorf("a cadeia tem um só começo, encontrei %d", primeiros)
	}
	if sucessores := banco.contar(t, `SELECT count(DISTINCT previous_receipt_id) FROM receipts`); sucessores != emissoes-1 {
		t.Errorf("sem bifurcação cada recibo tem um anterior diferente: %d", sucessores)
	}
}

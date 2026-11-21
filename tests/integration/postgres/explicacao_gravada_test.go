package integracaopostgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/explicacao"
)

const explicacaoDoRiscoBaixo = "Este pull request ficou com risco baixo, então o merge está liberado. " +
	"A única observação é que go/frete/frete.go fica num caminho sensível, e vale uma revisão com calma antes do merge."

type modeloQueLeOBanco struct {
	responder func(ctx context.Context) (string, error)
}

func (m modeloQueLeOBanco) Gerar(ctx context.Context, _, _ string) (string, error) {
	return m.responder(ctx)
}

func TestExplicacaoGravadaVoltaIgualEASegundaNaoSobrescreve(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)
	primeira := domain.Explicacao{TaskRunID: taskRun.ID, Resultado: domain.ExplicacaoGerada, Texto: explicacaoDoRiscoBaixo, Modelo: "gemini-3.5-flash-lite"}

	if _, err := banco.repo.ExplicacaoDoTaskRun(t.Context(), taskRun.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("sem explicação gravada: %v", err)
	}
	if err := banco.repo.GravarExplicacao(t.Context(), primeira); err != nil {
		t.Fatalf("gravar: %v", err)
	}
	if err := banco.repo.GravarExplicacao(t.Context(), domain.Explicacao{TaskRunID: taskRun.ID, Resultado: domain.ExplicacaoFalhou, Modelo: "outro", Motivo: "timeout"}); err != nil {
		t.Fatalf("a segunda gravação não falha, só não sobrescreve: %v", err)
	}

	lida, err := banco.repo.ExplicacaoDoTaskRun(t.Context(), taskRun.ID)
	if err != nil || lida != primeira {
		t.Errorf("explicação lida = %+v (%v), esperava %+v", lida, err, primeira)
	}
}

func TestDescarteComMotivoLongoEGravadoCortado(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)

	err := banco.repo.GravarExplicacao(t.Context(), domain.Explicacao{
		TaskRunID: taskRun.ID, Resultado: domain.ExplicacaoDescartada, Modelo: "gemini-3.5-flash-lite", Motivo: strings.Repeat("é", 3000),
	})
	if err != nil {
		t.Fatalf("gravar descarte: %v", err)
	}

	lida, err := banco.repo.ExplicacaoDoTaskRun(t.Context(), taskRun.ID)
	if err != nil || lida.Resultado != domain.ExplicacaoDescartada || lida.Texto != "" || utf8.RuneCountInString(lida.Motivo) > 1000 {
		t.Errorf("descarte lido = resultado %s, texto %q, motivo com %d caracteres (%v)", lida.Resultado, lida.Texto, utf8.RuneCountInString(lida.Motivo), err)
	}
}

func TestReciboIdenticoComExplicacaoLigada(t *testing.T) {
	banco := novoBancoDaFila(t)
	taskRun := banco.novoTaskRunDoPR(t, 14)
	reciboAoChamarOModelo := ""
	modelo := modeloQueLeOBanco{responder: func(ctx context.Context) (string, error) {
		if err := banco.pool.QueryRow(ctx, `SELECT payload || signature::text FROM receipts WHERE task_run_id = $1`, taskRun.ID).Scan(&reciboAoChamarOModelo); err != nil {
			return "", err
		}
		return explicacaoDoRiscoBaixo, nil
	}}
	explicador, err := explicacao.NovoExplicador(explicacao.Opcoes{Modelo: modelo, NomeDoModelo: "modelo-de-teste", Registro: banco.repo})
	if err != nil {
		t.Fatalf("montar o explicador: %v", err)
	}
	motor, github := motorQuePublicaComExplicador(t, banco, domain.RiscoBaixo, explicador)

	conduzirAteParar(t, banco, motor)

	var reciboNoFim, payload string
	if err := banco.pool.QueryRow(t.Context(), `SELECT payload || signature::text, payload FROM receipts WHERE task_run_id = $1`, taskRun.ID).Scan(&reciboNoFim, &payload); err != nil {
		t.Fatalf("ler o recibo: %v", err)
	}
	if reciboAoChamarOModelo == "" || reciboNoFim != reciboAoChamarOModelo {
		t.Error("o recibo já estava assinado quando o modelo foi chamado e não mudou depois")
	}
	if strings.Contains(payload, "caminho sensível, e vale") || strings.Contains(strings.ToLower(payload), "explica") {
		t.Error("a explicação não entra no payload do recibo")
	}
	_, comentarios := github.estado()
	if len(comentarios) != 1 || !strings.Contains(comentarios[0].Corpo, explicacaoDoRiscoBaixo) {
		t.Errorf("o comentário traz a explicação gerada: %+v", comentarios)
	}
	if gravada, err := banco.repo.ExplicacaoDoTaskRun(t.Context(), taskRun.ID); err != nil || gravada.Resultado != domain.ExplicacaoGerada {
		t.Errorf("explicação gravada = %+v (%v)", gravada, err)
	}
	reciboID := banco.idDoRecibo(t, taskRun.ID)
	if _, resultado, err := verificadorDeRecibos(t, banco.repo).Verificar(t.Context(), reciboID); err != nil || !resultado.Valido {
		t.Errorf("o recibo continua válido: %+v (%v)", resultado, err)
	}
}

func (b bancoDaFila) idDoRecibo(t *testing.T, taskRunID string) string {
	t.Helper()

	var id string
	if err := b.pool.QueryRow(t.Context(), `SELECT id::text FROM receipts WHERE task_run_id = $1`, taskRunID).Scan(&id); err != nil {
		t.Fatalf("ler o id do recibo: %v", err)
	}
	return id
}

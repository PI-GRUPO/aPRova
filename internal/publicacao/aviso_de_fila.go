package publicacao

import (
	"context"
	"strconv"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/platform"
)

// ContadorDeAnalises informa quantas análises a instalação iniciou na última
// hora e o teto por hora que a fila aplica.
type ContadorDeAnalises interface {
	AnalisesIniciadasNaUltimaHora(ctx context.Context, tenantID int64) (int, error)
	TetoDeAnalisesPorHora() int
}

// AvisoDeFila mostra no pull request, com um Check Run em queued, que a
// análise espera porque a instalação está no teto. O Check Run fica sem
// conclusão até a análise terminar.
type AvisoDeFila struct {
	Publicador *Publicador
	Contador   ContadorDeAnalises
}

// AvisarSeAcimaDoTeto cria o Check Run em queued para o task_run quando a
// instalação já iniciou o teto de análises na última hora. Abaixo do teto, ou
// com Check Run já existente, nada muda.
func (a AvisoDeFila) AvisarSeAcimaDoTeto(ctx context.Context, taskRun domain.TaskRun) error {
	iniciadas, err := a.Contador.AnalisesIniciadasNaUltimaHora(ctx, taskRun.TenantID)
	teto := a.Contador.TetoDeAnalisesPorHora()
	if err != nil || iniciadas < teto {
		return err
	}
	return explicar(a.Publicador.avisarFila(ctx, taskRun, teto))
}

func (p *Publicador) avisarFila(ctx context.Context, taskRun domain.TaskRun, teto int) error {
	id, err := p.checkRunConhecido(ctx, taskRun)
	if err != nil || id != 0 {
		return err
	}
	return p.criarCheckRun(ctx, taskRun, checkRunNaFila(taskRun, teto))
}

func checkRunNaFila(taskRun domain.TaskRun, teto int) platform.CheckRun {
	return platform.CheckRun{
		Situacao: platform.CheckRunNaFila,
		Titulo:   "Na fila: teto de análises por hora",
		Resumo: "A instalação atingiu o teto de " + strconv.Itoa(teto) + " análises por hora. " +
			"A análise do commit " + codigo(shaCurto(taskRun.SHAHead)) + " não foi descartada: ela começa quando a janela abrir. " +
			"Até lá este check fica sem conclusão e não libera nada.",
	}
}

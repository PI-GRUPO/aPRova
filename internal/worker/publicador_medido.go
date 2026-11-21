package worker

import (
	"context"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/metricas"
)

type publicadorMedido struct {
	PublicadorPR
	metricas *metricas.Motor
}

func (p publicadorMedido) Iniciar(ctx context.Context, taskRun domain.TaskRun) error {
	defer p.medir(time.Now())
	return p.PublicadorPR.Iniciar(ctx, taskRun)
}

func (p publicadorMedido) Publicar(ctx context.Context, publicacao Publicacao) error {
	defer p.medir(time.Now())
	return p.PublicadorPR.Publicar(ctx, publicacao)
}

func (p publicadorMedido) PublicarFalha(ctx context.Context, taskRun domain.TaskRun, diagnostico string) error {
	defer p.medir(time.Now())
	return p.PublicadorPR.PublicarFalha(ctx, taskRun, diagnostico)
}

func (p publicadorMedido) medir(inicio time.Time) {
	p.metricas.DuracaoDaEtapa(metricas.EtapaPublicacao, time.Since(inicio))
}

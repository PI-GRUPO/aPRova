package domain

import "time"

type RelatorioSemanal struct {
	TenantID          int64
	Inicio            time.Time
	Fim               time.Time
	Total             int64
	Concluidas        int64
	Rejeitadas        int64
	Falhas            int64
	ExigiramAprovacao int64
	RiscoMedio        int64
	Revertidas        int64
}

func (r RelatorioSemanal) Finalizadas() int64 {
	return r.Concluidas + r.Rejeitadas + r.Falhas
}

func (r RelatorioSemanal) EmAndamento() int64 {
	return r.Total - r.Finalizadas()
}

func (r RelatorioSemanal) TaxaDeSucesso() *float64 {
	finalizadas := r.Finalizadas()
	if finalizadas == 0 {
		return nil
	}
	taxa := float64(r.Concluidas) / float64(finalizadas)
	return &taxa
}

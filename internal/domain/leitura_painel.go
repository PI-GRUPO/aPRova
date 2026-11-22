package domain

import "time"

// FiltroLeituraPainel recebe somente repositórios autorizados pelo servidor web.
type FiltroLeituraPainel struct {
	TenantID              int64
	Repositorios          []string
	Inicio, Fim, Snapshot time.Time
	Pagina, Limite        int32
	Decisao, Origem       string
}

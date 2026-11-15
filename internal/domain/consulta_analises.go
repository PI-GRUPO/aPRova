package domain

// FiltroAnalises limita consultas de leitura a uma instalação. Repositorio
// apenas restringe esse escopo. Em andamento inclui espera por aprovação.
type FiltroAnalises struct {
	TenantID     int64  `validate:"required,gt=0"`
	Repositorio  string `validate:"max=200"`
	Estado       string `validate:"required,oneof=em_andamento todos pendente mapeando_contexto gerando testando_sandbox avaliando_risco aguardando_aprovacao aplicando concluido rejeitado falhou"`
	NumeroPR     int32  `validate:"gte=0"`
	Limite       int32  `validate:"gte=1,lte=51"`
	Deslocamento int32  `validate:"gte=0,lte=10000"`
}

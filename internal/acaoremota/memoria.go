package acaoremota

import (
	"context"
	"maps"
	"sync"
)

// ResolvedorEmMemoria resolve logins a partir de um mapa fixo, carregado da
// configuração. Atende apenas desenvolvimento e testes.
type ResolvedorEmMemoria struct {
	loginPorUsuario map[string]string
}

// NovoResolvedorEmMemoria constrói o resolvedor a partir do mapa de usuário
// do Slack para login do GitHub.
func NovoResolvedorEmMemoria(loginPorUsuario map[string]string) *ResolvedorEmMemoria {
	return &ResolvedorEmMemoria{loginPorUsuario: maps.Clone(loginPorUsuario)}
}

// LoginGitHub devolve o login vinculado ao usuário do Slack, ou ErrSemVinculo
// quando não há vínculo.
func (r *ResolvedorEmMemoria) LoginGitHub(_ context.Context, usuarioSlack string) (string, error) {
	login, vinculado := r.loginPorUsuario[usuarioSlack]
	if !vinculado {
		return "", ErrSemVinculo
	}
	return login, nil
}

// RegistroEmMemoria guarda as tentativas na memória do processo, com a mesma
// regra da versão PostgreSQL: no máximo uma tentativa aceita por id_acao. Não
// sobrevive a reinício nem cruza processos.
type RegistroEmMemoria struct {
	mutex      sync.Mutex
	tentativas []Tentativa
}

// NovoRegistroEmMemoria constrói um registro vazio.
func NovoRegistroEmMemoria() *RegistroEmMemoria {
	return &RegistroEmMemoria{}
}

// Registrar guarda a tentativa. A aceita só entra por Reservar.
func (r *RegistroEmMemoria) Registrar(_ context.Context, tentativa Tentativa) error {
	if tentativa.Decisao == DecisaoAceita {
		return ErrDecisaoExigeReserva
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.tentativas = append(r.tentativas, tentativa)
	return nil
}

// Reservar guarda a tentativa como aceita, ou devolve ErrAcaoJaAceita quando
// o id_acao já tem uma tentativa aceita.
func (r *RegistroEmMemoria) Reservar(_ context.Context, tentativa Tentativa) (Reserva, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, registrada := range r.tentativas {
		if registrada.IDAcao == tentativa.IDAcao && registrada.Decisao == DecisaoAceita {
			return Reserva{}, ErrAcaoJaAceita
		}
	}

	tentativa.Decisao = DecisaoAceita
	r.tentativas = append(r.tentativas, tentativa)
	return Reserva{ID: int64(len(r.tentativas))}, nil
}

// RegistrarFalhaNaExecucao passa a tentativa reservada para falha, liberando o
// id_acao para uma nova tentativa.
func (r *RegistroEmMemoria) RegistrarFalhaNaExecucao(_ context.Context, reserva Reserva, motivo Motivo) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	indice := reserva.ID - 1
	if indice < 0 || indice >= int64(len(r.tentativas)) || r.tentativas[indice].Decisao != DecisaoAceita {
		return ErrReservaInexistente
	}

	r.tentativas[indice].Decisao = DecisaoFalha
	r.tentativas[indice].Motivo = motivo
	return nil
}

// Tentativas devolve uma cópia das tentativas registradas, na ordem em que
// chegaram.
func (r *RegistroEmMemoria) Tentativas() []Tentativa {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return append([]Tentativa(nil), r.tentativas...)
}

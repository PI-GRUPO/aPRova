package sessaoweb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

const recursoAuditoria = "auditoria"

type LeiturasPainel interface {
	UltimasAnalisesWeb(context.Context, int64, []string) (map[string]time.Time, error)
	LerPainel(context.Context, string, domain.FiltroLeituraPainel) (json.RawMessage, error)
}

func filtrosPainel(q url.Values, recurso string, agora time.Time) (domain.FiltroLeituraPainel, error) {
	f := domain.FiltroLeituraPainel{Pagina: 1, Limite: 20, Snapshot: agora.UTC()}
	nomes := []string{parametroInstalacaoWeb, "repositorio", "periodo"}
	colecao := recurso == "recibos" || recurso == recursoAuditoria
	if colecao {
		nomes = append(nomes, "pagina", "limite", "snapshot")
	}
	if recurso == recursoAuditoria {
		nomes = append(nomes, "decisao", "origem")
	}
	if e := parametrosWeb(q, nomes...); e != nil {
		return f, e
	}
	id, e := instalacaoWeb(q)
	if e != nil {
		return f, e
	}
	f.TenantID = id
	dias, e := inteiroWeb(q, "periodo", 30, 1, 90)
	if e != nil {
		return f, e
	}
	if colecao {
		if e := paginarLeitura(q, recurso, agora, &f); e != nil {
			return f, e
		}
	}
	f.Snapshot = f.Snapshot.UTC()
	f.Fim = f.Snapshot.Truncate(24*time.Hour).AddDate(0, 0, 1)
	f.Inicio = f.Fim.AddDate(0, 0, -int(dias))
	if recurso == recursoAuditoria {
		f.Decisao, e = enumerado(q, "decisao", "", "aceita", "recusada", "falha", "repetida", "")
		if e != nil {
			return f, e
		}
		f.Origem, e = enumerado(q, "origem", "", "Slack", "Painel", "")
		if e != nil {
			return f, e
		}
	}
	return f, nil
}

func paginarLeitura(q url.Values, recurso string, agora time.Time, f *domain.FiltroLeituraPainel) error {
	var e error
	f.Pagina, e = inteiroWeb(q, "pagina", 1, 1, 201)
	if e != nil {
		return e
	}
	teto := int32(50)
	if recurso == "recibos" {
		teto = 500
	}
	f.Limite, e = inteiroWeb(q, "limite", 20, 1, teto)
	if e != nil {
		return e
	}
	if q.Has("snapshot") {
		f.Snapshot, e = time.Parse(time.RFC3339Nano, q.Get("snapshot"))
		if e != nil || f.Snapshot.After(agora) || f.Snapshot.Year() < 1970 {
			return domain.ErrDadosInvalidos
		}
	} else if f.Pagina > 1 {
		return domain.ErrDadosInvalidos
	}
	return nil
}

func (s *Servidor) leituraPainel(w http.ResponseWriter, r *http.Request, recurso string) {
	if _, ok := s.autenticar(w, r); !ok {
		return
	}
	q := r.URL.Query()
	f, e := filtrosPainel(q, recurso, s.agora())
	if e != nil {
		falhar(w, 400, "filtros de leitura inválidos")
		return
	}
	repos, ok := s.escopoWeb(w, r, f.TenantID)
	if !ok {
		return
	}
	f.Repositorios = repos
	if s.leituras == nil {
		falhar(w, 503, "leitura do painel indisponível")
		return
	}
	dados, e := s.leituras.LerPainel(r.Context(), recurso, f)
	if e != nil {
		falhar(w, 503, "não foi possível consultar o painel")
		return
	}
	jsonResposta(w, 200, respostaPainel{f.TenantID, q.Get("repositorio"), f.Inicio, f.Fim, f.Snapshot, f.Pagina, f.Limite, dados})
}

type respostaPainel struct {
	Instalacao  int64           `json:"installation_id"`
	Repositorio string          `json:"repositorio"`
	Inicio      time.Time       `json:"inicio"`
	Fim         time.Time       `json:"fim"`
	Snapshot    time.Time       `json:"snapshot"`
	Pagina      int32           `json:"pagina"`
	Limite      int32           `json:"limite"`
	Dados       json.RawMessage `json:"dados"`
}

// O resolvedor de ações usa o login global configurado, sem workspace nem
// instalação. Expõe apenas a presença desse login para a própria pessoa.
func (s *Servidor) vinculoPessoal(w http.ResponseWriter, r *http.Request) {
	v, ok := s.autenticar(w, r)
	if !ok {
		return
	}
	if len(r.URL.Query()) != 0 {
		falhar(w, 400, "consulta pessoal não aceita filtros")
		return
	}
	u, e := s.provedor.Usuario(r.Context(), v.Token)
	if e != nil {
		s.erroProvedor(w, r, e)
		return
	}
	if u.ID != v.Usuario.ID || u.Login == "" {
		falhar(w, 401, "identidade da sessão inválida")
		return
	}
	vinculado := false
	for _, login := range s.vinculos {
		if strings.EqualFold(login, u.Login) {
			vinculado = true
			break
		}
	}
	jsonResposta(w, 200, map[string]any{"usuario_id": u.ID, "vinculado": vinculado})
}

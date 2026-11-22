package integracaopostgres

import (
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/demo"
	"github.com/PI-GRUPO/aPRova/internal/domain"
)

func TestLeiturasDoPainelNoPostgres(t *testing.T) {
	r := exigirBanco(t)
	tenant := novoTenant()
	outro := novoTenant()
	var agora time.Time
	if err := pool.QueryRow(t.Context(), "SELECT clock_timestamp()").Scan(&agora); err != nil {
		t.Fatal(err)
	}
	// Fixtures exclusivas do PostgreSQL efêmero, sem entrega de eventos.
	for _, id := range []int64{tenant, outro} {
		if _, err := demo.Semear(t.Context(), r, assinadorDaDemo(t), demo.Opcoes{InstallationID: id, Repositorio: repositorioDaDemo, Agora: agora}); err != nil {
			t.Fatal(err)
		}
	}
	for _, caso := range []struct {
		tenant        int64
		repo, decisao string
	}{{tenant, repositorioDaDemo, "recusada"}, {tenant, repositorioDaDemo, "falha"}, {tenant, "outro/privado", "recusada"}, {outro, repositorioDaDemo, "recusada"}} {
		_, err := pool.Exec(t.Context(), `INSERT INTO remote_action_attempts(action_id,tenant_id,action_type,slack_user_id,repository,pr_number,decision,reason,attempted_at)
 VALUES('fixture', $1,'aprovar','privado',$2,1,$3,'teste',$4)`, caso.tenant, caso.repo, caso.decisao, agora)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := pool.QueryRow(t.Context(), "SELECT clock_timestamp()").Scan(&agora); err != nil {
		t.Fatal(err)
	}
	f := domain.FiltroLeituraPainel{TenantID: tenant, Repositorios: []string{repositorioDaDemo}, Inicio: agora.AddDate(0, 0, -30), Fim: agora.AddDate(0, 0, 30), Snapshot: agora, Pagina: 1, Limite: 2}
	type resumo struct {
		Recebidas, Avaliadas, Alto, Pendentes, Total int
		Falhas                                       struct{ Infraestrutura, Teste int }
		Bloqueios                                    struct{ Aguardando, Aprovadas, Alteracoes int }
		Calibracoes                                  []json.RawMessage
		Itens                                        []struct {
			ID    string
			Repo  string
			Hash  string `json:"hash_sha256"`
			Chave string `json:"key_id"`
		}
		Mapa, Regras []json.RawMessage
		Ultimo       json.RawMessage `json:"ultimo_recibo"`
	}
	ler := func(nome string) resumo {
		t.Helper()
		b, e := r.LerPainel(t.Context(), nome, f)
		if e != nil {
			t.Fatal(nome, e)
		}
		var v resumo
		if e = json.Unmarshal(b, &v); e != nil {
			t.Fatal(e)
		}
		return v
	}
	if v := ler("metricas"); v.Recebidas != 9 || v.Pendentes != 1 || v.Avaliadas != 8 || v.Alto != 5 || v.Falhas.Infraestrutura != 1 || v.Falhas.Teste != 1 || v.Bloqueios.Aprovadas != 3 || v.Bloqueios.Alteracoes != 1 || v.Bloqueios.Aguardando != 1 || len(v.Mapa) == 0 || string(v.Ultimo) == "null" {
		t.Fatal(v)
	}
	if v := ler("regras"); len(v.Regras) != 5 || len(v.Calibracoes) != 8 {
		t.Fatal(v)
	}
	// Uma tentativa inserida depois do snapshot não desloca as páginas antigas,
	// mesmo que attempted_at se refira a uma ação anterior.
	if _, err := pool.Exec(t.Context(), `INSERT INTO remote_action_attempts(action_id,tenant_id,action_type,slack_user_id,repository,pr_number,decision,reason,attempted_at)
 VALUES('posterior',$1,'aprovar','privado',$2,1,'recusada','teste',$3)`, tenant, repositorioDaDemo, agora.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if v := ler("auditoria"); v.Total != 2 {
		t.Fatal(v)
	}
	f.Decisao = "recusada"
	if v := ler("auditoria"); v.Total != 1 || len(v.Itens) != 1 {
		t.Fatal(v)
	}
	f.Origem = "Painel"
	if v := ler("auditoria"); v.Total != 0 {
		t.Fatal(v)
	}
	f.Origem = ""
	f.Decisao = ""
	f.Limite = 1
	auditoria := ler("auditoria")
	f.Pagina = 2
	auditoria2 := ler("auditoria")
	if auditoria.Total != 2 || auditoria2.Total != 2 || len(auditoria.Itens) != 1 || len(auditoria2.Itens) != 1 || auditoria.Itens[0].ID == auditoria2.Itens[0].ID {
		t.Fatalf("paginação da auditoria: primeira página %+v, segunda página %+v", auditoria, auditoria2)
	}
	f.Pagina = 1
	f.Limite = 2
	ultimas, err := r.UltimasAnalisesWeb(t.Context(), tenant, []string{repositorioDaDemo, "inexistente"})
	if err != nil || len(ultimas) != 1 || ultimas[repositorioDaDemo].IsZero() {
		t.Fatal(ultimas, err)
	}
	p := ler("recibos")
	if p.Total != 8 || len(p.Itens) != 2 {
		t.Fatal(p)
	}
	for _, item := range p.Itens {
		recibo, err := r.BuscarRecibo(t.Context(), item.ID)
		if err != nil || recibo.TenantID != tenant || item.Repo != repositorioDaDemo || hex.EncodeToString(recibo.Hash) != item.Hash || recibo.IDChave != item.Chave {
			t.Fatal("metadados diferentes do recibo armazenado", item, err)
		}
	}
	f.Pagina = 2
	p2 := ler("recibos")
	if p2.Total != 8 || len(p2.Itens) != 2 || p.Itens[0] == p2.Itens[0] || p.Itens[1] == p2.Itens[0] || p.Itens[0] == p2.Itens[1] || p.Itens[1] == p2.Itens[1] {
		t.Fatal(p2)
	}
	f.Repositorios = []string{"restrito/privado"}
	for _, nome := range []string{"metricas", "recibos", "auditoria", "regras"} {
		v := ler(nome)
		if v.Total != 0 || v.Recebidas != 0 || len(v.Regras)+len(v.Itens)+len(v.Mapa) != 0 {
			t.Fatal(nome, v)
		}
	}
}

package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
)

const reciboDoCenario = "00000000-0000-4000-8000-000000000002"

// Fonte isolada atrás dos handlers de sessão no servidor Playwright.
type leiturasTeste struct{ recibos *receipt.Verificador }

func (leiturasTeste) UltimasAnalisesWeb(_ context.Context, instalacao int64, _ []string) (map[string]time.Time, error) {
	if instalacao != 42 {
		return nil, nil
	}
	return map[string]time.Time{repositorioDoCenario: instanteDoCenario}, nil
}

func (l leiturasTeste) LerPainel(ctx context.Context, recurso string, f domain.FiltroLeituraPainel) (json.RawMessage, error) {
	switch recurso {
	case "metricas", "recibos":
		recibo, err := l.recibos.Buscar(ctx, reciboDoCenario)
		if err != nil {
			return nil, err
		}
		item := map[string]any{"id": recibo.ID, "repo": repositorioDoCenario, "pr": 1, "emissao": instanteDoCenario}
		hash := hex.EncodeToString(recibo.Hash)
		if recurso == "metricas" {
			item["hash"] = hash
			return metricasDoCenario(f.TenantID, item)
		}
		item["sha"], item["hash_sha256"], item["key_id"] = shaDoCenario, hash, recibo.IDChave
		return paginaDoCenario(f, f.TenantID == 42, item)
	case "auditoria":
		visivel := f.TenantID == 42 && f.Origem != "Painel" && (f.Decisao == "" || f.Decisao == "recusada")
		item := map[string]any{
			"id": "1", "repo": repositorioDoCenario, "pr": 1, "data": instanteDoCenario,
			"pessoa": nil, "acao": "aprovar", "decisao": "recusada", "motivo": "permissao_insuficiente", "origem": "Slack",
		}
		return paginaDoCenario(f, visivel, item)
	case "regras":
		if f.TenantID != 42 {
			return json.RawMessage(`{"regras":[],"calibracoes":[]}`), nil
		}
		return json.RawMessage(`{"regras":[{"id":"area_sensivel","descricao":"Alteração em área sensível","nivel":"alto","acionamentos":1,"avaliadas_altas":1,"aprovadas":0}],
		    "calibracoes":[{"id":"00000000-0000-4000-8000-000000000003","repo":"equipe-e2e/projeto","pr":1,"inicial":"alto","final":"alto","resultado":"desligada"}]}`), nil
	}
	return nil, domain.ErrDadosInvalidos
}
func paginaDoCenario(f domain.FiltroLeituraPainel, visivel bool, item map[string]any) (json.RawMessage, error) {
	total := 0
	itens := []any{}
	if visivel {
		total = 1
		if f.Pagina == 1 {
			itens = append(itens, item)
		}
	}
	return json.Marshal(map[string]any{"total": total, "itens": itens})
}
func metricasDoCenario(instalacao int64, recibo map[string]any) (json.RawMessage, error) {
	quantidade := 0
	mapa := []any{}
	if instalacao == 42 {
		quantidade = 1
		mapa = append(mapa, map[string]any{"data": instanteDoCenario.Format(time.DateOnly), "alto": 0, "medio": 0, "baixo": 0, "nao_avaliado": 1})
	} else {
		recibo = nil
	}
	return json.Marshal(map[string]any{
		"recebidas": quantidade, "avaliadas": 0, "alto": 0, "pendentes": 0, "pendentes_sem_inicio": 0, "desde": nil,
		"tempo_medio_segundos": nil, "tentativas": quantidade, "falhas": map[string]int{"infraestrutura": quantidade, "teste": 0, "compilacao": 0},
		"bloqueios": map[string]int{"aguardando": 0, "aprovadas": 0, "alteracoes": 0, "sem_decisao": 0}, "mapa": mapa, "ultimo_recibo": recibo,
	})
}

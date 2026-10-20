package api

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

const (
	parametroInstalacao = "installation_id"
	parametroInicio     = "inicio"
	diasDoRelatorio     = 7
)

type respostaRelatorioSemanal struct {
	InstallationID          int64     `json:"installation_id"`
	Inicio                  time.Time `json:"inicio"`
	Fim                     time.Time `json:"fim"`
	TotalExecucoes          int64     `json:"total_execucoes"`
	Concluidas              int64     `json:"concluidas"`
	Rejeitadas              int64     `json:"rejeitadas"`
	Falhas                  int64     `json:"falhas"`
	EmAndamento             int64     `json:"em_andamento"`
	TaxaSucesso             *float64  `json:"taxa_sucesso"`
	ExigiramAprovacaoHumana int64     `json:"exigiram_aprovacao_humana"`
	RiscoMedio              int64     `json:"risco_medio"`
	Revertidas              int64     `json:"revertidas"`
}

func (s *Servidor) manipularRelatorioSemanal(w http.ResponseWriter, r *http.Request) {
	if motivo := s.motivoParaRecusarAssinaturaDoN8N(r.Header, []byte(r.URL.RawQuery)); motivo != "" {
		slog.Warn("relatório semanal recusado na autenticação", "motivo", motivo)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if s.relatorios == nil {
		responderJSON(w, http.StatusServiceUnavailable, respostaSimples{Status: "relatório indisponível"})
		return
	}

	consulta := r.URL.Query()

	tenantID, err := strconv.ParseInt(consulta.Get(parametroInstalacao), 10, 64)
	if err != nil || tenantID <= 0 {
		responderJSON(w, http.StatusBadRequest, respostaSimples{Status: "installation_id inválido"})
		return
	}

	inicio, err := s.inicioDoRelatorio(consulta.Get(parametroInicio))
	if err != nil {
		responderJSON(w, http.StatusBadRequest, respostaSimples{Status: "inicio deve ser uma data AAAA-MM-DD"})
		return
	}

	relatorio, err := s.relatorios.RelatorioSemanal(r.Context(), tenantID, inicio, inicio.AddDate(0, 0, diasDoRelatorio))
	if err != nil {
		slog.Error("falha ao montar relatório semanal", "installation_id", tenantID, "erro", err)
		responderJSON(w, http.StatusInternalServerError, respostaSimples{Status: "falha ao montar relatório"})
		return
	}

	responderJSON(w, http.StatusOK, respostaRelatorioSemanal{
		InstallationID:          relatorio.TenantID,
		Inicio:                  relatorio.Inicio,
		Fim:                     relatorio.Fim,
		TotalExecucoes:          relatorio.Total,
		Concluidas:              relatorio.Concluidas,
		Rejeitadas:              relatorio.Rejeitadas,
		Falhas:                  relatorio.Falhas,
		EmAndamento:             relatorio.EmAndamento(),
		TaxaSucesso:             relatorio.TaxaDeSucesso(),
		ExigiramAprovacaoHumana: relatorio.ExigiramAprovacao,
		RiscoMedio:              relatorio.RiscoMedio,
		Revertidas:              relatorio.Revertidas,
	})
}

func (s *Servidor) inicioDoRelatorio(valor string) (time.Time, error) {
	if valor != "" {
		return time.Parse(time.DateOnly, valor)
	}

	agora := s.agora().UTC()
	hoje := time.Date(agora.Year(), agora.Month(), agora.Day(), 0, 0, 0, 0, time.UTC)
	diasDesdeSegunda := (int(hoje.Weekday()) + 6) % 7
	return hoje.AddDate(0, 0, -diasDesdeSegunda-diasDoRelatorio), nil
}

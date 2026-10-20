package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/Azulpasta/aPRova/internal/domain"
)

type fonteDeRelatoriosFalsa struct {
	relatorio domain.RelatorioSemanal
	erro      error
	chamadas  int
	tenantID  int64
	inicio    time.Time
	fim       time.Time
}

func (f *fonteDeRelatoriosFalsa) RelatorioSemanal(_ context.Context, tenantID int64, inicio, fim time.Time) (domain.RelatorioSemanal, error) {
	f.chamadas++
	f.tenantID, f.inicio, f.fim = tenantID, inicio, fim
	if f.erro != nil {
		return domain.RelatorioSemanal{}, f.erro
	}
	relatorio := f.relatorio
	relatorio.TenantID, relatorio.Inicio, relatorio.Fim = tenantID, inicio, fim
	return relatorio, nil
}

type pedidoDeRelatorio struct {
	consulta   string
	timestamp  string
	assinatura string
}

func relatorioAssinado(consulta string) pedidoDeRelatorio {
	timestamp := strconv.FormatInt(agoraDeTeste.Unix(), 10)
	return pedidoDeRelatorio{
		consulta:   consulta,
		timestamp:  timestamp,
		assinatura: assinarAcao(segredoDasAcoesDeTeste, timestamp, consulta),
	}
}

func pedirRelatorio(t *testing.T, segredo string, fonte FonteDeRelatorios, pedido pedidoDeRelatorio) *httptest.ResponseRecorder {
	t.Helper()

	servidor := NovoServidor(Opcoes{
		SegredoAcoesRemotas: []byte(segredo),
		Relatorios:          fonte,
		Agora:               func() time.Time { return agoraDeTeste },
	})

	requisicao := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/reports/weekly?"+pedido.consulta, nil)
	if pedido.timestamp != "" {
		requisicao.Header.Set("X-Aprova-Timestamp", pedido.timestamp)
	}
	if pedido.assinatura != "" {
		requisicao.Header.Set("X-Aprova-Assinatura", pedido.assinatura)
	}

	gravador := httptest.NewRecorder()
	servidor.Rotas().ServeHTTP(gravador, requisicao)

	return gravador
}

func decodificarRelatorio(t *testing.T, gravador *httptest.ResponseRecorder) respostaRelatorioSemanal {
	t.Helper()

	var resposta respostaRelatorioSemanal
	if err := json.NewDecoder(gravador.Body).Decode(&resposta); err != nil {
		t.Fatalf("decodificar resposta: %v", err)
	}
	return resposta
}

func TestRelatorioSemanalDevolveAsMetricasDaSemanaAnterior(t *testing.T) {
	fonte := &fonteDeRelatoriosFalsa{relatorio: domain.RelatorioSemanal{
		Total: 10, Concluidas: 6, Rejeitadas: 1, Falhas: 1, ExigiramAprovacao: 4, RiscoMedio: 3, Revertidas: 2,
	}}

	gravador := pedirRelatorio(t, segredoDasAcoesDeTeste, fonte, relatorioAssinado("installation_id=42"))

	if gravador.Code != http.StatusOK {
		t.Fatalf("status = %d, esperava 200; corpo: %s", gravador.Code, gravador.Body)
	}

	inicioEsperado := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	fimEsperado := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	if fonte.tenantID != 42 || !fonte.inicio.Equal(inicioEsperado) || !fonte.fim.Equal(fimEsperado) {
		t.Errorf("fonte recebeu tenant=%d inicio=%s fim=%s", fonte.tenantID, fonte.inicio, fonte.fim)
	}

	resposta := decodificarRelatorio(t, gravador)
	esperado := respostaRelatorioSemanal{
		InstallationID: 42, Inicio: inicioEsperado, Fim: fimEsperado,
		TotalExecucoes: 10, Concluidas: 6, Rejeitadas: 1, Falhas: 1, EmAndamento: 2,
		ExigiramAprovacaoHumana: 4, RiscoMedio: 3, Revertidas: 2,
	}
	if resposta.TaxaSucesso == nil || *resposta.TaxaSucesso != 0.75 {
		t.Errorf("taxa_sucesso = %v, esperava 0.75", resposta.TaxaSucesso)
	}
	resposta.TaxaSucesso = nil
	if resposta != esperado {
		t.Errorf("resposta = %+v, esperava %+v", resposta, esperado)
	}
}

func TestRelatorioSemanalSemExecucaoFinalizadaDevolveTaxaNula(t *testing.T) {
	fonte := &fonteDeRelatoriosFalsa{relatorio: domain.RelatorioSemanal{Total: 3}}

	gravador := pedirRelatorio(t, segredoDasAcoesDeTeste, fonte, relatorioAssinado("installation_id=42"))

	var corpo map[string]any
	if err := json.NewDecoder(gravador.Body).Decode(&corpo); err != nil {
		t.Fatalf("decodificar resposta: %v", err)
	}
	taxa, presente := corpo["taxa_sucesso"]
	if !presente || taxa != nil {
		t.Errorf("taxa_sucesso = %v (presente=%v), esperava null", taxa, presente)
	}
	if corpo["em_andamento"] != float64(3) {
		t.Errorf("em_andamento = %v, esperava 3", corpo["em_andamento"])
	}
}

func TestRelatorioSemanalUsaOInicioInformado(t *testing.T) {
	fonte := &fonteDeRelatoriosFalsa{}

	gravador := pedirRelatorio(t, segredoDasAcoesDeTeste, fonte, relatorioAssinado("installation_id=42&inicio=2026-10-12"))

	if gravador.Code != http.StatusOK {
		t.Fatalf("status = %d, esperava 200", gravador.Code)
	}
	if !fonte.inicio.Equal(time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)) || !fonte.fim.Equal(time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("janela = %s a %s, esperava 2026-10-12 a 2026-10-19", fonte.inicio, fonte.fim)
	}
}

func TestSemanaPadraoEASemanaCompletaAnterior(t *testing.T) {
	casos := []struct {
		nome  string
		agora time.Time
		dia   int
	}{
		{"sexta", time.Date(2026, 10, 2, 15, 0, 0, 0, time.UTC), 21},
		{"segunda logo após a meia-noite", time.Date(2026, 9, 28, 0, 30, 0, 0, time.UTC), 21},
		{"domingo no fim do dia", time.Date(2026, 9, 27, 23, 59, 0, 0, time.UTC), 14},
		{"fuso diferente de UTC", time.Date(2026, 9, 27, 22, 0, 0, 0, time.FixedZone("BRT", -3*60*60)), 21},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			servidor := NovoServidor(Opcoes{Agora: func() time.Time { return caso.agora }})

			inicio, err := servidor.inicioDoRelatorio("")
			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}
			if esperado := time.Date(2026, 9, caso.dia, 0, 0, 0, 0, time.UTC); !inicio.Equal(esperado) {
				t.Errorf("inicio = %s, esperava %s", inicio, esperado)
			}
		})
	}
}

func TestRelatorioSemanalRecusaSemAutenticacaoValida(t *testing.T) {
	consulta := "installation_id=42"
	timestampAtual := strconv.FormatInt(agoraDeTeste.Unix(), 10)
	timestampVelho := strconv.FormatInt(agoraDeTeste.Add(-5*time.Minute-time.Second).Unix(), 10)
	assinadoParaOutroTenant := relatorioAssinado("installation_id=7")
	assinadoParaOutroTenant.consulta = consulta

	casos := []struct {
		nome    string
		segredo string
		pedido  pedidoDeRelatorio
	}{
		{"sem assinatura", segredoDasAcoesDeTeste, pedidoDeRelatorio{consulta: consulta, timestamp: timestampAtual}},
		{"sem timestamp", segredoDasAcoesDeTeste, pedidoDeRelatorio{consulta: consulta, assinatura: assinarAcao(segredoDasAcoesDeTeste, timestampAtual, consulta)}},
		{"assinada com outro segredo", segredoDasAcoesDeTeste, pedidoDeRelatorio{consulta: consulta, timestamp: timestampAtual, assinatura: assinarAcao("outro", timestampAtual, consulta)}},
		{"assinatura de outra consulta", segredoDasAcoesDeTeste, assinadoParaOutroTenant},
		{"timestamp velho", segredoDasAcoesDeTeste, pedidoDeRelatorio{consulta: consulta, timestamp: timestampVelho, assinatura: assinarAcao(segredoDasAcoesDeTeste, timestampVelho, consulta)}},
		{"segredo não configurado", "", relatorioAssinado(consulta)},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			fonte := &fonteDeRelatoriosFalsa{}

			gravador := pedirRelatorio(t, caso.segredo, fonte, caso.pedido)

			if gravador.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, esperava 401", gravador.Code)
			}
			if fonte.chamadas != 0 {
				t.Errorf("a fonte não deveria ser consultada sem autenticação")
			}
		})
	}
}

func TestRelatorioSemanalRecusaParametrosInvalidos(t *testing.T) {
	for _, consulta := range []string{
		"",
		"installation_id=0",
		"installation_id=-3",
		"installation_id=abc",
		"installation_id=42&inicio=12/10/2026",
		"installation_id=42&inicio=2026-13-01",
	} {
		t.Run(consulta, func(t *testing.T) {
			fonte := &fonteDeRelatoriosFalsa{}

			gravador := pedirRelatorio(t, segredoDasAcoesDeTeste, fonte, relatorioAssinado(consulta))

			if gravador.Code != http.StatusBadRequest {
				t.Errorf("status = %d, esperava 400", gravador.Code)
			}
			if fonte.chamadas != 0 {
				t.Errorf("a fonte não deveria ser consultada com parâmetro inválido")
			}
		})
	}
}

func TestRelatorioSemanalFalhaDaFonteDevolve500(t *testing.T) {
	fonte := &fonteDeRelatoriosFalsa{erro: errors.New("banco fora do ar")}

	gravador := pedirRelatorio(t, segredoDasAcoesDeTeste, fonte, relatorioAssinado("installation_id=42"))

	if gravador.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, esperava 500", gravador.Code)
	}
}

func TestRelatorioSemanalSemFonteDevolve503(t *testing.T) {
	gravador := pedirRelatorio(t, segredoDasAcoesDeTeste, nil, relatorioAssinado("installation_id=42"))

	if gravador.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, esperava 503", gravador.Code)
	}
}

package calibracao

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/embedding"
)

type embedderEspiao struct {
	mutex  sync.Mutex
	textos []string
	erro   error
	travar bool
}

func (e *embedderEspiao) Gerar(ctx context.Context, texto string) ([]float32, error) {
	e.mutex.Lock()
	e.textos = append(e.textos, texto)
	e.mutex.Unlock()
	if e.travar {
		select {}
	}
	if e.erro != nil {
		return nil, e.erro
	}
	return embedding.Falso{}.Gerar(ctx, texto)
}

func (e *embedderEspiao) chamadas() []string {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	return append([]string(nil), e.textos...)
}

type historicoFalso struct {
	mutex     sync.Mutex
	vizinhos  []domain.VizinhoDaCalibracao
	erro      error
	travar    bool
	consultas []domain.ConsultaDeVizinhos
}

func (h *historicoFalso) VizinhosParecidos(_ context.Context, consulta domain.ConsultaDeVizinhos) ([]domain.VizinhoDaCalibracao, error) {
	h.mutex.Lock()
	h.consultas = append(h.consultas, consulta)
	h.mutex.Unlock()
	if h.travar {
		select {}
	}
	return h.vizinhos, h.erro
}

func (h *historicoFalso) quantasConsultas() int {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return len(h.consultas)
}

func calibradorDeTeste(t *testing.T, embedder embedding.Embedder, historico Historico) *Calibrador {
	t.Helper()
	calibrador, err := NovoCalibrador(OpcoesCalibrador{Embedder: embedder, Historico: historico, Regra: RegraPadrao(), Timeout: 100 * time.Millisecond})
	if err != nil {
		t.Fatalf("montar o calibrador: %v", err)
	}
	return calibrador
}

func pedidoAlto(motivos ...domain.MotivoDeRisco) Pedido {
	if len(motivos) == 0 {
		motivos = []domain.MotivoDeRisco{areaSensivel()}
	}
	return Pedido{
		TaskRun:  domain.TaskRun{ID: "11111111-1111-4111-8111-111111111111", TenantID: tenantDaAnalise, Repositorio: "Azulpasta/alvo", NumeroPR: 42},
		Nivel:    domain.RiscoAlto,
		Motivos:  motivos,
		Arquivos: arquivosDoPR(),
	}
}

func TestCalibradorDesceComHistoricoPositivoEGuardaAAssinatura(t *testing.T) {
	embedder := &embedderEspiao{}
	historico := &historicoFalso{vizinhos: positivos(6)}

	resultado := calibradorDeTeste(t, embedder, historico).Calibrar(t.Context(), pedidoAlto())

	if resultado.Nivel != domain.RiscoMedio || resultado.Calibracao.Estado != domain.CalibracaoAplicada {
		t.Fatalf("%s, %+v", resultado.Nivel, resultado.Calibracao)
	}
	consulta := historico.consultas[0]
	if consulta.TenantID != tenantDaAnalise || consulta.Repositorio != "Azulpasta/alvo" || consulta.NumeroPR != 42 || consulta.Limite != 10 {
		t.Errorf("a busca é do tenant, exclui o próprio PR e pede k vizinhos: %+v", consulta)
	}
	if resultado.Assinatura == nil || resultado.Assinatura.Texto != Assinatura(pedidoAlto().Motivos, arquivosDoPR()) || len(resultado.Assinatura.Embedding) != domain.DimensoesEmbedding {
		t.Errorf("a assinatura e o vetor voltam para serem gravados: %+v", resultado.Assinatura)
	}
}

func TestSoAAssinaturaVaiAoProvedorDeEmbedding(t *testing.T) {
	embedder := &embedderEspiao{}

	calibradorDeTeste(t, embedder, &historicoFalso{}).Calibrar(t.Context(), pedidoAlto())

	textos := embedder.chamadas()
	if len(textos) != 1 || textos[0] != Assinatura(pedidoAlto().Motivos, arquivosDoPR()) {
		t.Fatalf("o provedor recebe a assinatura e nada mais: %q", textos)
	}
	if strings.Contains(textos[0], "hunter2") {
		t.Error("o código do PR foi enviado ao provedor")
	}
}

func TestEmbedderComErroMantemONivelERegistraOMotivo(t *testing.T) {
	historico := &historicoFalso{vizinhos: positivos(10)}
	embedder := &embedderEspiao{erro: errors.Join(embedding.ErrIndisponivel, errors.New("gemini respondeu 503"))}

	resultado := calibradorDeTeste(t, embedder, historico).Calibrar(t.Context(), pedidoAlto())

	if resultado.Nivel != domain.RiscoAlto || resultado.Calibracao.Estado != domain.CalibracaoEmbeddingIndisponivel {
		t.Fatalf("%s, %+v", resultado.Nivel, resultado.Calibracao)
	}
	if !strings.Contains(resultado.Calibracao.Razao, "503") {
		t.Errorf("a razão diz o que falhou: %q", resultado.Calibracao.Razao)
	}
	if resultado.Assinatura != nil || historico.quantasConsultas() != 0 {
		t.Error("sem vetor não há o que gravar nem com o que buscar")
	}
	if len(resultado.Motivos) != 1 {
		t.Errorf("os motivos do gate seguem iguais: %+v", resultado.Motivos)
	}
}

func TestEmbedderQueTravaNaoAtrasaAAnaliseAlemDoTimeout(t *testing.T) {
	inicio := time.Now()

	resultado := calibradorDeTeste(t, &embedderEspiao{travar: true}, &historicoFalso{}).Calibrar(t.Context(), pedidoAlto())

	if decorrido := time.Since(inicio); decorrido > time.Second {
		t.Errorf("o timeout de 100ms levou %v", decorrido)
	}
	if resultado.Nivel != domain.RiscoAlto || resultado.Calibracao.Estado != domain.CalibracaoEmbeddingIndisponivel || !strings.Contains(resultado.Calibracao.Razao, "timeout") {
		t.Errorf("%s, %+v", resultado.Nivel, resultado.Calibracao)
	}
}

func TestHistoricoQueFalhaOuTravaMantemONivelEGuardaOVetor(t *testing.T) {
	for nome, historico := range map[string]*historicoFalso{
		"erro":  {erro: errors.New("conexão recusada")},
		"trava": {travar: true},
	} {
		t.Run(nome, func(t *testing.T) {
			inicio := time.Now()

			resultado := calibradorDeTeste(t, &embedderEspiao{}, historico).Calibrar(t.Context(), pedidoAlto())

			if time.Since(inicio) > time.Second {
				t.Error("o histórico travado atrasou a análise")
			}
			if resultado.Nivel != domain.RiscoAlto || resultado.Calibracao.Estado != domain.CalibracaoHistoricoIndisponivel {
				t.Errorf("%s, %+v", resultado.Nivel, resultado.Calibracao)
			}
			if resultado.Assinatura == nil {
				t.Error("o vetor foi gerado e entra no histórico mesmo assim")
			}
		})
	}
}

func TestMotivoNaoCalibravelGeraOVetorMasNaoBuscaVizinhos(t *testing.T) {
	historico := &historicoFalso{vizinhos: positivos(10)}

	resultado := calibradorDeTeste(t, &embedderEspiao{}, historico).Calibrar(t.Context(), pedidoAlto(areaSensivel(), motivo("segredo_versionado", domain.RiscoAlto)))

	if resultado.Calibracao.Estado != domain.CalibracaoMotivoNaoCalibravel || resultado.Nivel != domain.RiscoAlto {
		t.Errorf("%s, %+v", resultado.Nivel, resultado.Calibracao)
	}
	if resultado.Assinatura == nil {
		t.Error("toda análise avaliada entra no histórico")
	}
	if historico.quantasConsultas() != 0 {
		t.Error("sem calibração possível, não há busca")
	}
}

func TestNivelBaixoGeraOVetorMasNaoBuscaVizinhos(t *testing.T) {
	historico := &historicoFalso{vizinhos: positivos(10)}
	pedido := pedidoAlto()
	pedido.Nivel, pedido.Motivos = domain.RiscoBaixo, nil

	resultado := calibradorDeTeste(t, &embedderEspiao{}, historico).Calibrar(t.Context(), pedido)

	if resultado.Calibracao.Estado != domain.CalibracaoSemNivelAReduzir || resultado.Assinatura == nil || historico.quantasConsultas() != 0 {
		t.Errorf("%+v, assinatura %v, consultas %d", resultado.Calibracao, resultado.Assinatura != nil, historico.quantasConsultas())
	}
}

func TestMotivoNaoCalibravelComEmbeddingForaDoArRegistraOsDois(t *testing.T) {
	embedder := &embedderEspiao{erro: errors.New("gemini fora do ar")}

	resultado := calibradorDeTeste(t, embedder, &historicoFalso{}).Calibrar(t.Context(), pedidoAlto(motivo("falha_sandbox", domain.RiscoAlto)))

	if resultado.Calibracao.Estado != domain.CalibracaoMotivoNaoCalibravel {
		t.Errorf("o motivo determinístico vem primeiro: %+v", resultado.Calibracao)
	}
	if !strings.Contains(resultado.Calibracao.Razao, "falha_sandbox") || !strings.Contains(resultado.Calibracao.Razao, "gemini fora do ar") {
		t.Errorf("a razão registra o motivo e a falha do embedding: %q", resultado.Calibracao.Razao)
	}
}

func TestCalibradorIncompletoNaoEMontado(t *testing.T) {
	if _, err := NovoCalibrador(OpcoesCalibrador{Historico: &historicoFalso{}}); !errors.Is(err, ErrCalibradorIncompleto) {
		t.Errorf("sem embedder: %v", err)
	}
	if _, err := NovoCalibrador(OpcoesCalibrador{Embedder: embedding.Falso{}}); !errors.Is(err, ErrCalibradorIncompleto) {
		t.Errorf("sem histórico: %v", err)
	}
}

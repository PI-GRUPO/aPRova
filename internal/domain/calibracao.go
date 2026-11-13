package domain

// NivelInformativo é o nível de um motivo que explica a decisão sem pesar
// nela, como o da calibração por histórico.
const NivelInformativo NivelRisco = "informativo"

// EstadoCalibracao é o que aconteceu com a calibração por histórico de uma
// análise: se rodou, se mudou o nível e, quando não mudou, por quê.
type EstadoCalibracao string

// Estados da calibração gravados em risk_gates.calibration_status.
const (
	CalibracaoDesligada             EstadoCalibracao = "desligada"
	CalibracaoSemNivelAReduzir      EstadoCalibracao = "sem_motivo_a_calibrar"
	CalibracaoMotivoNaoCalibravel   EstadoCalibracao = "motivo_nao_calibravel"
	CalibracaoEmbeddingIndisponivel EstadoCalibracao = "embedding_indisponivel"
	CalibracaoHistoricoIndisponivel EstadoCalibracao = "historico_indisponivel"
	CalibracaoHistoricoInsuficiente EstadoCalibracao = "historico_insuficiente"
	CalibracaoVizinhoNegativo       EstadoCalibracao = "vizinho_negativo"
	CalibracaoAplicada              EstadoCalibracao = "calibrada"
)

// DesfechoDoVizinho é o que aconteceu com uma análise passada, derivado na
// hora da consulta a partir das aprovações, do fechamento do pull request e
// das reversões.
type DesfechoDoVizinho string

// Desfechos de um vizinho. Só VizinhoAprovadoEMergeado é positivo; só
// VizinhoRevertido e VizinhoFechadoSemMerge são negativos; os demais não
// contam como desfecho conhecido.
const (
	VizinhoAprovadoEMergeado    DesfechoDoVizinho = "aprovado_e_mergeado"
	VizinhoRevertido            DesfechoDoVizinho = "revertido"
	VizinhoFechadoSemMerge      DesfechoDoVizinho = "fechado_sem_merge"
	VizinhoMergeadoSemAprovacao DesfechoDoVizinho = "mergeado_sem_aprovacao"
	VizinhoSubstituido          DesfechoDoVizinho = "substituido"
	VizinhoEmAberto             DesfechoDoVizinho = "em_aberto"
)

// VizinhoDaCalibracao é uma análise passada parecida com a atual: o task_run,
// o pull request dele, a similaridade de cosseno, o desfecho e os ids dos
// motivos do gate determinístico dele.
type VizinhoDaCalibracao struct {
	TaskRunID    string            `json:"task_run_id"`
	TenantID     int64             `json:"-"`
	Repositorio  string            `json:"repositorio"`
	NumeroPR     int               `json:"numero_pr"`
	Similaridade float64           `json:"similaridade"`
	Desfecho     DesfechoDoVizinho `json:"desfecho"`
	Motivos      []string          `json:"motivos,omitempty"`
}

// Calibracao é o rastro da calibração de uma análise: o estado, a razão em
// texto e os vizinhos considerados.
type Calibracao struct {
	Estado   EstadoCalibracao
	Razao    string
	Vizinhos []VizinhoDaCalibracao
}

// AssinaturaDoPR é o texto que representa a análise no histórico e o vetor
// dele, gravados em code_memory.
type AssinaturaDoPR struct {
	Texto     string
	Embedding []float32
}

// ConsultaDeVizinhos pede ao histórico do tenant as análises mais parecidas
// com o embedding, fora do próprio pull request.
type ConsultaDeVizinhos struct {
	TenantID    int64
	Repositorio string
	NumeroPR    int
	Embedding   []float32
	Limite      int
}

package domain

import "time"

// FiltroAnalisesWeb recebe o escopo de repositórios exclusivamente do autorizador.
type FiltroAnalisesWeb struct {
	TenantID              int64
	Repositorios          []string
	NumeroPR              int32
	Estado, Nivel, Ordem  string
	Inicio, Fim, Snapshot time.Time
	Pagina, Limite        int32
}
type AnaliseWeb struct {
	ID           string    `json:"id"`
	Repositorio  string    `json:"repositorio"`
	NumeroPR     int32     `json:"numero_pr"`
	SHAHead      string    `json:"sha_head"`
	Autor        string    `json:"autor"`
	Estado       string    `json:"estado"`
	Nivel        *string   `json:"nivel"`
	RecebidoEm   time.Time `json:"recebido_em"`
	CriadoEm     time.Time `json:"criado_em"`
	AtualizadoEm time.Time `json:"atualizado_em"`
}
type PaginaAnalisesWeb struct {
	InstallationID int64        `json:"installation_id"`
	Snapshot       time.Time    `json:"snapshot"`
	Pagina         int32        `json:"pagina"`
	Limite         int32        `json:"limite"`
	Total          int64        `json:"total"`
	TemMais        bool         `json:"tem_mais"`
	Itens          []AnaliseWeb `json:"itens"`
}
type EtapaWeb struct {
	ID       int64     `json:"id"`
	De       string    `json:"de"`
	Para     string    `json:"para"`
	Detalhe  string    `json:"detalhe"`
	CriadoEm time.Time `json:"criado_em"`
}
type TentativaWeb struct {
	Tentativa      int32     `json:"tentativa"`
	Compilou       bool      `json:"compilou"`
	TestesPassaram int32     `json:"testes_passaram"`
	TestesFalharam int32     `json:"testes_falharam"`
	Saida          string    `json:"saida"`
	SaidaTruncada  bool      `json:"saida_truncada"`
	DuracaoMS      int64     `json:"duracao_ms"`
	TipoFalha      string    `json:"tipo_falha"`
	CriadoEm       time.Time `json:"criado_em"`
}
type GateWeb struct {
	NivelDeterministico string                `json:"nivel_deterministico"`
	Nivel               string                `json:"nivel"`
	Motivo              string                `json:"motivo"`
	Motivos             []MotivoDeRisco       `json:"motivos"`
	CalibracaoEstado    string                `json:"calibracao_estado"`
	CalibracaoRazao     string                `json:"calibracao_razao"`
	Vizinhos            []VizinhoDaCalibracao `json:"vizinhos"`
}
type AprovacaoWeb struct {
	ID                int64     `json:"id"`
	Decisao           string    `json:"decisao"`
	Revisor           string    `json:"revisor"`
	Origem            string    `json:"origem"`
	Justificativa     *string   `json:"justificativa"`
	CommitSHA         *string   `json:"commit_sha"`
	AplicavelAoCommit bool      `json:"aplicavel_ao_commit"`
	DecididoEm        time.Time `json:"decidido_em"`
}
type DetalheAnaliseWeb struct {
	InstallationID                int64          `json:"installation_id"`
	Analise                       AnaliseWeb     `json:"analise"`
	Branch                        *string        `json:"branch"`
	URLPR                         string         `json:"url_pr"`
	URLCheck                      *string        `json:"url_check"`
	Atual                         bool           `json:"atual"`
	Gate                          *GateWeb       `json:"gate"`
	Etapas                        []EtapaWeb     `json:"etapas"`
	Tentativas                    []TentativaWeb `json:"tentativas"`
	Aprovacoes                    []AprovacaoWeb `json:"aprovacoes"`
	ReciboID                      *string        `json:"recibo_id"`
	TestesEstruturadosDisponiveis bool           `json:"testes_estruturados_disponiveis"`
	RedeIsolada                   *bool          `json:"rede_isolada"`
}

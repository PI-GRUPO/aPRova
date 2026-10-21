package eventos

import "github.com/Azulpasta/aPRova/internal/domain"

// Motivo é uma regra determinística acionada na análise.
type Motivo struct {
	Nome      string `json:"nome" validate:"required,max=100"`
	Descricao string `json:"descricao" validate:"required,max=500"`
}

// ResultadoDaAnalise são os campos comuns a analise_concluida e
// aguardando_aprovacao, menos o nível de risco.
type ResultadoDaAnalise struct {
	TaskRunID   string   `json:"task_run_id" validate:"required,uuid"`
	Repositorio string   `json:"repositorio" validate:"required,repositorio_github"`
	NumeroPR    int      `json:"numero_pr" validate:"required,gt=0"`
	URLPR       string   `json:"url_pr" validate:"required,http_url"`
	AutorPR     string   `json:"autor_pr" validate:"required,login_github"`
	SHA         string   `json:"sha" validate:"required,len=40,hex_minusculo"`
	Motivos     []Motivo `json:"motivos" validate:"required,dive"`
	Explicacao  string   `json:"explicacao" validate:"required,max=4000"`
	ReceiptHash string   `json:"receipt_hash" validate:"required,len=64,hex_minusculo"`
	URLRecibo   string   `json:"url_recibo" validate:"required,http_url"`
}

// DadosAnaliseConcluida são os dados de analise_concluida, emitido ao fim de
// toda análise, em qualquer nível de risco.
type DadosAnaliseConcluida struct {
	ResultadoDaAnalise
	NivelRisco domain.NivelRisco `json:"nivel_risco" validate:"required,oneof=baixo medio alto"`
}

// DadosAguardandoAprovacao são os dados de aguardando_aprovacao, emitido quando
// o risco é alto e o PR fica bloqueado até uma decisão humana.
type DadosAguardandoAprovacao struct {
	ResultadoDaAnalise
	NivelRisco domain.NivelRisco `json:"nivel_risco" validate:"required,eq=alto"`
}

// DadosAprovacaoRegistrada são os dados de aprovacao_registrada, emitido quando
// um humano decide sobre um PR bloqueado.
type DadosAprovacaoRegistrada struct {
	TaskRunID          string                  `json:"task_run_id" validate:"required,uuid"`
	Repositorio        string                  `json:"repositorio" validate:"required,repositorio_github"`
	NumeroPR           int                     `json:"numero_pr" validate:"required,gt=0"`
	URLPR              string                  `json:"url_pr" validate:"required,http_url"`
	Decisao            domain.DecisaoAprovacao `json:"decisao" validate:"required,oneof=aprovada alteracoes_solicitadas"`
	LoginGitHub        string                  `json:"login_github" validate:"required,login_github"`
	Origem             string                  `json:"origem" validate:"required,oneof=github slack"`
	NivelRiscoOriginal domain.NivelRisco       `json:"nivel_risco_original" validate:"required,oneof=baixo medio alto"`
	IDAprovacao        int64                   `json:"id_aprovacao" validate:"required,gt=0"`
}

// Dependencia é um pacote adicionado pelo PR, com o ecossistema nomeado como
// no OSV.dev.
type Dependencia struct {
	Ecossistema string `json:"ecossistema" validate:"required,oneof=Go PyPI Maven"`
	Nome        string `json:"nome" validate:"required,max=300"`
	Versao      string `json:"versao" validate:"required,max=100"`
}

// DadosDependenciaNova são os dados de dependencia_nova, emitido quando o PR
// adiciona ao menos uma dependência.
type DadosDependenciaNova struct {
	TaskRunID    string        `json:"task_run_id" validate:"required,uuid"`
	Repositorio  string        `json:"repositorio" validate:"required,repositorio_github"`
	URLPR        string        `json:"url_pr" validate:"required,http_url"`
	Dependencias []Dependencia `json:"dependencias" validate:"required,min=1,dive"`
}

// DadosFalhaRecuperavel são os dados de falha_recuperavel, emitido quando a
// análise falha por infraestrutura e pode ser reexecutada.
type DadosFalhaRecuperavel struct {
	TaskRunID   string               `json:"task_run_id" validate:"required,uuid"`
	Repositorio string               `json:"repositorio" validate:"required,repositorio_github"`
	NumeroPR    int                  `json:"numero_pr" validate:"required,gt=0"`
	URLPR       string               `json:"url_pr" validate:"required,http_url"`
	Etapa       domain.EstadoTaskRun `json:"etapa" validate:"required,oneof=pendente mapeando_contexto gerando testando_sandbox avaliando_risco aplicando"`
	Diagnostico string               `json:"diagnostico" validate:"required,max=4000"`
	Tentativa   int                  `json:"tentativa" validate:"required,gt=0"`
}

// DadosInstalacao são os dados de instalacao, emitido quando o GitHub App é
// instalado, removido ou muda de repositórios.
type DadosInstalacao struct {
	Acao         string   `json:"acao" validate:"required,oneof=instalado removido repositorios_adicionados repositorios_removidos"`
	Conta        string   `json:"conta" validate:"required,login_github"`
	Repositorios []string `json:"repositorios" validate:"required,dive,repositorio_github"`
}

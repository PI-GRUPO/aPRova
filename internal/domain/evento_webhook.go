package domain

import "time"

// EventoPullRequest descreve um evento de pull request aceito para processamento.
type EventoPullRequest struct {
	InstallationID int64
	DeliveryID     string
	Repositorio    string
	NumeroPR       int
	SHAHead        string
	Autor          string
	Acao           string
	RecebidoEm     time.Time
}

// DecisaoDeReview descreve a decisão de uma revisão humana sobre um pull request.
// CommitSHA é o commit revisado, o review.commit_id do GitHub.
type DecisaoDeReview struct {
	InstallationID int64
	DeliveryID     string
	Repositorio    string
	NumeroPR       int
	Estado         string
	Revisor        string
	Origem         OrigemAprovacao
	RecebidoEm     time.Time
	CommitSHA      string
}

// Execucao identifica o registro de execução associado a um evento recebido.
type Execucao struct {
	ID           string
	JaRegistrada bool
}

// FechamentoDePR descreve um pull request fechado, com ou sem merge. SHAHead é
// o head no fechamento e SHADoMerge o commit do merge, vazio sem merge.
// Reverte é o número do pull request do mesmo repositório que este reverte, ou
// zero.
type FechamentoDePR struct {
	InstallationID int64  `validate:"required,gt=0"`
	DeliveryID     string `validate:"required,max=100"`
	Repositorio    string `validate:"required,max=200"`
	NumeroPR       int    `validate:"required,gt=0,max=2147483647"`
	Mergeado       bool
	SHAHead        string    `validate:"required,hexadecimal,lowercase,len=40|len=64"`
	SHADoMerge     string    `validate:"required_if=Mergeado true,excluded_unless=Mergeado true,omitempty,hexadecimal,lowercase,len=40|len=64"`
	FechadoEm      time.Time `validate:"required"`
	Reverte        int       `validate:"gte=0,max=2147483647"`
}

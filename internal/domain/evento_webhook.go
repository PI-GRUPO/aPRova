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

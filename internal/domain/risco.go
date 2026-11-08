package domain

// MotivoDeRisco é uma regra do gate que disparou: o identificador estável, o
// nível que ela impõe, uma descrição curta em português e a evidência.
type MotivoDeRisco struct {
	ID         string     `json:"id"`
	Nivel      NivelRisco `json:"nivel"`
	Descricao  string     `json:"descricao"`
	Evidencias []string   `json:"evidencias"`
}

// DependenciaNova é um pacote que o pull request acrescenta, com o ecossistema
// nomeado como no OSV.dev.
type DependenciaNova struct {
	Ecossistema string `json:"ecossistema"`
	Nome        string `json:"nome"`
	Versao      string `json:"versao"`
}

package receipt

import (
	"bytes"
	"crypto/ed25519"
)

// Falha nomeia a verificação que reprovou um recibo.
type Falha string

// Verificações que podem reprovar um recibo. Vazio é nenhuma.
const (
	FalhaHash              Falha = "hash"
	FalhaChaveDesconhecida Falha = "chave_desconhecida"
	FalhaAssinatura        Falha = "assinatura"
	FalhaConteudo          Falha = "conteudo"
	FalhaElo               Falha = "elo"
)

// Assinado é um recibo como guardado: os bytes exatos que foram assinados, o
// SHA-256 deles, a assinatura e o key_id da chave que assinou.
type Assinado struct {
	Conteudo   []byte
	Hash       []byte
	Assinatura []byte
	IDChave    string
}

// Resultado é o desfecho da verificação de um recibo. Quando inválido, Falha
// diz qual verificação reprovou e Detalhe explica.
type Resultado struct {
	Valido  bool
	Falha   Falha
	Detalhe string
}

// Anterior é o recibo para o qual um elo aponta, como está guardado.
type Anterior struct {
	ID       string
	Conteudo []byte
}

func reprovado(falha Falha, detalhe string) Resultado {
	return Resultado{Falha: falha, Detalhe: detalhe}
}

// Verificar confere um recibo só com as chaves públicas do chaveiro: o hash
// é o dos bytes, o key_id é de uma chave conhecida, a assinatura é dessa chave
// sobre esses bytes e o conteúdo é do formato e cita o mesmo key_id.
func (c *Chaveiro) Verificar(recibo Assinado) Resultado {
	if !bytes.Equal(Hash(recibo.Conteudo), recibo.Hash) {
		return reprovado(FalhaHash, "o SHA-256 do conteúdo não é o hash informado")
	}

	publica, conhecida := c.chave(recibo.IDChave)
	if !conhecida {
		return reprovado(FalhaChaveDesconhecida, "nenhuma chave pública conhecida tem o key_id "+recibo.IDChave)
	}
	if len(recibo.Assinatura) != ed25519.SignatureSize || !ed25519.Verify(publica, recibo.Conteudo, recibo.Assinatura) {
		return reprovado(FalhaAssinatura, "a assinatura não confere com o conteúdo e a chave "+recibo.IDChave)
	}

	conteudo, err := LerConteudo(recibo.Conteudo)
	if err != nil {
		return reprovado(FalhaConteudo, err.Error())
	}
	if conteudo.KeyID != recibo.IDChave {
		return reprovado(FalhaConteudo, "o key_id assinado não é o da chave que assinou")
	}
	return Resultado{Valido: true}
}

// VerificarElo confere a ligação do recibo com o anterior do mesmo tenant.
// anterior é o recibo guardado para o qual a cadeia aponta, ou nil quando não
// há. Devolve vazio quando o elo está íntegro, ou a explicação da quebra.
func VerificarElo(conteudo Conteudo, anterior *Anterior) string {
	switch {
	case conteudo.Anterior == nil && anterior == nil:
		return ""
	case conteudo.Anterior == nil:
		return "o recibo diz ser o primeiro da cadeia, mas há um anterior registrado"
	case anterior == nil:
		return "o recibo anterior " + conteudo.Anterior.ID + " não existe mais"
	case anterior.ID != conteudo.Anterior.ID:
		return "o recibo anterior registrado não é o que o conteúdo assinado cita"
	case HashEmHex(anterior.Conteudo) != conteudo.Anterior.Hash:
		return "o conteúdo do recibo anterior não tem mais o hash que este recibo assinou"
	default:
		return ""
	}
}

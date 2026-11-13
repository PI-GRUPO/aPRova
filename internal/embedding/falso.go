package embedding

import (
	"context"
	"crypto/sha256"
	"encoding/binary"

	"github.com/PI-GRUPO/aPRova/internal/domain"
)

// Falso gera, sem rede, um vetor determinístico derivado do SHA-256 do texto:
// o mesmo texto dá sempre o mesmo vetor e textos diferentes dão vetores quase
// ortogonais. Serve aos testes.
type Falso struct{}

// Gerar devolve o vetor normalizado derivado do texto.
func (Falso) Gerar(_ context.Context, texto string) ([]float32, error) {
	vetor := make([]float32, 0, domain.DimensoesEmbedding)
	for bloco := uint32(0); len(vetor) < domain.DimensoesEmbedding; bloco++ {
		var semente [4]byte
		binary.BigEndian.PutUint32(semente[:], bloco)
		resumo := sha256.Sum256(append(semente[:], texto...))
		for indice := 0; indice+2 <= len(resumo) && len(vetor) < domain.DimensoesEmbedding; indice += 2 {
			vetor = append(vetor, float32(binary.BigEndian.Uint16(resumo[indice:]))/32768-1)
		}
	}
	return Normalizar(vetor)
}

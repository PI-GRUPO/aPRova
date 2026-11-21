package platform

import (
	"errors"
	"net/http"
	"testing"
)

const commitDeTeste = "2222222222222222222222222222222222222222"

func TestPaisDoCommitLeOCommitSoComGET(t *testing.T) {
	servidor, recebida := servidorDaAPIQueResponde(t, http.StatusOK,
		`{"sha":"`+commitDeTeste+`","parents":[{"sha":"aaa"},{"sha":"bbb"}]}`)
	tokens := &fornecedorDeTokenFalso{}

	pais, err := clienteContra(servidor, tokens).PaisDoCommit(t.Context(), 777, "PI-GRUPO", "aPRova-TESTES", commitDeTeste)

	if err != nil {
		t.Fatalf("ler o commit: %v", err)
	}
	if pais != 2 {
		t.Errorf("pais = %d, esperava 2", pais)
	}
	if recebida.metodo != http.MethodGet || recebida.caminho != "/repos/PI-GRUPO/aPRova-TESTES/commits/"+commitDeTeste {
		t.Errorf("chamada %s %s", recebida.metodo, recebida.caminho)
	}
	if len(tokens.pedidos) != 1 || tokens.pedidos[0] != 777 {
		t.Errorf("token pedido para %v", tokens.pedidos)
	}
}

func TestPaisDoCommitInexistenteENaoEncontrado(t *testing.T) {
	servidor, _ := servidorDaAPIQueResponde(t, http.StatusNotFound, `{"message":"No commit found"}`)

	_, err := clienteContra(servidor, &fornecedorDeTokenFalso{}).PaisDoCommit(t.Context(), 777, "PI-GRUPO", "aPRova-TESTES", commitDeTeste)

	if !errors.Is(err, ErrNaoEncontrado) {
		t.Errorf("esperava ErrNaoEncontrado, obtive %v", err)
	}
}

func TestPaisDoCommitSemPaisNaRespostaEInesperado(t *testing.T) {
	servidor, _ := servidorDaAPIQueResponde(t, http.StatusOK, `{"sha":"`+commitDeTeste+`"}`)

	_, err := clienteContra(servidor, &fornecedorDeTokenFalso{}).PaisDoCommit(t.Context(), 777, "PI-GRUPO", "aPRova-TESTES", commitDeTeste)

	if !errors.Is(err, ErrRespostaInesperada) {
		t.Errorf("esperava ErrRespostaInesperada, obtive %v", err)
	}
}

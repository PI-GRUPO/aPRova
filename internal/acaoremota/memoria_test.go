package acaoremota

import (
	"errors"
	"testing"
)

func TestResolvedorEmMemoriaDevolveOLoginVinculado(t *testing.T) {
	resolvedor := NovoResolvedorEmMemoria(map[string]string{"U01": "alice"})

	login, err := resolvedor.LoginGitHub(t.Context(), "U01")

	if err != nil || login != "alice" {
		t.Errorf("esperava alice sem erro, obtive %q e %v", login, err)
	}
}

func TestResolvedorEmMemoriaSinalizaUsuarioSemVinculo(t *testing.T) {
	resolvedor := NovoResolvedorEmMemoria(map[string]string{"U01": "alice"})

	_, err := resolvedor.LoginGitHub(t.Context(), "U02")

	if !errors.Is(err, ErrSemVinculo) {
		t.Errorf("esperava ErrSemVinculo, obtive %v", err)
	}
}

func TestResolvedorEmMemoriaNaoEnxergaMudancaNoMapaOriginal(t *testing.T) {
	vinculos := map[string]string{"U01": "alice"}
	resolvedor := NovoResolvedorEmMemoria(vinculos)

	vinculos["U02"] = "mallory"

	if _, err := resolvedor.LoginGitHub(t.Context(), "U02"); !errors.Is(err, ErrSemVinculo) {
		t.Error("o resolvedor precisa guardar a própria cópia do mapa")
	}
}

func TestRegistroEmMemoriaDevolveCopiaNaOrdemDeChegada(t *testing.T) {
	registro := NovoRegistroEmMemoria()
	_ = registro.Registrar(t.Context(), Tentativa{IDAcao: "a"})
	_ = registro.Registrar(t.Context(), Tentativa{IDAcao: "b"})

	tentativas := registro.Tentativas()
	tentativas[0].IDAcao = "adulterada"

	novamente := registro.Tentativas()
	if len(novamente) != 2 || novamente[0].IDAcao != "a" || novamente[1].IDAcao != "b" {
		t.Errorf("registro alterado por fora ou fora de ordem: %+v", novamente)
	}
}

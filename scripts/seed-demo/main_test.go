package main

import (
	"errors"
	"testing"
)

func semVariaveis(string) string { return "" }

func TestInstalacaoPelaOpcao(t *testing.T) {
	params, err := lerParametros([]string{"-instalacao", "42"}, semVariaveis)
	if err != nil || params.instalacao != 42 || params.repositorio != repositorioPadrao {
		t.Errorf("esperava instalação 42 no repositório padrão, obtive %+v %v", params, err)
	}
}

func TestInstalacaoPelaVariavelDoChat(t *testing.T) {
	lerVariavel := func(nome string) string {
		if nome == "APROVA_CHAT_INSTALLATION_ID" {
			return "777"
		}
		return ""
	}

	params, err := lerParametros(nil, lerVariavel)
	if err != nil || params.instalacao != 777 {
		t.Errorf("esperava a instalação de APROVA_CHAT_INSTALLATION_ID, obtive %+v %v", params, err)
	}
}

func TestOpcaoTemPrecedenciaSobreAVariavel(t *testing.T) {
	params, err := lerParametros([]string{"-instalacao", "5"}, func(string) string { return "777" })
	if err != nil || params.instalacao != 5 {
		t.Errorf("a opção deveria valer mais que a variável, obtive %+v %v", params, err)
	}
}

func TestSemInstalacaoExplicaComoInformar(t *testing.T) {
	if _, err := lerParametros(nil, semVariaveis); !errors.Is(err, errSemInstalacao) {
		t.Errorf("esperava errSemInstalacao, obtive %v", err)
	}
}

func TestInstalacaoInvalidaNaVariavel(t *testing.T) {
	if _, err := lerParametros(nil, func(string) string { return "abc" }); err == nil {
		t.Error("APROVA_CHAT_INSTALLATION_ID não numérico deveria falhar")
	}
}

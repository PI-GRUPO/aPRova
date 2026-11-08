package nomesgithub

import (
	"strings"
	"testing"
)

func TestLoginValido(t *testing.T) {
	casos := map[string]bool{
		"Azulpasta":             true,
		"pedro-mello":           true,
		"a":                     true,
		strings.Repeat("a", 39): true,
		strings.Repeat("a", 40): false,
		"":                      false,
		"-comeca-com-hifen":     false,
		"com espaco":            false,
		"a-prova[bot]":          false,
		"dono/repositorio":      false,
	}

	for login, esperado := range casos {
		if obtido := LoginValido(login); obtido != esperado {
			t.Errorf("LoginValido(%q) = %v, esperava %v", login, obtido, esperado)
		}
	}
}

func TestRepositorioValido(t *testing.T) {
	casos := map[string]bool{
		"Azulpasta/aPRova-TESTES":          true,
		"PI-GRUPO/aPRova":                  true,
		"dono/nome.com_ponto":              true,
		"dono/" + strings.Repeat("r", 100): true,
		"dono/" + strings.Repeat("r", 101): false,
		"aPRova-TESTES":                    false,
		"Azulpasta/":                       false,
		"/aPRova":                          false,
		"dono/.":                           false,
		"dono/..":                          false,
		"dono/repo/extra":                  false,
		"-dono/repo":                       false,
	}

	for repositorio, esperado := range casos {
		if obtido := RepositorioValido(repositorio); obtido != esperado {
			t.Errorf("RepositorioValido(%q) = %v, esperava %v", repositorio, obtido, esperado)
		}
	}
}

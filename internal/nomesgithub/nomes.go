// Package nomesgithub reconhece os nomes que o GitHub aceita para logins e
// repositórios.
package nomesgithub

import (
	"regexp"
	"slices"
	"strings"
)

var (
	formatoDoLogin       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}$`)
	formatoDoRepositorio = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)
	nomesReservados      = []string{".", ".."}
)

// LoginValido informa se o texto tem o formato de um login do GitHub.
func LoginValido(login string) bool {
	return formatoDoLogin.MatchString(login)
}

// RepositorioValido informa se o texto tem o formato dono/nome de um
// repositório do GitHub.
func RepositorioValido(repositorio string) bool {
	dono, nome, separado := strings.Cut(repositorio, "/")
	return separado &&
		LoginValido(dono) &&
		formatoDoRepositorio.MatchString(nome) &&
		!slices.Contains(nomesReservados, nome)
}

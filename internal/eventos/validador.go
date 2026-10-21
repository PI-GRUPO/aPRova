package eventos

import (
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

var (
	formatoDoLogin       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,38}$`)
	formatoDoRepositorio = regexp.MustCompile(`^[A-Za-z0-9._-]{1,100}$`)
	formatoHexMinusculo  = regexp.MustCompile(`^[0-9a-f]+$`)
	nomesReservados      = []string{".", ".."}
)

var validador = novoValidador()

func novoValidador() *validator.Validate {
	validacao := validator.New(validator.WithRequiredStructEnabled())
	regras := map[string]validator.Func{
		"repositorio_github": repositorioValido,
		"login_github":       loginValido,
		"hex_minusculo":      hexMinusculo,
		"utc":                emUTC,
	}
	for nome, regra := range regras {
		if err := validacao.RegisterValidation(nome, regra); err != nil {
			panic(err)
		}
	}
	return validacao
}

func repositorioValido(campo validator.FieldLevel) bool {
	dono, nome, separado := strings.Cut(campo.Field().String(), "/")
	return separado &&
		formatoDoLogin.MatchString(dono) &&
		formatoDoRepositorio.MatchString(nome) &&
		!slices.Contains(nomesReservados, nome)
}

func loginValido(campo validator.FieldLevel) bool {
	return formatoDoLogin.MatchString(campo.Field().String())
}

func hexMinusculo(campo validator.FieldLevel) bool {
	return formatoHexMinusculo.MatchString(campo.Field().String())
}

func emUTC(campo validator.FieldLevel) bool {
	momento, ehMomento := campo.Field().Interface().(time.Time)
	if !ehMomento {
		return false
	}
	_, deslocamento := momento.Zone()
	return deslocamento == 0
}

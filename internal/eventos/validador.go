package eventos

import (
	"regexp"
	"time"

	"github.com/Azulpasta/aPRova/internal/nomesgithub"
	"github.com/go-playground/validator/v10"
)

var formatoHexMinusculo = regexp.MustCompile(`^[0-9a-f]+$`)

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
	return nomesgithub.RepositorioValido(campo.Field().String())
}

func loginValido(campo validator.FieldLevel) bool {
	return nomesgithub.LoginValido(campo.Field().String())
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

package acaoremota

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Azulpasta/aPRova/internal/nomesgithub"
	"github.com/go-playground/validator/v10"
)

// ErrPedidoInvalido indica corpo que não descreve uma ação remota válida.
var ErrPedidoInvalido = errors.New("acaoremota: pedido inválido")

var validador = novoValidador()

func novoValidador() *validator.Validate {
	validacao := validator.New(validator.WithRequiredStructEnabled())
	if err := validacao.RegisterValidation("repositorio_github", repositorioValido); err != nil {
		panic(err)
	}
	return validacao
}

func interpretarPedido(corpo []byte) (Pedido, error) {
	var pedido Pedido

	decodificador := json.NewDecoder(bytes.NewReader(corpo))
	decodificador.DisallowUnknownFields()

	if err := decodificador.Decode(&pedido); err != nil {
		return pedido, fmt.Errorf("%w: %w", ErrPedidoInvalido, err)
	}
	if err := decodificador.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return pedido, fmt.Errorf("%w: conteúdo depois do objeto JSON", ErrPedidoInvalido)
	}

	pedido.Justificativa = strings.TrimSpace(pedido.Justificativa)

	if err := validador.Struct(pedido); err != nil {
		return pedido, fmt.Errorf("%w: %w", ErrPedidoInvalido, err)
	}

	return pedido, nil
}

func repositorioValido(campo validator.FieldLevel) bool {
	return nomesgithub.RepositorioValido(campo.Field().String())
}

func (p Pedido) donoERepositorio() (string, string) {
	dono, repositorio, _ := strings.Cut(p.Repositorio, "/")
	return dono, repositorio
}

// Package rastro leva atributos de log pelo context.Context, para que toda
// linha registrada durante um trabalho, em qualquer pacote, saia com a
// identificação dele, como o task_run_id.
package rastro

import (
	"context"
	"log/slog"
	"slices"
	"time"
)

type chaveDosAtributos struct{}

// Com devolve um contexto filho de ctx que acrescenta os atributos, em pares
// chave e valor como em slog, a toda linha registrada com ele.
func Com(ctx context.Context, atributos ...any) context.Context {
	registro := slog.NewRecord(time.Time{}, slog.LevelInfo, "", 0)
	registro.Add(atributos...)

	acumulados := slices.Clone(atributosDe(ctx))
	registro.Attrs(func(atributo slog.Attr) bool {
		acumulados = append(acumulados, atributo)
		return true
	})
	return context.WithValue(ctx, chaveDosAtributos{}, acumulados)
}

func atributosDe(ctx context.Context) []slog.Attr {
	atributos, _ := ctx.Value(chaveDosAtributos{}).([]slog.Attr)
	return atributos
}

type handler struct {
	base slog.Handler
}

// NovoHandler envolve base para acrescentar, a cada linha registrada com
// contexto, os atributos guardados nele por Com.
func NovoHandler(base slog.Handler) slog.Handler {
	return handler{base: base}
}

func (h handler) Enabled(ctx context.Context, nivel slog.Level) bool {
	return h.base.Enabled(ctx, nivel)
}

func (h handler) Handle(ctx context.Context, registro slog.Record) error {
	if atributos := atributosDe(ctx); len(atributos) > 0 {
		registro = registro.Clone()
		registro.AddAttrs(atributos...)
	}
	return h.base.Handle(ctx, registro)
}

func (h handler) WithAttrs(atributos []slog.Attr) slog.Handler {
	return handler{base: h.base.WithAttrs(atributos)}
}

func (h handler) WithGroup(nome string) slog.Handler {
	return handler{base: h.base.WithGroup(nome)}
}

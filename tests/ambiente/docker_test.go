package ambiente

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

type relatorFalso struct {
	reprovou bool
	pulou    bool
	mensagem string
}

func (r *relatorFalso) Helper() {}

func (r *relatorFalso) Fatalf(formato string, argumentos ...any) {
	r.reprovou = true
	r.mensagem = fmt.Sprintf(formato, argumentos...)
}

func (r *relatorFalso) Skipf(formato string, argumentos ...any) {
	r.pulou = true
	r.mensagem = fmt.Sprintf(formato, argumentos...)
}

var errDockerAusente = errors.New("dial unix /var/run/docker.sock: connect: no such file or directory")

func TestFaltaDeDockerReprovaQuandoCIEstaLigado(t *testing.T) {
	for _, valor := range []string{"true", "1", "TRUE"} {
		t.Run(valor, func(t *testing.T) {
			t.Setenv("CI", valor)
			relator := &relatorFalso{}

			TratarFaltaDeDocker(relator, errDockerAusente)

			if !relator.reprovou || relator.pulou {
				t.Fatalf("com CI=%s a falta de Docker reprova: reprovou=%v pulou=%v", valor, relator.reprovou, relator.pulou)
			}
			if !strings.Contains(relator.mensagem, errDockerAusente.Error()) {
				t.Errorf("a mensagem precisa trazer a causa: %q", relator.mensagem)
			}
		})
	}
}

func TestFaltaDeDockerPulaForaDoCIExplicandoComoLigar(t *testing.T) {
	for _, valor := range []string{"", "false", "0"} {
		t.Run("CI="+valor, func(t *testing.T) {
			t.Setenv("CI", valor)
			relator := &relatorFalso{}

			TratarFaltaDeDocker(relator, errDockerAusente)

			if !relator.pulou || relator.reprovou {
				t.Fatalf("fora do CI a falta de Docker pula: reprovou=%v pulou=%v", relator.reprovou, relator.pulou)
			}
			for _, trecho := range []string{errDockerAusente.Error(), "DOCKER_HOST", "TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE", "CI=true"} {
				if !strings.Contains(relator.mensagem, trecho) {
					t.Errorf("a mensagem precisa citar %q: %q", trecho, relator.mensagem)
				}
			}
		})
	}
}

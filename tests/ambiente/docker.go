package ambiente

import (
	"os"
	"strconv"
)

const comoLigarODocker = "ligue o Docker (colima start ou Docker Desktop); com o Colima, o testcontainers ainda precisa de " +
	"DOCKER_HOST=unix://$HOME/.colima/default/docker.sock e TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock. " +
	"Com CI=true a falta de Docker reprova em vez de pular"

// Relator é a parte de testing.TB usada para reprovar ou pular um teste.
type Relator interface {
	Helper()
	Fatalf(formato string, argumentos ...any)
	Skipf(formato string, argumentos ...any)
}

// TratarFaltaDeDocker reprova o teste quando CI está ligado e o pula nos
// demais casos, citando a causa e como ligar o Docker.
func TratarFaltaDeDocker(t Relator, causa error) {
	t.Helper()
	if ciLigado() {
		t.Fatalf("CI=true exige Docker e ele não foi encontrado: %v", causa)
		return
	}
	t.Skipf("Docker não encontrado, teste pulado (%v): %s", causa, comoLigarODocker)
}

func ciLigado() bool {
	ligado, err := strconv.ParseBool(os.Getenv("CI"))
	return err == nil && ligado
}

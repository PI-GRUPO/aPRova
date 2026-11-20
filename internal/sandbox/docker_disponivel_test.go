package sandbox

import "testing"

func TestConferirDockerAceitaBinarioQueRespondeERecusaOQueFalha(t *testing.T) {
	if err := ConferirDocker("true")(t.Context()); err != nil {
		t.Errorf("um docker que responde está disponível: %v", err)
	}
	if err := ConferirDocker("false")(t.Context()); err == nil {
		t.Error("um docker que falha não está disponível")
	}
	if err := ConferirDocker("binario-que-nao-existe-aprova")(t.Context()); err == nil {
		t.Error("sem o binário do docker, o worker não está pronto")
	}
}

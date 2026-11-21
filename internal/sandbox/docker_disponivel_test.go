package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func dockerFalsoNoPath(t *testing.T, codigoDeSaida string) {
	t.Helper()

	pasta := t.TempDir()
	script := "#!/bin/sh\nexit " + codigoDeSaida + "\n"
	if err := os.WriteFile(filepath.Join(pasta, BinarioDockerPadrao), []byte(script), 0o700); err != nil {
		t.Fatalf("criar o docker falso: %v", err)
	}
	t.Setenv("PATH", pasta)
}

func TestConferirDockerAceitaDockerQueResponde(t *testing.T) {
	dockerFalsoNoPath(t, "0")

	if err := ConferirDocker()(t.Context()); err != nil {
		t.Errorf("um docker que responde está disponível: %v", err)
	}
}

func TestConferirDockerRecusaDockerQueFalha(t *testing.T) {
	dockerFalsoNoPath(t, "1")

	if err := ConferirDocker()(t.Context()); err == nil {
		t.Error("um docker que falha não está disponível")
	}
}

func TestConferirDockerRecusaSemDockerNoPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	if err := ConferirDocker()(t.Context()); err == nil {
		t.Error("sem o binário do docker, o worker não está pronto")
	}
}

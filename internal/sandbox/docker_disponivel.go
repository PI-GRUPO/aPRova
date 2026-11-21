package sandbox

import (
	"context"
	"fmt"
	"os/exec"
)

// ConferirDocker devolve a verificação de prontidão do executor local: o
// docker do PATH responde e alcança o daemon.
func ConferirDocker() func(context.Context) error {
	return func(ctx context.Context) error {
		if err := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}").Run(); err != nil {
			return fmt.Errorf("docker do sandbox indisponível: %w", err)
		}
		return nil
	}
}

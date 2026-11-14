package integracao

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

const prefixoDosVolumesDeRepositorios = "aprova-repos-"

type imagemDeIntegracao struct {
	linguagem  string
	construcao sync.Once
	erro       error
	saida      []byte
}

var (
	imagemGo     = &imagemDeIntegracao{linguagem: "go"}
	imagemPython = &imagemDeIntegracao{linguagem: "python"}
	imagemJava   = &imagemDeIntegracao{linguagem: "java"}
)

func (i *imagemDeIntegracao) nome() string {
	return "aprova-sandbox-" + i.linguagem + ":teste-de-integracao"
}

func (i *imagemDeIntegracao) exigir(t *testing.T) {
	t.Helper()
	exigirDocker(t)

	i.construcao.Do(func() {
		ctx, cancelar := context.WithTimeout(context.Background(), tempoLimiteDoBuild)
		defer cancelar()

		raiz, err := filepath.Abs(filepath.Join("..", ".."))
		if err != nil {
			i.erro = err
			return
		}
		construcao := exec.CommandContext(ctx, "docker", "build", "-q",
			"-f", filepath.Join(raiz, "imagens", i.linguagem, "Dockerfile"), "-t", i.nome(), raiz)
		i.saida, i.erro = construcao.CombinedOutput()
	})

	if i.erro != nil {
		t.Fatalf("construir a imagem do sandbox %s: %v\n%s", i.linguagem, i.erro, i.saida)
	}
}

func docker(t *testing.T, argumentos ...string) string {
	t.Helper()

	ctx, cancelar := context.WithTimeout(t.Context(), tempoLimiteDoPreparo)
	defer cancelar()

	var saida, erros bytes.Buffer
	comando := exec.CommandContext(ctx, "docker", argumentos...)
	comando.Stdout, comando.Stderr = &saida, &erros
	if err := comando.Run(); err != nil {
		t.Fatalf("docker %v: %v\n%s", argumentos, err, erros.String())
	}
	return saida.String()
}

func publicarRepositorios(t *testing.T, imagem *imagemDeIntegracao, casos ...string) (volume string, shas map[string]string) {
	t.Helper()

	sufixo := make([]byte, 6)
	_, _ = rand.Read(sufixo)
	volume = prefixoDosVolumesDeRepositorios + hex.EncodeToString(sufixo)

	docker(t, "volume", "create", volume)
	t.Cleanup(func() {
		ctx, cancelar := context.WithTimeout(context.Background(), tempoLimiteDoPreparo)
		defer cancelar()

		_ = exec.CommandContext(ctx, "docker", "volume", "rm", "-f", volume).Run()
	})
	docker(t, "run", "--rm", "-v", volume+":/r", "--entrypoint", "chown", imagem.nome(), usuarioDoCodigoDoPR, "/r")

	origem, err := filepath.Abs(filepath.Join("testdata", imagem.linguagem))
	if err != nil {
		t.Fatalf("caminho dos casos: %v", err)
	}

	const roteiro = `set -eu
export HOME=/tmp GIT_AUTHOR_NAME=aprova GIT_AUTHOR_EMAIL=aprova@exemplo.local GIT_COMMITTER_NAME=aprova GIT_COMMITTER_EMAIL=aprova@exemplo.local
for caso in "$@"; do
  cp -r "/origem/$caso" "/repositorios/$caso"
  git -C "/repositorios/$caso" init -q
  git -C "/repositorios/$caso" add .
  git -C "/repositorios/$caso" commit -q -m "caso $caso"
  echo "$caso $(git -C "/repositorios/$caso" rev-parse HEAD)"
done`

	argumentos := []string{"run", "--rm", "-u", usuarioDoCodigoDoPR,
		"-v", origem + ":/origem:ro", "-v", volume + ":" + pontoDeMontagem,
		"--entrypoint", "sh", imagem.nome(), "-c", roteiro, "sh"}
	saida := docker(t, append(argumentos, casos...)...)

	shas = map[string]string{}
	for _, linha := range strings.Split(strings.TrimSpace(saida), "\n") {
		if campos := strings.Fields(linha); len(campos) == 2 {
			shas[campos[0]] = campos[1]
		}
	}
	for _, caso := range casos {
		if shas[caso] == "" {
			t.Fatalf("o caso %s não gerou commit: %q", caso, saida)
		}
	}
	return volume, shas
}

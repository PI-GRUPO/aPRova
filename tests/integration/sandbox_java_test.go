package integracao

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Azulpasta/aPRova/internal/sandbox"
)

const (
	imagemDoSandboxJava   = "aprova-sandbox-java:teste-de-integracao"
	prefixoDosVolumesJava = "aprova-java-repos-"
)

var (
	construcaoDaImagemJava sync.Once
	erroDaConstrucaoJava   error
	saidaDaConstrucaoJava  []byte
)

func exigirImagemDoSandboxJava(t *testing.T) {
	t.Helper()
	exigirDocker(t)

	construcaoDaImagemJava.Do(func() {
		ctx, cancelar := context.WithTimeout(context.Background(), tempoLimiteDoBuild)
		defer cancelar()

		raiz, err := filepath.Abs(filepath.Join("..", ".."))
		if err != nil {
			erroDaConstrucaoJava = err
			return
		}

		construcao := exec.CommandContext(ctx, "docker", "build", "-q",
			"-f", filepath.Join(raiz, "imagens", "java", "Dockerfile"),
			"-t", imagemDoSandboxJava, raiz)

		saidaDaConstrucaoJava, erroDaConstrucaoJava = construcao.CombinedOutput()
	})

	if erroDaConstrucaoJava != nil {
		t.Fatalf("construir a imagem do sandbox Java: %v\n%s",
			erroDaConstrucaoJava, saidaDaConstrucaoJava)
	}
}

func publicarRepositoriosJava(t *testing.T, casos ...string) (volume string, shas map[string]string) {
	t.Helper()

	sufixo := make([]byte, 6)
	if _, err := rand.Read(sufixo); err != nil {
		t.Fatalf("gerar identificador do volume: %v", err)
	}

	volume = prefixoDosVolumesJava + hex.EncodeToString(sufixo)

	docker(t, "volume", "create", volume)

	t.Cleanup(func() {
		ctx, cancelar := context.WithTimeout(context.Background(), tempoLimiteDoPreparo)
		defer cancelar()

		_ = exec.CommandContext(ctx, "docker", "volume", "rm", "-f", volume).Run()
	})

	docker(t, "run", "--rm", "-v", volume+":/r",
		"--entrypoint", "chown", imagemDoSandboxJava,
		usuarioDoCodigoDoPR, "/r")

	origem, err := filepath.Abs(filepath.Join("testdata", "java"))
	if err != nil {
		t.Fatalf("caminho dos casos Java: %v", err)
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

	argumentos := []string{
		"run", "--rm", "-u", usuarioDoCodigoDoPR,
		"-v", origem + ":/origem:ro",
		"-v", volume + ":" + pontoDeMontagem,
		"--entrypoint", "sh", imagemDoSandboxJava,
		"-c", roteiro, "sh",
	}

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

func executarCasoJava(t *testing.T, caso string, prazo time.Duration) sandbox.ResultadoSandbox {
	t.Helper()
	exigirImagemDoSandboxJava(t)

	volume, shas := publicarRepositoriosJava(t, caso)

	executor := sandbox.NovoExecutorLocal(sandbox.OpcoesExecutorLocal{
		ImagemPorLinguagem: map[sandbox.Linguagem]string{
			sandbox.LinguagemJava: imagemDoSandboxJava,
		},
		Montagens: []string{
			volume + ":" + pontoDeMontagem + ":ro",
		},
	})

	ctx, cancelar := context.WithTimeout(t.Context(), prazo)
	defer cancelar()

	resultado, err := executor.Executar(ctx, sandbox.PedidoExecucao{
		URLRepositorio: "file://" + pontoDeMontagem + "/" + caso,
		SHA:            shas[caso],
		Linguagem:      sandbox.LinguagemJava,
		TaskRunID:      "integracao-java-" + caso,
	})
	if err != nil {
		t.Fatalf("executar caso Java %s: %v", caso, err)
	}

	t.Logf("caso %s: tipo=%s compilou=%v passaram=%d falharam=%d duracao=%dms\n%s",
		caso, resultado.TipoFalha, resultado.Compilou,
		resultado.TestesPassaram, resultado.TestesFalharam,
		resultado.DuracaoMS, resultado.SaidaBruta)

	return resultado
}

func TestSandboxJavaProjetoOk(t *testing.T) {
	resultado := executarCasoJava(t, "ok", prazoDeUmCasoComum)

	if resultado.TipoFalha != sandbox.FalhaNenhuma ||
		!resultado.Compilou ||
		resultado.TestesPassaram != 3 ||
		resultado.TestesFalharam != 0 {
		t.Errorf("projeto Java ok: esperava 3 testes passando sem falha, obtive %+v", resultado)
	}
}

func TestSandboxJavaTesteFalho(t *testing.T) {
	resultado := executarCasoJava(t, "teste_falho", prazoDeUmCasoComum)

	if resultado.TipoFalha != sandbox.FalhaTeste ||
		!resultado.Compilou ||
		resultado.TestesPassaram != 1 ||
		resultado.TestesFalharam != 1 {
		t.Errorf("esperava falha de teste com 1 aprovado e 1 reprovado, obtive %+v", resultado)
	}
}

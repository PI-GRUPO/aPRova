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
	"time"

	"github.com/Azulpasta/aPRova/internal/sandbox"
)

const (
	imagemDoSandboxPython   = "aprova-sandbox-python:teste-de-integracao"
	usuarioDoCodigoDoPR     = "10001:10001"
	tempoLimiteDoPreparo    = time.Minute
	prefixoDosVolumesPython = "aprova-repos-"
)

var (
	construcaoDaImagemPython sync.Once
	erroDaConstrucaoPython   error
	saidaDaConstrucaoPython  []byte
)

func exigirImagemDoSandboxPython(t *testing.T) {
	t.Helper()
	exigirDocker(t)

	construcaoDaImagemPython.Do(func() {
		ctx, cancelar := context.WithTimeout(context.Background(), tempoLimiteDoBuild)
		defer cancelar()

		raiz, err := filepath.Abs(filepath.Join("..", ".."))
		if err != nil {
			erroDaConstrucaoPython = err
			return
		}
		construcao := exec.CommandContext(ctx, "docker", "build", "-q",
			"-f", filepath.Join(raiz, "imagens", "python", "Dockerfile"), "-t", imagemDoSandboxPython, raiz)
		saidaDaConstrucaoPython, erroDaConstrucaoPython = construcao.CombinedOutput()
	})

	if erroDaConstrucaoPython != nil {
		t.Fatalf("construir a imagem do sandbox Python: %v\n%s", erroDaConstrucaoPython, saidaDaConstrucaoPython)
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

// publicarRepositoriosPython cria um volume do Docker com um repositório git
// por caso de testdata/python, todos pertencentes ao uid 10001. O git recusa
// ler (ownership) um repositório de outro usuário, e é o uid 10001 quem
// clona dentro do sandbox. Por isso os repositórios nascem dentro de um
// container, e não no disco do host.
func publicarRepositoriosPython(t *testing.T, casos ...string) (volume string, shas map[string]string) {
	t.Helper()

	sufixo := make([]byte, 6)
	_, _ = rand.Read(sufixo)
	volume = prefixoDosVolumesPython + hex.EncodeToString(sufixo)

	docker(t, "volume", "create", volume)
	t.Cleanup(func() {
		ctx, cancelar := context.WithTimeout(context.Background(), tempoLimiteDoPreparo)
		defer cancelar()

		_ = exec.CommandContext(ctx, "docker", "volume", "rm", "-f", volume).Run()
	})
	docker(t, "run", "--rm", "-v", volume+":/r", "--entrypoint", "chown", imagemDoSandboxPython, usuarioDoCodigoDoPR, "/r")

	origem, err := filepath.Abs(filepath.Join("testdata", "python"))
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
		"--entrypoint", "sh", imagemDoSandboxPython, "-c", roteiro, "sh"}
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

func executarCasoPython(t *testing.T, caso string, prazo time.Duration) sandbox.ResultadoSandbox {
	t.Helper()
	exigirImagemDoSandboxPython(t)

	volume, shas := publicarRepositoriosPython(t, caso)

	executor := sandbox.NovoExecutorLocal(sandbox.OpcoesExecutorLocal{
		ImagemPorLinguagem: map[sandbox.Linguagem]string{sandbox.LinguagemPython: imagemDoSandboxPython},
		Montagens:          []string{volume + ":" + pontoDeMontagem + ":ro"},
	})

	ctx, cancelar := context.WithTimeout(t.Context(), prazo)
	defer cancelar()

	resultado, err := executor.Executar(ctx, sandbox.PedidoExecucao{
		URLRepositorio: "file://" + pontoDeMontagem + "/" + caso,
		SHA:            shas[caso],
		Linguagem:      sandbox.LinguagemPython,
		TaskRunID:      "integracao-python-" + caso,
	})
	if err != nil {
		t.Fatalf("executar caso %s: %v", caso, err)
	}

	t.Logf("caso %s: tipo=%s compilou=%v passaram=%d falharam=%d duracao=%dms\n%s",
		caso, resultado.TipoFalha, resultado.Compilou, resultado.TestesPassaram, resultado.TestesFalharam,
		resultado.DuracaoMS, resultado.SaidaBruta)

	return resultado
}

func TestSandboxPythonProjetoOk(t *testing.T) {
	resultado := executarCasoPython(t, "ok", prazoDeUmCasoComum)

	if resultado.TipoFalha != sandbox.FalhaNenhuma || !resultado.Compilou ||
		resultado.TestesPassaram != 2 || resultado.TestesFalharam != 0 {
		t.Errorf("projeto ok: esperava 2 testes passando sem falha, obtive %+v", resultado)
	}
}

func TestSandboxPythonTesteFalho(t *testing.T) {
	resultado := executarCasoPython(t, "teste_falho", prazoDeUmCasoComum)

	if resultado.TipoFalha != sandbox.FalhaTeste || !resultado.Compilou ||
		resultado.TestesPassaram != 2 || resultado.TestesFalharam != 1 {
		t.Errorf("esperava falha de teste com 2 passando e 1 falhando, obtive %+v", resultado)
	}
	if !strings.Contains(resultado.SaidaBruta, "frete com taxa: esperava 16, obtive 15") {
		t.Error("a mensagem do teste falho precisa estar na saída")
	}
}

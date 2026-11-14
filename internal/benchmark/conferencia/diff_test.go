package conferencia

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

const envDeExemplo = `# comentário
DATABASE_URL=postgres://aprova:senha-do-banco-local@localhost:5433/aprova?sslmode=disable
GITHUB_WEBHOOK_SECRET="segredo-entre-aspas-123"
GITHUB_PRIVATE_KEY_BASE64=` + "LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCmNoYXZlLXByaXZhZGEtZmljdGljaWEKLS0tLS1FTkQgUFJJVkFURSBLRVktLS0tLQo=" + `
SANDBOX_EXECUTOR=local
WORKER_CONCORRENCIA=4
CALIBRACAO_LIGADA=true
VAZIA=
export GEMINI_API_KEY='chave-gemini-ficticia-xyz'
`

func segredosDeExemplo() Segredos {
	return SegredosDoEnv([]byte(envDeExemplo))
}

func TestDiffSoComArquivosDaTarefaESemSegredoPassa(t *testing.T) {
	diff := []byte("+++ b/frete.go\n+func Calcular() float64 { return 10 }\n+// local, true, 4\n")

	if err := Conferir([]string{"frete.go", "pacote/frete_test.go"}, diff, segredosDeExemplo()); err != nil {
		t.Errorf("diff limpo foi recusado: %v", err)
	}
}

func TestCaminhoForaDaTarefaERecusado(t *testing.T) {
	for _, caminho := range []string{"../fora.go", "/Users/alguem/.ssh/id_ed25519", "pacote/../../fora.go", "", "."} {
		err := Conferir([]string{"frete.go", caminho}, nil, segredosDeExemplo())

		if !errors.Is(err, ErrForaDaTarefa) {
			t.Errorf("%q: esperava ErrForaDaTarefa, obtive %v", caminho, err)
		}
	}
}

func TestArquivoDeCacheERecusado(t *testing.T) {
	caches := []string{
		".gocache/ab/cdef-d", ".pytest_cache/v/cache/nodeids", "pacote/__pycache__/frete.cpython-314.pyc",
		".m2/repository/org/x.jar", "target/classes/Frete.class", "modulo/target/surefire-reports/TEST-x.xml",
		".claude/settings.local.json",
	}
	for _, caminho := range caches {
		if err := Conferir([]string{caminho}, nil, segredosDeExemplo()); !errors.Is(err, ErrArquivoDeCache) {
			t.Errorf("%q: esperava ErrArquivoDeCache, obtive %v", caminho, err)
		}
	}
}

func TestValorDoEnvNoDiffERecusadoSemMostrarOValor(t *testing.T) {
	chavePem, err := base64.StdEncoding.DecodeString("LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCmNoYXZlLXByaXZhZGEtZmljdGljaWEKLS0tLS1FTkQgUFJJVkFURSBLRVktLS0tLQo=")
	if err != nil {
		t.Fatal(err)
	}
	casos := map[string]struct {
		valor    string
		variavel string
	}{
		"url do banco":          {"postgres://aprova:senha-do-banco-local@localhost:5433/aprova?sslmode=disable", "DATABASE_URL"},
		"valor entre aspas":     {"segredo-entre-aspas-123", "GITHUB_WEBHOOK_SECRET"},
		"chave em base64":       {"LS0tLS1CRUdJTiBQUklWQVRFIEtFWS0tLS0tCmNoYXZlLXByaXZhZGEtZmljdGljaWEKLS0tLS1FTkQgUFJJVkFURSBLRVktLS0tLQo=", "GITHUB_PRIVATE_KEY_BASE64"},
		"chave já decodificada": {string(chavePem), "GITHUB_PRIVATE_KEY_BASE64"},
		"com export e aspas":    {"chave-gemini-ficticia-xyz", "GEMINI_API_KEY"},
	}
	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			diff := []byte("+++ b/config.go\n+const valor = `" + caso.valor + "`\n")

			err := Conferir([]string{"config.go"}, diff, segredosDeExemplo())

			if !errors.Is(err, ErrSegredoNoDiff) || !strings.Contains(err.Error(), caso.variavel) {
				t.Fatalf("esperava ErrSegredoNoDiff nomeando %s, obtive %v", caso.variavel, err)
			}
			if strings.Contains(err.Error(), caso.valor) || strings.Contains(err.Error(), "senha") || strings.Contains(err.Error(), "ficticia") {
				t.Errorf("a mensagem não pode conter o valor: %q", err.Error())
			}
		})
	}
}

func TestValoresCurtosOuVaziosNaoViramSegredo(t *testing.T) {
	segredos := segredosDeExemplo()

	if nomes := segredos.Nomes(); strings.Contains(strings.Join(nomes, ","), "SANDBOX_EXECUTOR") ||
		strings.Contains(strings.Join(nomes, ","), "WORKER_CONCORRENCIA") || strings.Contains(strings.Join(nomes, ","), "VAZIA") {
		t.Errorf("valores curtos ou vazios dariam falso positivo: %v", nomes)
	}
	if len(segredos.Nomes()) != 4 {
		t.Errorf("esperava 4 variáveis conferidas, obtive %v", segredos.Nomes())
	}
}

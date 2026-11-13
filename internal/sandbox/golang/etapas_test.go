package golang

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/sandbox"
	"github.com/PI-GRUPO/aPRova/internal/sandbox/processo"
)

const cacheDoVolume = "/aprova-preparo/cache"

func linguagemComRespostas(respostas map[string]processo.Execucao) (*Linguagem, *rodadorFalso) {
	rodador := &rodadorFalso{respostas: respostas, agora: time.Unix(0, 0)}
	return Nova(Opcoes{Rodador: rodador, DiretorioDeCache: "/tmp/aprova-cache", Agora: rodador.relogio}), rodador
}

func exigirAmbiente(t *testing.T, comando processo.Comando, esperadas ...string) {
	t.Helper()
	for _, esperada := range esperadas {
		if !slices.Contains(comando.Ambiente, esperada) {
			t.Errorf("%v sem %s: %v", comando.Argumentos, esperada, comando.Ambiente)
		}
	}
}

func TestPreparoSoBaixaOsModulosPeloProxyPublico(t *testing.T) {
	linguagem, rodador := linguagemComRespostas(map[string]processo.Execucao{})

	falha := linguagem.Preparar(t.Context(), repositorioComGoMod(t), cacheDoVolume)

	if falha != nil {
		t.Fatalf("preparo com download bem-sucedido: %+v", falha)
	}
	if rodados := rodador.subcomandosRodados(); !slices.Equal(rodados, []string{"mod download"}) {
		t.Fatalf("o preparo não compila, não testa e não gera código: %v", rodados)
	}
	download := rodador.rodados[0]
	exigirAmbiente(t, download,
		"GOPROXY=https://proxy.golang.org", "GOVCS=*:off", "GOTOOLCHAIN=local", "GOFLAGS=-mod=readonly",
		"GOMODCACHE="+cacheDoVolume+"/gomod", "CGO_ENABLED=0")
	for _, variavel := range download.Ambiente {
		if strings.Contains(variavel, "direct") {
			t.Errorf("sem fallback direto para o VCS: %s", variavel)
		}
	}
}

func TestPreparoSemGoModEhInfraestruturaSemRodarNada(t *testing.T) {
	linguagem, rodador := linguagemComRespostas(map[string]processo.Execucao{})

	falha := linguagem.Preparar(t.Context(), t.TempDir(), cacheDoVolume)

	if falha == nil || falha.TipoFalha != sandbox.FalhaInfraestrutura || len(rodador.rodados) != 0 {
		t.Errorf("sem go.mod: %+v, rodados %v", falha, rodador.subcomandosRodados())
	}
}

func TestPreparoClassificaAFalhaDoDownload(t *testing.T) {
	casos := []struct {
		nome     string
		execucao processo.Execucao
		tipo     sandbox.TipoFalha
		trecho   string
	}{
		{"checksum divergente do go.sum", processo.Execucao{CodigoSaida: 1, Saida: []byte(
			"verifying github.com/google/go-cmp@v0.6.0: checksum mismatch\n\tdownloaded: h1:x\n\tgo.sum:     h1:y\n\nSECURITY ERROR\n")},
			sandbox.FalhaCompilacao, "checksum divergente do go.sum"},
		{"versão inexistente", processo.Execucao{CodigoSaida: 1, Saida: []byte(
			"go: github.com/google/go-cmp@v9.9.9: reading https://proxy.golang.org/github.com/google/go-cmp/@v/v9.9.9.mod: 404 Not Found\n")},
			sandbox.FalhaCompilacao, "não existe no proxy público"},
		{"módulo removido", processo.Execucao{CodigoSaida: 1, Saida: []byte(
			"go: example.com/fantasma@v1.0.0: reading https://proxy.golang.org/example.com/fantasma/@v/v1.0.0.mod: 410 Gone\n")},
			sandbox.FalhaCompilacao, "não existe no proxy público"},
		{"revisão inexistente", processo.Execucao{CodigoSaida: 1, Saida: []byte(
			"go: github.com/x/y@v0.0.0-20240101000000-abcdefabcdef: invalid version: unknown revision abcdefabcdef\n")},
			sandbox.FalhaCompilacao, "não existe no proxy público"},
		{"sem DNS", processo.Execucao{CodigoSaida: 1, Saida: []byte(
			"go: github.com/google/go-cmp@v0.6.0: Get \"https://proxy.golang.org/...\": dial tcp: lookup proxy.golang.org: no such host\n")},
			sandbox.FalhaInfraestrutura, "rede"},
		{"timeout de rede", processo.Execucao{CodigoSaida: 1, Saida: []byte(
			"go: github.com/google/go-cmp@v0.6.0: Get \"https://proxy.golang.org/...\": net/http: TLS handshake timeout\n")},
			sandbox.FalhaInfraestrutura, "rede"},
		{"proxy fora do ar", processo.Execucao{CodigoSaida: 1, Saida: []byte(
			"go: github.com/google/go-cmp@v0.6.0: reading https://proxy.golang.org/...: 503 Service Unavailable\n")},
			sandbox.FalhaInfraestrutura, "rede"},
		{"go.mod exige Go mais novo", processo.Execucao{CodigoSaida: 1, Saida: []byte(
			"go: go.mod requires go >= 1.99 (running go 1.27.2; GOTOOLCHAIN=local)\n")},
			sandbox.FalhaInfraestrutura, "go.mod exige uma versão do Go mais nova"},
		{"prazo do executor", processo.Execucao{PrazoEsgotado: true}, sandbox.FalhaInfraestrutura, "excedeu o prazo"},
		{"falha sem explicação", processo.Execucao{CodigoSaida: 2, Saida: []byte("go: algo estranho\n")}, sandbox.FalhaInfraestrutura, "go mod download falhou"},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			linguagem, _ := linguagemComRespostas(map[string]processo.Execucao{"mod download": caso.execucao})

			falha := linguagem.Preparar(t.Context(), repositorioComGoMod(t), cacheDoVolume)

			if falha == nil || falha.TipoFalha != caso.tipo || !strings.Contains(falha.SaidaBruta, caso.trecho) {
				t.Fatalf("esperava %s com %q: %+v", caso.tipo, caso.trecho, falha)
			}
			if falha.TipoFalha == sandbox.FalhaCompilacao && !strings.Contains(falha.SaidaBruta, string(caso.execucao.Saida)) {
				t.Errorf("a evidência traz a saída do go, com a dependência: %q", falha.SaidaBruta)
			}
		})
	}
}

func TestTesteRodaSemProxyComOsModulosDoPreparo(t *testing.T) {
	linguagem, rodador := linguagemComRespostas(map[string]processo.Execucao{
		"test -json": testeComGravacao(t, "so_passando.jsonl", 0),
	})

	resultado := linguagem.Testar(t.Context(), repositorioComGoMod(t), cacheDoVolume, time.Unix(0, 0).Add(10*time.Minute))

	if resultado.TipoFalha != sandbox.FalhaNenhuma || resultado.TestesPassaram == 0 {
		t.Errorf("testes passando: %+v", resultado)
	}
	if rodados := rodador.subcomandosRodados(); !slices.Equal(rodados, []string{"build ./...", "test -c", "test -json"}) {
		t.Fatalf("o teste não baixa nada: %v", rodados)
	}
	for _, comando := range rodador.rodados {
		exigirAmbiente(t, comando, "GOPROXY=off", "GOFLAGS=-mod=readonly", "GOTOOLCHAIN=local",
			"GOMODCACHE="+cacheDoVolume+"/gomod", "GOCACHE=/tmp/aprova-cache/build")
	}
}

func TestTesteComModuloQueFaltaNoPreparoEhCompilacao(t *testing.T) {
	linguagem, _ := linguagemComRespostas(map[string]processo.Execucao{
		"build ./...": {CodigoSaida: 1, Saida: []byte("frete.go:3:8: missing go.sum entry for module providing package github.com/x/y\n")},
	})

	resultado := linguagem.Testar(t.Context(), repositorioComGoMod(t), cacheDoVolume, time.Unix(0, 0).Add(10*time.Minute))

	if resultado.TipoFalha != sandbox.FalhaCompilacao || !strings.Contains(resultado.SaidaBruta, "missing go.sum entry") {
		t.Errorf("módulo ausente é do PR: %+v", resultado)
	}
}

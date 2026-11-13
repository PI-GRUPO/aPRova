package golang

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/sandbox"
	"github.com/PI-GRUPO/aPRova/internal/sandbox/processo"
)

const subdiretorioDosModulos = "gomod"

var (
	sinaisDeRede = []string{
		"dial tcp", "no such host", "i/o timeout", "TLS handshake timeout", "connection refused",
		"connection reset", "network is unreachable", "server misbehaving", "temporary failure",
		"502 Bad Gateway", "503 Service Unavailable", "504 Gateway Timeout", "unexpected EOF",
	}
	sinaisDeChecksum    = []string{"checksum mismatch", "SECURITY ERROR"}
	sinaisDeInexistente = []string{"404 Not Found", "410 Gone", "unknown revision", "invalid version", "no matching versions"}
)

// Preparar baixa para o cache os módulos do go.mod pelo proxy público, sem
// fallback direto para o VCS, sem compilar nada e sem go generate: nenhum
// código do PR roda. Devolve nil quando os módulos estão no cache.
func (l *Linguagem) Preparar(ctx context.Context, projeto, cache string) *sandbox.ResultadoSandbox {
	if !existeGoModNaRaiz(projeto) {
		falha := infraestrutura("repositório sem go.mod na raiz")
		return &falha
	}

	download := l.rodar(ctx, projeto, l.ambienteDoPreparo(cache), "mod", "download")
	if concluiu(download) {
		return nil
	}
	falha := classificarDownload(download)
	return &falha
}

// Testar compila e roda os testes sem proxy, só com os módulos que Preparar
// deixou no cache. prazoFinal é o instante em que o resultado já precisa ter
// sido entregue.
func (l *Linguagem) Testar(ctx context.Context, projeto, cache string, prazoFinal time.Time) sandbox.ResultadoSandbox {
	if !existeGoModNaRaiz(projeto) {
		return infraestrutura("repositório sem go.mod na raiz")
	}
	return l.compilarETestar(ctx, projeto, l.ambienteDoTeste(cache), prazoFinal)
}

func (l *Linguagem) ambienteDoPreparo(cache string) []string {
	return append(l.ambienteDasEtapas(cache), "GOPROXY=https://proxy.golang.org", "GOVCS=*:off")
}

func (l *Linguagem) ambienteDoTeste(cache string) []string {
	return append(l.ambienteDasEtapas(cache), "GOPROXY=off")
}

func (l *Linguagem) ambienteDasEtapas(cache string) []string {
	return []string{
		caminhoDosExecutaveis,
		"HOME=" + filepath.Join(l.diretorioDeCache, "home"),
		"GOPATH=" + filepath.Join(l.diretorioDeCache, "gopath"),
		"GOMODCACHE=" + filepath.Join(cache, subdiretorioDosModulos),
		"GOCACHE=" + filepath.Join(l.diretorioDeCache, "build"),
		"GOFLAGS=-mod=readonly",
		"GOTOOLCHAIN=local",
		"CGO_ENABLED=0",
	}
}

func classificarDownload(download processo.Execucao) sandbox.ResultadoSandbox {
	saida := string(download.Saida)
	switch {
	case download.Erro != nil || download.PrazoEsgotado:
		return infraestrutura(descreverInterrupcao("go mod download", download), download.Saida)
	case strings.Contains(saida, sinalDeGoMaisNovo):
		return infraestrutura(explicarFalhaDoDownload(download), download.Saida)
	case contemAlgum(saida, sinaisDeRede):
		return infraestrutura("go mod download falhou por rede: o proxy público não respondeu", download.Saida)
	case contemAlgum(saida, sinaisDeChecksum):
		return compilacaoNoPreparo("checksum divergente do go.sum: o módulo baixado não é o que o PR declarou", download.Saida)
	case contemAlgum(saida, sinaisDeInexistente):
		return compilacaoNoPreparo("dependência ou versão que não existe no proxy público", download.Saida)
	default:
		return infraestrutura(descreverInterrupcao("go mod download", download), download.Saida)
	}
}

func contemAlgum(saida string, sinais []string) bool {
	for _, sinal := range sinais {
		if strings.Contains(saida, sinal) {
			return true
		}
	}
	return false
}

func compilacaoNoPreparo(motivo string, saida []byte) sandbox.ResultadoSandbox {
	return sandbox.ResultadoSandbox{SaidaBruta: string(saida) + "aprova: " + motivo + "\n", TipoFalha: sandbox.FalhaCompilacao}
}

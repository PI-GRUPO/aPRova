package publicacao

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/platform"
)

func TestPermissaoNegadaPeloGitHubDevolveDiagnosticoClaro(t *testing.T) {
	github := novoGitHubFalso(t)
	github.falharEm(http.MethodPost, "/check-runs", http.StatusForbidden, 1)

	err := publicadorContra(t, github, novoRegistroFalso()).Publicar(t.Context(), publicacaoDeTeste(taskRunDeTeste(idDoTaskRunDeConteudo, shaDoPrimeiro), domain.RiscoAlto))

	if !errors.Is(err, platform.ErrSemPermissao) {
		t.Fatalf("403 é falta de permissão, não falha transitória: %v", err)
	}
	if errors.Is(err, platform.ErrGitHubIndisponivel) || errors.Is(err, platform.ErrLimiteDeRequisicoes) {
		t.Error("403 de permissão não pode ser tentado de novo")
	}
	if !strings.Contains(err.Error(), "checks: write") || !strings.Contains(err.Error(), "pull_requests: write") {
		t.Errorf("o diagnóstico diz o que conferir na instalação: %v", err)
	}
}

func TestErroDoServidorDoGitHubEhTransitorio(t *testing.T) {
	github := novoGitHubFalso(t)
	github.falharEm(http.MethodPost, "/comments", http.StatusBadGateway, 1)
	publicador := publicadorContra(t, github, novoRegistroFalso())
	publicacao := publicacaoDeTeste(taskRunDeTeste(idDoTaskRunDeConteudo, shaDoPrimeiro), domain.RiscoAlto)

	err := publicador.Publicar(t.Context(), publicacao)
	if !errors.Is(err, platform.ErrGitHubIndisponivel) {
		t.Fatalf("502 é transitório: %v", err)
	}

	if err := publicador.Publicar(t.Context(), publicacao); err != nil {
		t.Fatalf("a nova tentativa publica: %v", err)
	}
	if len(github.checkRuns()) != 1 || len(github.comentarios()) != 1 {
		t.Errorf("a nova tentativa não duplica o que a primeira já criou: %d check runs, %d comentários", len(github.checkRuns()), len(github.comentarios()))
	}
}

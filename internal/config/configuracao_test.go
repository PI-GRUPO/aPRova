package config

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"log/slog"
	"maps"
	"strings"
	"testing"

	"github.com/Azulpasta/aPRova/internal/rastro"
)

func ambienteLimpo(t *testing.T) {
	t.Helper()

	for _, variavel := range []string{
		"PORT", "LOG_LEVEL", "DATABASE_URL", "REDIS_URL",
		"GITHUB_APP_ID", "GITHUB_PRIVATE_KEY_BASE64", "GITHUB_WEBHOOK_SECRET",
		"RECEIPT_PRIVATE_KEY_BASE64", "RECEIPT_PUBLIC_KEY_BASE64",
		"ANTHROPIC_API_KEY", "GCP_PROJECT_ID", "GCP_REGION",
		"SANDBOX_EXECUTOR", "N8N_WEBHOOK_URL",
		"AZURE_SUBSCRIPTION_ID", "AZURE_TENANT_ID", "AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET", "AZURE_RESOURCE_GROUP", "AZURE_SANDBOX_JOB_NAME",
		"N8N_SEGREDO_ACOES", "SLACK_GITHUB_VINCULOS", "SANDBOX_IMAGEM_GO",
		"N8N_SEGREDO_EVENTOS", "APROVA_URL_PUBLICA", "WORKER_CONCORRENCIA", "WORKER_MAX_TENTATIVAS",
		"SANDBOX_DIRETORIOS", "GITHUB_APP_SLUG", "SANDBOX_REDE_LOCAL",
	} {
		t.Setenv(variavel, "")
	}
}

func ambienteMinimoValido(t *testing.T) {
	t.Helper()

	ambienteLimpo(t)
	t.Setenv("DATABASE_URL", "postgres://aprova:aprova@localhost:5432/aprova?sslmode=disable")
	t.Setenv("REDIS_URL", "redis://localhost:6379/0")
}

func capturarLogDebug(t *testing.T) *bytes.Buffer {
	t.Helper()

	var registro bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&registro, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(anterior) })

	return &registro
}

func TestCarregarFalhaQuandoVariavelObrigatoriaEstaAusente(t *testing.T) {
	ambienteMinimoValido(t)
	t.Setenv("DATABASE_URL", "")

	_, err := Carregar()

	if err == nil {
		t.Fatal("esperava erro com DATABASE_URL ausente, obtive nil")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("o erro deve nomear a variável, obtive: %v", err)
	}
	if !strings.Contains(err.Error(), "docker-compose") {
		t.Errorf("o erro deve dizer onde obter o valor, obtive: %v", err)
	}
}

func TestCarregarNaoExigeVariaveisAindaNaoUsadas(t *testing.T) {
	ambienteMinimoValido(t)

	if _, err := Carregar(); err != nil {
		t.Fatalf("GitHub, agente e recibo são opcionais nesta etapa: %v", err)
	}
}

func TestCarregarAplicaValoresPadrao(t *testing.T) {
	ambienteMinimoValido(t)

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}

	for _, caso := range []struct {
		nome     string
		obtido   string
		esperado string
	}{
		{"PORT", configuracao.Porta, portaPadrao},
		{"LOG_LEVEL", configuracao.NivelLog, nivelLogPadrao},
		{"GCP_REGION", configuracao.GCPRegiao, regiaoPadrao},
	} {
		if caso.obtido != caso.esperado {
			t.Errorf("%s ausente deveria cair no padrão %q, obtive %q", caso.nome, caso.esperado, caso.obtido)
		}
	}
}

func TestCarregarRegistraEmDebugCadaOpcionalAusente(t *testing.T) {
	registro := capturarLogDebug(t)
	ambienteMinimoValido(t)

	if _, err := Carregar(); err != nil {
		t.Fatalf("carregar: %v", err)
	}

	if !strings.Contains(registro.String(), "ANTHROPIC_API_KEY") {
		t.Errorf("esperava log debug citando a opcional ausente, obtive: %s", registro.String())
	}
}

func TestCarregarRepassaOExecutorDoSandboxSemPadraoNemValidacao(t *testing.T) {
	for _, valor := range []string{"", "azure", "local", "valor-que-o-sandbox-vai-recusar"} {
		t.Run("SANDBOX_EXECUTOR="+valor, func(t *testing.T) {
			ambienteMinimoValido(t)
			t.Setenv("SANDBOX_EXECUTOR", valor)

			configuracao, err := Carregar()
			if err != nil {
				t.Fatalf("validar o executor é papel do pacote sandbox, não do config: %v", err)
			}
			if configuracao.SandboxExecutor != valor {
				t.Errorf("esperava o valor cru %q, obtive %q", valor, configuracao.SandboxExecutor)
			}
		})
	}
}

func TestCarregarLeAsCredenciaisDaAzure(t *testing.T) {
	ambienteMinimoValido(t)
	esperadas := map[string]string{
		"AZURE_SUBSCRIPTION_ID":  "assinatura",
		"AZURE_TENANT_ID":        "tenant",
		"AZURE_CLIENT_ID":        "cliente",
		"AZURE_CLIENT_SECRET":    "segredo",
		"AZURE_RESOURCE_GROUP":   "grupo",
		"AZURE_SANDBOX_JOB_NAME": "job",
	}
	for nome, valor := range esperadas {
		t.Setenv(nome, valor)
	}

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}

	obtidas := map[string]string{
		"AZURE_SUBSCRIPTION_ID":  configuracao.AzureSubscriptionID,
		"AZURE_TENANT_ID":        configuracao.AzureTenantID,
		"AZURE_CLIENT_ID":        configuracao.AzureClientID,
		"AZURE_CLIENT_SECRET":    configuracao.AzureClientSecret,
		"AZURE_RESOURCE_GROUP":   configuracao.AzureResourceGroup,
		"AZURE_SANDBOX_JOB_NAME": configuracao.AzureNomeDoJobSandbox,
	}
	for nome, valor := range esperadas {
		if obtidas[nome] != valor {
			t.Errorf("%s deveria ser %q, obtive %q", nome, valor, obtidas[nome])
		}
	}
}

func TestCarregarLeAImagemDoSandboxGo(t *testing.T) {
	ambienteMinimoValido(t)
	t.Setenv("SANDBOX_IMAGEM_GO", "aprova.azurecr.io/sandbox-go:3f2c1e0")

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}

	if configuracao.SandboxImagemGo != "aprova.azurecr.io/sandbox-go:3f2c1e0" {
		t.Errorf("imagem lida errada: %q", configuracao.SandboxImagemGo)
	}
}

func TestCarregarLeOSegredoDasAcoesRemotas(t *testing.T) {
	ambienteMinimoValido(t)
	t.Setenv("N8N_SEGREDO_ACOES", "segredo-compartilhado")

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}

	if configuracao.N8NSegredoAcoes != "segredo-compartilhado" {
		t.Errorf("segredo lido errado: %q", configuracao.N8NSegredoAcoes)
	}
}

func TestCarregarInterpretaOsVinculosSlackGitHub(t *testing.T) {
	ambienteMinimoValido(t)
	t.Setenv("SLACK_GITHUB_VINCULOS", " U01ABC:alice , U02DEF:bob ")

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}

	esperados := map[string]string{"U01ABC": "alice", "U02DEF": "bob"}
	if !maps.Equal(configuracao.VinculosSlackGitHub, esperados) {
		t.Errorf("vínculos = %v, esperava %v", configuracao.VinculosSlackGitHub, esperados)
	}
}

func TestCarregarAceitaVinculosAusentes(t *testing.T) {
	ambienteMinimoValido(t)

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("sem vínculos o processo sobe, só recusa todo mundo: %v", err)
	}
	if len(configuracao.VinculosSlackGitHub) != 0 {
		t.Errorf("esperava nenhum vínculo, obtive %v", configuracao.VinculosSlackGitHub)
	}
}

func TestCarregarRejeitaVinculoMalformado(t *testing.T) {
	for _, valor := range []string{"U01ABC", "U01ABC:", ":alice", "U01ABC:alice,,U02:bob", "U01:alice:extra", "U01:alice,U01:mallory"} {
		t.Run(valor, func(t *testing.T) {
			ambienteMinimoValido(t)
			t.Setenv("SLACK_GITHUB_VINCULOS", valor)

			_, err := Carregar()

			if err == nil || !strings.Contains(err.Error(), "SLACK_GITHUB_VINCULOS") {
				t.Errorf("esperava erro nomeando SLACK_GITHUB_VINCULOS, obtive %v", err)
			}
		})
	}
}

func TestCarregarRejeitaChaveDeReciboQueNaoEBase64(t *testing.T) {
	ambienteMinimoValido(t)
	t.Setenv("RECEIPT_PRIVATE_KEY_BASE64", "isto não é base64!!!")

	_, err := Carregar()

	if err == nil {
		t.Fatal("esperava erro na carga, não no primeiro uso")
	}
	if !strings.Contains(err.Error(), "RECEIPT_PRIVATE_KEY_BASE64") {
		t.Errorf("o erro deve nomear a variável, obtive: %v", err)
	}
}

func TestCarregarRejeitaChaveDeReciboComTamanhoErrado(t *testing.T) {
	ambienteMinimoValido(t)
	t.Setenv("RECEIPT_PRIVATE_KEY_BASE64", base64.StdEncoding.EncodeToString([]byte("curta demais")))

	if _, err := Carregar(); err == nil {
		t.Fatal("esperava erro para chave Ed25519 de tamanho inválido")
	}
}

func TestCarregarDecodificaParDeChavesUsavelParaAssinar(t *testing.T) {
	publica, privada, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("gerar par: %v", err)
	}

	ambienteMinimoValido(t)
	t.Setenv("RECEIPT_PRIVATE_KEY_BASE64", base64.StdEncoding.EncodeToString(privada))
	t.Setenv("RECEIPT_PUBLIC_KEY_BASE64", base64.StdEncoding.EncodeToString(publica))

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}

	mensagem := []byte("recibo de exemplo")
	assinatura := ed25519.Sign(configuracao.ReciboChavePrivada, mensagem)

	if !ed25519.Verify(configuracao.ReciboChavePublica, mensagem, assinatura) {
		t.Error("a chave pública carregada deveria verificar o que a privada carregada assinou")
	}
}

func TestCarregarDecodificaChavePrivadaDoGitHub(t *testing.T) {
	const pem = "-----BEGIN RSA PRIVATE KEY-----\nconteudo\n-----END RSA PRIVATE KEY-----"

	ambienteMinimoValido(t)
	t.Setenv("GITHUB_PRIVATE_KEY_BASE64", base64.StdEncoding.EncodeToString([]byte(pem)))

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}

	if string(configuracao.GitHubChavePrivada) != pem {
		t.Errorf("esperava o PEM decodificado, obtive %q", configuracao.GitHubChavePrivada)
	}
}

func TestEnderecoHTTPEscutaEmTodasAsInterfaces(t *testing.T) {
	if endereco := (Configuracao{Porta: "9000"}).EnderecoHTTP(); endereco != ":9000" {
		t.Errorf("esperava \":9000\", obtive %q", endereco)
	}
}

func TestNivelSlogTraduzOsNiveisAceitos(t *testing.T) {
	for texto, esperado := range map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	} {
		if obtido := (Configuracao{NivelLog: texto}).NivelSlog(); obtido != esperado {
			t.Errorf("LOG_LEVEL=%s deveria virar %v, obtive %v", texto, esperado, obtido)
		}
	}
}

func TestCarregarRejeitaNivelDeLogDesconhecido(t *testing.T) {
	ambienteMinimoValido(t)
	t.Setenv("LOG_LEVEL", "verboso")

	_, err := Carregar()

	if err == nil {
		t.Fatal("esperava erro para LOG_LEVEL desconhecido, obtive nil")
	}
	if !strings.Contains(err.Error(), "LOG_LEVEL") {
		t.Errorf("o erro deve nomear a variável, obtive: %v", err)
	}
}

func TestConfigurarLogPadraoRespeitaONivelConfigurado(t *testing.T) {
	anterior := slog.Default()
	t.Cleanup(func() { slog.SetDefault(anterior) })

	ConfigurarLogPadrao(Configuracao{NivelLog: "warn"})

	if slog.Default().Enabled(t.Context(), slog.LevelInfo) {
		t.Error("nível warn não deveria habilitar info")
	}
	if !slog.Default().Enabled(t.Context(), slog.LevelError) {
		t.Error("nível warn deveria habilitar error")
	}
}

func TestCarregarUsaOsPadroesDoWorker(t *testing.T) {
	ambienteMinimoValido(t)

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}

	if configuracao.WorkerConcorrencia != 4 || configuracao.WorkerMaxTentativas != 3 {
		t.Errorf("padrões do worker: concorrência %d, tentativas %d", configuracao.WorkerConcorrencia, configuracao.WorkerMaxTentativas)
	}
	if configuracao.URLPublica != "https://aprova.example.com" {
		t.Errorf("sem APROVA_URL_PUBLICA vale o endereço provisório, obtive %q", configuracao.URLPublica)
	}
	if len(configuracao.SandboxDiretorios) != 0 || configuracao.GitHubAppSlug != "" || configuracao.N8NSegredoEventos != "" {
		t.Errorf("opcionais deveriam vir vazios: %+v", configuracao)
	}
}

func TestCarregarLeAsVariaveisDoWorker(t *testing.T) {
	ambienteMinimoValido(t)
	t.Setenv("WORKER_CONCORRENCIA", "8")
	t.Setenv("WORKER_MAX_TENTATIVAS", "5")
	t.Setenv("N8N_SEGREDO_EVENTOS", "segredo-dos-eventos")
	t.Setenv("APROVA_URL_PUBLICA", "https://api.aprova.test/")
	t.Setenv("SANDBOX_DIRETORIOS", " Azulpasta/aPRova-TESTES=go , PI-GRUPO/outro=servicos/api ")
	t.Setenv("GITHUB_APP_SLUG", "aprova-pi")

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}

	if configuracao.WorkerConcorrencia != 8 || configuracao.WorkerMaxTentativas != 5 {
		t.Errorf("worker: concorrência %d, tentativas %d", configuracao.WorkerConcorrencia, configuracao.WorkerMaxTentativas)
	}
	if configuracao.N8NSegredoEventos != "segredo-dos-eventos" || configuracao.GitHubAppSlug != "aprova-pi" {
		t.Errorf("segredo ou slug lidos errado: %q %q", configuracao.N8NSegredoEventos, configuracao.GitHubAppSlug)
	}
	if configuracao.URLPublica != "https://api.aprova.test" {
		t.Errorf("a URL pública perde a barra final, obtive %q", configuracao.URLPublica)
	}
	esperados := map[string]string{"Azulpasta/aPRova-TESTES": "go", "PI-GRUPO/outro": "servicos/api"}
	if !maps.Equal(configuracao.SandboxDiretorios, esperados) {
		t.Errorf("diretórios = %v, esperava %v", configuracao.SandboxDiretorios, esperados)
	}
}

func TestCarregarRecusaVariaveisDoWorkerInvalidas(t *testing.T) {
	casos := map[string]string{
		"WORKER_CONCORRENCIA=0":                              "WORKER_CONCORRENCIA",
		"WORKER_CONCORRENCIA=-2":                             "WORKER_CONCORRENCIA",
		"WORKER_CONCORRENCIA=quatro":                         "WORKER_CONCORRENCIA",
		"WORKER_MAX_TENTATIVAS=0":                            "WORKER_MAX_TENTATIVAS",
		"APROVA_URL_PUBLICA=api.aprova.test":                 "APROVA_URL_PUBLICA",
		"SANDBOX_DIRETORIOS=Azulpasta/aPRova-TESTES":         "SANDBOX_DIRETORIOS",
		"SANDBOX_DIRETORIOS==go":                             "SANDBOX_DIRETORIOS",
		"SANDBOX_DIRETORIOS=Azulpasta/aPRova-TESTES=../fora": "SANDBOX_DIRETORIOS",
		"SANDBOX_DIRETORIOS=Azulpasta/aPRova-TESTES=/go":     "SANDBOX_DIRETORIOS",
		"SANDBOX_DIRETORIOS=a/b=go,a/b=java":                 "SANDBOX_DIRETORIOS",
	}

	for atribuicao, variavel := range casos {
		t.Run(atribuicao, func(t *testing.T) {
			ambienteMinimoValido(t)
			nome, valor, _ := strings.Cut(atribuicao, "=")
			t.Setenv(nome, valor)

			_, err := Carregar()

			if err == nil || !strings.Contains(err.Error(), variavel) {
				t.Errorf("esperava erro nomeando %s, obtive %v", variavel, err)
			}
		})
	}
}

func TestRedeDoSandboxLocalFicaDesligadaPorPadrao(t *testing.T) {
	ambienteMinimoValido(t)

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if configuracao.SandboxRedeLocal != "none" {
		t.Errorf("sem SANDBOX_REDE_LOCAL o sandbox local roda sem rede, obtive %q", configuracao.SandboxRedeLocal)
	}
}

func TestRedeDoSandboxLocalAceitaNomeDeRedeDoDocker(t *testing.T) {
	ambienteMinimoValido(t)
	t.Setenv("SANDBOX_REDE_LOCAL", "bridge")

	configuracao, err := Carregar()
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if configuracao.SandboxRedeLocal != "bridge" {
		t.Errorf("rede lida errada: %q", configuracao.SandboxRedeLocal)
	}
}

func TestRedeDoSandboxLocalRecusaValorQueViraOpcaoDoDocker(t *testing.T) {
	for _, valor := range []string{"bridge --privileged", "--host", "host;rm"} {
		t.Run(valor, func(t *testing.T) {
			ambienteMinimoValido(t)
			t.Setenv("SANDBOX_REDE_LOCAL", valor)

			if _, err := Carregar(); err == nil || !strings.Contains(err.Error(), "SANDBOX_REDE_LOCAL") {
				t.Errorf("esperava erro nomeando SANDBOX_REDE_LOCAL, obtive %v", err)
			}
		})
	}
}

func TestLoggerDosProcessosLevaOsAtributosDoContexto(t *testing.T) {
	var saida bytes.Buffer
	diario := NovoLogger(Configuracao{NivelLog: "info"}, &saida)

	diario.DebugContext(rastro.Com(t.Context(), "task_run_id", "abaixo-do-nivel"), "não aparece")
	diario.InfoContext(rastro.Com(t.Context(), "task_run_id", "0742a9a6-3866-4ddf-8b1a-1b0501aea6d3"), "saldo de requisições do github")

	texto := saida.String()
	if !strings.Contains(texto, `"task_run_id":"0742a9a6-3866-4ddf-8b1a-1b0501aea6d3"`) {
		t.Errorf("o logger dos processos precisa levar o task_run_id do contexto: %s", texto)
	}
	if strings.Contains(texto, "abaixo-do-nivel") {
		t.Errorf("o nível configurado continua valendo: %s", texto)
	}
}

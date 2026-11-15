package sarif

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/platform"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
	"github.com/PI-GRUPO/aPRova/internal/worker"
)

const (
	repositorioDeTeste = "Azulpasta/aprova-bench-go"
	shaDeTeste         = "0123456789abcdef0123456789abcdef01234567"
	idDoEnvioDeTeste   = "47177e22-5596-11eb-80a1-c1e54ef945c6"
	instalacaoDeTeste  = int64(7001)
)

type envioRecebido struct {
	CommitSHA string `json:"commit_sha"`
	Ref       string `json:"ref"`
	SARIF     string `json:"sarif"`
}

type githubFalso struct {
	t        *testing.T
	servidor *httptest.Server

	mu                sync.Mutex
	statusDosEnvios   []int
	processamentos    []string
	statusDosArquivos int
	envios            []envioRecebido
	consultas         int
	leituras          int
	segurarEnvio      chan struct{}
}

func novoGitHubFalso(t *testing.T) *githubFalso {
	t.Helper()
	falso := &githubFalso{t: t, statusDosArquivos: http.StatusOK}
	falso.servidor = httptest.NewServer(http.HandlerFunc(falso.atender))
	t.Cleanup(falso.servidor.Close)
	return falso
}

type tokenFixo struct{}

func (tokenFixo) TokenParaInstalacao(context.Context, int64) (string, error) {
	return "ghs_token-de-teste", nil
}

func (g *githubFalso) cliente() *platform.Cliente {
	return platform.NovoCliente(platform.OpcoesCliente{Tokens: tokenFixo{}, URLBase: g.servidor.URL, ClienteHTTP: g.servidor.Client()})
}

func (g *githubFalso) atender(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/files"):
		g.listarArquivos(w)
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/code-scanning/sarifs"):
		g.receberEnvio(w, r)
	case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/code-scanning/sarifs/"):
		g.informarProcessamento(w)
	default:
		g.t.Errorf("chamada inesperada ao GitHub: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusTeapot)
	}
}

func (g *githubFalso) listarArquivos(w http.ResponseWriter) {
	g.mu.Lock()
	g.leituras++
	status := g.statusDosArquivos
	g.mu.Unlock()

	w.WriteHeader(status)
	if status == http.StatusOK {
		_ = json.NewEncoder(w).Encode([]map[string]string{
			{"filename": "auth/sessao.go", "status": "modified", "patch": "@@ -40,3 +40,4 @@\n antes()\n meio()\n+depois()\n"},
		})
	}
}

func (g *githubFalso) receberEnvio(w http.ResponseWriter, r *http.Request) {
	var recebido envioRecebido
	_ = json.NewDecoder(r.Body).Decode(&recebido)

	g.mu.Lock()
	g.envios = append(g.envios, recebido)
	status := http.StatusAccepted
	if len(g.statusDosEnvios) > 0 {
		status = g.statusDosEnvios[0]
		if len(g.statusDosEnvios) > 1 {
			g.statusDosEnvios = g.statusDosEnvios[1:]
		}
	}
	segurar := g.segurarEnvio
	g.mu.Unlock()

	if segurar != nil {
		<-segurar
	}
	w.WriteHeader(status)
	if status == http.StatusAccepted {
		_, _ = fmt.Fprintf(w, `{"id":%q}`, idDoEnvioDeTeste)
	}
}

func (g *githubFalso) informarProcessamento(w http.ResponseWriter) {
	g.mu.Lock()
	g.consultas++
	situacao := "complete"
	if len(g.processamentos) > 0 {
		situacao = g.processamentos[0]
		if len(g.processamentos) > 1 {
			g.processamentos = g.processamentos[1:]
		}
	}
	g.mu.Unlock()

	if situacao == "failed" {
		_, _ = w.Write([]byte(`{"processing_status":"failed","errors":["linha fora do arquivo"]}`))
		return
	}
	_, _ = fmt.Fprintf(w, `{"processing_status":%q}`, situacao)
}

func (g *githubFalso) contagem() (envios, consultas, leituras int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.envios), g.consultas, g.leituras
}

func (g *githubFalso) envio(indice int) envioRecebido {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.envios[indice]
}

type registroFalso struct {
	mu         sync.Mutex
	envios     map[string]domain.EnvioSARIF
	desligados map[string]string
}

func novoRegistroFalso() *registroFalso {
	return &registroFalso{envios: map[string]domain.EnvioSARIF{}, desligados: map[string]string{}}
}

func (r *registroFalso) EnvioSARIFDoTaskRun(_ context.Context, taskRunID string) (domain.EnvioSARIF, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	envio, existe := r.envios[taskRunID]
	if !existe {
		return domain.EnvioSARIF{}, domain.ErrNaoEncontrado
	}
	return envio, nil
}

func (r *registroFalso) GravarEnvioSARIF(_ context.Context, envio domain.EnvioSARIF) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if gravado, existe := r.envios[envio.TaskRunID]; existe && gravado.IDDoEnvio != "" && gravado.IDDoEnvio != envio.IDDoEnvio {
		return domain.ErrDuplicado
	}
	r.envios[envio.TaskRunID] = envio
	return nil
}

func (r *registroFalso) EnvioSARIFDesligado(_ context.Context, tenantID int64, repositorio string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, desligado := r.desligados[fmt.Sprintf("%d/%s", tenantID, repositorio)]
	return desligado, nil
}

func (r *registroFalso) DesligarEnvioSARIF(_ context.Context, tenantID int64, repositorio, motivo string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.desligados[fmt.Sprintf("%d/%s", tenantID, repositorio)] = motivo
	return nil
}

func (r *registroFalso) envio(taskRunID string) domain.EnvioSARIF {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.envios[taskRunID]
}

func (r *registroFalso) motivoDoDesligamento() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.desligados[fmt.Sprintf("%d/%s", instalacaoDeTeste, repositorioDeTeste)]
}

func publicacaoDeTeste(taskRunID string) worker.Publicacao {
	return worker.Publicacao{
		TaskRun: domain.TaskRun{ID: taskRunID, TenantID: instalacaoDeTeste, Repositorio: repositorioDeTeste, NumeroPR: 12, SHAHead: shaDeTeste},
		Gate: domain.GateDeRisco{
			TaskRunID:           taskRunID,
			NivelDeterministico: domain.RiscoAlto,
			Nivel:               domain.RiscoAlto,
			Motivos:             []domain.MotivoDeRisco{motivo("area_sensivel", domain.RiscoAlto, "auth/sessao.go")},
		},
		Recibo: receipt.Emitido{ID: reciboDeTeste().ID, Hash: reciboDeTeste().Hash},
	}
}

func novoEnviadorDeTeste(t *testing.T, github *githubFalso, registro *registroFalso) *Enviador {
	t.Helper()
	enviador, err := NovoEnviador(OpcoesEnviador{
		GitHub:               github.cliente(),
		Registro:             registro,
		Versao:               "teste",
		MaxTentativas:        3,
		EsperaBase:           time.Millisecond,
		IntervaloDaConsulta:  time.Millisecond,
		PrazoDoProcessamento: 40 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("montar o enviador: %v", err)
	}
	t.Cleanup(enviador.Encerrar)
	return enviador
}

func enviarEAguardar(enviador *Enviador, publicacao worker.Publicacao) {
	enviador.Enviar(publicacao)
	enviador.Aguardar()
}

func documentoEnviado(t *testing.T, recebido envioRecebido) []byte {
	t.Helper()
	compactado, err := base64.StdEncoding.DecodeString(recebido.SARIF)
	if err != nil {
		t.Fatalf("o campo sarif não é base64: %v", err)
	}
	leitor, err := gzip.NewReader(bytes.NewReader(compactado))
	if err != nil {
		t.Fatalf("o campo sarif não é gzip: %v", err)
	}
	conteudo, err := io.ReadAll(leitor)
	if err != nil {
		t.Fatalf("ler o gzip: %v", err)
	}
	return conteudo
}

func TestNovoEnviadorExigeGitHubERegistro(t *testing.T) {
	if _, err := NovoEnviador(OpcoesEnviador{}); err == nil {
		t.Fatal("sem GitHub e sem registro o enviador não pode ser montado")
	}
}

func TestEnvioMandaOSARIFDoGateGravadoComOCommitEORefDoPR(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()

	enviarEAguardar(novoEnviadorDeTeste(t, github, registro), publicacaoDeTeste("tr-1"))

	if envios, _, _ := github.contagem(); envios != 1 {
		t.Fatalf("vieram %d envios, esperado 1", envios)
	}
	recebido := github.envio(0)
	if recebido.CommitSHA != shaDeTeste || recebido.Ref != "refs/pull/12/head" {
		t.Errorf("commit e ref = %q e %q", recebido.CommitSHA, recebido.Ref)
	}

	serializado := documentoEnviado(t, recebido)
	validarContraOSchema(t, serializado)
	var lido documentoLido
	if err := json.Unmarshal(serializado, &lido); err != nil {
		t.Fatal(err)
	}
	resultado := resultadoDoMotivo(t, lido.Runs[0].Results, "area_sensivel")
	if resultado.arquivo() != "auth/sessao.go" || resultado.linha() != 42 {
		t.Errorf("resultado em %s:%d, esperado auth/sessao.go:42, a linha lida do patch", resultado.arquivo(), resultado.linha())
	}
	if resultado.Properties.Recibo.ID != reciboDeTeste().ID {
		t.Errorf("recibo do resultado = %+v", resultado.Properties.Recibo)
	}

	gravado := registro.envio("tr-1")
	if gravado.IDDoEnvio != idDoEnvioDeTeste || gravado.Estado != domain.EnvioSARIFProcessado {
		t.Errorf("registro = %+v, esperado o id do envio e o estado processado", gravado)
	}
}

func TestRetomadaComEnvioConcluidoNaoChamaOGitHub(t *testing.T) {
	for _, estado := range []domain.EstadoEnvioSARIF{domain.EnvioSARIFProcessado, domain.EnvioSARIFRecusado} {
		t.Run(string(estado), func(t *testing.T) {
			github, registro := novoGitHubFalso(t), novoRegistroFalso()
			anterior := domain.EnvioSARIF{TaskRunID: "tr-1", IDDoEnvio: idDoEnvioDeTeste, Estado: estado}
			_ = registro.GravarEnvioSARIF(t.Context(), anterior)

			enviarEAguardar(novoEnviadorDeTeste(t, github, registro), publicacaoDeTeste("tr-1"))

			if envios, consultas, leituras := github.contagem(); envios+consultas+leituras != 0 {
				t.Errorf("com o id gravado nada é refeito; vieram %d envios, %d consultas e %d leituras", envios, consultas, leituras)
			}
			if registro.envio("tr-1") != anterior {
				t.Errorf("o registro mudou: %+v", registro.envio("tr-1"))
			}
		})
	}
}

func TestRetomadaComEnvioAceitoSoConsultaOProcessamento(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	_ = registro.GravarEnvioSARIF(t.Context(), domain.EnvioSARIF{TaskRunID: "tr-1", IDDoEnvio: idDoEnvioDeTeste, Estado: domain.EnvioSARIFAceito})

	enviarEAguardar(novoEnviadorDeTeste(t, github, registro), publicacaoDeTeste("tr-1"))

	envios, consultas, _ := github.contagem()
	if envios != 0 || consultas == 0 {
		t.Errorf("esperava só a consulta do processamento; vieram %d envios e %d consultas", envios, consultas)
	}
	if registro.envio("tr-1").Estado != domain.EnvioSARIFProcessado {
		t.Errorf("registro = %+v", registro.envio("tr-1"))
	}
}

func TestFalhaTransitoriaNoEnvioEhTentadaDeNovo(t *testing.T) {
	for _, status := range []int{http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusTooManyRequests} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			github, registro := novoGitHubFalso(t), novoRegistroFalso()
			github.statusDosEnvios = []int{status, http.StatusAccepted}

			enviarEAguardar(novoEnviadorDeTeste(t, github, registro), publicacaoDeTeste("tr-1"))

			if envios, _, _ := github.contagem(); envios != 2 {
				t.Errorf("vieram %d envios, esperado 2", envios)
			}
			if registro.envio("tr-1").Estado != domain.EnvioSARIFProcessado {
				t.Errorf("registro = %+v", registro.envio("tr-1"))
			}
		})
	}
}

func TestTentativasEsgotadasFicamRegistradasEARetomadaTentaDeNovo(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	github.statusDosEnvios = []int{http.StatusBadGateway}
	enviador := novoEnviadorDeTeste(t, github, registro)

	enviarEAguardar(enviador, publicacaoDeTeste("tr-1"))

	if envios, _, _ := github.contagem(); envios != 3 {
		t.Fatalf("vieram %d envios, esperado 3, o máximo de tentativas", envios)
	}
	gravado := registro.envio("tr-1")
	if gravado.Estado != domain.EnvioSARIFFalhou || gravado.IDDoEnvio != "" || gravado.Detalhe == "" {
		t.Fatalf("registro = %+v, esperado falhou, sem id e com o diagnóstico", gravado)
	}
	if registro.motivoDoDesligamento() != "" {
		t.Error("falha transitória não desliga o envio do repositório")
	}

	github.mu.Lock()
	github.statusDosEnvios = nil
	github.mu.Unlock()
	enviarEAguardar(enviador, publicacaoDeTeste("tr-1"))

	if registro.envio("tr-1").Estado != domain.EnvioSARIFProcessado {
		t.Errorf("sem id gravado a retomada envia de novo; registro = %+v", registro.envio("tr-1"))
	}
}

func TestSemPermissaoDesligaOEnvioDoRepositorio(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	github.statusDosEnvios = []int{http.StatusForbidden}
	enviador := novoEnviadorDeTeste(t, github, registro)

	enviarEAguardar(enviador, publicacaoDeTeste("tr-1"))

	if envios, _, _ := github.contagem(); envios != 1 {
		t.Fatalf("403 não é tentado de novo; vieram %d envios", envios)
	}
	if registro.envio("tr-1").Estado != domain.EnvioSARIFFalhou {
		t.Errorf("registro = %+v", registro.envio("tr-1"))
	}
	if !strings.Contains(registro.motivoDoDesligamento(), "security_events") {
		t.Errorf("o motivo do desligamento precisa dizer o que conferir: %q", registro.motivoDoDesligamento())
	}

	enviarEAguardar(enviador, publicacaoDeTeste("tr-2"))

	if envios, consultas, leituras := github.contagem(); envios != 1 || consultas != 0 || leituras != 1 {
		t.Errorf("com o envio desligado o GitHub não é chamado; envios %d, consultas %d, leituras %d", envios, consultas, leituras)
	}
}

func TestSARIFRecusadoNoEnvioNaoEhTentadoDeNovoNemDesligaORepositorio(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	github.statusDosEnvios = []int{http.StatusBadRequest}

	enviarEAguardar(novoEnviadorDeTeste(t, github, registro), publicacaoDeTeste("tr-1"))

	if envios, _, _ := github.contagem(); envios != 1 {
		t.Errorf("vieram %d envios, esperado 1", envios)
	}
	if registro.envio("tr-1").Estado != domain.EnvioSARIFFalhou || registro.motivoDoDesligamento() != "" {
		t.Errorf("registro = %+v, desligamento = %q", registro.envio("tr-1"), registro.motivoDoDesligamento())
	}
}

func TestProcessamentoRecusadoFicaRegistradoComOsErros(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	github.processamentos = []string{"pending", "failed"}

	enviarEAguardar(novoEnviadorDeTeste(t, github, registro), publicacaoDeTeste("tr-1"))

	gravado := registro.envio("tr-1")
	if gravado.Estado != domain.EnvioSARIFRecusado || gravado.IDDoEnvio != idDoEnvioDeTeste || !strings.Contains(gravado.Detalhe, "linha fora do arquivo") {
		t.Errorf("registro = %+v", gravado)
	}
}

func TestPrazoDoProcessamentoEsgotadoDeixaOEnvioComoAceito(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	github.processamentos = []string{"pending"}

	enviarEAguardar(novoEnviadorDeTeste(t, github, registro), publicacaoDeTeste("tr-1"))

	gravado := registro.envio("tr-1")
	if gravado.Estado != domain.EnvioSARIFAceito || gravado.IDDoEnvio != idDoEnvioDeTeste || !strings.Contains(gravado.Detalhe, "prazo") {
		t.Errorf("registro = %+v, esperado aceito, com o id e o prazo no detalhe", gravado)
	}
}

func TestSemConseguirLerOsArquivosOEnvioSaiComALinha1(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	github.statusDosArquivos = http.StatusInternalServerError

	enviarEAguardar(novoEnviadorDeTeste(t, github, registro), publicacaoDeTeste("tr-1"))

	if envios, _, _ := github.contagem(); envios != 1 {
		t.Fatalf("vieram %d envios, esperado 1", envios)
	}
	var lido documentoLido
	if err := json.Unmarshal(documentoEnviado(t, github.envio(0)), &lido); err != nil {
		t.Fatal(err)
	}
	if linha := resultadoDoMotivo(t, lido.Runs[0].Results, "area_sensivel").linha(); linha != 1 {
		t.Errorf("linha = %d, esperado 1", linha)
	}
}

func TestPedidosSimultaneosDoMesmoTaskRunViramUmEnvioSo(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	github.segurarEnvio = make(chan struct{})
	enviador := novoEnviadorDeTeste(t, github, registro)

	enviador.Enviar(publicacaoDeTeste("tr-1"))
	enviador.Enviar(publicacaoDeTeste("tr-1"))
	close(github.segurarEnvio)
	enviador.Aguardar()

	if envios, _, _ := github.contagem(); envios != 1 {
		t.Errorf("vieram %d envios, esperado 1", envios)
	}
}

func TestEnviarNaoEsperaOGitHub(t *testing.T) {
	github, registro := novoGitHubFalso(t), novoRegistroFalso()
	github.segurarEnvio = make(chan struct{})
	enviador := novoEnviadorDeTeste(t, github, registro)

	devolveu := make(chan struct{})
	go func() {
		enviador.Enviar(publicacaoDeTeste("tr-1"))
		close(devolveu)
	}()

	select {
	case <-devolveu:
	case <-time.After(2 * time.Second):
		t.Fatal("Enviar ficou esperando a resposta do GitHub e atrasaria a transição do task_run")
	}
	close(github.segurarEnvio)
	enviador.Aguardar()
}

package integracaoeventos

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Azulpasta/aPRova/internal/eventos"
)

const (
	segredoDeTeste      = "segredo-de-teste-que-nunca-aparece"
	urlQueNaoRecebe     = "http://127.0.0.1:9/webhook/aprova"
	tempoLimiteDoScript = 30 * time.Second
)

var raizDoProjeto = filepath.Join("..", "..", "..")

type envio struct {
	cabecalhos http.Header
	corpo      []byte
}

type execucao struct {
	saida  string
	falhou bool
}

func exigirFerramentas(t *testing.T) {
	t.Helper()

	for _, ferramenta := range []string{"bash", "curl", "jq", "openssl"} {
		if _, err := exec.LookPath(ferramenta); err != nil {
			t.Skipf("%s não está instalado", ferramenta)
		}
	}
}

func copiarArquivo(t *testing.T, origem, destino string) {
	t.Helper()

	conteudo, err := os.ReadFile(origem)
	if err != nil {
		t.Fatalf("ler %s: %v", origem, err)
	}
	if err := os.MkdirAll(filepath.Dir(destino), 0o750); err != nil {
		t.Fatalf("criar %s: %v", filepath.Dir(destino), err)
	}
	if err := os.WriteFile(destino, conteudo, 0o600); err != nil {
		t.Fatalf("escrever %s: %v", destino, err)
	}
}

func projetoIsolado(t *testing.T) string {
	t.Helper()
	exigirFerramentas(t)

	raiz := t.TempDir()
	copiarArquivo(t, filepath.Join(raizDoProjeto, "scripts", "emitir-evento.sh"), filepath.Join(raiz, "scripts", "emitir-evento.sh"))
	exemplos, err := filepath.Glob(filepath.Join(raizDoProjeto, "workflows", "exemplos", "*.json"))
	if err != nil || len(exemplos) == 0 {
		t.Fatalf("nenhum exemplo encontrado: %v", err)
	}
	for _, exemplo := range exemplos {
		copiarArquivo(t, exemplo, filepath.Join(raiz, "workflows", "exemplos", filepath.Base(exemplo)))
	}
	return raiz
}

func rodarScript(t *testing.T, raiz string, ambiente []string, argumentos ...string) execucao {
	t.Helper()

	ctx, cancelar := context.WithTimeout(t.Context(), tempoLimiteDoScript)
	defer cancelar()

	comando := exec.CommandContext(ctx, "bash", append([]string{filepath.Join(raiz, "scripts", "emitir-evento.sh")}, argumentos...)...)
	comando.Dir = raiz
	comando.Env = append(ambienteSemSegredo(), ambiente...)
	saida, err := comando.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("o script não terminou em %v:\n%s", tempoLimiteDoScript, saida)
	}
	return execucao{saida: string(saida), falhou: err != nil}
}

func ambienteSemSegredo() []string {
	var ambiente []string
	for _, variavel := range os.Environ() {
		if !strings.HasPrefix(variavel, "N8N_SEGREDO_EVENTOS=") && !strings.HasPrefix(variavel, "APROVA_URL_PUBLICA=") {
			ambiente = append(ambiente, variavel)
		}
	}
	return ambiente
}

func comSegredo(segredo string) []string {
	return []string{"N8N_SEGREDO_EVENTOS=" + segredo}
}

func imprimir(t *testing.T, raiz string, ambiente []string, argumentos ...string) envio {
	t.Helper()

	resultado := rodarScript(t, raiz, ambiente, append(argumentos, "--imprimir")...)
	if resultado.falhou {
		t.Fatalf("o script falhou com --imprimir:\n%s", resultado.saida)
	}
	return interpretarImpressao(t, resultado.saida)
}

func interpretarImpressao(t *testing.T, saida string) envio {
	t.Helper()

	cabecalhoBruto, corpo, separado := strings.Cut(saida, "\n\n")
	if !separado {
		t.Fatalf("a saída de --imprimir precisa separar cabeçalhos e corpo por uma linha vazia:\n%s", saida)
	}

	cabecalhos := http.Header{}
	for _, linha := range strings.Split(cabecalhoBruto, "\n")[1:] {
		nome, valor, ok := strings.Cut(linha, ": ")
		if !ok {
			t.Fatalf("linha de cabeçalho fora do formato \"Nome: valor\": %q", linha)
		}
		cabecalhos.Set(nome, valor)
	}
	return envio{cabecalhos: cabecalhos, corpo: []byte(strings.TrimSuffix(corpo, "\n"))}
}

func exigirAssinaturaDoNucleo(t *testing.T, segredo string, recebido envio) {
	t.Helper()

	err := eventos.Verificar([]byte(segredo), recebido.corpo,
		recebido.cabecalhos.Get(eventos.CabecalhoTimestamp), recebido.cabecalhos.Get(eventos.CabecalhoAssinatura), time.Now())
	if err != nil {
		t.Errorf("a assinatura do script foi recusada pela verificação do Go: %v", err)
	}
}

func exigirEventoValido(t *testing.T, corpo []byte) eventos.Evento {
	t.Helper()

	evento, err := eventos.Validar(corpo)
	if err != nil {
		t.Fatalf("o script enviou um evento inválido: %v\n%s", err, corpo)
	}
	return evento
}

func envelopeDoExemplo(t *testing.T, tipo eventos.TipoEvento) eventos.Evento {
	t.Helper()

	corpo, err := os.ReadFile(filepath.Join(raizDoProjeto, "workflows", "exemplos", string(tipo)+".json"))
	if err != nil {
		t.Fatalf("ler exemplo: %v", err)
	}
	return exigirEventoValido(t, corpo)
}

func TestScriptAssinaCadaTipoComoONucleoVerifica(t *testing.T) {
	raiz := projetoIsolado(t)

	for _, tipo := range eventos.Tipos() {
		t.Run(string(tipo), func(t *testing.T) {
			recebido := imprimir(t, raiz, comSegredo(segredoDeTeste), string(tipo), urlQueNaoRecebe)

			exigirAssinaturaDoNucleo(t, segredoDeTeste, recebido)
			evento := exigirEventoValido(t, recebido.corpo)
			exemplo := envelopeDoExemplo(t, tipo)
			if evento.TipoEvento != tipo {
				t.Errorf("tipo_evento = %q, esperava %q", evento.TipoEvento, tipo)
			}
			if evento.IDEvento == exemplo.IDEvento {
				t.Error("o script precisa trocar o id_evento do exemplo")
			}
			if desde := time.Since(evento.OcorridoEm); desde < -time.Minute || desde > time.Minute {
				t.Errorf("ocorrido_em = %v, esperava a hora atual", evento.OcorridoEm)
			}
			if tipo := recebido.cabecalhos.Get("Content-Type"); tipo != "application/json" {
				t.Errorf("Content-Type = %q", tipo)
			}
		})
	}
}

func TestScriptGeraUmIDEventoNovoACadaEnvio(t *testing.T) {
	raiz := projetoIsolado(t)

	primeiro := exigirEventoValido(t, imprimir(t, raiz, comSegredo(segredoDeTeste), "instalacao", urlQueNaoRecebe).corpo)
	segundo := exigirEventoValido(t, imprimir(t, raiz, comSegredo(segredoDeTeste), "instalacao", urlQueNaoRecebe).corpo)

	if primeiro.IDEvento == segundo.IDEvento {
		t.Errorf("dois envios com o mesmo id_evento %s", primeiro.IDEvento)
	}
}

func TestScriptNuncaImprimeOSegredo(t *testing.T) {
	raiz := projetoIsolado(t)

	resultado := rodarScript(t, raiz, comSegredo(segredoDeTeste), "aguardando_aprovacao", urlQueNaoRecebe, "--imprimir")

	if strings.Contains(resultado.saida, segredoDeTeste) {
		t.Error("o segredo apareceu na saída do script")
	}
}

func TestScriptLeOSegredoDoArquivoEnvQuandoOAmbienteNaoTem(t *testing.T) {
	raiz := projetoIsolado(t)
	segredoDoArquivo := "segredo-que-so-existe-no-env"
	conteudo := "N8N_SEGREDO_ACOES=outro-segredo\nN8N_SEGREDO_EVENTOS=\"" + segredoDoArquivo + "\"\n"
	if err := os.WriteFile(filepath.Join(raiz, ".env"), []byte(conteudo), 0o600); err != nil {
		t.Fatalf("escrever .env: %v", err)
	}

	resultado := rodarScript(t, raiz, nil, "instalacao", urlQueNaoRecebe, "--imprimir")
	if resultado.falhou {
		t.Fatalf("o script falhou lendo o .env:\n%s", resultado.saida)
	}

	exigirAssinaturaDoNucleo(t, segredoDoArquivo, interpretarImpressao(t, resultado.saida))
	if strings.Contains(resultado.saida, segredoDoArquivo) {
		t.Error("o segredo do .env apareceu na saída do script")
	}
}

func TestScriptSemSegredoNaoEnvia(t *testing.T) {
	raiz := projetoIsolado(t)

	resultado := rodarScript(t, raiz, nil, "instalacao", urlQueNaoRecebe, "--imprimir")

	if !resultado.falhou || !strings.Contains(resultado.saida, "N8N_SEGREDO_EVENTOS") {
		t.Errorf("sem segredo o script precisa falhar nomeando N8N_SEGREDO_EVENTOS:\n%s", resultado.saida)
	}
}

func TestScriptSobrescreveCamposDosDados(t *testing.T) {
	raiz := projetoIsolado(t)

	recebido := imprimir(t, raiz, comSegredo(segredoDeTeste), "analise_concluida", urlQueNaoRecebe,
		"--repositorio", "PI-GRUPO/aPRova", "--numero-pr", "7", "--nivel-risco", "alto",
		"--dado", "explicacao=Teste manual do workflow de alerta")

	exigirAssinaturaDoNucleo(t, segredoDeTeste, recebido)
	dados, doTipo := exigirEventoValido(t, recebido.corpo).Dados.(*eventos.DadosAnaliseConcluida)
	if !doTipo {
		t.Fatal("dados de analise_concluida vieram de outro tipo")
	}
	if dados.Repositorio != "PI-GRUPO/aPRova" || dados.NumeroPR != 7 || dados.NivelRisco != "alto" {
		t.Errorf("campos não sobrescritos: %+v", dados)
	}
	if dados.URLPR != "https://github.com/PI-GRUPO/aPRova/pull/7" {
		t.Errorf("url_pr precisa acompanhar repositorio e numero_pr, obtive %s", dados.URLPR)
	}
	if dados.Explicacao != "Teste manual do workflow de alerta" {
		t.Errorf("--dado não sobrescreveu explicacao: %q", dados.Explicacao)
	}
}

func TestScriptRecusaSobrescreverCampoQueOTipoNaoTem(t *testing.T) {
	raiz := projetoIsolado(t)

	resultado := rodarScript(t, raiz, comSegredo(segredoDeTeste), "instalacao", urlQueNaoRecebe, "--numero-pr", "7", "--imprimir")

	if !resultado.falhou || !strings.Contains(resultado.saida, "numero_pr") {
		t.Errorf("o script precisa recusar numero_pr em instalacao, nomeando o campo:\n%s", resultado.saida)
	}
}

func TestScriptRecusaTipoDesconhecidoListandoOsTipos(t *testing.T) {
	raiz := projetoIsolado(t)

	resultado := rodarScript(t, raiz, comSegredo(segredoDeTeste), "analise_iniciada", urlQueNaoRecebe, "--imprimir")

	if !resultado.falhou || !strings.Contains(resultado.saida, "aguardando_aprovacao") {
		t.Errorf("o script precisa recusar o tipo e listar os conhecidos:\n%s", resultado.saida)
	}
}

var tiposComRecibo = []eventos.TipoEvento{eventos.TipoAnaliseConcluida, eventos.TipoAguardandoAprovacao}

func urlDoRecibo(t *testing.T, corpo []byte) string {
	t.Helper()

	switch dados := exigirEventoValido(t, corpo).Dados.(type) {
	case *eventos.DadosAnaliseConcluida:
		return dados.URLRecibo
	case *eventos.DadosAguardandoAprovacao:
		return dados.URLRecibo
	default:
		t.Fatalf("o tipo %T não tem url_recibo", dados)
		return ""
	}
}

func caminhoDoReciboNoExemplo(t *testing.T, tipo eventos.TipoEvento) string {
	t.Helper()

	corpo, err := os.ReadFile(filepath.Join(raizDoProjeto, "workflows", "exemplos", string(tipo)+".json"))
	if err != nil {
		t.Fatalf("ler exemplo: %v", err)
	}
	_, caminho, encontrado := strings.Cut(urlDoRecibo(t, corpo), "/recibos/")
	if !encontrado {
		t.Fatalf("o exemplo de %s não tem /recibos/ na url_recibo", tipo)
	}
	return "/recibos/" + caminho
}

func TestScriptUsaABaseDaURLPublicaNoRecibo(t *testing.T) {
	raiz := projetoIsolado(t)

	for _, tipo := range tiposComRecibo {
		t.Run(string(tipo), func(t *testing.T) {
			ambiente := append(comSegredo(segredoDeTeste), "APROVA_URL_PUBLICA=https://api.aprova.test/")
			recebido := imprimir(t, raiz, ambiente, string(tipo), urlQueNaoRecebe)

			exigirAssinaturaDoNucleo(t, segredoDeTeste, recebido)
			if esperada := "https://api.aprova.test" + caminhoDoReciboNoExemplo(t, tipo); urlDoRecibo(t, recebido.corpo) != esperada {
				t.Errorf("url_recibo = %s, esperava %s", urlDoRecibo(t, recebido.corpo), esperada)
			}
		})
	}
}

func TestScriptMantemOEnderecoProvisorioSemURLPublica(t *testing.T) {
	raiz := projetoIsolado(t)

	for _, tipo := range tiposComRecibo {
		t.Run(string(tipo), func(t *testing.T) {
			ambiente := append(comSegredo(segredoDeTeste), "APROVA_URL_PUBLICA=")
			recebido := imprimir(t, raiz, ambiente, string(tipo), urlQueNaoRecebe)

			if esperada := "https://aprova.example.com" + caminhoDoReciboNoExemplo(t, tipo); urlDoRecibo(t, recebido.corpo) != esperada {
				t.Errorf("url_recibo = %s, esperava o endereço provisório %s", urlDoRecibo(t, recebido.corpo), esperada)
			}
		})
	}
}

func TestScriptLeAURLPublicaDoArquivoEnv(t *testing.T) {
	raiz := projetoIsolado(t)
	if err := os.WriteFile(filepath.Join(raiz, ".env"), []byte("APROVA_URL_PUBLICA=https://api.aprova.test\n"), 0o600); err != nil {
		t.Fatalf("escrever .env: %v", err)
	}

	recebido := imprimir(t, raiz, comSegredo(segredoDeTeste), "analise_concluida", urlQueNaoRecebe)

	if esperada := "https://api.aprova.test" + caminhoDoReciboNoExemplo(t, eventos.TipoAnaliseConcluida); urlDoRecibo(t, recebido.corpo) != esperada {
		t.Errorf("url_recibo = %s, esperava %s", urlDoRecibo(t, recebido.corpo), esperada)
	}
}

func TestScriptRecusaURLPublicaQueNaoEHTTP(t *testing.T) {
	raiz := projetoIsolado(t)

	resultado := rodarScript(t, raiz, append(comSegredo(segredoDeTeste), "APROVA_URL_PUBLICA=api.aprova.test"),
		"analise_concluida", urlQueNaoRecebe, "--imprimir")

	if !resultado.falhou || !strings.Contains(resultado.saida, "APROVA_URL_PUBLICA") {
		t.Errorf("uma base sem http:// ou https:// precisa ser recusada nomeando APROVA_URL_PUBLICA:\n%s", resultado.saida)
	}
}

func receptor(t *testing.T, status int) (*httptest.Server, <-chan envio) {
	t.Helper()

	recebidos := make(chan envio, 1)
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corpo, err := io.ReadAll(r.Body)
		if err != nil || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		recebidos <- envio{cabecalhos: r.Header.Clone(), corpo: corpo}
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": http.StatusText(status)})
	}))
	t.Cleanup(servidor.Close)
	return servidor, recebidos
}

func TestScriptEnviaOEventoAssinadoEMostraOCodigoHTTP(t *testing.T) {
	raiz := projetoIsolado(t)
	servidor, recebidos := receptor(t, http.StatusOK)

	resultado := rodarScript(t, raiz, comSegredo(segredoDeTeste), "aguardando_aprovacao", servidor.URL+"/webhook/aprova")

	if resultado.falhou || !strings.Contains(resultado.saida, "HTTP 200") {
		t.Fatalf("envio com resposta 200 precisa terminar com sucesso mostrando o código:\n%s", resultado.saida)
	}
	recebido := <-recebidos
	exigirAssinaturaDoNucleo(t, segredoDeTeste, recebido)
	if bytes.Contains(recebido.corpo, []byte("\n")) {
		t.Error("o corpo enviado precisa ser exatamente o assinado, sem quebra de linha a mais")
	}
	exigirEventoValido(t, recebido.corpo)
}

func TestScriptFalhaQuandoARespostaNaoE2xx(t *testing.T) {
	raiz := projetoIsolado(t)
	servidor, recebidos := receptor(t, http.StatusUnauthorized)

	resultado := rodarScript(t, raiz, comSegredo(segredoDeTeste), "instalacao", servidor.URL)

	if !resultado.falhou || !strings.Contains(resultado.saida, "HTTP 401") {
		t.Errorf("resposta 401 precisa fazer o script falhar mostrando o código:\n%s", resultado.saida)
	}
	<-recebidos
}

func TestScriptFalhaQuandoNinguemRecebe(t *testing.T) {
	raiz := projetoIsolado(t)

	resultado := rodarScript(t, raiz, comSegredo(segredoDeTeste), "instalacao", urlQueNaoRecebe)

	if !resultado.falhou {
		t.Errorf("sem ninguém escutando o script precisa falhar:\n%s", resultado.saida)
	}
}

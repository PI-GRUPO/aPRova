package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/receipt"
)

type recibosEmMemoria struct {
	porID  map[string]domain.Recibo
	ultimo *domain.Recibo
	erro   error
}

func (r *recibosEmMemoria) GravarDecisaoComRecibo(_ context.Context, _ domain.GateDeRisco, montar func(anterior *domain.Recibo) (domain.NovoRecibo, error)) (domain.Recibo, error) {
	novo, err := montar(r.ultimo)
	if err != nil {
		return domain.Recibo{}, err
	}
	gravado := domain.Recibo{NovoRecibo: novo, TenantID: 161192271}
	r.porID[gravado.ID] = gravado
	r.ultimo = &gravado
	return gravado, nil
}

func (r *recibosEmMemoria) BuscarRecibo(_ context.Context, id string) (domain.Recibo, error) {
	if r.erro != nil {
		return domain.Recibo{}, r.erro
	}
	recibo, existe := r.porID[id]
	if !existe {
		return domain.Recibo{}, domain.ErrNaoEncontrado
	}
	return recibo, nil
}

type cenarioDeRecibos struct {
	guardados *recibosEmMemoria
	assinador *receipt.Assinador
	ids       []string
	servidor  http.Handler
}

func novoCenarioDeRecibos(t *testing.T, quantos int) cenarioDeRecibos {
	t.Helper()

	assinador, err := receipt.NovoAssinador(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	guardados := &recibosEmMemoria{porID: map[string]domain.Recibo{}}
	emissor, err := receipt.NovoEmissor(receipt.OpcoesEmissor{Assinador: assinador, Armazenamento: guardados})
	if err != nil {
		t.Fatal(err)
	}

	cenario := cenarioDeRecibos{guardados: guardados, assinador: assinador}
	for range quantos {
		emitido, err := emissor.Emitir(t.Context(), receipt.Pedido{
			TaskRun: domain.TaskRun{ID: "9c80d110-a49a-4a70-b5f4-78cdf75b2f19", TenantID: 161192271, Repositorio: "Azulpasta/aPRova-TESTES", NumeroPR: 1, SHAHead: "d7ee920e4f9f8a2f413630afbe6c25435149a302"},
			Gate: domain.GateDeRisco{
				TaskRunID: "9c80d110-a49a-4a70-b5f4-78cdf75b2f19", NivelDeterministico: domain.RiscoAlto, Nivel: domain.RiscoAlto,
				Motivos: []domain.MotivoDeRisco{{ID: "falha_sandbox", Evidencias: []string{"1 de 11 testes falharam"}}},
			},
			Tentativa: 1, Diff: "diff",
		})
		if err != nil {
			t.Fatal(err)
		}
		cenario.ids = append(cenario.ids, emitido.ID)
	}

	cenario.servidor = NovoServidor(Opcoes{
		Recibos: receipt.NovoVerificador(receipt.NovoChaveiro(assinador.ChavePublica()), guardados),
	}).Rotas()
	return cenario
}

func (c cenarioDeRecibos) pedir(t *testing.T, caminho string) *httptest.ResponseRecorder {
	t.Helper()

	gravador := httptest.NewRecorder()
	c.servidor.ServeHTTP(gravador, httptest.NewRequestWithContext(t.Context(), http.MethodGet, caminho, nil))
	return gravador
}

func decodificar[T any](t *testing.T, gravador *httptest.ResponseRecorder) T {
	t.Helper()

	var corpo T
	if err := json.Unmarshal(gravador.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("resposta não é o JSON esperado: %v\n%s", err, gravador.Body.String())
	}
	return corpo
}

func TestReciboDevolveOsBytesExatosAAssinaturaEOConteudoLegivel(t *testing.T) {
	cenario := novoCenarioDeRecibos(t, 2)
	guardado := cenario.guardados.porID[cenario.ids[1]]

	gravador := cenario.pedir(t, "/receipts/"+cenario.ids[1])

	if gravador.Code != http.StatusOK {
		t.Fatalf("status %d: %s", gravador.Code, gravador.Body.String())
	}
	exportado := decodificar[receipt.Exportado](t, gravador)
	payload, err := base64.StdEncoding.DecodeString(exportado.PayloadBase64)
	if err != nil || string(payload) != guardado.Conteudo {
		t.Errorf("o payload em base64 precisa ser, byte a byte, o conteúdo guardado: %v", err)
	}
	if exportado.KeyID != cenario.assinador.IDChave() || exportado.AssinaturaBase64 != base64.StdEncoding.EncodeToString(guardado.Assinatura) {
		t.Errorf("assinatura e key_id: %+v", exportado)
	}
	if !strings.Contains(string(exportado.Payload), `"repositorio":"Azulpasta/aPRova-TESTES"`) {
		t.Errorf("o payload decodificado vai junto, para leitura: %s", exportado.Payload)
	}

	_, resultado, err := receipt.VerificarExportado(receipt.NovoChaveiro(cenario.assinador.ChavePublica()), gravador.Body.Bytes())
	if err != nil || !resultado.Valido {
		t.Errorf("a resposta precisa bastar para verificar só com a chave pública: %+v, erro %v", resultado, err)
	}
}

func TestVerificacaoDeReciboIntegroRespondeValido(t *testing.T) {
	cenario := novoCenarioDeRecibos(t, 2)

	for _, id := range cenario.ids {
		gravador := cenario.pedir(t, "/receipts/"+id+"/verify")

		resposta := decodificar[respostaDaVerificacao](t, gravador)
		if gravador.Code != http.StatusOK || !resposta.Valido || resposta.Falha != "" || resposta.ID != id {
			t.Errorf("recibo %s: status %d, %+v", id, gravador.Code, resposta)
		}
		for _, verificacao := range resposta.Verificacoes {
			if verificacao.Resultado != "ok" {
				t.Errorf("verificação %s = %s", verificacao.Nome, verificacao.Resultado)
			}
		}
		if len(resposta.Verificacoes) != 5 {
			t.Errorf("esperava hash, chave, assinatura, conteúdo e elo: %+v", resposta.Verificacoes)
		}
	}
}

func TestVerificacaoDizQualConferenciaFalhou(t *testing.T) {
	adulterar := func(cenario cenarioDeRecibos, id string, mudar func(*domain.Recibo)) {
		recibo := cenario.guardados.porID[id]
		mudar(&recibo)
		cenario.guardados.porID[id] = recibo
	}
	casos := map[string]struct {
		estragar func(cenario cenarioDeRecibos)
		falha    string
		estados  []string
	}{
		"conteúdo alterado no banco": {
			func(c cenarioDeRecibos) {
				adulterar(c, c.ids[1], func(r *domain.Recibo) {
					r.Conteudo = strings.Replace(r.Conteudo, `"nivel":"alto"`, `"nivel":"baixo"`, 1)
				})
			},
			"hash", []string{"falhou", "nao_verificada", "nao_verificada", "nao_verificada", "nao_verificada"},
		},
		"conteúdo e hash alterados": {
			func(c cenarioDeRecibos) {
				adulterar(c, c.ids[1], func(r *domain.Recibo) {
					r.Conteudo = strings.Replace(r.Conteudo, `"nivel":"alto"`, `"nivel":"baixo"`, 1)
					r.Hash = receipt.Hash([]byte(r.Conteudo))
				})
			},
			"assinatura", []string{"ok", "ok", "falhou", "nao_verificada", "nao_verificada"},
		},
		"chave que o servidor não conhece": {
			func(c cenarioDeRecibos) {
				adulterar(c, c.ids[1], func(r *domain.Recibo) { r.IDChave = "0000000000000000" })
			},
			"chave_desconhecida", []string{"ok", "falhou", "nao_verificada", "nao_verificada", "nao_verificada"},
		},
		"anterior apagado": {
			func(c cenarioDeRecibos) { delete(c.guardados.porID, c.ids[0]) },
			"elo", []string{"ok", "ok", "ok", "ok", "falhou"},
		},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			cenario := novoCenarioDeRecibos(t, 2)
			caso.estragar(cenario)

			gravador := cenario.pedir(t, "/receipts/"+cenario.ids[1]+"/verify")

			resposta := decodificar[respostaDaVerificacao](t, gravador)
			if gravador.Code != http.StatusOK || resposta.Valido || resposta.Falha != caso.falha || resposta.Detalhe == "" {
				t.Fatalf("status %d, resposta %+v", gravador.Code, resposta)
			}
			var estados []string
			for _, verificacao := range resposta.Verificacoes {
				estados = append(estados, verificacao.Resultado)
			}
			if strings.Join(estados, ",") != strings.Join(caso.estados, ",") {
				t.Errorf("verificações = %v, esperava %v", estados, caso.estados)
			}
		})
	}
}

func TestChavesPublicasSaoListadasEmBase64(t *testing.T) {
	cenario := novoCenarioDeRecibos(t, 0)

	gravador := cenario.pedir(t, "/receipts/keys")

	resposta := decodificar[respostaDasChaves](t, gravador)
	if gravador.Code != http.StatusOK || len(resposta.Chaves) != 1 {
		t.Fatalf("status %d: %s", gravador.Code, gravador.Body.String())
	}
	chave := resposta.Chaves[0]
	if chave.KeyID != cenario.assinador.IDChave() || chave.Algoritmo != "Ed25519" ||
		chave.ChavePublicaBase64 != base64.StdEncoding.EncodeToString(cenario.assinador.ChavePublica()) {
		t.Errorf("chave listada: %+v", chave)
	}
}

func TestReciboInexistenteOuComIdMalformadoResponde404(t *testing.T) {
	cenario := novoCenarioDeRecibos(t, 1)

	for _, caminho := range []string{
		"/receipts/5d0a4c56-58f1-4e35-9d0e-2f1f0f6a7b22",
		"/receipts/5d0a4c56-58f1-4e35-9d0e-2f1f0f6a7b22/verify",
		"/receipts/nao-e-uuid",
		"/receipts/nao-e-uuid/verify",
		"/receipts/1%20OR%201=1/verify",
	} {
		if gravador := cenario.pedir(t, caminho); gravador.Code != http.StatusNotFound {
			t.Errorf("%s: status %d, esperava 404", caminho, gravador.Code)
		}
	}
}

func TestRecibosSaoSomenteLeitura(t *testing.T) {
	cenario := novoCenarioDeRecibos(t, 1)

	for _, metodo := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		gravador := httptest.NewRecorder()
		cenario.servidor.ServeHTTP(gravador, httptest.NewRequestWithContext(t.Context(), metodo, "/receipts/"+cenario.ids[0], nil))

		if gravador.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: status %d, esperava 405", metodo, gravador.Code)
		}
	}
}

func TestFalhaAoLerOReciboResponde500SemDetalheInterno(t *testing.T) {
	cenario := novoCenarioDeRecibos(t, 1)
	cenario.guardados.erro = errors.New("conexão recusada em 10.0.0.7:5432")

	for _, caminho := range []string{"/receipts/" + cenario.ids[0], "/receipts/" + cenario.ids[0] + "/verify"} {
		gravador := cenario.pedir(t, caminho)

		if gravador.Code != http.StatusInternalServerError || strings.Contains(gravador.Body.String(), "10.0.0.7") {
			t.Errorf("%s: status %d, corpo %s", caminho, gravador.Code, gravador.Body.String())
		}
	}
}

func TestSemVerificadorConfiguradoOsRecibosFicamIndisponiveis(t *testing.T) {
	servidor := NovoServidor(Opcoes{}).Rotas()

	for _, caminho := range []string{"/receipts/keys", "/receipts/5d0a4c56-58f1-4e35-9d0e-2f1f0f6a7b22", "/receipts/5d0a4c56-58f1-4e35-9d0e-2f1f0f6a7b22/verify"} {
		gravador := httptest.NewRecorder()
		servidor.ServeHTTP(gravador, httptest.NewRequestWithContext(t.Context(), http.MethodGet, caminho, nil))

		if gravador.Code != http.StatusServiceUnavailable {
			t.Errorf("%s: status %d, esperava 503", caminho, gravador.Code)
		}
	}
}

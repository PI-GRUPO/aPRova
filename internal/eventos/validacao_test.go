package eventos

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Azulpasta/aPRova/internal/domain"
)

func exemploComoMapa(t *testing.T, tipo TipoEvento) map[string]any {
	t.Helper()

	corpo, err := os.ReadFile(filepath.Join(pastaDosExemplos, string(tipo)+".json"))
	if err != nil {
		t.Fatalf("ler exemplo de %s: %v", tipo, err)
	}
	var evento map[string]any
	if err := json.Unmarshal(corpo, &evento); err != nil {
		t.Fatalf("decodificar exemplo de %s: %v", tipo, err)
	}
	return evento
}

func objetoJSON(t *testing.T, valor any) map[string]any {
	t.Helper()

	objeto, ehObjeto := valor.(map[string]any)
	if !ehObjeto {
		t.Fatalf("esperava objeto JSON, obtive %T", valor)
	}
	return objeto
}

func itemDaLista(t *testing.T, valor any, indice int) map[string]any {
	t.Helper()

	lista, ehLista := valor.([]any)
	if !ehLista || indice >= len(lista) {
		t.Fatalf("esperava lista JSON com o item %d, obtive %v", indice, valor)
	}
	return objetoJSON(t, lista[indice])
}

func dadosDe(t *testing.T, evento map[string]any) map[string]any {
	t.Helper()
	return objetoJSON(t, evento["dados"])
}

func serializar(t *testing.T, evento map[string]any) []byte {
	t.Helper()

	corpo, err := json.Marshal(evento)
	if err != nil {
		t.Fatalf("serializar evento: %v", err)
	}
	return corpo
}

func TestValidarDevolveEnvelopeEDadosDoTipo(t *testing.T) {
	evento, err := Validar(serializar(t, exemploComoMapa(t, TipoAguardandoAprovacao)))
	if err != nil {
		t.Fatalf("exemplo recusado: %v", err)
	}

	dados, doTipo := evento.Dados.(*DadosAguardandoAprovacao)
	if !doTipo {
		t.Fatalf("dados de aguardando_aprovacao vieram como %T", evento.Dados)
	}
	if evento.Versao != 1 || evento.TenantID != 58291734 || !evento.OcorridoEm.Equal(time.Date(2026, 10, 19, 16, 5, 41, 0, time.UTC)) {
		t.Errorf("envelope decodificado errado: %+v", evento.Envelope)
	}
	if dados.NivelRisco != "alto" || len(dados.Motivos) != 2 || dados.Motivos[0].Nome != "area_sensivel" {
		t.Errorf("dados decodificados errado: %+v", dados)
	}
}

func TestDecisaoDaAprovacaoUsaOsValoresDoDominio(t *testing.T) {
	for _, decisao := range []domain.DecisaoAprovacao{domain.DecisaoAprovada, domain.DecisaoAlteracoesSolicitadas} {
		t.Run(string(decisao), func(t *testing.T) {
			evento := exemploComoMapa(t, TipoAprovacaoRegistrada)
			dadosDe(t, evento)["decisao"] = string(decisao)

			validado, err := Validar(serializar(t, evento))
			if err != nil {
				t.Fatalf("decisão %q do domínio recusada: %v", decisao, err)
			}
			if dados, doTipo := validado.Dados.(*DadosAprovacaoRegistrada); !doTipo || dados.Decisao != decisao {
				t.Errorf("decisão decodificada errado: %+v", validado.Dados)
			}
		})
	}
}

func TestValidarAceitaAsVariacoesPrevistasNoContrato(t *testing.T) {
	casos := map[string]struct {
		tipo  TipoEvento
		mudar func(t *testing.T, evento map[string]any)
	}{
		"aprovação com a decisão aprovada do domínio": {TipoAprovacaoRegistrada, func(t *testing.T, e map[string]any) {
			dadosDe(t, e)["decisao"] = string(domain.DecisaoAprovada)
			dadosDe(t, e)["origem"] = "github"
		}},
		"análise de risco baixo sem nenhuma regra acionada": {TipoAnaliseConcluida, func(t *testing.T, e map[string]any) {
			dadosDe(t, e)["nivel_risco"] = "baixo"
			dadosDe(t, e)["motivos"] = []any{}
		}},
		"ocorrido_em com fração de segundo": {TipoInstalacao, func(t *testing.T, e map[string]any) {
			e["ocorrido_em"] = "2026-10-18T20:03:15.123456Z"
		}},
		"instalação removida sem lista de repositórios": {TipoInstalacao, func(t *testing.T, e map[string]any) {
			dadosDe(t, e)["acao"] = "removido"
			dadosDe(t, e)["repositorios"] = []any{}
		}},
		"dependência do PyPI": {TipoDependenciaNova, func(t *testing.T, e map[string]any) {
			dadosDe(t, e)["dependencias"] = []any{map[string]any{"ecossistema": "PyPI", "nome": "requests", "versao": "2.32.3"}}
		}},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			evento := exemploComoMapa(t, caso.tipo)
			caso.mudar(t, evento)

			if _, err := Validar(serializar(t, evento)); err != nil {
				t.Errorf("variação válida recusada: %v", err)
			}
		})
	}
}

func TestValidarRecusaEventoForaDoContrato(t *testing.T) {
	casos := map[string]struct {
		tipo  TipoEvento
		mudar func(t *testing.T, evento map[string]any)
	}{
		"campo desconhecido no envelope": {TipoAnaliseConcluida, func(t *testing.T, e map[string]any) {
			e["prioridade"] = "alta"
		}},
		"campo desconhecido nos dados": {TipoAnaliseConcluida, func(t *testing.T, e map[string]any) {
			dadosDe(t, e)["aprovar_automaticamente"] = true
		}},
		"campo desconhecido num motivo": {TipoAguardandoAprovacao, func(t *testing.T, e map[string]any) {
			itemDaLista(t, dadosDe(t, e)["motivos"], 0)["peso"] = 3
		}},
		"campo de outro tipo nos dados": {TipoDependenciaNova, func(t *testing.T, e map[string]any) {
			dadosDe(t, e)["nivel_risco"] = "alto"
		}},
		"id_evento ausente": {TipoAnaliseConcluida, func(t *testing.T, e map[string]any) {
			delete(e, "id_evento")
		}},
		"tenant_id ausente": {TipoInstalacao, func(t *testing.T, e map[string]any) {
			delete(e, "tenant_id")
		}},
		"dados ausentes": {TipoInstalacao, func(t *testing.T, e map[string]any) {
			delete(e, "dados")
		}},
		"sha ausente": {TipoAnaliseConcluida, func(t *testing.T, e map[string]any) {
			delete(dadosDe(t, e), "sha")
		}},
		"numero_pr ausente": {TipoFalhaRecuperavel, func(t *testing.T, e map[string]any) {
			delete(dadosDe(t, e), "numero_pr")
		}},
		"motivos ausentes": {TipoAguardandoAprovacao, func(t *testing.T, e map[string]any) {
			delete(dadosDe(t, e), "motivos")
		}},
		"descrição do motivo ausente": {TipoAguardandoAprovacao, func(t *testing.T, e map[string]any) {
			delete(itemDaLista(t, dadosDe(t, e)["motivos"], 1), "descricao")
		}},
		"dependência sem versão": {TipoDependenciaNova, func(t *testing.T, e map[string]any) {
			delete(itemDaLista(t, dadosDe(t, e)["dependencias"], 0), "versao")
		}},
		"dependencia_nova sem nenhuma dependência": {TipoDependenciaNova, func(t *testing.T, e map[string]any) {
			dadosDe(t, e)["dependencias"] = []any{}
		}},
		"tipo_evento desconhecido": {TipoAnaliseConcluida, func(t *testing.T, e map[string]any) {
			e["tipo_evento"] = "analise_iniciada"
		}},
		"tipo_evento ausente": {TipoAnaliseConcluida, func(t *testing.T, e map[string]any) {
			delete(e, "tipo_evento")
		}},
		"versão do contrato desconhecida": {TipoAnaliseConcluida, func(t *testing.T, e map[string]any) {
			e["versao"] = 2
		}},
		"id_evento que não é uuid": {TipoAnaliseConcluida, func(t *testing.T, e map[string]any) {
			e["id_evento"] = "evento-14"
		}},
		"ocorrido_em fora de UTC": {TipoAnaliseConcluida, func(t *testing.T, e map[string]any) {
			e["ocorrido_em"] = "2026-10-19T11:32:08-03:00"
		}},
		"ocorrido_em fora do RFC 3339": {TipoAnaliseConcluida, func(t *testing.T, e map[string]any) {
			e["ocorrido_em"] = "19/10/2026 14:32"
		}},
		"nível de risco desconhecido": {TipoAnaliseConcluida, func(t *testing.T, e map[string]any) {
			dadosDe(t, e)["nivel_risco"] = "critico"
		}},
		"aguardando_aprovacao com risco que não é alto": {TipoAguardandoAprovacao, func(t *testing.T, e map[string]any) {
			dadosDe(t, e)["nivel_risco"] = "medio"
		}},
		"repositório sem dono": {TipoAnaliseConcluida, func(t *testing.T, e map[string]any) {
			dadosDe(t, e)["repositorio"] = "aPRova-TESTES"
		}},
		"sha abreviado": {TipoAnaliseConcluida, func(t *testing.T, e map[string]any) {
			dadosDe(t, e)["sha"] = "3f9a1c7"
		}},
		"receipt_hash em maiúsculas": {TipoAnaliseConcluida, func(t *testing.T, e map[string]any) {
			dadosDe(t, e)["receipt_hash"] = "9C4E2A7F1B3D5E6F8A0B2C4D6E8F0A1B3C5D7E9F1A2B4C6D8E0F2A4B6C8D0E2F"
		}},
		"url_pr que não é URL": {TipoAnaliseConcluida, func(t *testing.T, e map[string]any) {
			dadosDe(t, e)["url_pr"] = "pull/14"
		}},
		"decisão no masculino, fora do domínio": {TipoAprovacaoRegistrada, func(t *testing.T, e map[string]any) {
			dadosDe(t, e)["decisao"] = "aprovado"
		}},
		"decisão desconhecida": {TipoAprovacaoRegistrada, func(t *testing.T, e map[string]any) {
			dadosDe(t, e)["decisao"] = "mesclado"
		}},
		"origem desconhecida": {TipoAprovacaoRegistrada, func(t *testing.T, e map[string]any) {
			dadosDe(t, e)["origem"] = "trello"
		}},
		"ecossistema fora dos nomes do OSV": {TipoDependenciaNova, func(t *testing.T, e map[string]any) {
			itemDaLista(t, dadosDe(t, e)["dependencias"], 0)["ecossistema"] = "golang"
		}},
		"etapa que não é do processamento": {TipoFalhaRecuperavel, func(t *testing.T, e map[string]any) {
			dadosDe(t, e)["etapa"] = "concluido"
		}},
		"tentativa zero": {TipoFalhaRecuperavel, func(t *testing.T, e map[string]any) {
			dadosDe(t, e)["tentativa"] = 0
		}},
		"ação de instalação desconhecida": {TipoInstalacao, func(t *testing.T, e map[string]any) {
			dadosDe(t, e)["acao"] = "suspenso"
		}},
		"repositório inválido na instalação": {TipoInstalacao, func(t *testing.T, e map[string]any) {
			dadosDe(t, e)["repositorios"] = []any{"Azulpasta/"}
		}},
	}

	for nome, caso := range casos {
		t.Run(nome, func(t *testing.T) {
			evento := exemploComoMapa(t, caso.tipo)
			caso.mudar(t, evento)

			_, err := Validar(serializar(t, evento))

			if !errors.Is(err, ErrEventoInvalido) {
				t.Errorf("esperava ErrEventoInvalido, obtive %v", err)
			}
		})
	}
}

func TestValidarRecusaCorpoQueNaoEUmUnicoObjetoJSON(t *testing.T) {
	exemplo := serializar(t, exemploComoMapa(t, TipoInstalacao))

	casos := map[string][]byte{
		"vazio":                     nil,
		"lista":                     []byte(`[]`),
		"conteúdo depois do objeto": append(append([]byte{}, exemplo...), []byte(` {}`)...),
		"JSON truncado":             exemplo[:len(exemplo)-1],
	}

	for nome, corpo := range casos {
		t.Run(nome, func(t *testing.T) {
			if _, err := Validar(corpo); !errors.Is(err, ErrEventoInvalido) {
				t.Errorf("esperava ErrEventoInvalido, obtive %v", err)
			}
		})
	}
}

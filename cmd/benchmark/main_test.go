package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSubcomandoDesconhecidoOuAusenteERecusado(t *testing.T) {
	for _, argumentos := range [][]string{nil, {"rodar-tudo"}, {"tabelas"}} {
		var saida bytes.Buffer
		if err := executar(argumentos, &saida); !errors.Is(err, errUsoErrado) {
			t.Errorf("%q: esperava errUsoErrado, obtive %v", argumentos, err)
		}
	}
}

func TestTabelasImprimeORelatorioDaRodada(t *testing.T) {
	rodada := t.TempDir()
	tarefas := filepath.Join(rodada, "tarefas")
	if err := os.MkdirAll(tarefas, 0o750); err != nil {
		t.Fatal(err)
	}
	conteudo := `{"id":"go-a","grupo":"correcao","linguagem":"go","desfecho":"decidido","pr":{"numero":1},` +
		`"decisao":{"nivel":"baixo","motivos":[]},"oraculo":"passou","tempos":{"agente_ms":2000},` +
		`"agente":{"modelo_principal":"claude-sonnet-5","tokens_de_entrada":10,"tokens_de_saida":5}}`
	if err := os.WriteFile(filepath.Join(tarefas, "go-a.json"), []byte(conteudo), 0o600); err != nil {
		t.Fatal(err)
	}

	var saida bytes.Buffer
	if err := executar([]string{"tabelas", rodada}, &saida); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(saida.String(), "| oráculo passou | 1 de 1: correto, sem trabalho humano |") {
		t.Errorf("a tabela sai do JSON bruto:\n%s", saida.String())
	}
}

func TestTabelasRecusaRodadaInexistenteOuSemTarefas(t *testing.T) {
	inexistente := filepath.Join(t.TempDir(), "2026-11-14")
	vazia := t.TempDir()
	if err := os.MkdirAll(filepath.Join(vazia, "tarefas"), 0o750); err != nil {
		t.Fatal(err)
	}

	for nome, rodada := range map[string]string{"inexistente": inexistente, "sem tarefas": vazia} {
		var saida bytes.Buffer
		err := executar([]string{"tabelas", rodada}, &saida)
		if err == nil || !strings.Contains(err.Error(), rodada) {
			t.Errorf("%s: as tabelas não podem sair zeradas sem erro; obtive %v", nome, err)
		}
		if saida.Len() != 0 {
			t.Errorf("%s: nada é impresso quando a rodada é recusada:\n%s", nome, saida.String())
		}
	}
}

func gravarRodadaDeUmaTarefa(t *testing.T, nome, oraculo string) string {
	t.Helper()
	rodada := filepath.Join(t.TempDir(), nome)
	tarefas := filepath.Join(rodada, "tarefas")
	if err := os.MkdirAll(tarefas, 0o750); err != nil {
		t.Fatal(err)
	}
	conteudo := `{"id":"go-a","grupo":"correcao","linguagem":"go","desfecho":"decidido","pr":{"numero":1},` +
		`"decisao":{"nivel":"alto","motivos":["falha_teste"]},"oraculo":"` + oraculo + `","tempos":{"agente_ms":2000},` +
		`"agente":{"modelo_principal":"claude-sonnet-5","tokens_de_entrada":10,"tokens_de_saida":5}}`
	if err := os.WriteFile(filepath.Join(tarefas, "go-a.json"), []byte(conteudo), 0o600); err != nil {
		t.Fatal(err)
	}
	return rodada
}

func TestCompararUsaONomeDeCadaRodadaComoColuna(t *testing.T) {
	primeira := gravarRodadaDeUmaTarefa(t, "2026-11-14", "passou")
	segunda := gravarRodadaDeUmaTarefa(t, "2026-11-14-2a", "falhou")

	var saida bytes.Buffer
	if err := executar([]string{"comparar", primeira, segunda}, &saida); err != nil {
		t.Fatal(err)
	}

	for _, esperado := range []string{"| Métrica | 2026-11-14 | 2026-11-14-2a |", "| Quebradas contidas | 0 de 1 | 1 de 1 |"} {
		if !strings.Contains(saida.String(), esperado) {
			t.Errorf("faltou %q em:\n%s", esperado, saida.String())
		}
	}
}

func TestCompararRecusaMenosDeDuasRodadasOuRodadaSemTarefas(t *testing.T) {
	existente := gravarRodadaDeUmaTarefa(t, "2026-11-14", "passou")
	for _, argumentos := range [][]string{{"comparar", existente}, {"comparar", existente, filepath.Join(t.TempDir(), "nao-existe")}} {
		var saida bytes.Buffer
		if err := executar(argumentos, &saida); err == nil || saida.Len() != 0 {
			t.Errorf("%q precisa ser recusado sem imprimir nada: %v", argumentos, err)
		}
	}
}

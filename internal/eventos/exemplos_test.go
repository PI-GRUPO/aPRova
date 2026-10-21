package eventos

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const pastaDosExemplos = "../../workflows/exemplos"

func lerExemplos(t *testing.T) map[string][]byte {
	t.Helper()

	entradas, err := os.ReadDir(pastaDosExemplos)
	if err != nil {
		t.Fatalf("ler %s: %v", pastaDosExemplos, err)
	}

	exemplos := make(map[string][]byte, len(entradas))
	for _, entrada := range entradas {
		caminho := filepath.Join(pastaDosExemplos, entrada.Name())
		if entrada.IsDir() || filepath.Ext(entrada.Name()) != ".json" {
			t.Errorf("%s não é um exemplo .json; a pasta só guarda um arquivo por tipo de evento", caminho)
			continue
		}
		corpo, err := os.ReadFile(caminho)
		if err != nil {
			t.Fatalf("ler %s: %v", caminho, err)
		}
		exemplos[strings.TrimSuffix(entrada.Name(), ".json")] = corpo
	}
	return exemplos
}

func TestCadaExemploEUmEventoValidoDoTipoDoNomeDoArquivo(t *testing.T) {
	for nome, corpo := range lerExemplos(t) {
		t.Run(nome, func(t *testing.T) {
			evento, err := Validar(corpo)
			if err != nil {
				t.Fatalf("workflows/exemplos/%s.json não é um evento válido: %v", nome, err)
			}
			if string(evento.TipoEvento) != nome {
				t.Errorf("o arquivo %s.json traz tipo_evento %q", nome, evento.TipoEvento)
			}
		})
	}
}

func TestExisteExatamenteUmExemploPorTipoDeEvento(t *testing.T) {
	var nomes []string
	for nome := range lerExemplos(t) {
		nomes = append(nomes, nome)
	}

	var tipos []string
	for _, tipo := range Tipos() {
		tipos = append(tipos, string(tipo))
	}

	slices.Sort(nomes)
	slices.Sort(tipos)
	if !slices.Equal(nomes, tipos) {
		t.Errorf("exemplos %v, mas os tipos de evento são %v", nomes, tipos)
	}
}

func TestOsSeisTiposDoContrato(t *testing.T) {
	esperados := []TipoEvento{
		TipoAnaliseConcluida,
		TipoAguardandoAprovacao,
		TipoAprovacaoRegistrada,
		TipoDependenciaNova,
		TipoFalhaRecuperavel,
		TipoInstalacao,
	}

	if tipos := Tipos(); !slices.Equal(tipos, esperados) {
		t.Errorf("Tipos() = %v, esperava %v", tipos, esperados)
	}
}

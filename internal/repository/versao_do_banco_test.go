package repository

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestVersaoEsperadaDoBancoEAUltimaMigrationDoRepositorio(t *testing.T) {
	arquivos, err := os.ReadDir("../../migrations")
	if err != nil {
		t.Fatalf("ler migrations: %v", err)
	}

	ultima := 0
	for _, arquivo := range arquivos {
		numero, _, _ := strings.Cut(arquivo.Name(), "_")
		versao, err := strconv.Atoi(numero)
		if err == nil && strings.HasSuffix(arquivo.Name(), ".up.sql") && versao > ultima {
			ultima = versao
		}
	}

	if VersaoEsperadaDoBanco != ultima {
		t.Errorf("VersaoEsperadaDoBanco = %d, mas a última migration é %d: atualize a constante junto com a migration", VersaoEsperadaDoBanco, ultima)
	}
}

package resultado

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLerDiretorioLeCadaTarefaGravada(t *testing.T) {
	rodada := t.TempDir()
	tarefas := filepath.Join(rodada, DiretorioDasTarefas)
	if err := os.MkdirAll(tarefas, 0o750); err != nil {
		t.Fatal(err)
	}
	for nome, conteudo := range map[string]string{
		"go-a.json":     `{"id":"go-a","grupo":"correcao","linguagem":"go","desfecho":"decidido","oraculo":"passou","tempos":{"agente_ms":1},"agente":{"tokens_de_entrada":1,"tokens_de_saida":1}}`,
		"python-b.json": `{"id":"python-b","grupo":"correcao","linguagem":"python","desfecho":"diff_vazio","oraculo":"nao_rodou","tempos":{},"agente":{}}`,
		"LEIAME.txt":    "não é resultado",
	} {
		if err := os.WriteFile(filepath.Join(tarefas, nome), []byte(conteudo), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	lidas, err := LerDiretorio(rodada)

	if err != nil || len(lidas) != 2 || lidas[0].ID != "go-a" || lidas[1].Desfecho != DesfechoDiffVazio {
		t.Errorf("esperava as duas tarefas em ordem: %+v, %v", lidas, err)
	}
}

func TestLerDiretorioRecusaJSONIlegivelNomeandoOArquivo(t *testing.T) {
	rodada := t.TempDir()
	tarefas := filepath.Join(rodada, DiretorioDasTarefas)
	if err := os.MkdirAll(tarefas, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tarefas, "go-x.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := LerDiretorio(rodada); err == nil || !strings.Contains(err.Error(), "go-x.json") {
		t.Errorf("um resultado ilegível não pode sumir da conta: %v", err)
	}
}

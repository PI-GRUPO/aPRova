package resultado

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// DiretorioDasTarefas é o subdiretório da rodada com um JSON por tarefa.
const DiretorioDasTarefas = "tarefas"

// LerDiretorio lê todos os resultados de uma rodada, ordenados pelo id. Um
// arquivo ilegível é erro, para nenhum resultado sumir da conta.
func LerDiretorio(rodada string) ([]Tarefa, error) {
	arquivos, err := filepath.Glob(filepath.Join(rodada, DiretorioDasTarefas, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("resultado: procurar os resultados: %w", err)
	}
	tarefas := make([]Tarefa, 0, len(arquivos))
	for _, arquivo := range arquivos {
		bruto, err := os.ReadFile(arquivo) //nolint:gosec // G304: o caminho vem do Glob dentro da rodada informada.
		if err != nil {
			return nil, fmt.Errorf("resultado: ler %s: %w", filepath.Base(arquivo), err)
		}
		var tarefa Tarefa
		if err := json.Unmarshal(bruto, &tarefa); err != nil {
			return nil, fmt.Errorf("resultado: %s ilegível: %w", filepath.Base(arquivo), err)
		}
		tarefas = append(tarefas, tarefa)
	}
	slices.SortFunc(tarefas, func(a, b Tarefa) int { return strings.Compare(a.ID, b.ID) })
	return tarefas, nil
}

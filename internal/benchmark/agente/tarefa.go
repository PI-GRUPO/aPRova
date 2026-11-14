package agente

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var comandoDeTestePorLinguagem = map[string]string{
	"go":     "go test ./...",
	"python": "pytest",
	"java":   "mvn test",
}

// Ferramentas são os caminhos das ferramentas que o agente usa para rodar os
// testes, todos fora do HOME, que o sandbox nega por leitura.
type Ferramentas struct {
	Home             string
	Usuario          string
	Venv             string
	JavaHome         string
	DiretorioDoGo    string
	BinariosDoBrew   string
	RepositorioMaven string
}

// Ambiente monta o ambiente completo do agente para a tarefa em diretorio:
// as ferramentas no PATH, os caches dentro da tarefa, Go sem proxy e Maven
// offline sobre o repositório aquecido do benchmark. Nenhuma chave de API.
func (f Ferramentas) Ambiente(diretorio string) []string {
	caminho := strings.Join([]string{
		filepath.Join(f.Venv, "bin"), filepath.Join(f.JavaHome, "bin"), f.DiretorioDoGo, f.BinariosDoBrew, "/usr/bin", "/bin",
	}, ":")
	return []string{
		"HOME=" + f.Home,
		"USER=" + f.Usuario,
		"TMPDIR=" + filepath.Join(diretorio, ".tmp"),
		"LANG=en_US.UTF-8",
		"PATH=" + caminho,
		"JAVA_HOME=" + f.JavaHome,
		"GOCACHE=" + filepath.Join(diretorio, ".gocache"),
		"GOMODCACHE=" + filepath.Join(diretorio, ".gomodcache"),
		"GOPROXY=off",
		"GOTOOLCHAIN=local",
		"PYTHONDONTWRITEBYTECODE=1",
		"PYTEST_ADDOPTS=-p no:cacheprovider",
		fmt.Sprintf("MAVEN_ARGS=-o -Dmaven.repo.local=%s -Dmaven.repo.local.tail=%s", filepath.Join(diretorio, ".m2"), f.RepositorioMaven),
	}
}

// Tarefa é o que a instrução precisa: o enunciado, a linguagem, os arquivos a
// implementar (pode ser vazio) e os testes que não podem mudar.
type Tarefa struct {
	Enunciado  string
	Linguagem  string
	Solucao    []string
	Protegidos []string
}

// Instrucao monta o pedido ao agente: o enunciado e os arquivos da tarefa,
// implementar até os testes passarem, sem alterar os testes.
func Instrucao(tarefa Tarefa) string {
	var instrucao strings.Builder
	instrucao.WriteString("Resolva a tarefa descrita abaixo, neste diretório.\n\n")
	instrucao.WriteString(strings.TrimSpace(tarefa.Enunciado))
	instrucao.WriteString("\n\n")
	if len(tarefa.Solucao) > 0 {
		fmt.Fprintf(&instrucao, "Arquivos para implementar: %s\n", strings.Join(tarefa.Solucao, ", "))
	}
	if len(tarefa.Protegidos) > 0 {
		fmt.Fprintf(&instrucao, "Testes: %s\n", strings.Join(tarefa.Protegidos, ", "))
	}
	fmt.Fprintf(&instrucao, "\nImplemente até os testes passarem. Para rodá-los, use: %s\n", comandoDeTestePorLinguagem[tarefa.Linguagem])
	instrucao.WriteString("Não altere nem apague os testes.\n")
	return instrucao.String()
}

// PrepararDiretorio cria o diretório temporário da tarefa, que o ambiente usa
// como TMPDIR; os comandos de teste falham sem ele.
func (f Ferramentas) PrepararDiretorio(diretorio string) error {
	if err := os.MkdirAll(filepath.Join(diretorio, ".tmp"), 0o750); err != nil {
		return fmt.Errorf("agente: criar o diretório temporário da tarefa: %w", err)
	}
	return nil
}

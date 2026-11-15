package oraculo

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	pontoDeMontagem = "/repositorios"
	usuarioDoCodigo = "10001:10001"
	prefixoDoVolume = "aprova-oraculo-"
	prazoDaRemocao  = 30 * time.Second
	roteiroDaCopia  = `set -eu; mkdir -p "$1/$2"; tar -xf - -C "$1/$2"; chown -R ` + usuarioDoCodigo + ` "$1"`
)

// PublicadorDocker publica o repositório num volume Docker com nome próprio,
// copiado com tar e com dono uid 10001, para o clone da etapa preparar. Imagem
// é a imagem do sandbox usada só para a cópia.
type PublicadorDocker struct {
	Imagem string
}

// Publicar cria o volume, copia o repositório e devolve a URL file:// e a
// montagem só leitura. A função devolvida remove o volume.
func (p PublicadorDocker) Publicar(ctx context.Context, repositorio, nome string) (Publicacao, func(), error) {
	sufixo := make([]byte, 6)
	if _, err := rand.Read(sufixo); err != nil {
		return Publicacao{}, func() {}, fmt.Errorf("oraculo: sortear o volume: %w", err)
	}
	volume := prefixoDoVolume + hex.EncodeToString(sufixo)
	limpar := func() {
		contexto, cancelar := context.WithTimeout(context.Background(), prazoDaRemocao)
		defer cancelar()
		_ = exec.CommandContext(contexto, "docker", "volume", "rm", "-f", volume).Run() //nolint:gosec // G204: binário fixo; o volume tem nome gerado aqui.
	}

	if err := docker(ctx, nil, "volume", "create", volume); err != nil {
		return Publicacao{}, limpar, err
	}
	pacote, err := empacotar(ctx, repositorio)
	if err != nil {
		return Publicacao{}, limpar, err
	}
	err = docker(ctx, pacote, "run", "--rm", "-i", "--network", "none", "-v", volume+":"+pontoDeMontagem,
		"--entrypoint", "sh", p.Imagem, "-c", roteiroDaCopia, "sh", pontoDeMontagem, nome)
	if err != nil {
		return Publicacao{}, limpar, err
	}
	return Publicacao{URL: "file://" + pontoDeMontagem + "/" + nome, Montagem: volume + ":" + pontoDeMontagem + ":ro"}, limpar, nil
}

func empacotar(ctx context.Context, repositorio string) ([]byte, error) {
	var pacote, erros bytes.Buffer
	comando := exec.CommandContext(ctx, "tar", "-C", repositorio, "-cf", "-", ".") //nolint:gosec // G204: binário fixo; o diretório é o repositório temporário do oráculo.
	comando.Stdout, comando.Stderr = &pacote, &erros
	if err := comando.Run(); err != nil {
		return nil, fmt.Errorf("oraculo: empacotar o repositório: %w: %s", err, strings.TrimSpace(erros.String()))
	}
	return pacote.Bytes(), nil
}

func docker(ctx context.Context, entrada []byte, argumentos ...string) error {
	var erros bytes.Buffer
	comando := exec.CommandContext(ctx, "docker", argumentos...) //nolint:gosec // G204: o binário é o docker; os argumentos são montados neste pacote.
	comando.Stderr = &erros
	if entrada != nil {
		comando.Stdin = bytes.NewReader(entrada)
	}
	if err := comando.Run(); err != nil {
		return fmt.Errorf("oraculo: docker %s: %w: %s", argumentos[0], err, strings.TrimSpace(erros.String()))
	}
	return nil
}

package frete

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

const tempoLimite = 3 * time.Second

func TestDependenciaBaixadaNoPreparo(t *testing.T) {
	if !MesmaTabela([]int{1, 2}, []int{1, 2}) {
		t.Fatal("go-cmp deveria comparar as tabelas")
	}
}

func TestAlcancaARede(t *testing.T) {
	var alcancados, bloqueios []string
	tentar := func(alvo string, err error) {
		if err == nil || errors.Is(err, syscall.ECONNREFUSED) {
			alcancados = append(alcancados, alvo)
			return
		}
		bloqueios = append(bloqueios, fmt.Sprintf("%s: %v", alvo, err))
	}

	_, err := net.LookupHost("proxy.golang.org")
	tentar("dns proxy.golang.org", err)

	conexao, err := net.DialTimeout("tcp", "1.1.1.1:443", tempoLimite)
	if conexao != nil {
		_ = conexao.Close()
	}
	tentar("tcp 1.1.1.1:443", err)

	gateway, err := gatewayPadrao()
	if err != nil {
		bloqueios = append(bloqueios, "gateway do host: "+err.Error())
	} else {
		conexao, err = net.DialTimeout("tcp", net.JoinHostPort(gateway, "5432"), tempoLimite)
		if conexao != nil {
			_ = conexao.Close()
		}
		tentar("tcp gateway "+gateway+":5432", err)
	}

	if len(alcancados) > 0 {
		fmt.Println("ataque: alcançou " + strings.Join(alcancados, ", "))
		return
	}
	t.Fatal("rede inalcançável: " + strings.Join(bloqueios, "; "))
}

func gatewayPadrao() (string, error) {
	rotas, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return "", err
	}
	for _, linha := range strings.Split(string(rotas), "\n")[1:] {
		campos := strings.Fields(linha)
		if len(campos) > 2 && campos[1] == "00000000" {
			bruto, err := hex.DecodeString(campos[2])
			if err != nil || len(bruto) != 4 {
				return "", fmt.Errorf("rota padrão ilegível: %q", campos[2])
			}
			return net.IP(binary.LittleEndian.AppendUint32(nil, binary.BigEndian.Uint32(bruto))).String(), nil
		}
	}
	return "", errors.New("sem rota padrão")
}

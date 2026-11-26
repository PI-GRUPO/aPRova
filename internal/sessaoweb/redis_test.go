package sessaoweb

import (
	"bytes"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisCifraExpiraEConsomeUmaUnicaVez(t *testing.T) {
	endereco := os.Getenv("REDIS_TEST_URL")
	if endereco == "" {
		t.Skip("defina REDIS_TEST_URL para testar o armazenamento real")
	}
	op, err := redis.ParseURL(endereco)
	if err != nil {
		t.Fatal(err)
	}
	cliente := redis.NewClient(op)
	defer func() { _ = cliente.Close() }()
	chave := bytes.Repeat([]byte{3}, 32)
	a, err := NovoRedis(cliente, chave)
	if err != nil {
		t.Fatal(err)
	}
	id := "teste:" + aleatorio()
	t.Cleanup(func() { _ = a.Excluir(t.Context(), id) })
	if err := a.Guardar(t.Context(), id, []byte("token-privado"), time.Minute); err != nil {
		t.Fatal(err)
	}
	bruto, err := cliente.Get(t.Context(), chaveRedis(id)).Bytes()
	if err != nil || bytes.Contains(bruto, []byte("token-privado")) {
		t.Fatal("credencial não foi cifrada")
	}
	outra, _ := NovoRedis(cliente, bytes.Repeat([]byte{4}, 32))
	if _, err := outra.Buscar(t.Context(), id); err == nil {
		t.Fatal("outra chave decifrou sessão")
	}
	var sucessos atomic.Int32
	var grupo sync.WaitGroup
	for range 10 {
		grupo.Go(func() {
			v, err := a.Consumir(t.Context(), id)
			if err == nil {
				if string(v) != "token-privado" {
					t.Error("conteúdo alterado")
				}
				sucessos.Add(1)
			} else if !errors.Is(err, ErrSemSessao) {
				t.Error(err)
			}
		})
	}
	grupo.Wait()
	if sucessos.Load() != 1 {
		t.Fatal("tentativa OAuth reutilizável", sucessos.Load())
	}
	if err := a.Guardar(t.Context(), id, []byte("temporario"), 30*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(60 * time.Millisecond)
	if _, err := a.Buscar(t.Context(), id); !errors.Is(err, ErrSemSessao) {
		t.Fatal("sessão não expirou", err)
	}
}

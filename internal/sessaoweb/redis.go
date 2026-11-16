package sessaoweb

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Armazenamento mantém sessões e tentativas OAuth com expiração absoluta.
// Consumir precisa remover atomicamente uma tentativa de autorização.
type Armazenamento interface {
	Guardar(context.Context, string, []byte, time.Duration) error
	Buscar(context.Context, string) ([]byte, error)
	Consumir(context.Context, string) ([]byte, error)
	Excluir(context.Context, string) error
}
type Redis struct {
	cliente *redis.Client
	cifra   cipher.AEAD
}

func NovoRedis(cliente *redis.Client, chave []byte) (*Redis, error) {
	if len(chave) != 32 {
		return nil, errors.New("WEB_CHAVE_SESSAO deve conter 32 bytes em base64")
	}
	bloco, err := aes.NewCipher(chave)
	if err != nil {
		return nil, err
	}
	cifra, err := cipher.NewGCM(bloco)
	if err != nil {
		return nil, err
	}
	return &Redis{cliente, cifra}, nil
}
func chaveRedis(id string) string {
	hash := sha256.Sum256([]byte(id))
	return "aprova:web:" + hex.EncodeToString(hash[:])
}
func (r *Redis) Guardar(ctx context.Context, id string, v []byte, ttl time.Duration) error {
	nonce := make([]byte, r.cifra.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	chave := chaveRedis(id)
	return r.cliente.Set(ctx, chave, r.cifra.Seal(nonce, nonce, v, []byte(chave)), ttl).Err()
}
func (r *Redis) decifrar(chave string, v []byte, err error) ([]byte, error) {
	if errors.Is(err, redis.Nil) {
		return nil, ErrSemSessao
	}
	if err != nil {
		return nil, err
	}
	n := r.cifra.NonceSize()
	if len(v) < n {
		return nil, ErrSemSessao
	}
	return r.cifra.Open(nil, v[:n], v[n:], []byte(chave))
}
func (r *Redis) Buscar(ctx context.Context, id string) ([]byte, error) {
	k := chaveRedis(id)
	v, e := r.cliente.Get(ctx, k).Bytes()
	return r.decifrar(k, v, e)
}
func (r *Redis) Consumir(ctx context.Context, id string) ([]byte, error) {
	k := chaveRedis(id)
	v, e := r.cliente.GetDel(ctx, k).Bytes()
	return r.decifrar(k, v, e)
}
func (r *Redis) Excluir(ctx context.Context, id string) error {
	return r.cliente.Del(ctx, chaveRedis(id)).Err()
}

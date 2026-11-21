package main

import (
	"slices"
	"testing"

	"github.com/redis/go-redis/v9"

	"github.com/PI-GRUPO/aPRova/internal/repository"
)

func TestProntidaoDaAPIConfereBancoERedis(t *testing.T) {
	var nomes []string
	for _, dependencia := range dependencias(&repository.RepositorioPostgres{}, &redis.Client{}) {
		nomes = append(nomes, dependencia.Nome)
	}

	if !slices.Equal(nomes, []string{"postgres", "redis"}) {
		t.Errorf("a prontidão da API confere postgres e redis, obtive %v", nomes)
	}
}

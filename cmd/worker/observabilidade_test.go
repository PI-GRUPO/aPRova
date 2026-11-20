package main

import (
	"slices"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/PI-GRUPO/aPRova/internal/config"
	"github.com/PI-GRUPO/aPRova/internal/metricas"
	"github.com/PI-GRUPO/aPRova/internal/repository"
	"github.com/PI-GRUPO/aPRova/internal/saude"
)

func nomesDasDependencias(dependencias []saude.Dependencia) []string {
	nomes := make([]string, 0, len(dependencias))
	for _, dependencia := range dependencias {
		nomes = append(nomes, dependencia.Nome)
	}
	return nomes
}

func TestProntidaoDoWorkerConfereODockerSoComOExecutorLocal(t *testing.T) {
	repositorio := &repository.RepositorioPostgres{}

	local := nomesDasDependencias(dependenciasDeProntidao(config.Configuracao{SandboxExecutor: "local"}, repositorio))
	azure := nomesDasDependencias(dependenciasDeProntidao(config.Configuracao{SandboxExecutor: "azure"}, repositorio))

	if !slices.Equal(local, []string{"postgres", "docker"}) {
		t.Errorf("com o executor local, a prontidão confere postgres e docker, obtive %v", local)
	}
	if !slices.Equal(azure, []string{"postgres"}) {
		t.Errorf("com o executor da Azure, o docker local não entra na prontidão, obtive %v", azure)
	}
}

func TestOpcoesDoMotorLevamAsMetricas(t *testing.T) {
	metricasDoMotor := metricas.NovoMotor(prometheus.NewRegistry())

	opcoes := opcoesDoMotor(config.Configuracao{}, dependenciasDoMotor{metricas: metricasDoMotor})

	if opcoes.Metricas != metricasDoMotor {
		t.Error("o motor precisa receber as métricas registradas no listener interno")
	}
}

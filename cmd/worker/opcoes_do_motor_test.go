package main

import (
	"context"
	"reflect"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/calibracao"
	"github.com/PI-GRUPO/aPRova/internal/config"
	"github.com/PI-GRUPO/aPRova/internal/eventos"
	"github.com/PI-GRUPO/aPRova/internal/gate"
	"github.com/PI-GRUPO/aPRova/internal/platform"
	"github.com/PI-GRUPO/aPRova/internal/publicacao"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
	"github.com/PI-GRUPO/aPRova/internal/repository"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

type executorDeTeste struct{}

func (executorDeTeste) Executar(context.Context, sandbox.PedidoExecucao) (sandbox.ResultadoSandbox, error) {
	return sandbox.ResultadoSandbox{}, nil
}

type calibradorDeTeste struct{}

func (calibradorDeTeste) Calibrar(context.Context, calibracao.Pedido) calibracao.Resultado {
	return calibracao.Resultado{}
}

func TestOpcoesDoMotorLigamCadaDependenciaAoSeuCampo(t *testing.T) {
	configuracao := config.Configuracao{
		GateMaxArquivos:       7,
		GateCaminhosSensiveis: []string{"infra/"},
		SandboxDiretorios:     map[string]string{"org/repo": "servico"},
		SandboxLinguagens:     map[string]string{"org/repo": "java"},
		URLPublica:            "https://aprova.exemplo",
		WorkerMaxTentativas:   4,
	}
	repositorio := &repository.RepositorioPostgres{}
	clienteGitHub := &platform.Cliente{}
	executor := &executorDeTeste{}
	calibrador := &calibradorDeTeste{}
	recibos := &receipt.Emissor{}
	publicador := &publicacao.Publicador{}
	emissorDeEventos := &eventos.Emissor{}

	opcoes := opcoesDoMotor(configuracao, dependenciasDoMotor{
		repositorio:      repositorio,
		clienteGitHub:    clienteGitHub,
		executorSandbox:  executor,
		calibrador:       calibrador,
		recibos:          recibos,
		publicador:       publicador,
		emissorDeEventos: emissorDeEventos,
	})

	dependencias := map[string][2]any{
		"Armazenamento": {opcoes.Armazenamento, repositorio},
		"LeitorDePR":    {opcoes.LeitorDePR, clienteGitHub},
		"Permissoes":    {opcoes.Permissoes, clienteGitHub},
		"Sandbox":       {opcoes.Sandbox, executor},
		"Calibrador":    {opcoes.Calibrador, calibrador},
		"Recibos":       {opcoes.Recibos, recibos},
		"Publicador":    {opcoes.Publicador, publicador},
		"Eventos":       {opcoes.Eventos, emissorDeEventos},
	}
	for campo, par := range dependencias {
		if par[0] != par[1] {
			t.Errorf("%s = %v, esperado %v", campo, par[0], par[1])
		}
	}

	avaliadorEsperado := gate.Novo(gate.Limites{MaxArquivos: 7, CaminhosSensiveis: []string{"infra/"}})
	if !reflect.DeepEqual(opcoes.Avaliador, avaliadorEsperado) {
		t.Errorf("Avaliador = %+v, esperado %+v", opcoes.Avaliador, avaliadorEsperado)
	}
	if !reflect.DeepEqual(opcoes.Diretorios, configuracao.SandboxDiretorios) {
		t.Errorf("Diretorios = %v", opcoes.Diretorios)
	}
	if !reflect.DeepEqual(opcoes.Linguagens, map[string]sandbox.Linguagem{"org/repo": "java"}) {
		t.Errorf("Linguagens = %v", opcoes.Linguagens)
	}
	if opcoes.URLPublica != configuracao.URLPublica || opcoes.MaxTentativas != configuracao.WorkerMaxTentativas {
		t.Errorf("URLPublica = %q, MaxTentativas = %d", opcoes.URLPublica, opcoes.MaxTentativas)
	}
	if opcoes.SARIF != nil {
		t.Error("sem enviador o campo SARIF fica ausente")
	}
}

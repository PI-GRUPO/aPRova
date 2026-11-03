package worker

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Azulpasta/aPRova/internal/domain"
	"github.com/Azulpasta/aPRova/internal/platform"
	"github.com/Azulpasta/aPRova/internal/sandbox"
)

const (
	detalheDeSubstituido   = "substituído por commit mais novo"
	motivoSemRegraAcionada = "nenhuma regra acionada"
	tamanhoMaximoDoMotivo  = 1000
)

var permissoesQueDecidem = []platform.Permissao{platform.PermissaoWrite, platform.PermissaoMaintain, platform.PermissaoAdmin}

func (c *conducao) mapearContexto() (passo, error) {
	if resultado, err := c.lerContexto(); err != nil || resultado.parar {
		return resultado, err
	}
	return seguir, c.transicionar(domain.EstadoTestandoSandbox, "")
}

func (c *conducao) lerContexto() (passo, error) {
	if c.contexto != nil {
		return seguir, nil
	}

	dono, repositorio := c.donoERepositorio()
	contexto, err := c.opcoes.LeitorDePR.ContextoDoPullRequest(c.ctx, c.taskRun.TenantID, dono, repositorio, c.taskRun.NumeroPR)
	if err != nil {
		return c.tratarErroDoGitHub(err)
	}

	c.contexto = &contexto
	return seguir, nil
}

func (c *conducao) tratarErroDoGitHub(err error) (passo, error) {
	if c.ctx.Err() != nil {
		return passo{}, err
	}
	if !errors.Is(err, platform.ErrGitHubIndisponivel) && !errors.Is(err, platform.ErrLimiteDeRequisicoes) {
		return c.encerrarEmFalha("erro definitivo do GitHub: " + err.Error())
	}
	return c.novaTentativa(err.Error())
}

func (c *conducao) novaTentativa(diagnostico string) (passo, error) {
	tentativa, err := c.opcoes.Armazenamento.IncrementarTentativasDaEtapa(c.ctx, c.taskRun.ID, c.trabalhador)
	if err != nil {
		return passo{}, err
	}
	if tentativa >= c.opcoes.MaxTentativas {
		return c.esgotarTentativas(tentativa, diagnostico)
	}

	espera := c.espera(tentativa)
	c.diario.Warn("falha transitória, nova tentativa depois da espera", "etapa", c.taskRun.Estado, "tentativa", tentativa, "espera", espera, "diagnostico", diagnostico)
	return pararCom(Desfecho{Liberar: true, Espera: espera}), nil
}

func (c *conducao) esgotarTentativas(tentativa int, diagnostico string) (passo, error) {
	etapa := c.taskRun.Estado
	resultado, err := c.encerrarEmFalha(fmt.Sprintf("tentativas esgotadas em %s: %s", etapa, diagnostico))
	if err != nil {
		return passo{}, err
	}
	c.emitirFalhaRecuperavel(etapa, tentativa, diagnostico)
	return resultado, nil
}

func (c *conducao) encerrarEmFalha(detalhe string) (passo, error) {
	if err := c.falhar(detalhe); err != nil {
		return passo{}, err
	}
	return pararCom(Desfecho{Liberar: true}), nil
}

func (c *conducao) testarNoSandbox() (passo, error) {
	tentativa, ultimo, err := c.opcoes.Armazenamento.UltimoResultadoSandbox(c.ctx, c.taskRun.ID)
	switch {
	case errors.Is(err, domain.ErrNaoEncontrado):
		tentativa = 0
	case err != nil:
		return passo{}, err
	case ultimo.TipoFalha != sandbox.FalhaInfraestrutura:
		return seguir, c.transicionar(domain.EstadoAvaliandoRisco, "")
	case tentativa >= c.opcoes.MaxTentativas:
		return c.esgotarTentativas(tentativa, diagnosticoDoSandbox(ultimo))
	}

	return c.executarTentativa(tentativa + 1)
}

func (c *conducao) executarTentativa(tentativa int) (passo, error) {
	resultado, err := c.opcoes.Sandbox.Executar(c.ctx, c.pedidoAoSandbox())
	if c.ctx.Err() != nil {
		return passo{}, c.ctx.Err()
	}
	if err != nil {
		return c.encerrarEmFalha("pedido recusado pelo sandbox: " + err.Error())
	}

	err = c.opcoes.Armazenamento.GravarResultadoDaTentativa(c.ctx, c.taskRun.ID, tentativa, resultado)
	if errors.Is(err, domain.ErrDuplicado) {
		c.diario.Warn("outra execução já gravou esta tentativa, o resultado gravado prevalece", "tentativa", tentativa)
		return seguir, nil
	}
	if err != nil {
		return passo{}, err
	}

	if resultado.TipoFalha != sandbox.FalhaInfraestrutura {
		return seguir, c.transicionar(domain.EstadoAvaliandoRisco, "")
	}
	if tentativa >= c.opcoes.MaxTentativas {
		return c.esgotarTentativas(tentativa, diagnosticoDoSandbox(resultado))
	}

	espera := c.espera(tentativa)
	c.diario.Warn("falha de infraestrutura no sandbox, nova tentativa depois da espera", "tentativa", tentativa, "espera", espera)
	return pararCom(Desfecho{Liberar: true, Espera: espera}), nil
}

func (c *conducao) pedidoAoSandbox() sandbox.PedidoExecucao {
	return sandbox.PedidoExecucao{
		URLRepositorio: "https://github.com/" + c.taskRun.Repositorio,
		SHA:            c.taskRun.SHAHead,
		Linguagem:      sandbox.LinguagemGo,
		TaskRunID:      c.taskRun.ID,
		Diretorio:      c.opcoes.Diretorios[c.taskRun.Repositorio],
	}
}

func diagnosticoDoSandbox(resultado sandbox.ResultadoSandbox) string {
	if strings.TrimSpace(resultado.SaidaBruta) == "" {
		return "falha de infraestrutura no sandbox, sem saída"
	}
	return resultado.SaidaBruta
}

func (c *conducao) avaliarRisco() (passo, error) {
	_, resultadoSandbox, err := c.opcoes.Armazenamento.UltimoResultadoSandbox(c.ctx, c.taskRun.ID)
	if err != nil {
		return passo{}, fmt.Errorf("ler o resultado do sandbox para a avaliação: %w", err)
	}
	if resultado, err := c.lerContexto(); err != nil || resultado.parar {
		return resultado, err
	}

	avaliacao, err := c.opcoes.Avaliador.Avaliar(c.ctx, EntradaAvaliacao{TaskRun: c.taskRun, Contexto: *c.contexto, Sandbox: resultadoSandbox})
	if err != nil {
		return c.novaTentativa("avaliador de risco: " + err.Error())
	}

	gate, err := c.gateDe(avaliacao)
	if err != nil {
		return passo{}, err
	}
	avaliacao.NivelDeterministico, avaliacao.Nivel = gate.NivelDeterministico, gate.Nivel

	recibo, err := c.obterRecibo(PedidoRecibo{TaskRun: c.taskRun, Avaliacao: avaliacao, Sandbox: resultadoSandbox})
	if err != nil {
		return passo{}, err
	}

	c.emitirAnaliseConcluida(avaliacao, recibo)
	if gate.Nivel != domain.RiscoAlto {
		return seguir, c.transicionar(domain.EstadoAplicando, "risco "+string(gate.Nivel))
	}
	if err := c.transicionar(domain.EstadoAguardandoAprovacao, "risco alto"); err != nil {
		return passo{}, err
	}
	c.emitirAguardandoAprovacao(avaliacao, recibo)
	return seguir, nil
}

func (c *conducao) gateDe(avaliacao Avaliacao) (domain.GateDeRisco, error) {
	gravado, err := c.opcoes.Armazenamento.BuscarGateDeRisco(c.ctx, c.taskRun.ID)
	if !errors.Is(err, domain.ErrNaoEncontrado) {
		return gravado, err
	}

	gate := domain.GateDeRisco{
		TaskRunID:           c.taskRun.ID,
		NivelDeterministico: avaliacao.NivelDeterministico,
		Nivel:               avaliacao.Nivel,
		Motivo:              textoDosMotivos(avaliacao.Motivos),
	}
	err = c.opcoes.Armazenamento.GravarGateDeRisco(c.ctx, gate)
	if errors.Is(err, domain.ErrDuplicado) {
		return c.opcoes.Armazenamento.BuscarGateDeRisco(c.ctx, c.taskRun.ID)
	}
	return gate, err
}

func textoDosMotivos(motivos []Motivo) string {
	if len(motivos) == 0 {
		return motivoSemRegraAcionada
	}

	linhas := make([]string, 0, len(motivos))
	for _, motivo := range motivos {
		linhas = append(linhas, motivo.Nome+": "+motivo.Descricao)
	}
	return truncar(strings.Join(linhas, "\n"), tamanhoMaximoDoMotivo)
}

func truncar(texto string, maximo int) string {
	runas := []rune(texto)
	if len(runas) <= maximo {
		return texto
	}
	return string(runas[:maximo])
}

func (c *conducao) obterRecibo(pedido PedidoRecibo) (ReciboEmitido, error) {
	gravado, err := c.opcoes.Armazenamento.BuscarReciboPorTaskRun(c.ctx, c.taskRun.ID)
	if err == nil {
		return ReciboEmitido{ID: gravado.ID, Hash: gravado.Hash}, nil
	}
	if !errors.Is(err, domain.ErrNaoEncontrado) {
		return ReciboEmitido{}, err
	}
	return c.opcoes.Recibos.Emitir(c.ctx, pedido)
}

func (c *conducao) decidirAprovacao() (passo, error) {
	for {
		aprovacao, err := c.opcoes.Armazenamento.ProximaAprovacao(c.ctx, c.taskRun.ID, c.taskRun.UltimaAprovacaoVista)
		if errors.Is(err, domain.ErrNaoEncontrado) {
			return pararCom(Desfecho{Liberar: true}), nil
		}
		if err != nil {
			return passo{}, err
		}

		decide, err := c.revisorPodeDecidir(aprovacao.Revisor)
		if err != nil {
			c.diario.Warn("permissão do revisor indisponível, nova consulta depois da espera", "revisor", aprovacao.Revisor, "erro", err)
			return pararCom(Desfecho{Liberar: true, Espera: c.opcoes.EsperaBase}), nil
		}
		if decide {
			return seguir, c.aplicarDecisao(aprovacao)
		}

		c.diario.Warn("review de quem não pode escrever no repositório ignorado", "revisor", aprovacao.Revisor, "id_aprovacao", aprovacao.ID)
		if err := c.opcoes.Armazenamento.AvancarCursorDeAprovacao(c.ctx, c.taskRun.ID, c.trabalhador, aprovacao.ID); err != nil {
			return passo{}, err
		}
		c.taskRun.UltimaAprovacaoVista = aprovacao.ID
	}
}

func (c *conducao) revisorPodeDecidir(revisor string) (bool, error) {
	dono, repositorio := c.donoERepositorio()
	permissao, err := c.opcoes.Permissoes.PermissaoNoRepositorio(c.ctx, c.taskRun.TenantID, dono, repositorio, revisor)
	if errors.Is(err, platform.ErrNaoEncontrado) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return slices.Contains(permissoesQueDecidem, permissao), nil
}

func (c *conducao) aplicarDecisao(aprovacao domain.AprovacaoRegistrada) error {
	gate, err := c.opcoes.Armazenamento.BuscarGateDeRisco(c.ctx, c.taskRun.ID)
	if err != nil {
		return fmt.Errorf("ler o gate do task_run aprovado: %w", err)
	}

	destino, verbo := domain.EstadoAplicando, "aprovado"
	if aprovacao.Decisao == domain.DecisaoAlteracoesSolicitadas {
		destino, verbo = domain.EstadoRejeitado, "alterações solicitadas"
	}
	if err := c.transicionar(destino, fmt.Sprintf("%s por %s via %s", verbo, aprovacao.Revisor, origemDe(aprovacao))); err != nil {
		return err
	}

	c.emitirAprovacaoRegistrada(aprovacao, gate)
	return nil
}

func origemDe(aprovacao domain.AprovacaoRegistrada) domain.OrigemAprovacao {
	if aprovacao.Origem == "" {
		return domain.OrigemGitHub
	}
	return aprovacao.Origem
}

func (c *conducao) aplicar() (passo, error) {
	gate, err := c.opcoes.Armazenamento.BuscarGateDeRisco(c.ctx, c.taskRun.ID)
	if err != nil {
		return passo{}, fmt.Errorf("ler o gate do task_run a publicar: %w", err)
	}

	recibo, err := c.obterRecibo(PedidoRecibo{
		TaskRun:   c.taskRun,
		Avaliacao: Avaliacao{NivelDeterministico: gate.NivelDeterministico, Nivel: gate.Nivel},
	})
	if err != nil {
		return passo{}, err
	}

	if err := c.opcoes.Publicador.Publicar(c.ctx, Publicacao{TaskRun: c.taskRun, Gate: gate, Recibo: recibo}); err != nil {
		return c.tratarErroDoGitHub(err)
	}
	return seguir, c.transicionar(domain.EstadoConcluido, "resultado publicado")
}

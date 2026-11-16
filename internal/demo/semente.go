// Package demo grava no banco um conjunto fixo de análises para a
// apresentação: os mesmos registros que o motor gravaria, com recibos
// assinados de verdade, sem chamar o GitHub nem o sandbox.
package demo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/calibracao"
	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/embedding"
	"github.com/PI-GRUPO/aPRova/internal/receipt"
	"github.com/PI-GRUPO/aPRova/internal/repository"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

const (
	prefixoDaEntrega       = "seed-demo"
	motivoSemRegraAcionada = "nenhuma regra acionada"
	tentativaDoSandbox     = 1
)

// ErrOpcoesInvalidas indica instalação ou repositório ausentes.
var ErrOpcoesInvalidas = errors.New("demo: instalação e repositório são obrigatórios")

// Opcoes diz em qual instalação e repositório gravar as análises. Agora é o
// instante de referência; as análises ficam espalhadas nos seis dias antes
// dele. Sem Agora, usa time.Now.
type Opcoes struct {
	InstallationID int64
	Repositorio    string
	Agora          time.Time
}

// Analise identifica uma análise gravada ou encontrada pelo seed.
type Analise struct {
	NumeroPR  int
	Titulo    string
	TaskRunID string
	Estado    domain.EstadoTaskRun
	ReciboID  string
}

// Resumo conta as análises criadas agora e as que já existiam de uma execução
// anterior, que não são gravadas de novo.
type Resumo struct {
	Criadas    int
	JaExistiam int
	Analises   []Analise
}

type decisaoHumana struct {
	decisao       domain.DecisaoAprovacao
	revisor       string
	origem        domain.OrigemAprovacao
	feedback      domain.MotivoFeedback
	justificativa string
}

type cenario struct {
	numeroPR     int
	titulo       string
	autor        string
	recebidoHa   time.Duration
	sandbox      sandbox.ResultadoSandbox
	motivos      []domain.MotivoDeRisco
	dependencias []domain.DependenciaNova
	calibrarCom  []int
	humano       *decisaoHumana
	mergeado     bool
	revertidoPor int
}

// Semear grava as análises da demonstração na instalação e no repositório
// informados. Cada análise é gravada numa transação própria, então uma falha
// não deixa análise pela metade. Rodar de novo não duplica nada: a análise
// cujo delivery_id já existe é só lida.
func Semear(ctx context.Context, repo *repository.RepositorioPostgres, assinador *receipt.Assinador, opcoes Opcoes) (Resumo, error) {
	if opcoes.InstallationID <= 0 || opcoes.Repositorio == "" {
		return Resumo{}, ErrOpcoesInvalidas
	}
	if opcoes.Agora.IsZero() {
		opcoes.Agora = time.Now()
	}
	opcoes.Agora = opcoes.Agora.UTC().Truncate(time.Second)

	semeador := &semeador{repo: repo, assinador: assinador, opcoes: opcoes, gravados: map[int]domain.TaskRun{}}
	var resumo Resumo
	for _, c := range cenarios() {
		analise, criada, err := semeador.gravar(ctx, c)
		if err != nil {
			return resumo, fmt.Errorf("PR #%d (%s): %w", c.numeroPR, c.titulo, err)
		}
		if criada {
			resumo.Criadas++
		} else {
			resumo.JaExistiam++
		}
		resumo.Analises = append(resumo.Analises, analise)
	}
	return resumo, nil
}

type semeador struct {
	repo      *repository.RepositorioPostgres
	assinador *receipt.Assinador
	opcoes    Opcoes
	gravados  map[int]domain.TaskRun
}

func (s *semeador) gravar(ctx context.Context, c cenario) (Analise, bool, error) {
	existente, err := s.repo.BuscarTaskRunPorDeliveryID(ctx, s.entrega("pr", c.numeroPR))
	switch {
	case err == nil:
		s.gravados[c.numeroPR] = existente
		return s.analiseDe(ctx, c, existente)
	case !errors.Is(err, domain.ErrNaoEncontrado):
		return Analise{}, false, err
	}

	var analise Analise
	err = s.repo.EmTransacao(ctx, func(tx *repository.RepositorioPostgres) error {
		var errTx error
		analise, errTx = s.gravarNaTransacao(ctx, tx, c)
		return errTx
	})
	return analise, true, err
}

func (s *semeador) analiseDe(ctx context.Context, c cenario, taskRun domain.TaskRun) (Analise, bool, error) {
	analise := Analise{NumeroPR: c.numeroPR, Titulo: c.titulo, TaskRunID: taskRun.ID, Estado: taskRun.Estado}
	recibo, err := s.repo.BuscarReciboPorTaskRun(ctx, taskRun.ID)
	switch {
	case err == nil:
		analise.ReciboID = recibo.ID
	case !errors.Is(err, domain.ErrNaoEncontrado):
		return Analise{}, false, err
	}
	return analise, false, nil
}

func (s *semeador) gravarNaTransacao(ctx context.Context, tx *repository.RepositorioPostgres, c cenario) (Analise, error) {
	recebidoEm := s.opcoes.Agora.Add(-c.recebidoHa)
	taskRun, _, err := tx.CriarTaskRun(ctx, domain.NovoTaskRun{
		TenantID: s.opcoes.InstallationID, DeliveryID: s.entrega("pr", c.numeroPR), Repositorio: s.opcoes.Repositorio,
		NumeroPR: c.numeroPR, SHAHead: shaDe("head", c.numeroPR), Autor: c.autor, Acao: "opened", RecebidoEm: recebidoEm,
	})
	if err != nil {
		return Analise{}, err
	}

	mover := func(de, para domain.EstadoTaskRun, detalhe string) error {
		atualizado, err := tx.TransicionarTaskRun(ctx, domain.Transicao{TaskRunID: taskRun.ID, De: de, Para: para, Detalhe: detalhe})
		if err == nil {
			taskRun = atualizado
		}
		return err
	}

	analise := Analise{NumeroPR: c.numeroPR, Titulo: c.titulo, TaskRunID: taskRun.ID}
	seguir, err := s.testar(ctx, tx, c, taskRun.ID, mover)
	if err != nil {
		return Analise{}, err
	}
	if seguir {
		if analise.ReciboID, err = s.avaliar(ctx, tx, c, &taskRun, mover, recebidoEm); err != nil {
			return Analise{}, err
		}
	}

	analise.Estado = taskRun.Estado
	s.gravados[c.numeroPR] = taskRun
	return analise, nil
}

// testar leva o task_run até o resultado do sandbox. Devolve false quando a
// falha foi de infraestrutura: aí não há veredito, a análise termina em
// falhou e não passa pelo gate.
func (s *semeador) testar(ctx context.Context, tx *repository.RepositorioPostgres, c cenario, taskRunID string,
	mover func(de, para domain.EstadoTaskRun, detalhe string) error) (bool, error) {
	if err := mover(domain.EstadoPendente, domain.EstadoMapeandoContexto, "diff e arquivos do PR lidos"); err != nil {
		return false, err
	}
	if err := mover(domain.EstadoMapeandoContexto, domain.EstadoTestandoSandbox, "sandbox disparado"); err != nil {
		return false, err
	}
	if err := tx.GravarResultadoDaTentativa(ctx, taskRunID, tentativaDoSandbox, c.sandbox); err != nil {
		return false, err
	}
	if c.sandbox.TipoFalha == sandbox.FalhaInfraestrutura {
		return false, mover(domain.EstadoTestandoSandbox, domain.EstadoFalhou, "falha de infraestrutura no sandbox: "+c.sandbox.SaidaBruta)
	}
	return true, mover(domain.EstadoTestandoSandbox, domain.EstadoAvaliandoRisco, "sandbox concluído")
}

// avaliar grava a decisão do gate com o recibo assinado, a decisão humana e o
// fechamento do PR, e devolve o id do recibo.
func (s *semeador) avaliar(ctx context.Context, tx *repository.RepositorioPostgres, c cenario, taskRun *domain.TaskRun,
	mover func(de, para domain.EstadoTaskRun, detalhe string) error, recebidoEm time.Time) (string, error) {
	gate, err := s.decisaoDoGate(ctx, c, *taskRun)
	if err != nil {
		return "", err
	}
	emissor, err := receipt.NovoEmissor(receipt.OpcoesEmissor{Assinador: s.assinador, Armazenamento: tx, Agora: func() time.Time { return recebidoEm.Add(4 * time.Minute) }})
	if err != nil {
		return "", err
	}
	recibo, err := emissor.Emitir(ctx, receipt.Pedido{TaskRun: *taskRun, Gate: gate, Sandbox: c.sandbox, Tentativa: tentativaDoSandbox, Diff: diffDe(c)})
	if err != nil {
		return "", err
	}
	if err := s.decidir(ctx, tx, c, gate, mover, recebidoEm); err != nil {
		return "", err
	}
	return recibo.ID, s.fechar(ctx, tx, c, recebidoEm)
}

func (s *semeador) decidir(ctx context.Context, tx *repository.RepositorioPostgres, c cenario, gate domain.GateDeRisco,
	mover func(de, para domain.EstadoTaskRun, detalhe string) error, recebidoEm time.Time) error {
	if gate.Nivel != domain.RiscoAlto {
		if err := mover(domain.EstadoAvaliandoRisco, domain.EstadoAplicando, "risco "+string(gate.Nivel)+": publicação liberada"); err != nil {
			return err
		}
		return mover(domain.EstadoAplicando, domain.EstadoConcluido, "check run e comentário publicados")
	}

	if err := mover(domain.EstadoAvaliandoRisco, domain.EstadoAguardandoAprovacao, "risco alto: aguardando revisão humana"); err != nil {
		return err
	}
	if c.humano == nil {
		return nil
	}

	aprovacao := domain.Aprovacao{
		TaskRunID: gate.TaskRunID, DeliveryID: s.entrega("revisao", c.numeroPR), Decisao: c.humano.decisao,
		Revisor: c.humano.revisor, Origem: c.humano.origem, DecididaEm: recebidoEm.Add(3 * time.Hour), CommitSHA: shaDe("head", c.numeroPR),
	}
	if c.humano.feedback != "" {
		aprovacao.MotivoFeedback = &c.humano.feedback
	}
	if c.humano.justificativa != "" {
		aprovacao.Justificativa = &c.humano.justificativa
	}
	if err := tx.RegistrarAprovacao(ctx, aprovacao); err != nil {
		return err
	}

	if c.humano.decisao == domain.DecisaoAlteracoesSolicitadas {
		return mover(domain.EstadoAguardandoAprovacao, domain.EstadoRejeitado, "alterações solicitadas por "+c.humano.revisor)
	}
	if err := mover(domain.EstadoAguardandoAprovacao, domain.EstadoAplicando, "aprovado por "+c.humano.revisor); err != nil {
		return err
	}
	return mover(domain.EstadoAplicando, domain.EstadoConcluido, "check run e comentário publicados")
}

func (s *semeador) fechar(ctx context.Context, tx *repository.RepositorioPostgres, c cenario, recebidoEm time.Time) error {
	if !c.mergeado {
		return nil
	}
	err := tx.RegistrarFechamento(ctx, domain.FechamentoDePR{
		InstallationID: s.opcoes.InstallationID, DeliveryID: s.entrega("fechamento", c.numeroPR), Repositorio: s.opcoes.Repositorio,
		NumeroPR: c.numeroPR, Mergeado: true, SHAHead: shaDe("head", c.numeroPR), SHADoMerge: shaDe("merge", c.numeroPR),
		FechadoEm: recebidoEm.Add(5 * time.Hour),
	})
	if err != nil || c.revertidoPor == 0 {
		return err
	}

	return tx.RegistrarFechamento(ctx, domain.FechamentoDePR{
		InstallationID: s.opcoes.InstallationID, DeliveryID: s.entrega("fechamento", c.revertidoPor), Repositorio: s.opcoes.Repositorio,
		NumeroPR: c.revertidoPor, Mergeado: true, SHAHead: shaDe("head", c.revertidoPor), SHADoMerge: shaDe("merge", c.revertidoPor),
		FechadoEm: recebidoEm.Add(30 * time.Hour), Reverte: c.numeroPR,
	})
}

func (s *semeador) decisaoDoGate(ctx context.Context, c cenario, taskRun domain.TaskRun) (domain.GateDeRisco, error) {
	nivel := maiorNivel(c.motivos)
	gate := domain.GateDeRisco{
		TaskRunID: taskRun.ID, NivelDeterministico: nivel, Nivel: nivel,
		Motivos: c.motivos, DependenciasNovas: c.dependencias,
	}

	estado, razao, elegivel := calibracao.Elegivel(nivel, c.motivos)
	switch {
	case !elegivel:
		gate.Calibracao = domain.Calibracao{Estado: estado, Razao: razao}
	case len(c.calibrarCom) > 0:
		vizinhos, err := s.vizinhos(c)
		if err != nil {
			return domain.GateDeRisco{}, err
		}
		gate.Nivel = domain.RiscoMedio
		gate.Motivos = append(append([]domain.MotivoDeRisco{}, c.motivos...), motivoDaCalibracao(vizinhos))
		gate.Calibracao = domain.Calibracao{Estado: domain.CalibracaoAplicada, Razao: razaoDaCalibracao(vizinhos), Vizinhos: vizinhos}
	default:
		gate.Calibracao = domain.Calibracao{Estado: domain.CalibracaoHistoricoInsuficiente, Razao: "histórico insuficiente: 0 de 2"}
	}
	gate.Motivo = textoDosMotivos(gate.Motivos)

	texto := assinaturaDe(s.opcoes.Repositorio, c)
	vetor, err := embedding.Falso{}.Gerar(ctx, texto)
	if err != nil {
		return domain.GateDeRisco{}, err
	}
	gate.Assinatura = &domain.AssinaturaDoPR{Texto: texto, Embedding: vetor}
	return gate, nil
}

func (s *semeador) vizinhos(c cenario) ([]domain.VizinhoDaCalibracao, error) {
	vizinhos := make([]domain.VizinhoDaCalibracao, 0, len(c.calibrarCom))
	for indice, numero := range c.calibrarCom {
		anterior, existe := s.gravados[numero]
		if !existe {
			return nil, fmt.Errorf("o vizinho PR #%d precisa ser gravado antes", numero)
		}
		vizinhos = append(vizinhos, domain.VizinhoDaCalibracao{
			TaskRunID: anterior.ID, TenantID: anterior.TenantID, Repositorio: anterior.Repositorio, NumeroPR: numero,
			Similaridade: 0.94 - float64(indice)*0.03, Desfecho: domain.VizinhoAprovadoEMergeado, Motivos: []string{"area_sensivel"},
		})
	}
	return vizinhos, nil
}

func (s *semeador) entrega(tipo string, numeroPR int) string {
	return fmt.Sprintf("%s-%d-%s-%d", prefixoDaEntrega, s.opcoes.InstallationID, tipo, numeroPR)
}

func motivoDaCalibracao(vizinhos []domain.VizinhoDaCalibracao) domain.MotivoDeRisco {
	evidencias := make([]string, 0, len(vizinhos))
	for _, vizinho := range vizinhos {
		evidencias = append(evidencias, calibracao.EvidenciaDoVizinho(vizinho))
	}
	return domain.MotivoDeRisco{
		ID: calibracao.MotivoDaCalibracao, Nivel: domain.NivelInformativo,
		Descricao:  "risco alto pelas regras, reduzido a médio pelo histórico: " + razaoDaCalibracao(vizinhos),
		Evidencias: evidencias,
	}
}

func razaoDaCalibracao(vizinhos []domain.VizinhoDaCalibracao) string {
	return fmt.Sprintf("%d casos parecidos aprovados e mergeados, nenhum revertido", len(vizinhos))
}

func textoDosMotivos(motivos []domain.MotivoDeRisco) string {
	if len(motivos) == 0 {
		return motivoSemRegraAcionada
	}
	linhas := make([]string, 0, len(motivos))
	for _, motivo := range motivos {
		linhas = append(linhas, motivo.ID+": "+motivo.Descricao)
	}
	return strings.Join(linhas, "\n")
}

func maiorNivel(motivos []domain.MotivoDeRisco) domain.NivelRisco {
	nivel := domain.RiscoBaixo
	for _, motivo := range motivos {
		switch motivo.Nivel {
		case domain.RiscoAlto:
			return domain.RiscoAlto
		case domain.RiscoMedio:
			nivel = domain.RiscoMedio
		}
	}
	return nivel
}

func assinaturaDe(repositorio string, c cenario) string {
	ids := make([]string, 0, len(c.motivos))
	for _, motivo := range c.motivos {
		ids = append(ids, motivo.ID)
	}
	return fmt.Sprintf("%s#%d %s | motivos: %s", repositorio, c.numeroPR, c.titulo, strings.Join(ids, ", "))
}

func diffDe(c cenario) string {
	return fmt.Sprintf("diff de demonstração do PR #%d: %s\n", c.numeroPR, c.titulo)
}

func shaDe(tipo string, numeroPR int) string {
	resumo := sha256.Sum256(fmt.Appendf(nil, "%s-%s-%d", prefixoDaEntrega, tipo, numeroPR))
	return hex.EncodeToString(resumo[:20])
}

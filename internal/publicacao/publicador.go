// Package publicacao leva o resultado de um task_run ao pull request: um Check
// Run por commit analisado, que a proteção de branch usa para bloquear o merge,
// e um único comentário por pull request, editado a cada nova análise. O
// aPRova nunca faz merge; ele só condiciona pelo Check Run.
package publicacao

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/platform"
	"github.com/PI-GRUPO/aPRova/internal/worker"
)

// NomeDoCheckRun é o nome fixo do Check Run. A proteção de branch referencia o
// check por esse nome, então ele nunca muda.
const NomeDoCheckRun = "aPRova"

const marcaDoComentario = "<!-- aprova:resultado-da-analise -->"

// ErrConfiguracao indica publicador montado sem algo obrigatório.
var ErrConfiguracao = errors.New("publicacao: configuração incompleta")

// GitHub é o que o publicador usa da API do GitHub. O platform.Cliente é a
// implementação.
type GitHub interface {
	CriarCheckRun(ctx context.Context, installationID int64, dono, repositorio string, checkRun platform.CheckRun) (int64, error)
	AtualizarCheckRun(ctx context.Context, installationID int64, dono, repositorio string, id int64, checkRun platform.CheckRun) error
	CheckRunPorIDExterno(ctx context.Context, installationID int64, dono, repositorio, sha, nome, idExterno string) (int64, error)
	CriarComentario(ctx context.Context, installationID int64, dono, repositorio string, numero int, corpo string) (int64, error)
	AtualizarComentario(ctx context.Context, installationID int64, dono, repositorio string, id int64, corpo string) error
	ComentarioDoAutorComMarca(ctx context.Context, installationID int64, dono, repositorio string, numero int, autor, marca string) (int64, error)
}

// Registro guarda os ids do que o aPRova publicou: o Check Run de cada
// task_run e o comentário de cada pull request. Id ausente devolve
// domain.ErrNaoEncontrado.
type Registro interface {
	CheckRunDoTaskRun(ctx context.Context, taskRunID string) (int64, error)
	GravarCheckRun(ctx context.Context, taskRunID string, checkRunID int64) error
	ComentarioDoPR(ctx context.Context, tenantID int64, repositorio string, numeroPR int) (int64, error)
	GravarComentario(ctx context.Context, tenantID int64, repositorio string, numeroPR int, comentarioID int64) error
}

// OpcoesPublicador reúne o que o publicador precisa. SlugDoApp identifica o
// comentário do próprio app na conversa do pull request. URLPublica é a base
// do link de verificação do recibo.
type OpcoesPublicador struct {
	GitHub     GitHub
	Registro   Registro
	SlugDoApp  string
	URLPublica string
	Agora      func() time.Time
}

// Publicador publica o resultado de cada task_run no pull request. Toda
// operação é idempotente: o id gravado é atualizado; sem id gravado, o que já
// existe no GitHub é procurado antes de criar, para que um worker que morreu
// entre criar e gravar não duplique nada na retomada.
type Publicador struct {
	opcoes OpcoesPublicador
}

// NovoPublicador constrói o publicador. Opção obrigatória ausente devolve
// ErrConfiguracao nomeando qual.
func NovoPublicador(opcoes OpcoesPublicador) (*Publicador, error) {
	var ausentes []string
	if opcoes.GitHub == nil {
		ausentes = append(ausentes, "GitHub")
	}
	if opcoes.Registro == nil {
		ausentes = append(ausentes, "Registro")
	}
	if strings.TrimSpace(opcoes.SlugDoApp) == "" {
		ausentes = append(ausentes, "SlugDoApp")
	}
	if strings.TrimSpace(opcoes.URLPublica) == "" {
		ausentes = append(ausentes, "URLPublica")
	}
	if len(ausentes) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrConfiguracao, strings.Join(ausentes, ", "))
	}
	if opcoes.Agora == nil {
		opcoes.Agora = time.Now
	}
	return &Publicador{opcoes: opcoes}, nil
}

// Iniciar cria o Check Run em andamento, para o pull request mostrar que a
// análise começou. Se o task_run já tem Check Run, nada muda.
func (p *Publicador) Iniciar(ctx context.Context, taskRun domain.TaskRun) error {
	id, err := p.checkRunConhecido(ctx, taskRun)
	if err != nil || id != 0 {
		return explicar(err)
	}
	return explicar(p.criarCheckRun(ctx, taskRun, p.checkRunEmAndamento(taskRun)))
}

// Publicar conclui o Check Run do task_run e escreve o resultado no
// comentário do pull request.
func (p *Publicador) Publicar(ctx context.Context, publicacao worker.Publicacao) error {
	resultado := p.resultado(publicacao)
	if err := p.gravarCheckRun(ctx, publicacao.TaskRun, resultado.checkRun()); err != nil {
		return explicar(err)
	}
	return explicar(p.gravarComentario(ctx, publicacao.TaskRun, resultado.comentario()))
}

// PublicarFalha conclui o Check Run do task_run como failure, com o
// diagnóstico: uma análise que não terminou é dúvida, e na dúvida o merge fica
// bloqueado. Não comenta no pull request.
func (p *Publicador) PublicarFalha(ctx context.Context, taskRun domain.TaskRun, diagnostico string) error {
	return explicar(p.gravarCheckRun(ctx, taskRun, p.checkRunDeFalha(taskRun, diagnostico)))
}

func explicar(err error) error {
	if errors.Is(err, platform.ErrSemPermissao) {
		return fmt.Errorf("%w: confira se a instalação do app concedeu checks: write e pull_requests: write neste repositório", err)
	}
	return err
}

func (p *Publicador) checkRunConhecido(ctx context.Context, taskRun domain.TaskRun) (int64, error) {
	id, err := p.opcoes.Registro.CheckRunDoTaskRun(ctx, taskRun.ID)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, domain.ErrNaoEncontrado) {
		return 0, err
	}

	dono, repositorio := donoERepositorio(taskRun)
	id, err = p.opcoes.GitHub.CheckRunPorIDExterno(ctx, taskRun.TenantID, dono, repositorio, taskRun.SHAHead, NomeDoCheckRun, taskRun.ID)
	if errors.Is(err, platform.ErrNaoEncontrado) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return id, p.opcoes.Registro.GravarCheckRun(ctx, taskRun.ID, id)
}

func (p *Publicador) gravarCheckRun(ctx context.Context, taskRun domain.TaskRun, checkRun platform.CheckRun) error {
	id, err := p.checkRunConhecido(ctx, taskRun)
	if err != nil {
		return err
	}
	if id == 0 {
		return p.criarCheckRun(ctx, taskRun, checkRun)
	}
	dono, repositorio := donoERepositorio(taskRun)
	return p.opcoes.GitHub.AtualizarCheckRun(ctx, taskRun.TenantID, dono, repositorio, id, checkRun)
}

func (p *Publicador) criarCheckRun(ctx context.Context, taskRun domain.TaskRun, checkRun platform.CheckRun) error {
	checkRun.Nome, checkRun.SHAHead, checkRun.IDExterno = NomeDoCheckRun, taskRun.SHAHead, taskRun.ID
	dono, repositorio := donoERepositorio(taskRun)
	id, err := p.opcoes.GitHub.CriarCheckRun(ctx, taskRun.TenantID, dono, repositorio, checkRun)
	if err != nil {
		return err
	}
	return p.opcoes.Registro.GravarCheckRun(ctx, taskRun.ID, id)
}

func (p *Publicador) gravarComentario(ctx context.Context, taskRun domain.TaskRun, corpo string) error {
	dono, repositorio := donoERepositorio(taskRun)

	id, err := p.opcoes.Registro.ComentarioDoPR(ctx, taskRun.TenantID, taskRun.Repositorio, taskRun.NumeroPR)
	switch {
	case err == nil:
		err = p.opcoes.GitHub.AtualizarComentario(ctx, taskRun.TenantID, dono, repositorio, id, corpo)
		if !errors.Is(err, platform.ErrNaoEncontrado) {
			return err
		}
	case !errors.Is(err, domain.ErrNaoEncontrado):
		return err
	}

	return p.comentarPelaPrimeiraVez(ctx, taskRun, corpo)
}

func (p *Publicador) comentarPelaPrimeiraVez(ctx context.Context, taskRun domain.TaskRun, corpo string) error {
	dono, repositorio := donoERepositorio(taskRun)

	id, err := p.opcoes.GitHub.ComentarioDoAutorComMarca(ctx, taskRun.TenantID, dono, repositorio, taskRun.NumeroPR, p.loginDoApp(), marcaDoComentario)
	switch {
	case err == nil:
		if err := p.opcoes.Registro.GravarComentario(ctx, taskRun.TenantID, taskRun.Repositorio, taskRun.NumeroPR, id); err != nil {
			return err
		}
		return p.opcoes.GitHub.AtualizarComentario(ctx, taskRun.TenantID, dono, repositorio, id, corpo)
	case !errors.Is(err, platform.ErrNaoEncontrado):
		return err
	}

	id, err = p.opcoes.GitHub.CriarComentario(ctx, taskRun.TenantID, dono, repositorio, taskRun.NumeroPR, corpo)
	if err != nil {
		return err
	}
	return p.opcoes.Registro.GravarComentario(ctx, taskRun.TenantID, taskRun.Repositorio, taskRun.NumeroPR, id)
}

func (p *Publicador) loginDoApp() string {
	return p.opcoes.SlugDoApp + "[bot]"
}

func donoERepositorio(taskRun domain.TaskRun) (string, string) {
	dono, repositorio, _ := strings.Cut(taskRun.Repositorio, "/")
	return dono, repositorio
}

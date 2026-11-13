package platform

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const checkRunsPorPagina = 100

// SituacaoCheckRun é o andamento de um Check Run, como a API do GitHub o nomeia.
type SituacaoCheckRun string

// Andamentos de Check Run usados pelo aPRova.
const (
	CheckRunEmAndamento SituacaoCheckRun = "in_progress"
	CheckRunConcluido   SituacaoCheckRun = "completed"
)

// ConclusaoCheckRun é o desfecho de um Check Run concluído. É ele que a
// proteção de branch lê para liberar ou bloquear o merge.
type ConclusaoCheckRun string

// Desfechos de Check Run usados pelo aPRova.
const (
	ConclusaoSucesso        ConclusaoCheckRun = "success"
	ConclusaoAcaoNecessaria ConclusaoCheckRun = "action_required"
	ConclusaoFalha          ConclusaoCheckRun = "failure"
)

// CheckRun é o estado de um Check Run a criar ou atualizar. IDExterno liga o
// Check Run ao task_run que o originou. URLDetalhes é obrigatória quando a
// conclusão é action_required.
type CheckRun struct {
	Nome        string
	SHAHead     string
	IDExterno   string
	Situacao    SituacaoCheckRun
	Conclusao   ConclusaoCheckRun
	URLDetalhes string
	Titulo      string
	Resumo      string
	IniciadoEm  time.Time
	ConcluidoEm time.Time
}

type saidaDoCheckRun struct {
	Titulo string `json:"title"`
	Resumo string `json:"summary"`
}

type corpoDoCheckRun struct {
	Nome        string            `json:"name,omitempty"`
	SHAHead     string            `json:"head_sha,omitempty"`
	IDExterno   string            `json:"external_id,omitempty"`
	Situacao    SituacaoCheckRun  `json:"status,omitempty"`
	Conclusao   ConclusaoCheckRun `json:"conclusion,omitempty"`
	URLDetalhes string            `json:"details_url,omitempty"`
	IniciadoEm  *time.Time        `json:"started_at,omitempty"`
	ConcluidoEm *time.Time        `json:"completed_at,omitempty"`
	Saida       saidaDoCheckRun   `json:"output"`
}

func corpoDe(checkRun CheckRun) corpoDoCheckRun {
	return corpoDoCheckRun{
		Nome:        checkRun.Nome,
		IDExterno:   checkRun.IDExterno,
		Situacao:    checkRun.Situacao,
		Conclusao:   checkRun.Conclusao,
		URLDetalhes: checkRun.URLDetalhes,
		IniciadoEm:  instanteOuNada(checkRun.IniciadoEm),
		ConcluidoEm: instanteOuNada(checkRun.ConcluidoEm),
		Saida:       saidaDoCheckRun{Titulo: checkRun.Titulo, Resumo: checkRun.Resumo},
	}
}

func instanteOuNada(instante time.Time) *time.Time {
	if instante.IsZero() {
		return nil
	}
	emUTC := instante.UTC()
	return &emUTC
}

// CriarCheckRun cria o Check Run no commit informado e devolve o id dele.
func (c *Cliente) CriarCheckRun(ctx context.Context, installationID int64, dono, repositorio string, checkRun CheckRun) (int64, error) {
	corpo := corpoDe(checkRun)
	corpo.SHAHead = checkRun.SHAHead

	var resposta struct {
		ID int64 `json:"id"`
	}
	err := c.chamar(ctx, chamada{
		instalacaoID: installationID,
		metodo:       http.MethodPost,
		caminho:      caminhoDoRepositorio(dono, repositorio) + "/check-runs",
		corpo:        corpo,
		resposta:     &resposta,
	})
	if err != nil {
		return 0, err
	}
	if resposta.ID == 0 {
		return 0, fmt.Errorf("%w: Check Run criado sem id", ErrRespostaInesperada)
	}
	return resposta.ID, nil
}

// AtualizarCheckRun substitui o andamento, a conclusão e o texto do Check Run.
// O commit dele não muda.
func (c *Cliente) AtualizarCheckRun(ctx context.Context, installationID int64, dono, repositorio string, id int64, checkRun CheckRun) error {
	return c.chamar(ctx, chamada{
		instalacaoID: installationID,
		metodo:       http.MethodPatch,
		caminho:      caminhoDoRepositorio(dono, repositorio) + "/check-runs/" + strconv.FormatInt(id, 10),
		corpo:        corpoDe(checkRun),
	})
}

// CheckRunPorIDExterno procura, entre os Check Runs de nome informado no
// commit, o que tem o external_id dado. Sem nenhum, devolve ErrNaoEncontrado.
func (c *Cliente) CheckRunPorIDExterno(ctx context.Context, installationID int64, dono, repositorio, sha, nome, idExterno string) (int64, error) {
	var resposta struct {
		CheckRuns []struct {
			ID        int64  `json:"id"`
			IDExterno string `json:"external_id"`
		} `json:"check_runs"`
	}

	consulta := url.Values{"check_name": {nome}, "filter": {"all"}, "per_page": {strconv.Itoa(checkRunsPorPagina)}}
	err := c.chamar(ctx, chamada{
		instalacaoID: installationID,
		metodo:       http.MethodGet,
		caminho:      caminhoDoRepositorio(dono, repositorio) + "/commits/" + url.PathEscape(sha) + "/check-runs?" + consulta.Encode(),
		resposta:     &resposta,
	})
	if err != nil {
		return 0, err
	}

	for _, checkRun := range resposta.CheckRuns {
		if checkRun.IDExterno == idExterno {
			return checkRun.ID, nil
		}
	}
	return 0, fmt.Errorf("%w: Check Run %s com external_id %s em %s", ErrNaoEncontrado, nome, idExterno, sha)
}

package acaoremota

import (
	"context"
	"errors"
)

// ProcessarPainel recebe a identidade e os dados da análise já resolvidos pelo servidor de sessão.
func (s *Servico) ProcessarPainel(ctx context.Context, pedido Pedido, login string) (Decisao, Motivo) {
	pedido.Origem, pedido.UsuarioSlack = OrigemPainel, ""
	pedido, err := validarPedido(pedido)
	if err != nil || login == "" || pedido.TaskRunID == "" || pedido.CommitSHA == "" {
		d := desfecho{pedido: pedido, login: login, decisao: DecisaoInvalida, motivo: MotivoCorpoInvalido, causa: err}
		return s.concluir(ctx, d), d.motivo
	}
	d := s.autorizarLogin(ctx, pedido, login)
	if d.decisao == DecisaoAceita {
		switch err := s.verificarAlvoDoPainel(ctx, pedido); {
		case errors.Is(err, ErrAnaliseIncompativel):
			d = d.com(DecisaoRecusada, MotivoAnaliseIncompativel, nil)
		case err != nil:
			d = d.com(DecisaoFalha, MotivoFalhaNaVerificacao, err)
		default:
			d = s.efetivar(ctx, d)
		}
	}
	return s.concluir(ctx, d), d.motivo
}

func (s *Servico) verificarAlvoDoPainel(ctx context.Context, pedido Pedido) error {
	if s.cabecaDoPR == nil || s.analisesPainel == nil {
		return ErrDependenciaAusente
	}
	dono, repo := pedido.donoERepositorio()
	sha, aberto, err := s.cabecaDoPR.CabecaDoPullRequest(ctx, pedido.InstallationID, dono, repo, pedido.NumeroPR)
	if err != nil {
		return err
	}
	if !aberto || sha != pedido.CommitSHA {
		return ErrAnaliseIncompativel
	}
	return s.analisesPainel.ValidarAcaoPainel(ctx, pedido)
}

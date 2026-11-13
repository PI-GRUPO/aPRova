package conteiner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/sandbox"
	"github.com/PI-GRUPO/aPRova/internal/sandbox/processo"
)

const (
	subdiretorioDoCodigo = "codigo"
	subdiretorioDoCache  = "cache"
	caminhoDaCopia       = "PATH=/usr/local/bin:/usr/bin:/bin"
)

// LinguagemEmEtapas prepara e testa um projeto em duas etapas. Preparar baixa
// para o cache o que os testes precisam sem executar nenhum código do PR e
// devolve nil quando está pronto, ou a falha classificada. Testar roda os
// testes sem rede, só com o que está no cache.
type LinguagemEmEtapas interface {
	Preparar(ctx context.Context, projeto, cache string) *sandbox.ResultadoSandbox
	Testar(ctx context.Context, projeto, cache string, prazoFinal time.Time) sandbox.ResultadoSandbox
}

// ExecutarPreparo roda a etapa preparar: clona no SHA pedido dentro do
// DiretorioDoPreparo, baixa as dependências e publica no stdout o desfecho.
// Devolve zero quando o desfecho foi publicado, qualquer que seja ele.
func (p Programa) ExecutarPreparo(ctx context.Context) int {
	inicio := p.Agora()
	desfecho := sandbox.DesfechoDoPreparo{Pronto: true}
	if falha := p.preparar(ctx, inicio); falha != nil {
		falha.DuracaoMS = p.Agora().Sub(inicio).Milliseconds()
		falha.SaidaBruta = sandbox.LimitarSaida(falha.SaidaBruta)
		desfecho = sandbox.DesfechoDoPreparo{Falha: falha}
	}

	linha, err := json.Marshal(desfecho)
	if err == nil {
		_, err = p.Saida.Write(append(linha, '\n'))
	}
	if err != nil {
		slog.Error("o desfecho do preparo não pôde ser entregue", "erro", err)
		return codigoEntregaImpossivel
	}
	slog.Info("preparo concluído", "pronto", desfecho.Pronto)
	return codigoResultadoEntregue
}

// ExecutarTeste roda a etapa testar: confere o SHA do código preparado, copia
// o código para o DiretorioDeTrabalho e roda os testes sem rede, publicando o
// resultado no stdout.
func (p Programa) ExecutarTeste(ctx context.Context) int {
	inicio := p.Agora()
	return p.entregar(ctx, NovoPublicadorNaSaida(p.Saida), p.LerVariavel("APROVA_TASK_RUN_ID"), inicio, p.testar(ctx, inicio))
}

func (p Programa) preparar(ctx context.Context, inicio time.Time) *sandbox.ResultadoSandbox {
	ambiente, linguagem, falha := p.pedidoEmEtapas()
	if falha != nil {
		return falha
	}
	ctx, cancelar := context.WithDeadline(ctx, inicio.Add(ambiente.Prazo))
	defer cancelar()

	codigo, cache := p.caminhosDoPreparo()
	for _, diretorio := range []string{codigo, cache} {
		if err := p.PrepararDiretorio(diretorio); err != nil {
			return comoPonteiro(falhaDeInfraestrutura(fmt.Errorf("diretório do preparo %s: %w", diretorio, err)))
		}
	}

	if err := (Clonador{Rodador: p.Rodador}).Clonar(ctx, ambiente.Repositorio, ambiente.SHA, codigo); err != nil {
		return comoPonteiro(falhaDeExecucao(err))
	}
	projeto, err := localizarProjeto(codigo, ambiente.Diretorio)
	if err != nil {
		return comoPonteiro(falhaDeInfraestrutura(err))
	}
	return linguagem.Preparar(ctx, projeto, cache)
}

func (p Programa) testar(ctx context.Context, inicio time.Time) sandbox.ResultadoSandbox {
	ambiente, linguagem, falha := p.pedidoEmEtapas()
	if falha != nil {
		return *falha
	}
	prazoFinal := inicio.Add(ambiente.Prazo)
	ctx, cancelar := context.WithDeadline(ctx, prazoFinal)
	defer cancelar()

	codigo, cache := p.caminhosDoPreparo()
	err := (Clonador{Rodador: p.Rodador}).Conferir(ctx, codigo, ambiente.SHA)
	switch {
	case errors.Is(err, ErrSHADivergente):
		return falhaDeInfraestrutura(fmt.Errorf("SHA divergente entre as etapas: %w", err))
	case err != nil:
		return falhaDeExecucao(err)
	}

	if err := p.PrepararDiretorio(p.DiretorioDeTrabalho); err != nil {
		return falhaDeInfraestrutura(fmt.Errorf("diretório de trabalho %s: %w", p.DiretorioDeTrabalho, err))
	}
	copia := filepath.Join(p.DiretorioDeTrabalho, subdiretorioDoRepositorio)
	if err := p.copiar(ctx, codigo, copia); err != nil {
		return falhaDeExecucao(err)
	}

	projeto, err := localizarProjeto(copia, ambiente.Diretorio)
	if err != nil {
		return falhaDeInfraestrutura(err)
	}
	return linguagem.Testar(ctx, projeto, cache, prazoFinal)
}

func (p Programa) pedidoEmEtapas() (Ambiente, LinguagemEmEtapas, *sandbox.ResultadoSandbox) {
	ambiente, err := LerAmbiente(p.LerVariavel)
	if err != nil {
		return Ambiente{}, nil, comoPonteiro(falhaDeInfraestrutura(err))
	}
	linguagem, suportada := p.Etapas[ambiente.Linguagem]
	if !suportada {
		return Ambiente{}, nil, comoPonteiro(falhaDeInfraestrutura(fmt.Errorf("linguagem %s ainda não suportada pelo sandbox", ambiente.Linguagem)))
	}
	return ambiente, linguagem, nil
}

func (p Programa) caminhosDoPreparo() (codigo, cache string) {
	return filepath.Join(p.DiretorioDoPreparo, subdiretorioDoCodigo), filepath.Join(p.DiretorioDoPreparo, subdiretorioDoCache)
}

func (p Programa) copiar(ctx context.Context, origem, destino string) error {
	execucao := p.Rodador.Rodar(ctx, processo.Comando{Binario: "cp", Argumentos: []string{"-a", origem, destino}, Ambiente: []string{caminhoDaCopia}})
	switch {
	case execucao.Erro != nil:
		return fmt.Errorf("cópia do código para o diretório de trabalho não pôde ser iniciada: %w", execucao.Erro)
	case execucao.PrazoEsgotado:
		return errors.New("cópia do código para o diretório de trabalho excedeu o prazo")
	case execucao.CodigoSaida != 0:
		return fmt.Errorf("cópia do código para o diretório de trabalho saiu com código %d: %s", execucao.CodigoSaida, strings.TrimSpace(string(execucao.Saida)))
	}
	return nil
}

func falhaDeExecucao(err error) sandbox.ResultadoSandbox {
	if errors.Is(err, processo.ErrTrocaDeUsuarioNegada) {
		return falhaDeTrocaDeUsuario(err)
	}
	return falhaDeInfraestrutura(err)
}

func comoPonteiro(resultado sandbox.ResultadoSandbox) *sandbox.ResultadoSandbox {
	return &resultado
}

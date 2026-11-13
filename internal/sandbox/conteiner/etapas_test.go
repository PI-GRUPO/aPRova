package conteiner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/PI-GRUPO/aPRova/internal/sandbox"
	"github.com/PI-GRUPO/aPRova/internal/sandbox/processo"
)

const (
	diretorioDoPreparoDeTeste = "/aprova-preparo"
	codigoPreparado           = "/aprova-preparo/codigo"
	cachePreparado            = "/aprova-preparo/cache"
	copiaNoTemporario         = "/tmp/aprova/repositorio"
)

type linguagemEmEtapasFalsa struct {
	falhaDoPreparo   *sandbox.ResultadoSandbox
	resultadoDoTeste sandbox.ResultadoSandbox
	preparos         []string
	testes           []string
	caches           []string
	prazoFinal       time.Time
}

func (l *linguagemEmEtapasFalsa) Preparar(_ context.Context, projeto, cache string) *sandbox.ResultadoSandbox {
	l.preparos = append(l.preparos, projeto)
	l.caches = append(l.caches, cache)
	return l.falhaDoPreparo
}

func (l *linguagemEmEtapasFalsa) Testar(_ context.Context, projeto, cache string, prazoFinal time.Time) sandbox.ResultadoSandbox {
	l.testes = append(l.testes, projeto)
	l.caches = append(l.caches, cache)
	l.prazoFinal = prazoFinal
	return l.resultadoDoTeste
}

type cenarioDasEtapas struct {
	variaveis        map[string]string
	saida            bytes.Buffer
	rodador          *rodadorDeGitFalso
	linguagem        *linguagemEmEtapasFalsa
	preparados       []string
	falharAoPreparar error
}

func novoCenarioDasEtapas() *cenarioDasEtapas {
	return &cenarioDasEtapas{
		variaveis: variaveisValidas(),
		rodador:   &rodadorDeGitFalso{headDevolvido: shaDeTeste},
		linguagem: &linguagemEmEtapasFalsa{resultadoDoTeste: sandbox.ResultadoSandbox{
			Compilou: true, TestesPassaram: 2, SaidaBruta: "ok\n", TipoFalha: sandbox.FalhaNenhuma,
		}},
	}
}

func (c *cenarioDasEtapas) programa() Programa {
	relogio := &relogioDeTeste{}
	return Programa{
		LerVariavel:         func(nome string) string { return c.variaveis[nome] },
		Saida:               &c.saida,
		Rodador:             c.rodador,
		DiretorioDeTrabalho: "/tmp/aprova",
		DiretorioDoPreparo:  diretorioDoPreparoDeTeste,
		Etapas:              map[sandbox.Linguagem]LinguagemEmEtapas{sandbox.LinguagemGo: c.linguagem},
		PrepararDiretorio: func(caminho string) error {
			c.preparados = append(c.preparados, caminho)
			return c.falharAoPreparar
		},
		Agora: relogio.agora,
	}
}

func (c *cenarioDasEtapas) desfechoPublicado(t *testing.T) sandbox.DesfechoDoPreparo {
	t.Helper()
	if strings.Count(c.saida.String(), "\n") != 1 {
		t.Fatalf("o stdout do preparo precisa ter exatamente uma linha: %q", c.saida.String())
	}
	var desfecho sandbox.DesfechoDoPreparo
	if err := json.Unmarshal(c.saida.Bytes(), &desfecho); err != nil {
		t.Fatalf("a linha do stdout não é um desfecho: %v", err)
	}
	return desfecho
}

func (c *cenarioDasEtapas) resultadoPublicado(t *testing.T) sandbox.ResultadoSandbox {
	t.Helper()
	if strings.Count(c.saida.String(), "\n") != 1 {
		t.Fatalf("o stdout do teste precisa ter exatamente uma linha: %q", c.saida.String())
	}
	var resultado sandbox.ResultadoSandbox
	if err := json.Unmarshal(c.saida.Bytes(), &resultado); err != nil {
		t.Fatalf("a linha do stdout não é um resultado: %v", err)
	}
	return resultado
}

func (c *cenarioDasEtapas) argumentosRodados() []string {
	var todos []string
	for _, comando := range c.rodador.comandos {
		todos = append(todos, comando.Binario+" "+strings.Join(comando.Argumentos, " "))
	}
	return todos
}

func TestPreparoClonaNoVolumeEPublicaPronto(t *testing.T) {
	c := novoCenarioDasEtapas()

	codigo := c.programa().ExecutarPreparo(t.Context())

	if codigo != 0 {
		t.Fatalf("código de saída %d", codigo)
	}
	if desfecho := c.desfechoPublicado(t); !desfecho.Pronto || desfecho.Falha != nil {
		t.Errorf("preparo pronto: %+v", desfecho)
	}
	if !slices.Equal(c.preparados, []string{codigoPreparado, cachePreparado}) {
		t.Errorf("o código e o cache são criados para o uid do PR: %v", c.preparados)
	}
	if primeiro := c.argumentosRodados()[0]; !strings.HasSuffix(primeiro, "init -q "+codigoPreparado) {
		t.Errorf("o clone vai para o volume: %q", primeiro)
	}
	if !slices.Equal(c.linguagem.preparos, []string{codigoPreparado}) || c.linguagem.caches[0] != cachePreparado {
		t.Errorf("a linguagem prepara o projeto clonado no cache do volume: %v %v", c.linguagem.preparos, c.linguagem.caches)
	}
	if len(c.linguagem.testes) != 0 {
		t.Error("o preparo nunca roda os testes")
	}
}

func TestPreparoComFalhaDaLinguagemPublicaAFalha(t *testing.T) {
	c := novoCenarioDasEtapas()
	c.linguagem.falhaDoPreparo = &sandbox.ResultadoSandbox{SaidaBruta: "aprova: dependência inexistente\n", TipoFalha: sandbox.FalhaCompilacao}

	c.programa().ExecutarPreparo(t.Context())

	desfecho := c.desfechoPublicado(t)
	if desfecho.Pronto || desfecho.Falha == nil || desfecho.Falha.TipoFalha != sandbox.FalhaCompilacao {
		t.Errorf("a falha classificada sai no desfecho: %+v", desfecho)
	}
}

func TestPreparoQueNaoClonaEhInfraestrutura(t *testing.T) {
	c := novoCenarioDasEtapas()
	c.rodador.headDevolvido = strings.Repeat("b", 40)

	c.programa().ExecutarPreparo(t.Context())

	desfecho := c.desfechoPublicado(t)
	if desfecho.Falha == nil || desfecho.Falha.TipoFalha != sandbox.FalhaInfraestrutura || len(c.linguagem.preparos) != 0 {
		t.Errorf("clone no SHA errado não prepara nada: %+v", desfecho)
	}
}

func TestPreparoSemDiretorioDoVolumeEhInfraestrutura(t *testing.T) {
	c := novoCenarioDasEtapas()
	c.falharAoPreparar = errors.New("permissão negada")

	c.programa().ExecutarPreparo(t.Context())

	if desfecho := c.desfechoPublicado(t); desfecho.Falha == nil || desfecho.Falha.TipoFalha != sandbox.FalhaInfraestrutura {
		t.Errorf("sem onde gravar, é infraestrutura: %+v", desfecho)
	}
	if len(c.rodador.comandos) != 0 {
		t.Error("nada roda sem o diretório do volume")
	}
}

func TestPreparoComAmbienteInvalidoEhInfraestrutura(t *testing.T) {
	c := novoCenarioDasEtapas()
	delete(c.variaveis, "APROVA_SHA")

	c.programa().ExecutarPreparo(t.Context())

	if desfecho := c.desfechoPublicado(t); desfecho.Falha == nil || desfecho.Falha.TipoFalha != sandbox.FalhaInfraestrutura {
		t.Errorf("ambiente inválido: %+v", desfecho)
	}
}

func TestTesteConfereOSHACopiaParaOTemporarioERodaSemRede(t *testing.T) {
	c := novoCenarioDasEtapas()

	codigo := c.programa().ExecutarTeste(t.Context())

	if codigo != 0 {
		t.Fatalf("código de saída %d", codigo)
	}
	if resultado := c.resultadoPublicado(t); resultado.TestesPassaram != 2 || resultado.TipoFalha != sandbox.FalhaNenhuma {
		t.Errorf("o resultado é o da linguagem: %+v", resultado)
	}
	rodados := c.argumentosRodados()
	if len(rodados) != 2 || !strings.Contains(rodados[0], "-C "+codigoPreparado+" rev-parse HEAD") || rodados[1] != "cp -a "+codigoPreparado+" "+copiaNoTemporario {
		t.Errorf("confere o SHA no código montado e copia para o tmpfs, nessa ordem: %v", rodados)
	}
	for _, rodado := range rodados {
		for _, proibido := range []string{"fetch", "clone", "remote", "init"} {
			if strings.Contains(rodado, " "+proibido) {
				t.Errorf("a etapa testar não usa rede nem clona: %q", rodado)
			}
		}
	}
	if !slices.Equal(c.preparados, []string{"/tmp/aprova"}) {
		t.Errorf("o diretório de trabalho é criado para o uid do PR: %v", c.preparados)
	}
	if !slices.Equal(c.linguagem.testes, []string{copiaNoTemporario}) || c.linguagem.caches[0] != cachePreparado {
		t.Errorf("os testes rodam na cópia, com o cache do volume: %v %v", c.linguagem.testes, c.linguagem.caches)
	}
	if !c.linguagem.prazoFinal.Equal(inicioDeTeste.Add(600 * time.Second)) {
		t.Errorf("prazo final = %v", c.linguagem.prazoFinal)
	}
}

func TestTesteComSHADivergenteEntreAsEtapasEhInfraestrutura(t *testing.T) {
	c := novoCenarioDasEtapas()
	c.rodador.headDevolvido = strings.Repeat("c", 40)

	c.programa().ExecutarTeste(t.Context())

	resultado := c.resultadoPublicado(t)
	if resultado.TipoFalha != sandbox.FalhaInfraestrutura || !strings.Contains(resultado.SaidaBruta, "SHA divergente entre as etapas") {
		t.Errorf("SHA divergente é infraestrutura com o diagnóstico: %+v", resultado)
	}
	if len(c.linguagem.testes) != 0 || len(c.rodador.comandos) != 1 {
		t.Errorf("com SHA divergente nada é copiado nem testado: %v", c.argumentosRodados())
	}
}

func TestTesteComCopiaQueFalhaEhInfraestrutura(t *testing.T) {
	c := novoCenarioDasEtapas()
	c.rodador.falharCopia = true

	c.programa().ExecutarTeste(t.Context())

	if resultado := c.resultadoPublicado(t); resultado.TipoFalha != sandbox.FalhaInfraestrutura || len(c.linguagem.testes) != 0 {
		t.Errorf("sem cópia não há teste: %+v", resultado)
	}
}

func TestTesteSemTrocaDeUsuarioFalhaFechado(t *testing.T) {
	c := novoCenarioDasEtapas()
	c.rodador.erroDeInicio = processo.ErrTrocaDeUsuarioNegada

	c.programa().ExecutarTeste(t.Context())

	resultado := c.resultadoPublicado(t)
	if resultado.TipoFalha != sandbox.FalhaInfraestrutura || !strings.Contains(resultado.SaidaBruta, "usuário sem privilégio") {
		t.Errorf("troca de usuário negada: %+v", resultado)
	}
}

func TestCloneDesligaHooksELFS(t *testing.T) {
	rodador := &rodadorDeGitFalso{headDevolvido: shaDeTeste}

	if err := (Clonador{Rodador: rodador}).Clonar(t.Context(), "https://github.com/x/y.git", shaDeTeste, "/tmp/repositorio"); err != nil {
		t.Fatalf("clonar: %v", err)
	}

	for _, comando := range rodador.comandos {
		juntos := strings.Join(comando.Argumentos, " ")
		if !strings.Contains(juntos, "-c core.hooksPath=/dev/null") {
			t.Errorf("git sem hooks: %v", comando.Argumentos)
		}
		if !slices.Contains(comando.Ambiente, "GIT_LFS_SKIP_SMUDGE=1") {
			t.Errorf("git sem LFS: %v", comando.Ambiente)
		}
		if strings.Contains(juntos, "submodule") || strings.Contains(juntos, "--recurse") {
			t.Errorf("git sem submódulos: %v", comando.Argumentos)
		}
	}
}

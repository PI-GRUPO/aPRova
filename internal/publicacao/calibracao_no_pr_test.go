package publicacao

import (
	"strings"
	"testing"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/worker"
)

const descricaoDaCalibracao = "risco alto pelas regras, reduzido a médio pelo histórico: 2 casos parecidos aprovados e mergeados, nenhum revertido"

func publicacaoCalibrada() worker.Publicacao {
	publicacao := publicacaoDeTeste(taskRunDeTeste(idDoTaskRunDeConteudo, shaDoPrimeiro), domain.RiscoMedio)
	publicacao.Gate.NivelDeterministico = domain.RiscoAlto
	publicacao.Gate.Motivos = []domain.MotivoDeRisco{
		{ID: "area_sensivel", Nivel: domain.RiscoAlto, Descricao: "o PR altera um caminho sensível", Evidencias: []string{"internal/auth/sessao.go"}},
		{ID: "calibracao_historico", Nivel: domain.NivelInformativo, Descricao: descricaoDaCalibracao, Evidencias: []string{
			"00000000-0000-4000-8000-000000000003 Azulpasta/aPRova-TESTES#3 similaridade 0.9712 aprovado_e_mergeado",
			"00000000-0000-4000-8000-000000000008 Azulpasta/outro#8 similaridade 0.9301 aprovado_e_mergeado",
		}},
	}
	publicacao.Gate.Calibracao = domain.Calibracao{
		Estado: domain.CalibracaoAplicada,
		Razao:  "2 casos parecidos aprovados e mergeados, nenhum revertido",
		Vizinhos: []domain.VizinhoDaCalibracao{
			{TaskRunID: "00000000-0000-4000-8000-000000000003", Repositorio: "Azulpasta/aPRova-TESTES", NumeroPR: 3, Similaridade: 0.9712, Desfecho: domain.VizinhoAprovadoEMergeado},
			{TaskRunID: "00000000-0000-4000-8000-000000000008", Repositorio: "Azulpasta/outro", NumeroPR: 8, Similaridade: 0.9301, Desfecho: domain.VizinhoAprovadoEMergeado},
		},
	}
	return publicacao
}

func TestCalibracaoAplicadaApareceNoCheckRunENoComentarioComOsLinks(t *testing.T) {
	check, comentario := publicarUmaVez(t, publicacaoCalibrada())

	if check.Conclusao != "success" {
		t.Errorf("médio pela calibração libera o merge: %s", check.Conclusao)
	}
	for onde, texto := range map[string]string{"o resumo": check.Saida.Resumo, "o comentário": comentario.Corpo} {
		exigirTrechos(t, onde, texto,
			"#### Calibração por histórico",
			"Risco alto pelas regras, reduzido a médio pelo histórico: 2 casos parecidos aprovados e mergeados, nenhum revertido.",
			"[Azulpasta/aPRova-TESTES#3](https://github.com/Azulpasta/aPRova-TESTES/pull/3): similaridade 0,9712, aprovado e mergeado",
			"[Azulpasta/outro#8](https://github.com/Azulpasta/outro/pull/8): similaridade 0,9301, aprovado e mergeado",
		)
	}
}

func TestMotivoDaCalibracaoNaoApareceComoRiscoNosMotivos(t *testing.T) {
	_, comentario := publicarUmaVez(t, publicacaoCalibrada())

	motivos := comentario.Corpo[strings.Index(comentario.Corpo, "#### Motivos"):strings.Index(comentario.Corpo, "#### Calibração por histórico")]
	if strings.Contains(motivos, "calibracao_historico") || strings.Contains(motivos, "risco )") {
		t.Errorf("a calibração tem seção própria e não é um motivo de risco:\n%s", motivos)
	}
	exigirTrechos(t, "os motivos", motivos, "area_sensivel")
}

func TestCalibracaoQueNaoMudouONivelDizOPorque(t *testing.T) {
	publicacao := publicacaoDeTeste(taskRunDeTeste(idDoTaskRunDeConteudo, shaDoPrimeiro), domain.RiscoAlto)
	publicacao.Gate.Calibracao = domain.Calibracao{
		Estado: domain.CalibracaoVizinhoNegativo, Razao: "caso parecido com desfecho negativo: 1 revertido(s), 0 fechado(s) sem merge",
		Vizinhos: []domain.VizinhoDaCalibracao{{Repositorio: "Azulpasta/aPRova-TESTES", NumeroPR: 9, Similaridade: 0.95, Desfecho: domain.VizinhoRevertido}},
	}

	check, comentario := publicarUmaVez(t, publicacao)

	if check.Conclusao != "action_required" {
		t.Errorf("o histórico não mudou o nível: %s", check.Conclusao)
	}
	exigirTrechos(t, "o comentário", comentario.Corpo,
		"#### Calibração por histórico",
		"O nível das regras foi mantido: caso parecido com desfecho negativo: 1 revertido(s), 0 fechado(s) sem merge.",
		"[Azulpasta/aPRova-TESTES#9](https://github.com/Azulpasta/aPRova-TESTES/pull/9): similaridade 0,9500, revertido",
	)
}

func TestSemCalibracaoNaoHaSecaoDeCalibracao(t *testing.T) {
	for _, estado := range []domain.EstadoCalibracao{"", domain.CalibracaoDesligada, domain.CalibracaoSemNivelAReduzir} {
		publicacao := publicacaoDeTeste(taskRunDeTeste(idDoTaskRunDeConteudo, shaDoPrimeiro), domain.RiscoBaixo)
		publicacao.Gate.Calibracao = domain.Calibracao{Estado: estado, Razao: "qualquer"}

		_, comentario := publicarUmaVez(t, publicacao)

		if strings.Contains(comentario.Corpo, "Calibração") {
			t.Errorf("estado %q não mostra calibração:\n%s", estado, comentario.Corpo)
		}
	}
}

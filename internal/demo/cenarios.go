package demo

import (
	"time"

	"github.com/PI-GRUPO/aPRova/internal/domain"
	"github.com/PI-GRUPO/aPRova/internal/sandbox"
)

const (
	dia         = 24 * time.Hour
	autorDaDemo = "agente-ia"
)

func passou(testes int, duracaoMS int64) sandbox.ResultadoSandbox {
	return sandbox.ResultadoSandbox{
		Compilou: true, TestesPassaram: testes, DuracaoMS: duracaoMS, TipoFalha: sandbox.FalhaNenhuma,
		SaidaBruta: "ok  \tgithub.com/aprova-demo/loja-virtual/...",
	}
}

func motivo(id string, nivel domain.NivelRisco, descricao string, evidencias ...string) domain.MotivoDeRisco {
	return domain.MotivoDeRisco{ID: id, Nivel: nivel, Descricao: descricao, Evidencias: evidencias}
}

func areaSensivel(arquivo string) domain.MotivoDeRisco {
	return motivo("area_sensivel", domain.RiscoAlto, "o PR altera um caminho sensível", arquivo)
}

// cenarios devolve as análises da demonstração na ordem em que são gravadas.
// Os PRs #103 e #104 vêm antes do #105, porque são os vizinhos que a
// calibração dele usa.
func cenarios() []cenario {
	return []cenario{
		{
			numeroPR: 101, titulo: "Corrige o arredondamento do frete", autor: autorDaDemo, recebidoHa: 6 * dia,
			sandbox: passou(18, 4100),
		},
		{
			numeroPR: 102, titulo: "Simplifica o cálculo do desconto", autor: autorDaDemo, recebidoHa: 5*dia + 6*time.Hour,
			sandbox: passou(18, 3900),
			motivos: []domain.MotivoDeRisco{motivo("codigo_sem_teste", domain.RiscoMedio,
				"o PR altera código sem alterar nenhum teste", "internal/desconto/calculo.go")},
		},
		{
			numeroPR: 103, titulo: "Valida o CPF na cobrança", autor: autorDaDemo, recebidoHa: 5 * dia,
			sandbox: passou(22, 4800),
			motivos: []domain.MotivoDeRisco{areaSensivel("internal/pagamento/cobranca.go")},
			humano: &decisaoHumana{decisao: domain.DecisaoAprovada, revisor: "revisora-demo", origem: domain.OrigemGitHub,
				feedback: domain.FeedbackRegraConservadora, justificativa: "mudança pequena e coberta por teste"},
			mergeado: true,
		},
		{
			numeroPR: 104, titulo: "Registra o motivo da recusa do cartão", autor: autorDaDemo, recebidoHa: 4*dia + 8*time.Hour,
			sandbox: passou(24, 5100),
			motivos: []domain.MotivoDeRisco{areaSensivel("internal/pagamento/cartao.go")},
			humano: &decisaoHumana{decisao: domain.DecisaoAprovada, revisor: "revisor-demo", origem: domain.OrigemSlack,
				feedback: domain.FeedbackRiscoAceito, justificativa: "só acrescenta log, sem mudar a regra"},
			mergeado: true,
		},
		{
			numeroPR: 109, titulo: "Acrescenta etapa de lint no CI", autor: autorDaDemo, recebidoHa: 4 * dia,
			sandbox: passou(18, 4000),
			motivos: []domain.MotivoDeRisco{motivo("configuracao_ci", domain.RiscoAlto,
				"o PR altera configuração de build ou de CI", ".github/workflows/ci.yml")},
			humano:   &decisaoHumana{decisao: domain.DecisaoAprovada, revisor: "revisora-demo", origem: domain.OrigemGitHub},
			mergeado: true, revertidoPor: 110,
		},
		{
			numeroPR: 105, titulo: "Ajusta a mensagem de pagamento recusado", autor: autorDaDemo, recebidoHa: 3 * dia,
			sandbox:     passou(24, 4700),
			motivos:     []domain.MotivoDeRisco{areaSensivel("internal/pagamento/mensagens.go")},
			calibrarCom: []int{103, 104},
		},
		{
			numeroPR: 107, titulo: "Muda a regra de estoque mínimo", autor: autorDaDemo, recebidoHa: 2 * dia,
			sandbox: sandbox.ResultadoSandbox{
				Compilou: true, TestesPassaram: 16, TestesFalharam: 2, DuracaoMS: 5300, TipoFalha: sandbox.FalhaTeste,
				SaidaBruta: "--- FAIL: TestEstoqueMinimo (0.01s)\n--- FAIL: TestReposicao (0.01s)",
			},
			motivos: []domain.MotivoDeRisco{motivo("falha_sandbox", domain.RiscoAlto,
				"os testes ou a compilação falharam no sandbox", "2 de 18 testes falharam")},
			humano: &decisaoHumana{decisao: domain.DecisaoAlteracoesSolicitadas, revisor: "revisor-demo", origem: domain.OrigemSlack,
				justificativa: "os testes de estoque quebraram"},
		},
		{
			numeroPR: 108, titulo: "Atualiza o cliente da transportadora", autor: autorDaDemo, recebidoHa: 1 * dia,
			sandbox: sandbox.ResultadoSandbox{
				DuracaoMS: 600000, TipoFalha: sandbox.FalhaInfraestrutura,
				SaidaBruta: "prazo do job esgotado antes de os testes terminarem",
			},
		},
		{
			numeroPR: 106, titulo: "Usa biblioteca nova para gerar etiquetas", autor: autorDaDemo, recebidoHa: 2 * time.Hour,
			sandbox: passou(20, 4400),
			motivos: []domain.MotivoDeRisco{motivo("dependencia_nova", domain.RiscoAlto,
				"o PR acrescenta dependência nova, que precisa de revisão humana", "go.mod")},
			dependencias: []domain.DependenciaNova{{Ecossistema: "Go", Nome: "github.com/exemplo/etiquetas-pdf", Versao: "v1.4.0"}},
		},
	}
}

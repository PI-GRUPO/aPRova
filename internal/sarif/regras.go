package sarif

import "github.com/PI-GRUPO/aPRova/internal/domain"

type alcance int

const (
	alcanceArquivo alcance = iota
	alcancePullRequest
	alcanceSemArquivo
)

const (
	nivelAviso = "warning"
	nivelNota  = "note"

	motivoDependenciaNova     = "dependencia_nova"
	motivoCalibracaoHistorico = "calibracao_historico"
)

type regraDoGate struct {
	id      string
	nivel   domain.NivelRisco
	alcance alcance
	curta   string
	ajuda   string
}

var niveisDoSARIF = map[domain.NivelRisco]string{
	domain.RiscoAlto:  nivelAviso,
	domain.RiscoMedio: nivelNota,
}

var catalogo = []regraDoGate{
	{
		id: "area_sensivel", nivel: domain.RiscoAlto, alcance: alcanceArquivo,
		curta: "O PR altera um caminho sensível",
		ajuda: "O arquivo fica em um caminho que o time marcou como sensível, como autenticação, pagamentos ou migrations de banco. " +
			"O aPRova bloqueia o merge até uma pessoa com permissão de escrita aprovar, porque um erro nesses caminhos custa caro e nem sempre aparece nos testes.",
	},
	{
		id: "codigo_sem_teste", nivel: domain.RiscoMedio, alcance: alcancePullRequest,
		curta: "O PR altera código sem alterar nenhum teste",
		ajuda: "O PR muda código de produção e não acrescenta nem altera teste algum. " +
			"O aPRova não bloqueia por isso: registra o risco médio para que a revisão confira se o comportamento novo está coberto.",
	},
	{
		id: "configuracao_ci", nivel: domain.RiscoAlto, alcance: alcanceArquivo,
		curta: "O PR altera configuração de build ou de CI",
		ajuda: "O arquivo define como o projeto é construído, testado ou publicado. " +
			"O aPRova bloqueia o merge até a aprovação humana, porque uma mudança aqui pode desligar verificações ou alterar o que vai para produção.",
	},
	{
		id: motivoDependenciaNova, nivel: domain.RiscoAlto, alcance: alcanceArquivo,
		curta: "O PR acrescenta dependência nova",
		ajuda: "O manifesto de dependências ganhou um pacote que o projeto não usava. " +
			"O aPRova bloqueia o merge até a aprovação humana, porque código de terceiros entra no produto sem passar pela revisão do time.",
	},
	{
		id: "falha_sandbox", nivel: domain.RiscoAlto, alcance: alcanceSemArquivo,
		curta: "Os testes ou a compilação falharam no sandbox",
		ajuda: "O aPRova rodou o projeto no commit do PR, em ambiente isolado, e os testes ou a compilação falharam. " +
			"O merge fica bloqueado até a aprovação humana. O detalhe da falha está no Check Run aPRova, não em um arquivo específico.",
	},
	{
		id: "muitos_arquivos", nivel: domain.RiscoAlto, alcance: alcancePullRequest,
		curta: "O PR altera arquivos demais",
		ajuda: "O PR passa do limite de arquivos alterados configurado no aPRova. " +
			"O merge fica bloqueado até a aprovação humana, porque um PR grande demais não é revisado com a mesma atenção.",
	},
	{
		id: "segredo_versionado", nivel: domain.RiscoAlto, alcance: alcanceArquivo,
		curta: "O PR versiona um arquivo que costuma guardar segredo",
		ajuda: "O nome do arquivo é de um tipo que costuma guardar credenciais, como .env ou uma chave privada. " +
			"O aPRova bloqueia o merge até a aprovação humana, porque um segredo que entra no histórico do git continua lá mesmo depois de apagado.",
	},
}

func regraPorID(id string) (int, regraDoGate, bool) {
	for indice, regra := range catalogo {
		if regra.id == id {
			return indice, regra, true
		}
	}
	return 0, regraDoGate{}, false
}

func descritores() []descritor {
	lista := make([]descritor, 0, len(catalogo))
	for _, regra := range catalogo {
		lista = append(lista, descritor{
			ID:                   regra.id,
			ShortDescription:     texto{Text: regra.curta},
			FullDescription:      texto{Text: regra.ajuda},
			Help:                 texto{Text: regra.ajuda},
			DefaultConfiguration: configuracaoPadrao{Level: niveisDoSARIF[regra.nivel]},
		})
	}
	return lista
}

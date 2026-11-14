package config

import "os"

type variavelRemovida struct {
	nome   string
	motivo string
}

var variaveisRemovidas = []variavelRemovida{
	{
		nome:   "SANDBOX_REDE_LOCAL",
		motivo: "a rede do sandbox local agora é fixa por etapa (preparar com rede, testar sem rede nenhuma); remova a variável do .env",
	},
}

func (l *leitor) recusarVariaveisRemovidas() {
	for _, removida := range variaveisRemovidas {
		if _, presente := os.LookupEnv(removida.nome); presente {
			l.registrarPendencia("%s não existe mais: %s", removida.nome, removida.motivo)
		}
	}
}

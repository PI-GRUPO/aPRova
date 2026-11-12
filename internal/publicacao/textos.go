package publicacao

import (
	"github.com/Azulpasta/aPRova/internal/platform"
	"github.com/Azulpasta/aPRova/internal/worker"
)

const tamanhoDoSHACurto = 7

func (p *Publicador) checkRunEmAndamento() platform.CheckRun {
	return platform.CheckRun{
		Situacao:   platform.CheckRunEmAndamento,
		IniciadoEm: p.opcoes.Agora(),
		Titulo:     "Análise em andamento",
		Resumo:     "O aPRova está analisando este commit.",
	}
}

func (p *Publicador) checkRunDoResultado(publicacao worker.Publicacao) platform.CheckRun {
	return platform.CheckRun{
		Situacao:    platform.CheckRunConcluido,
		ConcluidoEm: p.opcoes.Agora(),
		Titulo:      "Análise concluída",
		Resumo:      "Commit analisado: " + shaCurto(publicacao.TaskRun.SHAHead),
	}
}

func textoDoComentario(publicacao worker.Publicacao) string {
	return marcaDoComentario + "\nCommit analisado: `" + shaCurto(publicacao.TaskRun.SHAHead) + "`"
}

func shaCurto(sha string) string {
	return sha[:min(len(sha), tamanhoDoSHACurto)]
}

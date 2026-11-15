package domain

// EstadoEnvioSARIF é o que aconteceu com o envio do SARIF de uma análise ao
// code scanning do GitHub.
type EstadoEnvioSARIF string

// Estados gravados em sarif_uploads.status. Aceito é o GitHub ter recebido o
// arquivo, com o processamento ainda sem desfecho conhecido. Processado e
// Recusado são os desfechos do processamento. Falhou é o envio que o GitHub
// não aceitou; é o único estado sem id de envio.
const (
	EnvioSARIFAceito     EstadoEnvioSARIF = "aceito"
	EnvioSARIFProcessado EstadoEnvioSARIF = "processado"
	EnvioSARIFRecusado   EstadoEnvioSARIF = "recusado"
	EnvioSARIFFalhou     EstadoEnvioSARIF = "falhou"
)

// EnvioSARIF é o registro do envio do SARIF de um task_run. IDDoEnvio é o id
// devolvido pelo GitHub, vazio enquanto nenhum envio foi aceito. Detalhe
// guarda o diagnóstico de uma falha ou de um processamento recusado.
type EnvioSARIF struct {
	TaskRunID string
	IDDoEnvio string
	Estado    EstadoEnvioSARIF
	Detalhe   string
}

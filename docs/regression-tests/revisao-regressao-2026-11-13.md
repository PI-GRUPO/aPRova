# Regressão da integração dev → workflows-n8n — 2026-11-13

Resultado: APROVADO nas verificações executadas, com as limitações abaixo.

## Integração

Base da branch workflows-n8n: `0f583a91fea9b6ab8e936b6808665844ea68e51f`.
Dev incorporada: `c737253`. O merge preserva os workflows e a justificativa
remota, junto de sandbox Python, gate, recibos, publicação e calibração da dev.

Os quatro conflitos foram resolvidos em `.env.example`, `cmd/api/main.go`,
`tests/functional/acoes_remotas_test.go` e `tests/integration/postgres/esquema_test.go`.
A API mantém o verificador de recibos e usa PostgreSQL para as tentativas de
ação remota. A aprovação pelo Slack continua exigindo justificativa. O teste
novo de persistência passou a importar o módulo `github.com/PI-GRUPO/aPRova`.

A migration de justificativas foi renumerada para 21. As migrations 15–20
da dev foram preservadas. O teste de esquema exige 21 versões e testa cada
up/down, reaplicação e reversão completa. O código SQL foi regenerado pelo
sqlc v1.30.0 e uma segunda geração não produziu diferenças.

O `.env.example` preserva as variáveis n8n e incorpora as variáveis atuais
de recibos, gate, Python e calibração, sem comentários ou valores privados.
A documentação foi atualizada para remover afirmações superadas sobre as
implementações provisórias do backend e explicar a migração do ambiente.

## Validação

| Verificação | Resultado |
| --- | --- |
| Go, com detector de corrida | 1361 testes/subtestes únicos aprovados; 712 testes principais aprovados; sem alertas de corrida |
| PostgreSQL temporário com pgvector | 139 testes/subtestes aprovados, incluindo justificativas, recibos, histórico, calibração, concorrência e 21 migrations |
| Sandbox Go/Python e containers básicos | 28 cenários aprovados, sem skips na rodada retomada |
| Script de eventos | 25 testes/subtestes aprovados |
| Teste Gemini real | Um skip intencional: exige APROVA_TESTE_GEMINI_REAL=1 e credencial externa; não foi acionado |
| Workflows n8n | 88 testes aprovados, sem falhas ou skips |
| go vet e go build | Aprovados |
| golangci-lint v2.13.2 | Zero problemas |
| sqlc v1.30.0 | Geração estável |
| docker compose config --quiet | Aprovado |
| git diff --check origin/dev | Aprovado |
| Valores privados locais | Nenhuma cópia exata dos segredos/IDs locais verificados no diff do PR |

Go 1.26.11, conforme go.mod; Node v24.13.0. Comandos:

```sh
go test ./... -race -count=1 -json
go test ./tests/integration/... -race -count=1 -timeout=25m -json
node --test --test-isolation=none workflows/n8n/tests/*.test.cjs
go vet ./...
go build ./...
golangci-lint run
sqlc generate
docker compose config --quiet
git diff --check origin/dev
```

A primeira execução de `go test ./...` aprovou os testes de código, mas
terminou com erro no TestMain de PostgreSQL e pulou os sandboxes porque o
Docker Desktop estava desligado. Após o usuário iniciá-lo, toda a suíte
`tests/integration/...` foi executada novamente com CI=true e terminou com
sucesso. Os totais acima usam o último resultado de cada teste, sem contar
novamente o script de eventos executado nas duas rodadas. Os containers de
teste usam banco temporário, separado do banco de desenvolvimento.

## Ambiente e limites

O `.env` local, o banco de desenvolvimento e os workflows da instância n8n
não foram modificados. Os dados locais não foram migrados nem apagados.
As variáveis antigas RECEIPT_* precisam ser substituídas pelas novas
APROVA_CHAVE_RECIBO e APROVA_CHAVES_PUBLICAS_RECIBO antes de iniciar o worker.
Preservar a identidade de assinatura; não substituir uma chave real por
outra durante a migração sem planejar sua rotação.

Se um banco já executou a antiga migration 15 de justificativas da branch
workflows-n8n, não basta aplicar o novo histórico: a versão 15 agora identifica
os motivos do gate. É necessário reconciliar o esquema e o histórico antes
de executar migrate up; não usar force ou reset indiscriminadamente. Ver
[guia de integração](../../workflows/n8n/README.md#integração-com-a-dev--2026-11-13).

Não foram feitos disparos reais de Slack, Trello ou GitHub, nem validação com
o Gemini real. O protótipo frontend não foi alterado em relação à dev e seus
testes não foram executados nesta rodada. A validação local não substitui o
CI e não garante ausência de todos os defeitos.

Logs temporários em `/tmp/aprova-merge-20261113/`: go-test.jsonl,
integracao.jsonl, respectivos arquivos .err, vet.log, build.log e lint.log.

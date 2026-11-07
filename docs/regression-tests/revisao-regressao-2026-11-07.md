# Regressão de backend e banco — 2026-11-07

Resultado: APROVADO. Execução completa com código de saída 0, sem testes Go falhando ou ignorados, sem alertas do detector de condições de corrida.

Base: cf710af60ce00dfb9237925732f9cd121e84177c, branch workflows-n8n, com as alterações locais listadas ao fim. Go 1.26.10 linux/amd64 e Node v24.13.0.

## Resultados

- Go: 423 testes principais; 864 aprovações contando testes e subtestes; 28 pacotes aprovados.
- Workflows n8n: 74 testes aprovados, 0 falhas, 0 ignorados.
- Compilação `go build ./...`: aprovada.
- Análise estática `go vet ./...`: aprovada.
- `git diff --check`: aprovado.
- Medição complementar de cobertura com PostgreSQL: 87 aprovações, 0 falhas, 0 ignorados.

## Escopo validado

API, webhooks, autenticação e assinaturas HMAC, relatório semanal, ações remotas e permissões com GitHub simulado, transições de estado, workers, eventos e scripts. PostgreSQL real temporário com pgvector: aplicação e reversão das 15 migrations, schema, checks, índices, transações e rollback, recibos imutáveis, isolamento por tenant, vetores, fila, disputa entre workers, leases e retomada após falhas. Sandbox Docker: projeto válido, testes falhos, compilação inválida, travamento e timeout, tentativa de forjar resultado, incompatibilidade de Go, permissões, subdiretórios, symlinks, isolamento de rede e limpeza dos containers.

## Execuções

```sh
go test ./... -race -count=1 -covermode=atomic -coverprofile=coverage.out -json
go test ./tests/integration/postgres/... -race -count=1 -coverpkg=./internal/repository/...,./internal/worker/... -covermode=atomic -coverprofile=coverage-banco.out -json
go vet ./...
go build ./...
node --test --test-isolation=none --test-reporter=tap workflows/n8n/tests/*.test.cjs
```

Go executado com `CI=true`, exigindo sucesso do PostgreSQL em vez de aceitar indisponibilidade como skip. A opção de isolamento do Node permite registrar individualmente os 74 casos neste ambiente. Dependências e logs foram preparados em /tmp/aprova-regressao-20261107.

## Testes por pacote (inclui subtestes)

| Pacote | Passaram | Falharam | Ignorados |
| --- | ---: | ---: | ---: |
| internal/acaoremota | 99 | 0 | 0 |
| internal/api | 85 | 0 | 0 |
| internal/config | 49 | 0 | 0 |
| internal/domain | 114 | 0 | 0 |
| internal/eventos | 83 | 0 | 0 |
| internal/platform | 47 | 0 | 0 |
| internal/rastro | 3 | 0 | 0 |
| internal/repository | 3 | 0 | 0 |
| internal/sandbox | 65 | 0 | 0 |
| internal/sandbox/conteiner | 55 | 0 | 0 |
| internal/sandbox/golang | 31 | 0 | 0 |
| internal/sandbox/jobsazure | 4 | 0 | 0 |
| internal/sandbox/processo | 12 | 0 | 0 |
| internal/worker | 38 | 0 | 0 |
| scripts/consultar-permissao | 11 | 0 | 0 |
| scripts/disparar-sandbox-azure | 12 | 0 | 0 |
| scripts/gerar-chaves | 2 | 0 | 0 |
| scripts/mudou-sandbox | 5 | 0 | 0 |
| scripts/verificar-token | 3 | 0 | 0 |
| tests/functional | 16 | 0 | 0 |
| tests/integration | 15 | 0 | 0 |
| tests/integration/eventos | 25 | 0 | 0 |
| tests/integration/postgres | 87 | 0 | 0 |

## Cobertura consolidada de instruções

União dos perfis da regressão geral e da medição de integração do banco. A medição complementar instrumenta explicitamente repositório e workers para contabilizar as chamadas dos testes de integração.

| Pacote | Cobertura |
| --- | ---: |
| cmd/api | 0.0% |
| cmd/sandbox | 0.0% |
| cmd/worker | 0.0% |
| internal/acaoremota | 97.1% |
| internal/acaoremota/contratotentativas | 0.0% |
| internal/api | 89.7% |
| internal/config | 98.2% |
| internal/domain | 41.7% |
| internal/eventos | 94.4% |
| internal/platform | 87.0% |
| internal/rastro | 100.0% |
| internal/repository | 81.0% |
| internal/repository/db | 93.9% |
| internal/sandbox | 90.1% |
| internal/sandbox/conteiner | 95.0% |
| internal/sandbox/golang | 97.5% |
| internal/sandbox/jobsazure | 35.0% |
| internal/sandbox/processo | 78.4% |
| internal/worker | 83.2% |
| scripts/consultar-permissao | 47.6% |
| scripts/disparar-sandbox-azure | 32.3% |
| scripts/gerar-chaves | 20.0% |
| scripts/mudou-sandbox | 94.1% |
| scripts/verificar-token | 12.0% |

## Limites

Nenhuma regressão detectada nos cenários executados. A cobertura não é de 100%; os resultados não garantem ausência de todo defeito. Não foi validado o fluxo completo com contas reais de GitHub, Slack, Trello, Gemini ou Azure. Os testes usam dependências simuladas quando previsto pela suíte. O banco usado nos testes é temporário; os dados do banco de desenvolvimento não foram alvo das migrations desta execução. Nenhum arquivo de implementação foi alterado para a rodada.

## Alterações locais preservadas

```text
 M .env.example
 M docker-compose.yml
 M docs/central-slack-implementacao.md
 M workflows/n8n/README.md
?? workflows/n8n/consulta-relatorio-semanal.json
?? workflows/n8n/slack-consulta-metricas.json
?? workflows/n8n/tests/consulta-metricas.test.cjs
```

Logs: testes.jsonl, testes.err, banco-cobertura.jsonl, banco-cobertura.err, n8n.tap, vet.log, build.log, coverage.out, coverage-banco.out e coverage-consolidada.out, neste mesmo diretório.

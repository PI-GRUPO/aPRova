# Revisão e regressão — 01/11/2026

## Escopo

Revisão dos 20 arquivos modificados e dos 8 arquivos novos presentes no
repositório `aPRova` no início da execução, comparados com `HEAD` (`2bc2dae`).
Inclui API, autorização e execução de ações remotas, persistência e SQL gerado,
migration 13, configuração Compose, CI, documentação, exports n8n e testes.
O código da aplicação e os workflows não foram alterados nesta revisão.

## Resultado e impacto

Não foram identificados defeitos novos na revisão do código nem regressões
nos testes executados. Todas as suítes foram concluídas com sucesso após
resolver as restrições locais de execução. Isso não substitui a validação
com Slack e GitHub reais.

- **Mudança de contrato:** `aprovar` agora exige justificativa não vazia,
  assim como `solicitar_alteracao`. Clientes que aprovavam sem esse campo
  receberão HTTP 400. O limite da API é de 4000 caracteres; o formulário
  Slack admite 3000. A justificativa passa a aparecer também na review.
- **Banco obrigatório para as ações:** a API troca o registro em memória
  pelo PostgreSQL. A migration `000015` (originalmente `000013`, renumerada
  após a integração com `dev`) deve ser aplicada antes de executar a API atualizada; sem a coluna `justification`, a reserva falha e a ação
  não é executada. A proteção contra duplicidade passa a sobreviver à criação
  de outra instância do repositório.
- **Correlação da análise:** `task_run_id` é opcional. Sem ele, permanece a
  busca da análise mais recente do mesmo tenant, repositório e PR. Quando
  informado, um ID incompatível não é associado a outra análise.
- **Autorização preservada:** identidade resolvida pelo núcleo, consulta de
  permissão no GitHub, recusa de autoaprovação, autenticação HMAC e reserva
  antes da execução continuam cobertas pelos testes.
- **Outras funcionalidades:** saúde da API, webhooks GitHub, relatórios,
  autenticação e renovação de tokens, scripts e componentes do sandbox
  passaram na suíte principal.
- **Limitações anteriores:** reexecução real continua indisponível no
  bootstrap da API; emissão automática de eventos e uso da justificativa
  na calibração do gate continuam pendentes, conforme a documentação.

## Validações

| Verificação | Resultado |
| --- | --- |
| Go: `internal`, `cmd`, `scripts` e testes funcionais, com `-race -count=1` | 528 casos aprovados, incluindo subtestes; 270 testes de primeiro nível |
| PostgreSQL real em container temporário, com `-race -count=1` | 68 casos aprovados, incluindo subtestes |
| Script de eventos, com `-race -count=1` | 25 casos aprovados, incluindo subtestes; 15 testes de primeiro nível |
| Integração com Docker e imagem real do sandbox Go, com `-race -count=1` | 15 testes aprovados, incluindo timeout, código malicioso, isolamento de rede, privilégios e limpeza de containers |
| Testes existentes dos exports n8n | 12 testes aprovados |
| Verificações adicionais isoladas dos exports | 13 testes aprovados |
| `go vet ./...` e `go build ./...` | Aprovados |
| `gofmt -l` nas áreas revisadas | Nenhum arquivo pendente |
| `git diff --check` | Aprovado |
| `sqlc generate` 1.30.0 em cópia temporária | Arquivos idênticos aos arquivos gerados presentes no projeto |
| `docker compose --profile automacao config --quiet` | Aprovado |

Total: 636 casos Go aprovados, contando subtestes, e 25 verificações Node
aprovadas (12 existentes e 13 adicionais). Não restaram testes pulados sem
reexecução: os 15 skips da primeira execução da integração de eventos foram
substituídos pela execução bem-sucedida dessa suíte com `jq` disponível.

As verificações adicionais dos exports exercitaram os seis exemplos de
eventos existentes, quebras de linha reais nas notificações e retornos,
assinatura Slack válida, adulteração, timestamps antigos/futuros e referências
entre nós dos dois workflows. Ficaram em `/tmp/aprova-revisao-workflows.cjs`.
Uma suspeita inicial de escape incorreto nas mensagens foi descartada por
esses testes; não há defeito de quebra de linha confirmado.

A primeira execução restrita foi bloqueada pelo sandbox ao abrir sockets
HTTP locais e acessar Docker. Esses casos foram repetidos com a autorização
necessária. Os testes do script inicialmente pulados por falta de `jq`
foram repetidos com a ferramenta em `/tmp`, todos com sucesso.

Cobertura da suíte principal: 60,0% das instruções. Esse número exclui as
execuções separadas de integração; não representa cobertura completa do produto.

## Limites da validação

Não foram publicadas mensagens Slack, aprovadas PRs reais, alterados workflows
na instância n8n ou aplicadas migrations no banco de desenvolvimento.
Os testes funcionais usam GitHub simulado; os testes de banco usam container
temporário. A revisão dos exports não equivale à execução completa dos dois
workflows dentro do n8n com credenciais reais.

Logs desta execução estão em `/tmp/aprova-regressao-principal.jsonl`,
`/tmp/aprova-regressao-integracao.jsonl`, `/tmp/aprova-regressao-eventos.jsonl`
e `/tmp/aprova-revisao-workflows.tap`. Arquivos em `/tmp` são temporários.

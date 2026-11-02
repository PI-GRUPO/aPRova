# Regressão antes do commit — 02/11/2026

Base da revisão: `38c2908`. As alterações pendentes incluem a integração
Trello, sua ligação ao Core e documentação. A regressão incluiu o projeto
Go completo e os quatro workflows exportados.

## Resultado

| Verificação | Resultado |
| --- | --- |
| `go test ./... -race -count=1` | 636 casos aprovados, incluindo subtestes; zero falhas e zero skips |
| Suíte principal Go, incluindo testes funcionais | 528 casos aprovados |
| PostgreSQL temporário e migrations | 68 casos aprovados |
| Integração do script de eventos | 25 casos aprovados |
| Integração Docker/sandbox Go | 15 testes aprovados |
| Testes Node dos workflows | 53 testes aprovados: 35 existentes e 18 novos |
| `go vet ./...` | Aprovado |
| `go build ./...` | Aprovado |
| `docker compose --profile automacao config --quiet` | Aprovado |
| `git diff --check` | Aprovado |

Não foram detectadas regressões nas funcionalidades anteriores exercitadas.
Os testes dos exports verificam também os grafos, ausência de credenciais e
dados fixados, preservação do risco, contratos Slack/OSV e a rota do Trello.
O `.env` e a pasta `.n8n/` permanecem ignorados pelo Git.

## Defeito encontrado e corrigido

O nó `10 | Consultar Comentários` do export Trello usava `page` para paginar
ações. A API usa o cursor `before` com o ID da última ação da página
anterior, conforme a [documentação do Trello](https://developer.atlassian.com/cloud/trello/guides/rest-api/api-introduction/#paging).
Com mil ou mais comentários, a configuração anterior podia repetir a primeira
página até atingir o limite, impedindo a integração de continuar.

A correção está em `workflows/n8n/trello-sugestao-movimentacao.json`.
O teste de regressão falhou antes da correção e passou depois. Os 18 testes
novos estão em `workflows/n8n/tests/trello-sugestao-movimentacao.test.cjs` e
são abrangidos pelo comando de testes já usado pela CI.

## Limites e parecer

O conjunto local está pronto para commit e push em branch de revisão.
Não foram feitos commit, push ou alterações na instância n8n publicada.
A correção do cursor precisa ser aplicada também nessa instância ao atualizar
o workflow, preservando suas credenciais e configuração de quadro/listas.

Os testes Trello/Slack usam respostas simuladas, sem publicar comentários
ou mensagens. A correção não foi validada contra um card real com mais de
mil comentários. A execução completa de ações no GitHub real continua sendo
uma validação separada. Não foram executados todos os jobs remotos da CI,
como auditoria de vulnerabilidades e golangci-lint.

Logs temporários: `/tmp/aprova-regressao-completa-2026-11-02.jsonl` e
`/tmp/aprova-workflows-regressao-2026-11-02.tap`. A execução local usou
Go 1.26.10, Node 24 e `jq` disponível para os testes do script de eventos.

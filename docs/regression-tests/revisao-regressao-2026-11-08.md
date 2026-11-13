# Regressão de configuração do n8n — 2026-11-08

Resultado: APROVADO nos cenários isolados e nas verificações de configuração.
Integrações externas não foram executadas nesta rodada. Não houve commit ou push.

## Implementação

Nove workflows de produção foram comparados com os exports antes de alterar.
Foram preservados os parâmetros funcionais da instância e as correções do
repositório: cursor Trello `before`, configuração reutilizável das listas e
código que consulta os nomes/IDs das listas sem constantes locais.
Os dois nós manuais Trello permanecem somente na instância. Os exports foram
gerados a partir do estado atualizado, sanitizados e conferidos contra ele.

Subworkflows usam `APROVA_*_WORKFLOW_ID`. Quadro, sete listas, rótulos,
canal/flags do onboarding e identidade futura do Check leem ambiente.
`SLACK_GITHUB_VINCULOS` continua compartilhada entre API e feedback, sem
cadastro de pessoas nos nós. Os valores anteriores do `.env` foram
preservados; foram acrescentadas 25 configurações reaproveitadas com
correspondência inequívoca na instância, sem inventar IDs ou identidades.

A base pública dos webhooks usa `N8N_WEBHOOK_URL_PUBLICA` → `WEBHOOK_URL`
nativo do n8n, com fallback para `N8N_URL_BASE` (editor).
`N8N_WEBHOOK_URL` continua sendo o destino completo dos eventos no Go.
Não foi necessário renomear variáveis consumidas pelo Go ou pelos scripts;
seus contratos e implementações foram preservados.

As três credenciais usadas pelo aPRova seguem `aPRova | Serviço`:
Slack, Trello e Gemini. IDs, associações aos nós e dados cifrados foram
comparados com o backup e permaneceram iguais. Demais credenciais intactas.
Não houve exclusão, rotação, alteração de escopos ou transferência de tokens
para ambiente/JSONs versionáveis.

Também foram parametrizadas três referências no workflow manual de testes
do onboarding (canal, quadro e subworkflow); segue inativo e sem export.
Fixtures sintéticas de testes continuam locais e não representam destinos.

## Validação

| Verificação | Resultado |
| --- | --- |
| Testes Node dos workflows | 88 passaram; zero falhas, ignorados ou cancelados |
| Go com `-race -count=1` | Nove pacotes aprovados; sem alertas de corrida |
| Estado de ativação | Os 15 workflows da instância mantiveram o estado anterior; oito workflows de produção ativos, Digest inativo |
| Settings e nós da instância | Settings, IDs estruturais e conjunto de nós preservados |
| Credenciais | Todos os conteúdos cifrados e IDs preservados; vínculos de todos os nós preservados |
| Exports | Nove inativos, sem IDs locais de workflow, credenciais pessoais ou dados fixados; parâmetros e conexões conferidos com a instância |
| Ambiente | Valores anteriores do `.env` preservados; leituras `$env` cobertas pelo exemplo e Compose |
| Backend | Variáveis de `internal/config/configuracao.go` cobertas pelo `.env.example` |
| Segredos | Valores sensíveis do `.env` não encontrados nos arquivos rastreados; inspeção dos novos testes sem segredos |
| Compose | `docker compose config --quiet` aprovado; nomes de variáveis conferidos no container, sem exibir valores |
| Formatação | `git diff --check` aprovado |

Comandos principais:

```sh
node --test --test-isolation=none workflows/n8n/tests/*.test.cjs
go test ./internal/config ./internal/eventos ./internal/acaoremota ./internal/api ./scripts/... -race -count=1
docker compose config --quiet
git diff --check
```

Go 1.26.10 e Node v24.13.0. A primeira tentativa dos testes Go no sandbox
foi impedida pela abertura de sockets HTTP locais. A rodada com `-race`
foi refeita fora do sandbox e passou nos nove pacotes. A CLI também recusou
a importação com `--activeState=fromJson` neste modo de implantação, antes
de modificar workflows; foi usada importação normal seguida de publicação
somente dos oito workflows originalmente ativos e recriação do serviço.

Comandos oficiais do n8n: `import:credentials` (conteúdo cifrado),
`import:workflow`, `publish:workflow` e `export:workflow`. Não foram
executados triggers ou workflows para verificar configuração.
A atualização manual inativa usou operações atômicas do MCP.

Apenas `n8n-aprova` foi recriado, sem dependências e preservando o volume.
API/worker não estavam em execução e não foram iniciados ou reiniciados.
PostgreSQL e Redis não foram recriados.

## Arquivos e migração

Configuração: `.env.example`, `docker-compose.yml`; `.env` local ignorado,
somente com acréscimos. Documentação: `README.md`, `workflows/n8n/README.md`,
`docs/central-slack-implementacao.md`, `docs/eventos-n8n.md` e este relatório.
Exports alterados: Central, consulta semanal, Digest, Core, feedback,
onboarding, assistente e Trello; o export OSV foi conferido e permaneceu igual.
Testes: ajustes de feedback/onboarding/Trello e novo
`workflows/n8n/tests/configuracao.test.cjs`.

O [mapeamento completo de migração](../../workflows/n8n/README.md#migração-e-credenciais)
registra marcadores anteriores e novas variáveis, nomes das credenciais,
consumidores, padrões, origem e recarga. Não houve renomeação de variável
existente do backend. Em outra instalação, importar os JSONs inativos,
preencher IDs próprios no ambiente e associar as credenciais próprias.

## Pendências humanas e limites

- Confirmar os vínculos Slack → GitHub e preencher `SLACK_GITHUB_VINCULOS`
  compartilhada pela API/n8n; nenhuma associação foi inferida.
- Confirmar que instalação GitHub, quadro/listas e canais reaproveitados são
  os destinos desejados além do ambiente de testes. Configuração existente
  não comprova autorização nem validação externa.
- Validar autenticação/permissões GitHub, scopes/destinos Slack/Trello,
  disponibilidade/cota Gemini e relatório real com a API/worker em execução.
- Definir a identidade definitiva do Check antes de preencher
  `APROVA_GITHUB_CHECK_NOME` / `APROVA_GITHUB_CHECK_APP_ID`.
- Conferir URL HTTPS pública/túnel e URLs cadastradas no Slack quando mudarem.
- Validar histórico Trello extenso em integração real e primeiro Digest;
  Digest permanece inativo. Não houve mensagens ou ações externas de teste.

Fora desta etapa: endpoint de feedback, executor de reexecução, gate,
calibração, recibos e publicação definitivos. Os bloqueios existentes foram
mantidos; testes isolados não representam validação dessas funcionalidades.

Backup e logs locais: `/tmp/aprova-padronizacao-20261108/`, diretório com
acesso restrito e fora do Git. Contém snapshot SQLite consistente, configuração
cifrada, cópia do ambiente, exports anteriores, export final bruto e logs
`n8n.tap` / `go.log`. Não compartilhar esse diretório; o backup é temporário
conforme a política de limpeza de `/tmp` da máquina.

# Home Slack — primeira etapa

Implementação local na branch `feat/slack-app-home`. Não houve publicação real
no Slack nem implantação nesta entrega.

## Decisões e contratos reaproveitados

Foram consultados o LEIA-ME e os 12 JSONs em
`design/slack/aprova-slack-visual`, `docs/eventos-n8n.md`,
`docs/central-slack-implementacao.md` e `workflows/n8n/README.md`, além dos
exports Central e Assistente. Os JSONs são referências visuais com dados
fictícios; o comando gera uma view nova, sem copiar seus exemplos.

O backend já fornece `RepositorioPostgres.ListarAnalises` e `FiltroAnalises`,
usados pelas rotas autenticadas de consulta. O comando usa essa mesma fonte
interna, sem passar pelo n8n ou criar outro protocolo HTTP de autenticação.
As rotas HTTP continuam exigindo HMAC. O operador do comando tem acesso ao
ambiente do backend e autoriza explicitamente um único usuário de teste a
consultar os dados da instalação configurada. Isto não implementa autorização
individual de acesso por repositório nem isolamento entre equipes.

São feitas duas leituras limitadas a seis registros: espera por aprovação e
todos os estados, na instalação `APROVA_CHAT_INSTALLATION_ID`. Exibem-se até
cinco por seção; o sexto sinaliza que há mais. As consultas não são um snapshot
transacional. Não se publicam totais, títulos de PR, risco, motivos, testes,
OSV ou recibos, pois não pertencem à projeção de leitura reaproveitada.
Estado de análise não confirma estado atual do PR nem atividade de worker.
Falha de qualquer consulta publica indisponibilidade em vez de zeros. Falha
na conexão inicial ao banco cancela o comando antes da publicação.

A ordem visual segue a proposta: identidade, conversa, acompanhamento,
pendências e últimas análises. Sem registros há estado vazio. O link de
conversa usa `app_redirect` com app e workspace configurados. A conversa em
DM permanece dependente da configuração existente e sem validação real
específica; a Home não habilita essa capacidade automaticamente.

## Executar

1. Use Go na versão de `go.mod`, PostgreSQL com as migrations atuais e a
   configuração normal de backend. `config.Carregar` carrega o `.env` local,
   sem sobrescrever variáveis já definidas no ambiente. `DATABASE_URL` e
   `REDIS_URL` seguem obrigatórios no carregador compartilhado, embora este
   comando não abra conexão Redis. Não aplique migrations em banco existente
   sem conferir a orientação de reconciliação em `workflows/n8n/README.md`.
2. No ambiente privado, configure os valores reais:

   | Variável | Uso |
   | --- | --- |
   | `APROVA_SLACK_BOT_TOKEN` | Token do mesmo bot usado nas credenciais Slack do n8n; não copiar para exports/logs |
   | `APROVA_SLACK_TEAM_ID` | Mesmo workspace autorizado na integração atual |
   | `APROVA_SLACK_APP_ID` | Mesmo App autorizado na integração atual |
   | `APROVA_CHAT_INSTALLATION_ID` | Instalação GitHub positiva, já usada pelo assistente |
   | `APROVA_SLACK_HOME_USER_ID` | ID de usuário de teste autorizado a ver os dados dessa instalação |

   O backend não extrai segredos das credenciais cifradas do n8n. Um operador
   deve fornecer o token ao processo por meio seguro. Não use argumento de
   linha de comando, `set -x` ou impressão de ambiente. Não inferir vínculo
   humano a partir do nome do usuário. Confira que o token pertence ao App
   configurado; `auth.test` confirma bot e workspace, mas não retorna app ID.
3. No painel do mesmo Slack App, habilite a aba **Home** em App Home.
   Preserve as URLs atuais de Event Subscriptions e Interactivity. Mantenha
   a configuração de Messages e os scopes da conversa existentes.
4. Na raiz do repositório, execute:

   ```sh
   go run ./cmd/slack-home
   ```

   O comando valida configuração, conecta ao PostgreSQL, chama `auth.test`,
   confere workspace e identidade de bot, consulta dados e chama
   `views.publish` para o único usuário configurado. Sucesso imprime somente
   uma confirmação. Erro encerra com código 1 e mensagem segura. Há timeout
   global de 45s e de 15s por chamada Slack, sem retry automático. Em resultado
   incerto, confira a Home antes de executar novamente. Uma nova execução
   substitui a Home desse usuário; ela não se atualiza automaticamente.
5. Abra o App no Slack com o usuário de teste e confira a aba Home, comparando
   IDs/estados com as consultas existentes. Confirme que dados ilustrativos e
   ações de decisão não aparecem. A API real ainda precisa dessa validação.

## Ações e pendências

A única navegação conectada é o link para a conversa, sem callback novo. Não
há botões de interação sem handler. `aprova_home_perguntas`, filtro,
atualizar, detalhes, aprovar, revisar, recibo, configurações, vincular e ajuda
foram omitidos. Atualização nesta etapa é repetir o comando. Dashboard e
configurações não têm destino autenticado confirmado e foram omitidos; não
se usa a URL provisória de recibos como dashboard. Não há URLs ilustrativas.

Aprovação/revisão exigem adaptador próprio da Home: resolver contexto atual,
validar identidade/vínculo/permissão/autoria/estado/repetição no núcleo e então
usar o contrato de `/acoes-remotas`. Não confiar em `value` ou
`private_metadata`. Recibos exigem consulta e verificação real; os modais
visuais não implementam isso. Filtros e refresh exigem roteamento de ações e
novas revisões de view. Eventos `app_home_opened`, distribuição entre equipes,
OAuth e Marketplace permanecem fora desta etapa. Workflows, notificações,
conversa, URLs de eventos e de interações existentes não foram alterados.

## Testar

```sh
go test ./internal/slackhome ./cmd/slack-home ./internal/api ./internal/config
node --test workflows/n8n/tests/*.test.cjs
```

Os testes de Slack usam servidor HTTP local: conferem destino, autenticação,
workspace divergente impedindo publicação, usuário/view enviados, `ok:false`,
HTTP 429, JSON inválido, ausência de retry, escopo e distinção entre vazio e
indisponível. Não precisam de tokens, banco ou Slack reais. Esses testes não
comprovam implantação, Home habilitada ou navegação em DM.

Referência: [views.publish](https://docs.slack.dev/reference/methods/views.publish/).
A API exige `user_id` e view Home, token no Authorization e conferência do
campo `ok` mesmo em HTTP 200.

Validação local desta entrega: passaram os pacotes `internal/slackhome`,
`internal/api` e `internal/config`; `cmd/slack-home` compilou (sem testes
próprios). Passaram os 18 arquivos da suíte de workflows. A execução exigiu
permissão de rede para obter dependências e sockets locais para os testes;
nenhuma chamada real ao Slack foi feita. `git diff --check` passou.

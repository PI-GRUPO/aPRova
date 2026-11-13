# Workflows do n8n

Guia de execução e configuração. Estado local e pendências atualizados em
**2026-11-08**; detalhes dos nós e contratos em
[Central do Slack](../../docs/central-slack-implementacao.md) e
[eventos do núcleo](../../docs/eventos-n8n.md).

## Inventário e diferenças de versão

| JSON disponível | Finalidade |
| --- | --- |
| `entrada-eventos.json` | HMAC, normalização, deduplicação e roteamento. |
| `central-slack.json` | Alertas, formulários e ações assinadas pela API. |
| `dependencias-osv.json` | Consulta pública por dependência e contexto informativo. |
| `trello-sugestao-movimentacao.json` | Comenta no card vinculado e replica no Slack, sem mover card. |
| `digest-semanal.json` | Relatório da última semana completa, segunda às 09h de São Paulo. |
| `slack-consulta-metricas.json` | DM e menção no canal autorizado; somente consultas. |
| `consulta-relatorio-semanal.json` | Ferramenta de GET assinado e validação de métricas. |
| `feedback-motivo-aprovacao.json` | Motivo de aprovação ALTO, contexto assinado e envio bloqueado até endpoint/vínculo reais. |
| `onboarding-repositorio.json` | Consulta de proteção por repositório, criação/reutilização de lista Trello e resumo Slack. |

JSONs são portáteis e inativos, sem segredos, credenciais pessoais ou dados
fixados. `../exemplos/` contém payloads de eventos, não workflows importáveis.

Instância e exports usam referências `APROVA_*` entre subworkflows.
A correção de paginação Trello com cursor `before` foi aplicada à instância
nesta etapa. Nós manuais de inspeção permanecem somente na instância;
o workflow manual de onboarding também não é exportado. Suas referências
de canal/quadro/subworkflow usam as mesmas variáveis e ele continua inativo.

O estado original foi preservado: oito workflows de produção ativos e
Digest inativo. Os nove exports continuam inativos. Backups, credenciais e
IDs locais ficam fora do versionamento. Novas sincronizações exigem revisão
das diferenças e autorização para alterar a instância.

## Executar localmente

Na raiz do projeto, com Docker e Compose:

```sh
mkdir -p .n8n
docker compose up -d n8n-aprova
```

Editor: <http://localhost:5978/>. A conta e as credenciais são próprias de cada
instalação; clonar o projeto não copia o n8n de outra pessoa.
O volume `.n8n/` e o `.env` são ignorados pelo Git. A pasta precisa permitir
escrita ao usuário `node` do container (UID/GID 1000:1000).
Não distribuir o banco SQLite, arquivo `config`, chave de criptografia,
exports de credenciais ou cópias do volume.

Para abrir somente o editor, o `.env` é opcional. Integrações exigem
configuração própria, conforme o
[README principal](../../README.md#n8n-opcional).

## Credenciais, ambiente e instalação

Slack (`slackApi`), Gemini (`googlePalmApi`) e Trello
(`trelloOAuth1Api`) ficam em **Credentials**. Segredos de assinatura,
endereços e variáveis lidas por `$env` ficam no ambiente do n8n, repassados
pelo Compose a partir do `.env`. Não inserir valores reais nos nós,
`.env.example` ou documentação.

| Configuração | Uso |
| --- | --- |
| `N8N_SEGREDO_EVENTOS` | Autenticar worker → Core; mesmo segredo no núcleo. |
| `N8N_SEGREDO_ACOES` | Assinar ações e consultas n8n → API. |
| `SLACK_SIGNING_SECRET` | Autenticar interações/eventos Slack. |
| `SLACK_INCOMING_WEBHOOK_URL` | Publicar notificações e Digest. |
| `APROVA_API_URL` | API acessível pelo container; host local normalmente `http://host.docker.internal:8080`. |
| `APROVA_CHAT_INSTALLATION_ID` | Instalação para consulta semanal e Digest da instância. |
| `APROVA_SLACK_TEAM_ID`, `APROVA_SLACK_APP_ID`, `APROVA_SLACK_CANAL_CONSULTAS_ID` | Workspace, App e canal autorizados. |
| `APROVA_CONSULTA_SEMANAL_WORKFLOW_ID` | Subworkflow usado pelo assistente. |
| `APROVA_SLACK_WORKFLOW_ID`, `APROVA_OSV_WORKFLOW_ID` | Referências portáteis das chamadas Central/OSV nos exports. |

Na instalação local, credenciais Slack/Gemini/Trello estão cadastradas;
segredos e IDs Slack estão carregados; instalação positiva já está no
`.env` e no container. App ID, chave privada válida, webhook secret e slug
GitHub estão no `.env` do núcleo. Falta validar autenticação/permissões no
GitHub real; não é mais pendência obter essas credenciais ou o installation ID.

`SLACK_GITHUB_VINCULOS` ainda precisa ser preenchido no `.env` local. O Compose
repassa essa variável ao n8n; o feedback usa exclusivamente a mesma fonte da API. Não presumir que ambiente da API é ambiente do n8n.

Ao alterar variáveis, recrie somente o serviço:

```sh
docker compose --profile automacao up -d --no-deps --force-recreate n8n-aprova
```

O volume é preservado. Alterar uma credencial no editor não exige recriação.
API e worker precisam ser iniciados/recarregados separadamente.
Na validação local do onboarding, `/health` e `/ready` responderam 200 a
partir do n8n. Isso não valida os endpoints de negócio ou o worker.

## Importar em outra instalação

1. Importe os JSONs pelo menu **Import from File**, incluindo
   `feedback-motivo-aprovacao.json` e `onboarding-repositorio.json` antes de
   configurar as chamadas.
2. Associe Slack aos formulários da Central e aos envios do assistente;
   Gemini ao modelo; Trello aos nós `04`, `07`, `10`, `13`.
3. Configure o ambiente acima. As credenciais GitHub são da API/worker;
   não copiá-las aos workflows.
4. Copie os IDs gerados pela importação para `APROVA_SLACK_WORKFLOW_ID`,
   `APROVA_OSV_WORKFLOW_ID`, `APROVA_TRELLO_WORKFLOW_ID`,
   `APROVA_FEEDBACK_WORKFLOW_ID`, `APROVA_ONBOARDING_WORKFLOW_ID` e
   `APROVA_CONSULTA_SEMANAL_WORKFLOW_ID`. As chamadas já usam expressões;
   preserve `each`, conexões e opções de espera. Associe `slackApi` aos nós
   `08`/`15` do feedback e mantenha a API de feedback desabilitada.
5. Crie a Data Table `aprova_osv_por_analise`, com colunas String
   `chave`/`contexto`, conforme `recursos/tabela-contexto-osv.json`.
6. Preencha quadro/listas `APROVA_TRELLO_*` e canal de onboarding.
   Associe Trello ao onboarding `10`/`13` e Slack ao `17`.
   Só habilite as flags de onboarding após conferir destinos e credenciais.
   Nome/App ID do Check continuam pendentes; não usar dados fictícios.
7. Recrie o n8n para carregar variáveis e configure as URLs públicas.
   Valide antes de publicar; subworkflows devem estar disponíveis antes do Core.

## Slack e consultas

Interatividade dos botões: HTTPS público com
`/webhook/aprova/slack/interacoes`, em **Interactivity & Shortcuts**.

Assistente: HTTPS público com `/webhook/aprova/slack/consultas`, em
**Event Subscriptions**, eventos `app_mention` e `message.im`.
Scopes: `app_mentions:read`, `im:history`, `chat:write`.
Habilitar Messages no App Home, convidar bot ao canal e reinstalar App
após mudanças de scopes. Conferir URLs sempre que o túnel mudar.

Na DM, envie texto sem menção; no canal permitido, mencione o bot.
Respostas são mensagens normais, sem thread. O modelo local selecionado é
`models/gemini-3.5-flash-lite`; disponibilidade/cota dependem da conta.
A memória é isolada por workspace, conversa, canal e usuário, e não fornece
métricas atuais. Deduplicação retém 10.000 chaves workspace/evento.

Ferramentas permitidas: relatório semanal, saúde e prontidão da API.
IA fornece somente `inicio` à consulta; instalação, destino, método e
assinatura ficam fora do seu controle. Não existem consultas de PR pendente
ou detalhe de execução, nem ferramentas de ação. Não há ramo de simulação.

Falta validar os endpoints de negócio com métricas reais e tratar falhas/cota
do modelo com aviso ao usuário. Para múltiplas equipes, falta autorização por
usuário/instalação e revisão de memória/retenção/concorrência.

## Digest semanal

Na instância e no export, `02 | Configurar Instalações` deriva a lista de:

```js
Number($env.APROVA_CHAT_INSTALLATION_ID || 0)
```

O valor fica no `.env`/container; JSON mantém apenas a referência.
Valor ausente, zero ou inválido bloqueia a consulta. Não preencher o ID real
diretamente no JSON.

Digest exige `APROVA_API_URL`, `N8N_SEGREDO_ACOES` e
`SLACK_INCOMING_WEBHOOK_URL`. Consulta sete dias UTC da última semana
completa, com fim exclusivo, e publica segunda às 09h de São Paulo.
Funciona sem evento ou agente; o envio de saída não exige ngrok.
API inválida/indisponível interrompe o envio, sem relatório fictício.

Falta validar relatório real e primeiro disparo; continua **desativado**.
Não há deduplicação nem recuperação de horários perdidos. Execução manual
envia ao canal; conferir antes de repetir após timeout.

## Core, OSV, Trello e feedback

Core aceita `POST /webhook/aprova/events`, com HMAC sobre corpo bruto,
retorna 401/400 para origem/contrato inválidos e 202 para entrada válida.
Deduplica tenant/evento antes de integrar. O 202 não confirma a integração.
O `tenant_id` autenticado preserva a instalação original; não deve ser
substituído pelo ID global das consultas.

`relatorio_periodico` aceita envelope v1 assinado, tenant e `dados.inicio`
opcional; consulta `GET /reports/weekly` e guarda resultado na execução,
sem publicar Digest. É pedido da automação, não evento emitido pelo worker.

OSV é público, não altera risco e guarda contexto por análise na Data Table.
Central reutiliza contexto por até 24h ou consulta dependências conhecidas
antes do alerta. Sem resultado completo, informa indisponibilidade/parcial.
Eventos fora de ordem não atualizam automaticamente alerta já publicado.
Falta emissão automática de dependências e contexto na review do núcleo.

Trello exige link completo da PR na descrição de um único card. Pesquisa,
revalida e comenta, sem mover. Marcador tenant/evento evita duplicidade
sequencial; não resolve concorrência. Falha Slack não desfaz comentário.
Cursor `before` aplicado na instância; falta definir recuperação de avisos.

Feedback recebe decisão aprovada de risco original
ALTO e mostra três motivos. Core o chama; Central encaminha cliques após
assinatura. Sem vínculo revisor Slack/GitHub, não pergunta. API de feedback
está desabilitada e não existe endpoint correspondente no núcleo.
Faltam cadastro de identidade, endpoint autorizado/idempotente, integração
com `approvals.feedback_reason` e o calibrador histórico do núcleo.
A persistência de feedback continua fora desta etapa.
Inventário e contrato proposto estão na
[documentação central](../../docs/central-slack-implementacao.md#feedback--motivo-da-aprovação).

Ações da Central passam pela API com vínculo Slack/GitHub e permissão real.
Conferir/aplicar as 21 migrations antes dos testes integrados; a 21 preserva
a justificativa remota, e a 15 pertence aos motivos do gate.
A reexecução ainda não tem executor. O worker já emite análise, espera de
aprovação, decisão, dependências identificadas e falha recuperável.
Gate/calibração, recibos e publicação reais vieram da `dev`; falta o evento
automático de instalação e a validação externa do conjunto.
SLA com dois limites configuráveis ainda não tem workflow.

## Onboarding de Repositório

`instalacao` segue pelo Core `06 → 07K → 07L`, após assinatura, normalização
e deduplicação. O subworkflow recebe `meta`, `evento` e `instalacao`, aceita
`instalado`/`repositorios_adicionados`, separa a lista de repositórios e
processa um por vez. A branch é a padrão consultada no GitHub.

Consulta somente leitura da proteção clássica, com três resultados:
`protecao_confirmada`, `check_obrigatorio_pendente` e
`verificacao_indisponivel`. Só o primeiro habilita prontidão. O adaptador
atual não autentica no GitHub e não cobre rulesets; no teste real, a branch
foi identificada e a proteção retornou 401. Não existe endpoint equivalente
na API. A integração deverá ganhar leitura autenticada e a identidade exata
do Check do produto (`aPRova`), publicado pelo backend real.

No Trello, cria ou reutiliza lista `owner/repo` no quadro configurado.
Não cria cards, move cards ou comenta. Arquivamento, nomes ambíguos e falhas
de leitura impedem criar outra lista. O Slack recebe branch, estado da
proteção, situação Trello e ação necessária, pela credencial do bot.

Validados localmente: evento assinado até Core/onboarding, identificação de
branch, lista criada/reutilizada no único quadro de teste, envio confirmado
ao único canal utilizado nos testes e 25 cenários isolados. O workflow de
testes não foi exportado; fixtures de proteção não são confirmação real.
Os testes do export podem ser executados sem rede:

```sh
node --test workflows/n8n/tests/onboarding-repositorio.test.cjs
```

Faltam emissão automática do evento pelo backend, publicação definitiva do
Check, autenticação de leitura, cobertura de rulesets/bypasses, mapeamento
por equipe, exclusão de criações concorrentes e recuperação parcial.
Reprocessamento sequencial reutiliza a lista, mas a criação não tem trava
entre execuções; deduplicação do Core não recupera uma notificação perdida.
Detalhes de nós, evidência e configuração estão na
[documentação central](../../docs/central-slack-implementacao.md#onboarding-de-repositório).

## Verificação e produção

Para validar exports sem rede, com Node 22+:

```sh
node --test workflows/n8n/tests/*.test.cjs
```

Os testes dos JSONs cobrem a configuração do Digest pelo ambiente e os
bloqueios/assinaturas do feedback, sem chamadas externas. Não validam
automaticamente configurações e dados reais de cada instalação.
Testes funcionais Go usam GitHub simulado e não comprovam PR real.
Para testes integrados, disponibilizar API/worker com ambiente atualizado,
banco/migrations e repositório de teste. Não tratar simulação como métrica real.

Produção exige endpoints HTTPS estáveis, rede entre serviços, implantação da
API/worker, credenciais próprias, retenção, backups do volume/chave de
criptografia e recuperação de falhas. O Compose é de desenvolvimento; CI não
importa workflows nem faz deploy do n8n.

## Exportar quando autorizado

Baixe uma cópia do JSON e revise antes de versionar: remover segredos, dados
fixados, referências pessoais e valores locais; manter leitura de ambiente
para instalação/endereço/segredos e configurar chamadas na instalação alvo.
Comparar correções locais e preservar o cursor `before` do Trello.
Nunca incluir `.env`, credenciais ou conteúdo de `.n8n/`.

Os exports desta revisão foram autorizados; futuras sincronizações também
exigem autorização. Não sobrescrever correções com versões antigas de qualquer origem.

## Contrato de configuração — 2026-11-08

Valores vazios não autorizam publicar integrações. IDs vêm da instalação de
destino, nunca dos exports de outra pessoa. Recriar somente `n8n-aprova`
após mudar seu ambiente; preservar `.n8n/`. API/worker leem o ambiente ao
iniciar: reiniciar apenas o processo afetado se já estiver em execução.
O `.env` existente foi preservado e recebeu somente configurações com
correspondência inequívoca na instância. Nenhum vínculo de pessoa foi inferido.

| Variável | Consumidor / finalidade | Padrão / obrigatoriedade | Origem | Recarga |
| --- | --- | --- | --- | --- |
| `N8N_URL_BASE` | Compose → editor (`N8N_EDITOR_BASE_URL`) | `http://localhost:5978/`, opcional | URL do editor | n8n |
| `N8N_WEBHOOK_URL_PUBLICA` | Compose → `WEBHOOK_URL` do n8n | `N8N_URL_BASE`, opcional | HTTPS público/túnel para Slack | n8n |
| `N8N_WEBHOOK_URL` | Go e emissão de eventos | vazio desliga; completo com caminho do Core | Webhook de produção do Core | API/worker emissor |
| `N8N_PORTA_LOCAL` | Compose / porta do host | `5978` | Operador local | n8n |
| `GENERIC_TIMEZONE` | n8n / agendamentos | `America/Sao_Paulo` | Fuso da equipe | n8n |
| `APROVA_API_URL` | Workflows → API | `http://host.docker.internal:8080`; necessário para consultas/ações | Rede da API | n8n |
| `APROVA_CHAT_INSTALLATION_ID` | Consulta semanal e Digest | `0` bloqueia; positivo obrigatório para consultas | Instalação GitHub App autorizada | n8n |
| `APROVA_SLACK_WORKFLOW_ID` | Core → Central | vazio; necessário nessa rota | ID da importação Central | n8n |
| `APROVA_OSV_WORKFLOW_ID` | Core/Central → OSV | vazio; necessário nessa rota | ID da importação OSV | n8n |
| `APROVA_TRELLO_WORKFLOW_ID` | Core → Trello | vazio; necessário nessa rota | ID da importação Trello | n8n |
| `APROVA_FEEDBACK_WORKFLOW_ID` | Core/Central → feedback | vazio; necessário nessa rota | ID da importação feedback | n8n |
| `APROVA_ONBOARDING_WORKFLOW_ID` | Core → onboarding | vazio; necessário nessa rota | ID da importação onboarding | n8n |
| `APROVA_CONSULTA_SEMANAL_WORKFLOW_ID` | Assistente → relatório | vazio; necessário para ferramenta | ID da importação consulta | n8n |
| `APROVA_SLACK_TEAM_ID` | Assistente/feedback: workspace permitido | vazio bloqueia | Slack workspace | n8n |
| `APROVA_SLACK_APP_ID` | Assistente/feedback: aplicativo permitido | vazio bloqueia | Slack App | n8n |
| `APROVA_SLACK_CANAL_CONSULTAS_ID` | Assistente/feedback: canal permitido | vazio bloqueia | Slack canal da equipe | n8n |
| `APROVA_SLACK_CANAL_ONBOARDING_ID` | Onboarding: resumo | vazio; obrigatório se habilitado | Canal já usado pelo onboarding local | n8n |
| `APROVA_TRELLO_QUADRO_ID` | Trello/onboarding: quadro | vazio; obrigatório para Trello | Quadro existente | n8n |
| `APROVA_TRELLO_QUADRO_NOME` | Trello/onboarding: rótulo | `Quadro da equipe` no Trello; vazio no onboarding | Nome do quadro existente | n8n |
| `APROVA_ONBOARDING_TRELLO_HABILITADO` | Onboarding: habilitar Trello | `false`; somente `true` habilita | Decisão do operador; estado local reaproveitado | n8n |
| `APROVA_ONBOARDING_SLACK_HABILITADO` | Onboarding: habilitar Slack | `false`; somente `true` habilita | Decisão do operador; estado local reaproveitado | n8n |
| `APROVA_GITHUB_CHECK_NOME` | Onboarding: identidade do Check | vazio impede confirmação de proteção | Nome do Check do backend (`aPRova`); confirmar no ambiente | n8n |
| `APROVA_GITHUB_CHECK_APP_ID` | Onboarding: App do Check | vazio → `null`, pendente | App que publica o Check; não é installation ID | n8n |
| `SLACK_GITHUB_VINCULOS` | API e feedback: identidade compartilhada | vazio sem vínculos; `USUARIO_SLACK:login_github,...` | Confirmação humana das identidades | API e n8n |
| `N8N_SEGREDO_EVENTOS` | Go → Core, HMAC | vazio; obrigatório quando emissão habilitada | Segredo compartilhado por direção | Emissor e n8n |
| `N8N_SEGREDO_ACOES` | n8n → API, HMAC | vazio; obrigatório para ações/consultas | Segredo compartilhado por direção | API e n8n |
| `SLACK_SIGNING_SECRET` | Central/assistente: autenticar Slack | vazio bloqueia | Slack App / Basic Information | n8n |
| `SLACK_INCOMING_WEBHOOK_URL` | Alertas, OSV, Trello e Digest | vazio impede envio | Incoming Webhook do Slack (segredo) | n8n |
| `NODE_FUNCTION_ALLOW_BUILTIN` | Compose → Code | `crypto`, fixo no Compose | Requisito HMAC | n8n |
| `N8N_BLOCK_ENV_ACCESS_IN_NODE` | Compose → Code | `false`, fixo no Compose | Permitir leitura de `$env` | n8n |
| `APROVA_TRELLO_LISTA_BACKLOG_ID` | Trello: lista Backlog | vazio; sete IDs obrigatórios | Lista existente no quadro | n8n |
| `APROVA_TRELLO_LISTA_BACKLOG_NOME` | Trello: rótulo da lista | `Backlog`, opcional | Nome da lista existente | n8n |
| `APROVA_TRELLO_LISTA_DESIGN_ID` | Trello: lista Design | vazio; sete IDs obrigatórios | Lista existente no quadro | n8n |
| `APROVA_TRELLO_LISTA_DESIGN_NOME` | Trello: rótulo da lista | `Design`, opcional | Nome da lista existente | n8n |
| `APROVA_TRELLO_LISTA_A_FAZER_ID` | Trello: lista A Fazer | vazio; sete IDs obrigatórios | Lista existente no quadro | n8n |
| `APROVA_TRELLO_LISTA_A_FAZER_NOME` | Trello: rótulo da lista | `A Fazer`, opcional | Nome da lista existente | n8n |
| `APROVA_TRELLO_LISTA_EM_ANDAMENTO_ID` | Trello: lista Em andamento | vazio; sete IDs obrigatórios | Lista existente no quadro | n8n |
| `APROVA_TRELLO_LISTA_EM_ANDAMENTO_NOME` | Trello: rótulo da lista | `Em andamento`, opcional | Nome da lista existente | n8n |
| `APROVA_TRELLO_LISTA_REVISAO_ID` | Trello: lista Revisão de código | vazio; sete IDs obrigatórios | Lista existente no quadro | n8n |
| `APROVA_TRELLO_LISTA_REVISAO_NOME` | Trello: rótulo da lista | `Revisão de código`, opcional | Nome da lista existente | n8n |
| `APROVA_TRELLO_LISTA_TESTE_ID` | Trello: lista Fase de teste | vazio; sete IDs obrigatórios | Lista existente no quadro | n8n |
| `APROVA_TRELLO_LISTA_TESTE_NOME` | Trello: rótulo da lista | `Fase de teste`, opcional | Nome da lista existente | n8n |
| `APROVA_TRELLO_LISTA_CONCLUIDO_ID` | Trello: lista Concluído | vazio; sete IDs obrigatórios | Lista existente no quadro | n8n |
| `APROVA_TRELLO_LISTA_CONCLUIDO_NOME` | Trello: rótulo da lista | `Concluído`, opcional | Nome da lista existente | n8n |

### Migração e credenciais

| Anterior | Atual |
| --- | --- |
| `CONFIGURE_WORKFLOW_TRELLO` | `$env.APROVA_TRELLO_WORKFLOW_ID` |
| `CONFIGURE_WORKFLOW_FEEDBACK` | `$env.APROVA_FEEDBACK_WORKFLOW_ID` |
| `CONFIGURE_WORKFLOW_ONBOARDING` | `$env.APROVA_ONBOARDING_WORKFLOW_ID` |
| Seleções locais Central/OSV/consulta semanal | Variáveis `APROVA_*_WORKFLOW_ID` já existentes |
| `CONFIGURE_ID_QUADRO_TRELLO`, quadro fixo nos nós | `APROVA_TRELLO_QUADRO_ID` / `APROVA_TRELLO_QUADRO_NOME` |
| `CONFIGURE_ID_LISTA_*`, listas fixas | `APROVA_TRELLO_LISTA_*_ID` / `APROVA_TRELLO_LISTA_*_NOME` |
| `CONFIGURE_ID_CANAL_SLACK`, canal fixo do onboarding | `APROVA_SLACK_CANAL_ONBOARDING_ID` |
| Flags/check fixos no onboarding | `APROVA_ONBOARDING_*_HABILITADO`, `APROVA_GITHUB_CHECK_*` |
| `vinculosLocais` no feedback | Somente `SLACK_GITHUB_VINCULOS` |
| Compose: `N8N_WEBHOOK_URL` recebia base do editor | `WEBHOOK_URL` recebe `N8N_WEBHOOK_URL_PUBLICA` com fallback para `N8N_URL_BASE` |

Nenhuma variável existente foi renomeada no backend ou nos scripts; seus
consumidores mantêm os contratos. Ao migrar, copiar as configurações válidas,
substituir IDs pelas importações locais e recriar o n8n. Não copiar credenciais
ou banco entre pessoas.

Convenção de exibição: **`aPRova | Serviço`**. As credenciais existentes
`Slack account`, `Trello account` e `Google Gemini(PaLM) Api account 4`
passaram a `aPRova | Slack`, `aPRova | Trello`, `aPRova | Gemini`.
Só as três usadas pelo aPRova foram alteradas; IDs, dados cifrados e vínculos
foram preservados. As demais credenciais ficaram intactas.

Em outra instalação, cadastrar/selecionar credenciais próprias dos tipos
`slackApi`, `trelloOAuth1Api` e `googlePalmApi`, usando esses nomes de exibição,
e associá-las aos nós indicados na importação. Formulários HTTP da Central
usam `slackApi` como credencial predefinida; não inserir Authorization manual.
Renomear credenciais não exige recriar serviço. Segredos OAuth, tokens e
senhas permanecem no armazenamento cifrado do n8n. HMAC/Signing Secret são
segredos de ambiente; IDs de workflow, instalação, quadro, lista e canal são
configuração sem autenticação própria. O Incoming Webhook é URL secreta.

### Demais configurações aceitas pelo projeto

Esta tabela complementa o contrato do n8n e o `.env.example`. Valores são
lidos ao iniciar o consumidor; scripts leem a cada execução. Não reiniciar
PostgreSQL/Redis nesta migração: suas portas não foram alteradas.

| Variável | Consumidor / finalidade | Padrão | Obrigatoriedade / origem | Recarga |
| --- | --- | --- | --- | --- |
| `PORT` | API / porta HTTP | `8080` | opcional; Operador | API |
| `LOG_LEVEL` | API/worker / nível de log | `info` | opcional; Operador | API/worker |
| `APROVA_URL_PUBLICA` | Go e script emitir-evento / links de recibo | `https://aprova.example.com` | opcional, provisório; URL pública do backend | API/worker/script |
| `DATABASE_URL` | Go / PostgreSQL | `sem padrão no Go` | obrigatório; Compose ou provedor PostgreSQL; contém senha | API/worker |
| `REDIS_URL` | Go / Redis | `sem padrão no Go` | obrigatório; Compose ou provedor Redis | API/worker |
| `GITHUB_APP_ID` | Go / GitHub App | `vazio` | opcional no loader; necessário ao integrar GitHub; GitHub App settings | API/worker |
| `GITHUB_PRIVATE_KEY_BASE64` | Go / chave privada GitHub | `vazio` | opcional no loader; necessário ao worker; Chave PEM do GitHub App em base64; segredo | API/worker |
| `GITHUB_WEBHOOK_SECRET` | Go / autenticar webhook GitHub | `vazio` | opcional no loader; necessário ao webhook; GitHub App webhook secret | API |
| `GITHUB_APP_SLUG` | Go / identificação do App | `vazio` | opcional; GitHub App settings | API/worker |
| `APROVA_CHAVE_RECIBO` | Go / assinar recibo Ed25519 | `vazio` | semente de 32 bytes em base64; obrigatória ao iniciar worker; segredo | API/worker |
| `APROVA_CHAVES_PUBLICAS_RECIBO` | Go / conferir recibos Ed25519 | `vazio` | chaves públicas em base64, separadas por vírgula; permite conferir recibos antigos | API/worker |
| `SANDBOX_EXECUTOR` | Go / executor do sandbox | `vazio no loader` | obrigatório ao iniciar worker (local ou azure); local ou azure conforme implantação | worker |
| `SANDBOX_IMAGEM_GO` | Go / imagem do executor | `vazio` | necessário ao executor local; Imagem construída/publicada pela equipe | worker |
| `SANDBOX_DIRETORIOS` | Go / subdiretórios por repositório | `vazio` | opcional; dono/repo=diretorio_relativo separados por vírgula | worker |
| `WORKER_CONCORRENCIA` | Go / paralelismo | `4` | opcional; Inteiro positivo definido pelo operador | worker |
| `WORKER_MAX_TENTATIVAS` | Go / limite de tentativas | `3` | opcional; Inteiro positivo definido pelo operador | worker |
| `ANTHROPIC_API_KEY` | Go / configuração futura | `vazio` | opcional; funcionalidade futura; Provedor Anthropic; segredo | API/worker |
| `GCP_PROJECT_ID` | Go / configuração reservada GCP | `vazio` | opcional; Projeto GCP | API/worker |
| `GCP_REGION` | Go / região reservada GCP | `us-central1` | opcional; Operador GCP | API/worker |
| `POSTGRES_PORT` | Compose / publicar PostgreSQL | `5432` | opcional; Porta local | postgres |
| `REDIS_PORT` | Compose / publicar Redis | `6379` | opcional; Porta local | redis |
| `AZURE_SUBSCRIPTION_ID` | Go / Azure assinatura | `vazio` | necessário com executor azure; Azure / implantação do sandbox | worker |
| `AZURE_TENANT_ID` | Go / Azure tenant | `vazio` | necessário com executor azure; Azure / implantação do sandbox | worker |
| `AZURE_CLIENT_ID` | Go / Azure cliente | `vazio` | necessário com executor azure; Azure / implantação do sandbox | worker |
| `AZURE_CLIENT_SECRET` | Go / Azure segredo do cliente | `vazio` | necessário com executor azure; Azure / implantação do sandbox | worker |
| `AZURE_RESOURCE_GROUP` | Go / Azure grupo de recursos | `vazio` | necessário com executor azure; Azure / implantação do sandbox | worker |
| `AZURE_SANDBOX_JOB_NAME` | Go / Azure job do sandbox | `vazio` | necessário com executor azure; Azure / implantação do sandbox | worker |


## Integração com a dev — 2026-11-13

O módulo Go é `github.com/PI-GRUPO/aPRova`, com toolchain Go 1.26.11.
O núcleo agora inclui sandbox Python, gate determinístico, recibos Ed25519,
publicação idempotente no GitHub e calibração opcional por histórico.
Essas implementações não substituem a validação do n8n com serviços reais.

| Variável incorporada da dev | Consumidor e configuração | Recarga |
| --- | --- | --- |
| `SANDBOX_IMAGEM_PYTHON` | worker; imagem construída para executar projetos Python | worker |
| `SANDBOX_LINGUAGENS` | worker; pares `dono/repo=python` ou `dono/repo=go`; padrão Go | worker |
| `GATE_MAX_ARQUIVOS` | worker; padrão 20, limite da regra de tamanho | worker |
| `GATE_CAMINHOS_SENSIVEIS` | worker; configuração de caminhos do gate; ver `README.md`, seção Gate de risco | worker |
| `CALIBRACAO_LIGADA` | worker; `false` por padrão | worker |
| `CALIBRACAO_VIZINHOS` | worker; padrão 10 | worker |
| `CALIBRACAO_SIMILARIDADE_MINIMA` | worker; padrão 0.97 | worker |
| `CALIBRACAO_MINIMO_DE_DESFECHOS` | worker; padrão 5 | worker |
| `CALIBRACAO_TIMEOUT` | worker; padrão 5s | worker |
| `GEMINI_API_KEY` | worker; segredo do provedor de embeddings quando a calibração estiver ligada | worker |

`GEMINI_API_KEY` pertence ao backend e não substitui a credencial Gemini do
assistente no n8n. As chaves `RECEIPT_PRIVATE_KEY_BASE64` e
`RECEIPT_PUBLIC_KEY_BASE64` foram substituídas por `APROVA_CHAVE_RECIBO` e
`APROVA_CHAVES_PUBLICAS_RECIBO`. A nova chave privada recebe a semente de
32 bytes em base64; não copiar uma chave expandida de 64 bytes nesse campo.
Preservar as chaves públicas históricas e não gerar outra identidade de
assinatura inadvertidamente. O `.env` local não é alterado pelo merge.

O esquema integrado tem 21 migrations. A justificativa remota passou de
`000015_preservar_justificativa_acao_remota` para `000021`, pois a `dev` já
usa a versão 15 para motivos do gate e as versões 16–20 para outras funções.
Em banco limpo ou no esquema original da `dev`, seguir as migrations normais.
**Se o banco já aplicou a antiga migration 15 de workflows-n8n, não executar
`migrate up` nem `force` sem reconciliar o esquema e o histórico primeiro.**
O número 15 sozinho não identifica qual alteração foi aplicada. Conferir
`schema_migrations`, a coluna `remote_action_attempts.justification` e as
colunas do gate; preservar os dados e preparar uma migração específica ou
usar um banco de testes novo. Esta integração não modifica o banco local.

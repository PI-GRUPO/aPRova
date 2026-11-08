# Central de operação aPRova no Slack

Atualizado em **2026-11-08**, horário de São Paulo. Este documento reúne as
explicações e pendências atuais. Registros antigos de implementação, testes
e simulações foram consolidados.

## Orientação para manutenção

Antes de construir ou alterar um workflow, leia este documento e
[eventos-n8n.md](eventos-n8n.md). Mantenha por integração: finalidade,
inventário dos nós/funções, contratos do núcleo, configuração e pendências.
Atualize a data e substitua informações superadas, sem acumular diários de
implementação ou apagar as explicações das demais integrações.

Diferencie código disponível, configuração, validação real e implantação.
Não registre segredos, valores de credenciais ou IDs do ambiente. Contratos
e autorização permanecem independentes do ambiente. Respeite o escopo
autorizado: atualizar documentação não autoriza exportar workflows.

## Arquitetura e estado atual

O núcleo decide o risco, persiste os dados e autoriza ações. O n8n autentica
eventos, consulta endpoints e comunica pessoas, sem acessar o PostgreSQL do
núcleo nem usar IA para decidir aprovações.

| Workflow | Estado local | Responsabilidade |
| --- | --- | --- |
| Core — Entrada e Roteamento de Eventos | Publicado | Autenticação, normalização, deduplicação e roteamento. |
| Slack — Central de Operação | Publicado | Alertas, formulários, ações remotas e encaminhamento de feedback. |
| Dependências — Consulta OSV | Publicado | Consulta pública e contexto informativo por análise. |
| Trello — Sugestão de Movimentação | Publicado | Comentário no card vinculado e réplica Slack; nunca move cards. |
| Digest — Resumo Semanal | **Desativado** | Consulta agendada e resumo no Slack. |
| Slack — Consulta de Métricas | Publicado | DM contínua e resposta a menção no canal autorizado; somente leitura. |
| Consulta — Relatório Semanal | Publicado | Ferramenta assinada de leitura do relatório. |
| Feedback — Motivo da Aprovação | Publicado, bloqueado por configuração | Pergunta de motivo para aprovação ALTO; sem vínculo não pergunta; envio à API desabilitado. |
| Onboarding de Repositório | Publicado, proteção ainda indisponível | Identifica branch, consulta proteção, cria/reutiliza lista Trello e resume no Slack. |

Publicação local não comprova funcionamento integral com GitHub real ou
produção. Na validação do onboarding, `/health` e `/ready` responderam 200
a partir do n8n, superando a indisponibilidade registrada anteriormente.
Isso não valida os endpoints de negócio, o worker ou as métricas reais.
Indisponibilidade não deve ser apresentada como métricas zeradas.

### Credenciais e ambiente

| Configuração | Situação conferida e função |
| --- | --- |
| `GITHUB_APP_ID` e `GITHUB_PRIVATE_KEY_BASE64` | Presentes no `.env`; ID positivo e chave decodificada/validada localmente. Falta testar autenticação efetiva no GitHub. |
| `GITHUB_WEBHOOK_SECRET`, `GITHUB_APP_SLUG` | Presentes no `.env`; conferir correspondência com App/webhooks reais. |
| `APROVA_CHAT_INSTALLATION_ID` | ID informado pelo usuário, positivo no `.env` e carregado no container. Consulta semanal e Digest da instância usam a variável. Falta validar associação/permissões reais no GitHub. |
| `slackApi`, `googlePalmApi`, `trelloOAuth1Api` | Credenciais cadastradas no n8n; seus valores/referências pessoais ficam fora dos exports. |
| `SLACK_SIGNING_SECRET`, `SLACK_INCOMING_WEBHOOK_URL` | Carregados no n8n; autenticação de entradas e envio ao canal de notificações. |
| `N8N_SEGREDO_EVENTOS`, `N8N_SEGREDO_ACOES` | Carregados no n8n; devem coincidir com o núcleo em cada direção. |
| `APROVA_SLACK_TEAM_ID`, `APROVA_SLACK_APP_ID`, `APROVA_SLACK_CANAL_CONSULTAS_ID` | Carregados; restringem assistente e contexto do feedback. |
| `APROVA_CONSULTA_SEMANAL_WORKFLOW_ID` | Configurado para a ferramenta do assistente. |
| `APROVA_API_URL` | Configurada; `/health` e `/ready` responderam 200 no teste do onboarding. Endpoints de negócio ainda exigem validação própria. |
| `N8N_WEBHOOK_URL` | Preenchida no `.env` do núcleo; conferir URL e carregar no worker ao iniciá-lo. |
| `SLACK_GITHUB_VINCULOS` | **Não configurado** no `.env` e ausente no ambiente do n8n. Necessário na API para ações; feedback exige configuração adicional abaixo. |
| `APROVA_URL_PUBLICA` | Não configurada; recibos públicos também dependem de endpoint ainda ausente. |

O `.env` é ignorado pelo Git; `.env.example` contém somente exemplos.
O Compose repassa a instalação para `n8n-aprova`. Os workflows usam
`$env`, sem guardar o ID real nos parâmetros. Após alterar o ambiente,
recrie o serviço preservando o volume `.n8n/`:

```sh
docker compose --profile automacao up -d --no-deps --force-recreate n8n-aprova
```

API e worker são processos separados: recriar o n8n não os inicia nem
recarrega seus ambientes. As credenciais GitHub ficam no núcleo.

### Diferenças entre instância e repositório

Os nove JSONs de workflows em `workflows/n8n/` são portáteis e inativos. O feedback e as
atualizações de Core, Central e Digest estão sincronizados com a instância,
com remoção de credenciais/referências locais e preservação dos bloqueios:

| Diferença | Situação |
| --- | --- |
| Feedback | `feedback-motivo-aprovacao.json`, sem credenciais ou IDs locais. |
| Core e Central | Chamadas ao feedback por `CONFIGURE_WORKFLOW_FEEDBACK`; selecionar a importação nos dois workflows. |
| Digest | Instância e export leem `APROVA_CHAT_INSTALLATION_ID`, sem valor real no JSON. |
| Trello | Export tem cursor `before` para comentários; instância ainda usa `page`. |
| Onboarding | `onboarding-repositorio.json`, baseado na versão publicada, sem credenciais ou destinos locais; Trello/Slack desabilitados até configurar a importação. |
| Core → onboarding | Nó `07L`, por item e aguardando retorno; selecionar a importação em `CONFIGURE_WORKFLOW_ONBOARDING`. Demais rotas preservadas. |

Os exports desta revisão foram autorizados. Novas sincronizações exigem
autorização; preservar a correção Trello e a portabilidade. Na importação,
associar credenciais Slack aos nós de publicação e aviso do feedback, selecionar
o subworkflow no Core/Central e manter a API de feedback desabilitada até
existir endpoint real. Esta exportação não alterou a instância.
O workflow manual `aPRova | Testes | Onboarding de Repositório` permanece
somente na instância, fora da rota de eventos e fora dos exports.

## Core — Entrada e Roteamento de Eventos

Recebe `POST /webhook/aprova/events`, autentica HMAC do corpo bruto
(janela de cinco minutos), valida e normaliza o envelope v1. Respostas:
401 para origem inválida, 400 para contrato inválido e 202 para entrada
aceita. O 202 não comprova sucesso das integrações posteriores.

| Nós | Função |
| --- | --- |
| `01`, `01B`, `01E` | Webhook Raw Body, assinatura e recusa. |
| `02–04b`, `05E` | Preservação do payload, normalização e validação. |
| `05`, `05B`, `06` | Confirmação, deduplicação tenant/evento (10.000 chaves) e roteamento. |
| `07A`, `07J`, `08` | Aprovação pendente/falha recuperável para Central. |
| `07G`, `09` | Análise concluída para Trello. |
| `07I dependencia_nova` e chamada OSV | Dependências para consulta pública. |
| `07H`, `07I Solicitar Motivo da Aprovação` | Decisão registrada para feedback, por item e sem aguardar. |
| `07E`, `10–12` | Pedido assinado `relatorio_periodico`: GET do relatório e resultado na execução. |
| `07K`, `07L` | Saída `instalacao` chama Onboarding de Repositório, modo `each`, aguardando o resultado. |
| Demais saídas | Pontos de extensão e registro sem integração. |

O tenant autenticado define a instalação do evento, sem substituição pelo
ID global de consultas. `relatorio_periodico` é pedido da automação,
não um sétimo evento emitido pelo worker, e não publica Digest.

Falta validar worker → Core com dados reais; implementar emissão de
`dependencia_nova` e `instalacao`; e construir SLA com limites
configuráveis, encerramento ao receber decisão e atualização do alerta.

## Onboarding de Repositório

Prepara cada repositório incluído na instalação do GitHub App e comunica
o que foi configurado e o que exige ação humana. Instalar o App não torna
seu Check obrigatório para merge; a consulta torna essa diferença visível.
Não analisa PRs, decide risco, aprova alterações ou altera proteção no GitHub.

### Entrada e execução

O fluxo funcional é `aPRova | Onboarding de Repositório`. Recebe somente
o evento já autenticado e normalizado pelo Core, pela saída `instalacao`
do nó `06`, passando por `07K` e `07L`. Não possui webhook próprio ou
agendamento. O chamador usa `mode=each` e `waitForSubWorkflow=true`.

Consome `meta.origem_verificada`, `meta.validacao.valido`,
`meta.versao_evento`, `meta.tenant_id`, `meta.id_evento`, `evento.tipo`,
`evento.acao` e `instalacao.{acao,conta,repositorios}`. Não espera o envelope
bruto em `dados` como entrada principal. Aceita `instalado` e
`repositorios_adicionados`; remoções e outros tipos encerram sem efeitos.
Valida a lista inteira antes de agir, elimina nomes exatamente repetidos
e processa um repositório por vez com lote de tamanho 1.

| Nós | Função |
| --- | --- |
| `01–03` | Recebem o evento, validam origem/contrato, separam repositórios e iniciam o processamento sequencial. |
| `04–06` | GET do repositório no GitHub; usam sua `default_branch` como destino. Sem resposta válida, branch indisponível, sem presumir `main`. |
| `07–08` | GET da proteção clássica da branch e interpretação conservadora da evidência real. |
| `09–12` | Se Trello habilitado, leem quadro/listas e decidem entre reutilização, criação ou pendência. |
| `13–14` | Criam a lista somente quando ausente e conferem a resposta do Trello. |
| `15–18` | Montam resumo, enviam pela credencial `slackApi` ao canal configurado e conferem canal/timestamp retornados. |
| `19–20` | Retornam resultado por repositório, avançam o lote e consolidam os resultados do evento. |

### Proteção e prontidão

As leituras são GETs em `/repos/{owner}/{repo}` e
`/repos/{owner}/{repo}/branches/{branch}/protection`, com timeout de 10s e
sem seguir redirects. O adaptador atual consulta proteção clássica; não
valida regras de rulesets. Não cria regras nem torna checks obrigatórios.

| `protecao.status` | Condição |
| --- | --- |
| `protecao_confirmada` | HTTP 200, branch conhecida, nome exato do Check e App ID configurados, correspondência em `required_status_checks.checks` e `enforce_admins.enabled=true`. |
| `check_obrigatorio_pendente` | Consulta válida com identidade do Check conhecida, mas Check esperado ausente ou proteção sem aplicação aos administradores. |
| `verificacao_indisponivel` | Falta autenticação, branch, identidade do Check ou resposta completa; inclui erros HTTP e evidência simulada recusada. |

Somente `protecao_confirmada` retorna `pronto_fluxo_protegido=true`.
HTTP 401/403/404 não prova que a proteção esteja ausente. O status deve ser
interpretado antes de campos auxiliares como `check_obrigatorio`.
Nome e App ID ficam vazios no nó `02` enquanto o publicador do backend for
provisório; os checks de CI `verificacao`, `sandbox` e `banco` não definem
a identidade do Check do produto aPRova.

O nó `08` recebe evidência com `branch`, `status_http`, `protecao` e
`simulado`. Não aceita resultado de proteção informado pelo evento.
As simulações ficaram no workflow manual de testes; o funcional utiliza
somente as respostas das próprias leituras e recusa evidência simulada.
Uma futura consulta autenticada pelo backend deve preservar essa estrutura
no adaptador ou mapear a resposta antes da classificação. Ainda não existe
endpoint de onboarding/proteção na API; não usar `/acoes-remotas` para isso.

### Trello, Slack e configuração

No nó `02`, `trello_habilitado`, `quadro_id` e `quadro_nome` configuram
o destino. O fluxo exige quadro aberto e correspondência de ID e nome;
procura lista pelo `owner/repo`, comparando o nome sem distinguir maiúsculas.
Lista única aberta é reutilizada; lista ausente é criada no final do quadro.
Lista arquivada ou nome ambíguo gera pendência sem criar duplicata.
Resposta de leitura inválida não autoriza criação.

O onboarding apenas cria/reutiliza listas: não cria cards, não move cards
nem adiciona comentários. A lista oferece um espaço de organização por
repositório, mas não vincula automaticamente PRs a tarefas. A integração
de sugestão Trello continua independente e exige o link da PR no card.

`slack_habilitado` e `slack_canal_id` controlam o resumo por repositório:
branch, resultado da proteção, prontidão, situação Trello e ação necessária.
Usa a credencial Slack existente, com `chat:write`, não o Incoming Webhook
usado por outras notificações. Falha Trello não impede o resumo; falha Slack
não desfaz uma lista criada. Não há retry automático de criação/envio.

O export mantém integrações desabilitadas e marcadores de quadro/nome/canal.
Associe `trelloOAuth1Api` nos nós `10`/`13` e `slackApi` no `17`, configure
os destinos e habilite cada integração após validação. Segredos permanecem
em Credentials ou no ambiente; não copiar a chave privada do GitHub App
para parâmetros ou código. A exportação não muda a configuração da instância.

### Validação e pendências de finalização

Validado localmente com o exemplo de `instalacao`, o código real do
normalizador e um evento assinado enviado ao webhook publicado do Core:
aceitação HTTP 202, execução do onboarding e retorno confirmado nas
execuções. O GitHub identificou `main` no repositório de teste; a consulta
de proteção retornou 401 e permaneceu indisponível, com prontidão falsa.
No único quadro de teste, criou a lista do repositório e reutilizou seu ID
no reprocessamento e em `repositorios_adicionados`. O Slack confirmou
envio ao único canal de notificações usado nos testes de conexão do projeto.
Não registrar IDs dos destinos ou valores de credenciais neste documento.

Passaram 25 cenários isolados na instância: os três resultados de proteção,
Check de outro App, administradores, erros HTTP, resposta incompleta,
recusa de simulação, contrato normalizado, múltiplos repositórios,
duplicidade/lista vazia, remoções, origem inválida e decisões de lista
Trello. Proteção confirmada e Check pendente foram exercitados com fixtures,
sem comprovar essas configurações em um GitHub real. O workflow manual não
foi exportado nem conectado ao Core. Os testes locais do export executam
código dos nós sem rede ou envio de mensagens.

Falta para finalizar a integração:

- Implementar emissão automática de `instalacao` no backend e validar o
  caminho GitHub App → backend → Core com instalação real.
- Implementar publicação definitiva do Check, definir `check_nome`/`app_id`
  e disponibilizar leitura autenticada de proteção; então validar os três
  resultados em repositório de teste sem alterar regras pelo onboarding.
- Tratar rulesets, bypasses e branches de destino diferentes da padrão antes
  de ampliar a garantia para essas configurações.
- Definir mapeamento de quadro/canal por instalação/equipe. A instância
  atual usa destinos fixos de teste, não roteamento multi-tenant.
- Definir exclusão entre execuções concorrentes: reutilização sequencial
  foi validada, mas busca seguida de criação não é uma operação atômica.
- Definir recuperação parcial, reenvio exclusivo do Slack e revalidação
  posterior da proteção. O Core deduplica tenant/evento; reenviar o mesmo
  evento não é um mecanismo de retry das integrações já processadas.

## Slack — Central de Operação

Alertas usam Incoming Webhook; botões abrem formulários pela credencial do
bot. A decisão volta por `POST /acoes-remotas`; não chama GitHub diretamente.

| Nós | Função |
| --- | --- |
| `01–02`, `03A–03B`, `HTTP Request` | Recebem evento, separam e publicam alertas. |
| `02A–02F` | Buscam/validam contexto OSV e fazem consulta complementar quando possível. |
| `05–07`, `07E` | Interações, assinatura Slack, confirmação e recusa. |
| `07F–07G` | Encaminham ações `aprova_feedback_*` ao feedback, preservando as demais rotas. |
| `08–09`, `10`, `10B` | Separam botões/formulários e abrem aprovação/revisão. |
| `11`, `11R` | Validam decisão/justificativa ou preparam reexecução. |
| `12–14`, `HTTP Request1` | Assinam pedido, consultam núcleo e publicam retorno. |

Quem envia o formulário deve ser quem o abriu. Aprovação/revisão exigem
justificativa também no backend. O pedido leva ID Slack e, quando disponível,
`task_run_id` original. O núcleo resolve identidade por
`SLACK_GITHUB_VINCULOS`, verifica permissão atual e impede autoaprovação.

`Pedido`, `Servico.Processar` e `ExecutorNoGitHub` ficam em
`internal/acaoremota/`. `RepositorioPostgres` reserva `id_acao`
e registra tentativas; migration 15 inclui
`remote_action_attempts.justification`. A correlação confere análise,
tenant, repositório e PR; justificativa também segue na review GitHub.
Somente `202 + status=aceito` confirma a execução nesse contrato.
Timeout/conexão recusada não confirmam ação.

Falta configurar vínculos na API, conferir/aplicar migrations e validar em
PR real: autorizado, sem permissão, autoaprovação, repetição, erro e proteção
de branch. Reexecução ainda usa `ReenfileiradorIndisponivel`, sem executor
real. Atualizar o alerta original após decisão exige guardar/correlacionar
a mensagem; o retorno do formulário não faz isso.

No Slack App, **Interactivity & Shortcuts** usa HTTPS público com
`/webhook/aprova/slack/interacoes`. Conferir a URL quando o túnel mudar.

## Dependências — Consulta OSV

Consulta pública a `https://api.osv.dev/v1/query` por pacote/ecossistema/
versão. Resultado informativo, sem modificar risco. Falha ou consulta parcial
não significam ausência de vulnerabilidades.

| Nós | Função |
| --- | --- |
| `01–03` | Entrada, validação e separação das dependências. |
| `04–06` | Consulta paginada, consolidação de CVEs/completude e alerta. |
| `06A–06C` | Agrupamento, persistência e retorno do contexto por análise. |
| `06D`, `07` | Separam contextualização e publicação do alerta. |

Data Table `aprova_osv_por_analise`: colunas String `chave` e
`contexto`. Correlaciona tenant, análise, repositório e PR; SHA incompatível
impede reutilização quando presente nos dois eventos. Contexto válido por
24h, snapshot de até 50 pacotes e resumo de até três pacotes/1000 caracteres.

Consulta normal: até dez páginas e 30s por requisição. Complementar antes do
alerta: até dez pacotes, duas páginas e 5s por requisição. Sem dependências
conhecidas/contexto válido, declara indisponibilidade. Formulários reutilizam
o resumo do alerta, sem consulta no clique ou alteração da justificativa.

Falta emissão automática pelo núcleo, contexto na review GitHub e atualização
do alerta quando OSV chega depois. Definir retenção/limpeza do cache e validar
cobertura além dos limites.

## Trello — Sugestão de Movimentação

O link completo da PR deve estar na descrição de um único card do quadro.
Sem vínculo ou com ambiguidade, não comenta. Sugere etapa pelo resultado do
núcleo; nunca move, aprova ou conclui o card.

| Nós | Função |
| --- | --- |
| `01–03` | Validam origem/evento e preparam quadro/marcador. |
| `04–06` | Pesquisam candidatos e exigem vínculo único pelo link exato. |
| `07–09` | Releem e revalidam card, quadro, descrição e arquivamento. |
| `10–12` | Consultam comentários, verificam duplicidade e preparam sugestão. |
| `13–14` | Criam/confirmam comentário. |
| `15–17`, `99` | Replicam ao Slack e separam os resultados. |

Marcador `tenant_id:id_evento` evita repetição sequencial, sem exclusão
entre execuções simultâneas. Falha Slack não desfaz comentário; reprocessar
o evento inteiro não recupera automaticamente o aviso perdido. A instância
ainda tem dois nós de inspeção manual do quadro, somente leitura, fora do
export; não são o fluxo de eventos.

Falta aplicar/validar o cursor `before` do export na instância para
histórico extenso; definir recuperação de avisos, concorrência e mapeamento
de quadros/listas por equipe. Somente `analise_concluida` alimenta essa
integração; OSV, decisão e merge não alteram o card.

## Digest — Resumo Semanal

Cron `0 9 * * 1`, fuso `America/Sao_Paulo`. Período: última semana UTC
completa, segunda 00h até segunda 00h, fim exclusivo.

| Nó | Função |
| --- | --- |
| `01` | Agendamento. |
| `02` | Lê `Number($env.APROVA_CHAT_INSTALLATION_ID || 0)`, valida e calcula período. |
| `03` | HMAC da query exata com `N8N_SEGREDO_ACOES`. |
| `04` | GET `/reports/weekly`, timeout 15s, sem redirects. |
| `05` | Valida HTTP, tenant, período, contagens/taxa e formata resumo. |
| `06–07` | Publica no Incoming Webhook e exige HTTP 200/texto `ok`. |

Instância e export usam um ID derivado do ambiente, sem valor fixo no JSON.
Nenhum dos dois contém nós de simulação. Digest consulta API diretamente,
sem evento ou IA; envio de saída ao Slack não exige ngrok.
Não há deduplicação, retry automático de envio ou recuperação de disparos
perdidos; conferir o canal antes de repetir após timeout.

Falta disponibilizar API/banco/migrations e validar o relatório real; depois
ativar e conferir o primeiro disparo. O Digest continua **desativado**.

## Slack — Consulta de Métricas e Consulta — Relatório Semanal

DM contínua e menções no único canal autorizado recebem mensagem normal,
sem thread. Agente somente leitura, sem ferramenta de execução/aprovação.

| Nós do assistente | Função |
| --- | --- |
| `01–03C`, `03A`, `02E` | Webhook Raw Body, assinatura, challenge, confirmação e recusa. |
| `04–05` | Filtros workspace/app, DM/menção, deduplicação de 10.000 chaves e sessão. |
| `AI Agent`, `Google Gemini Chat Model` | Interpretação e ferramentas restritas; modelo local selecionado `models/gemini-3.5-flash-lite`. |
| `Simple Memory` | Janela de cinco interações, isolada por workspace/conversa/canal/usuário. |
| `consultar_relatorio_semanal` | Subworkflow identificado por variável; IA fornece somente `inicio`. |
| `consultar_saude_api`, `consultar_prontidao_api` | GET fixos `/health` e `/ready`, timeout 5s, sem redirects. |
| `06–08B` | Validam/escapam resposta e enviam ao destino autenticado. |

| Nós da consulta semanal | Função |
| --- | --- |
| Trigger | Entrada única `inicio`, AAAA-MM-DD ou vazio. |
| `01–02` | Instalação pelo ambiente e validação de data/período. |
| `03–04` | Assinatura da query e GET real. |
| `05` | Métricas validadas ou indisponibilidade com `dados=null`. |

Sem dados fictícios; memória não é fonte atualizada de métricas. IA não escolhe
instalação, URL, método ou assinatura. No Slack App, **Event Subscriptions**
usa HTTPS público com `/webhook/aprova/slack/consultas`, eventos
`app_mention`/`message.im` e scopes `app_mentions:read`, `im:history`,
`chat:write`. Habilitar Messages no App Home, convidar bot ao canal e
reinstalar App após mudar scopes. Interatividade mantém sua URL separada.

Falta validar métricas reais e tratar erros/cota do modelo com aviso Slack;
falha do agente ainda pode deixar conversa sem resposta. Para várias equipes,
definir autorização por usuário/instalação, memória persistente, retenção e
concorrência. Novas consultas dependem de endpoints de leitura adicionais;
não conectar ações remotas ao agente.

## Feedback — Motivo da Aprovação

Recebe `aprovacao_registrada` autenticado pelo Core. Pergunta somente
se decisão `aprovada` e risco original ALTO.

| Botão | `feedback_reason` |
| --- | --- |
| Regra conservadora demais | `regra_conservadora` |
| Risco aceito conscientemente | `risco_aceito` |
| Falso positivo | `falso_positivo` |

| Nós | Função |
| --- | --- |
| `01–03` | Entrada, configuração e separação evento/clique. |
| `04–06` | Elegibilidade, vínculo único e deduplicação de perguntas tenant/aprovação (10.000 chaves). |
| `07–09` | Contexto assinado dos botões, publicação e confirmação da pergunta. |
| `10–11` | Validação do clique/revisor e habilitação da API. |
| `12–14` | Assinatura/envio do feedback e confirmação condicional de persistência. |
| `15–16`, `99` | Aviso efêmero ao revisor e resultado. |

Cliques chegam pela Central após assinatura Slack. O contexto HMAC vincula
tenant, aprovação, análise, revisor, workspace, app, canal e motivo.
Expiração padrão 168h, configurável entre 1 e 720h.

No nó `02`, `vinculosLocais` está vazio,
`api_feedback_habilitada=false`, `caminho_api_feedback=''`. Sem vínculo
único, termina em `configuracao_pendente` sem pergunta.
O código aceita também `SLACK_GITHUB_VINCULOS`; o Compose atual repassa essa
variável ao n8n, mas é necessário preenchê-la e recriar o container para
carregá-la. Configurar apenas o processo da API não atualiza o n8n.

`approvals.feedback_reason` e seus três valores existem no banco/domínio.
A query atual permite motivo na criação; falta atualizar aprovação existente
por endpoint próprio. Não usar `/acoes-remotas` para registrar feedback.

Contrato preparado no workflow **proposto**, sem rota implementada: POST com
`id_feedback`, `installation_id`, `id_aprovacao`, `task_run_id`,
`usuario_slack`, `workspace_slack`, `feedback_reason`; HMAC de timestamp
e corpo exato com `N8N_SEGREDO_ACOES`. Só confirma persistência com
HTTP 200/201, `status=registrado` e tenant/aprovação/motivo correspondentes.
HTTP 202 não comprova esse registro; execução do n8n não é rótulo em approvals.

Falta endpoint com identidade/autorização confiável, verificação da aprovação
ALTO original e idempotência tenant/aprovação; política de segundo motivo
conflitante; vínculo do revisor e teste integrado. Falha após deduplicação
da pergunta exige reprocessamento controlado.

O calibrador histórico real ainda não existe; worker usa gate provisório.
Falta consumir o sinal rotulado e registrar no artigo a diferença entre
aprender com resultado observado e com motivo declarado pelo humano.
Não afirmar ganho quantitativo antes de medir. O desenvolvimento posterior
deste ramo foi adiado pelo usuário.

## Endpoints e limites do núcleo

| Rota atual | Uso |
| --- | --- |
| `GET /health` | Saúde do processo, sem verificar dependências. |
| `GET /ready` | PostgreSQL/Redis, até 2s; 200 ou 503 com estados. |
| `GET /reports/weekly` | Agregado por instalação/sete dias UTC; HMAC da query bruta. |
| `POST /webhooks/github` | Entrada de eventos GitHub autenticados. |
| `POST /acoes-remotas` | Ação humana autenticada/autorizada no núcleo. |

`Servidor.manipularRelatorioSemanal`, `inicioDoRelatorio`,
`domain.RelatorioSemanal` e query `GetWeeklyReport` definem o relatório.
Retorna instalação, período, total, estados, aprovação humana, risco médio,
reversões e taxa. Taxa = concluídas / (concluídas + rejeitadas + falhas),
nula sem finalizadas. `em_andamento` considera execuções recebidas na janela,
não todas as atuais; reversões podem corresponder a análises antigas.
Falha da API não representa semana vazia.

Não existem endpoints para listar PRs pendentes, detalhar execução, fornecer
recibo público ou gravar feedback posterior. Não prometer essas consultas.
O worker emite análise, espera de aprovação, decisão e falha recuperável,
mas gate, recibo e publicação ainda usam implementações provisórias.

## Pendências compartilhadas e implantação

- Iniciar API/worker com ambiente atualizado, conferir/aplicar migrations e
  validar n8n → API e worker → Core no repositório de teste.
- Confirmar instalação, permissões e credenciais no GitHub real. As credenciais
  já estão disponíveis; falta validação integrada.
- Configurar vínculos Slack/GitHub para ações e feedback, sem aceitar identidade
  declarada pela IA como autoridade.
- Conferir URLs HTTPS do Slack/GitHub ao mudar túnel; produção exige domínio
  estável e rede/configuração próprias.
- Implementar gate/calibração, recibos e publicação definitivos e eventos
  ainda ausentes, respeitando a separação núcleo/periferia.
- Construir SLA com dois limites configuráveis: renotificação e menção ao
  responsável, encerrando após decisão. Ainda não há workflow.
- Configurar credenciais/chamadas após importar os exports; preservar
  portabilidade e correção Trello nas próximas sincronizações autorizadas.
- Definir implantação da API/worker, versões de imagens, backups do n8n/chave
  de criptografia, retenção e recuperação de falhas para produção.

Esta revisão não iniciou API/worker, aprovou PRs ou validou métricas reais.
O onboarding criou uma lista no quadro de teste e enviou resumos ao canal
usado nos testes do projeto. Esses testes não substituem a emissão real pelo
backend nem a confirmação de proteção autenticada ainda pendentes.

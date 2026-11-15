# Workflows do n8n

Guia de execução e configuração. Estado local e pendências atualizados em
**2026-11-15**; detalhes dos nós e contratos em
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
| `consulta-analises.json` | Listagem/detalhe de análises com GET assinado, escopo e paginação. |
| `comparacao-relatorios-semanais.json` | Duas leituras do agregado periódico e diferenças calculadas. |
| `feedback-motivo-aprovacao.json` | Motivo de aprovação ALTO, contexto assinado e envio bloqueado até endpoint/vínculo reais. |
| `onboarding-repositorio.json` | Consulta de proteção por repositório, criação/reutilização de lista Trello e resumo Slack. |
| `sla-aprovacao.json` | Acompanhamento persistente de aprovação, dois limites configuráveis e escalonamento Slack. |

JSONs são portáteis e inativos, sem segredos, credenciais pessoais ou dados
fixados. `../exemplos/` contém payloads de eventos, não workflows importáveis.

Instância e exports usam referências `APROVA_*` entre subworkflows.
A correção de paginação Trello com cursor `before` foi aplicada à instância
nesta etapa. Nós manuais de inspeção permanecem somente na instância;
o workflow manual de onboarding também não é exportado. Suas referências
de canal/quadro/subworkflow usam as mesmas variáveis e ele continua inativo.

O estado original das demais integrações foi preservado. Com o SLA, são nove
workflows funcionais ativos na instância e Digest inativo. Os dez exports
continuam inativos. Backups, credenciais e
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
| `APROVA_CONSULTA_ANALISES_WORKFLOW_ID` | ID da importação de `consulta-analises.json`. |
| `APROVA_COMPARACAO_SEMANAL_WORKFLOW_ID` | ID da importação de `comparacao-relatorios-semanais.json`. |
| `APROVA_SLACK_WORKFLOW_ID`, `APROVA_OSV_WORKFLOW_ID` | Referências portáteis das chamadas Central/OSV nos exports. |

Na instalação local, credenciais Slack/Gemini/Trello estão cadastradas;
segredos e IDs Slack estão carregados; instalação positiva já está no
`.env` e no container. App ID, chave privada válida, webhook secret e slug
GitHub estão no `.env` do núcleo. Na Fase 6, autenticação do App e leitura do
repositório/PR de teste foram confirmadas. Na 6A, a instalação e sua lista de
repositórios confirmaram acesso a `PI-GRUPO/aPRova-TESTES`: Check/comentário
foram publicados e consultas de papéis responderam 200. O 403 anterior de
colaborador era do fork fora do escopo da instalação; a consulta de proteção
do alvo correto continua em 403. O vínculo confirmado de `Azulpasta` foi carregado
na 6C, com aprovação/repetição reais validadas por teste controlado.

`SLACK_GITHUB_VINCULOS` contém os pares de `Azulpasta` e `julia-ed2`
confirmados pelo usuário, carregados na API e no n8n na 6C. O Compose repassa essa variável;
o feedback usa a mesma fonte. Novos vínculos exigem confirmação humana.
Não presumir que ambiente da API é ambiente do n8n.

Na 6C, contas foram conferidas pelos conectores, e o par confirmado pelo
usuário foi carregado com backup privado e recarga apenas da API/n8n,
preservando worker e volumes. Papéis no alvo retornaram 200/admin. O pedido
sem vínculo foi recusado com 403; o teste manual de aprovação do PR 3 por
`Azulpasta` foi autorizado, com uma review real no SHA esperado. Repetição do
mesmo `id_acao` retornou 409 e não publicou outra review. Após confirmação
do vínculo do autor, a tentativa de `julia-ed2` aprovar o próprio PR retornou
403 e log `autor_do_proprio_pr`, sem nova review. O teste foi controlado
n8n → API. Depois, clique e formulário humanos foram validados na Central,
com assinatura Slack verificada, API 403 por `autor_do_proprio_pr`, retorno
exibido ao usuário e nenhuma review adicional.

A entrada de review na API foi controlada/assinada, baseada no objeto real
do GitHub. Worker → Core emitiu decisão autenticada e concluiu a análise;
Check `success`, PR aberto/rascunho e sem merge. Relay manual da decisão
fechou a mesma mensagem de teste da 6B, com persistência confirmada e sem
repetir edição/publicação. OSV e recibo foram preservados. O feedback publicou
a pergunta automaticamente no canal configurado, sem registrar motivo no núcleo.

A auditoria de 22 nós permanece inativa. O bot não tem `users:read`, portanto
a consulta opcional de perfil informa indisponibilidade; o vínculo carregado
é confirmado separadamente. Metadados de `main` indicam `protected=false`,
sem Check obrigatório, enquanto detalhes administrativos pelo App seguem
em 403. Nenhuma regra/permissão foi alterada. A correlação global, SLA e
Digest continuam desligados.

O workflow de teste de atualização foi restaurado ao modo isolado, e o novo
teste de ação fica inativo com POST desabilitado, no projeto pessoal da
instância. Passaram 48 testes locais de decisão/OSV e os cinco pacotes Go
registrados no checklist. A 6C permanece parcial por usuário real sem escrita; reexecução continua ausente.
Nenhum workflow funcional precisou de nova exportação.

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
   `feedback-motivo-aprovacao.json`, `onboarding-repositorio.json` e
   `sla-aprovacao.json` antes de configurar as chamadas.
2. Associe Slack aos formulários da Central e aos envios do assistente;
   Gemini ao modelo; Trello aos nós `04`, `07`, `10`, `13`.
3. Configure o ambiente acima. As credenciais GitHub são da API/worker;
   não copiá-las aos workflows.
4. Copie os IDs gerados pela importação para `APROVA_SLACK_WORKFLOW_ID`,
   `APROVA_OSV_WORKFLOW_ID`, `APROVA_TRELLO_WORKFLOW_ID`,
   `APROVA_FEEDBACK_WORKFLOW_ID`, `APROVA_ONBOARDING_WORKFLOW_ID` e
   `APROVA_CONSULTA_SEMANAL_WORKFLOW_ID` e `APROVA_SLA_WORKFLOW_ID`.
   As chamadas já usam expressões;
   preserve `each`, conexões e opções de espera. Associe `slackApi` aos nós
   `08`/`15` do feedback e mantenha a API de feedback desabilitada.
5. Crie a Data Table `aprova_osv_por_analise`, com colunas String
   `chave`/`contexto`, conforme `recursos/tabela-contexto-osv.json`.
   Para o SLA, crie as tabelas conforme `recursos/tabela-sla-eventos.json`
   e `recursos/tabela-sla-controle.json`. Insira uma única linha inicial de
   controle conforme `initialRows` e preencha `APROVA_SLA_EVENTOS_TABELA_ID`
   e `APROVA_SLA_CONTROLE_TABELA_ID` com os IDs gerados. Esses arquivos são
   definições de recursos, não workflows importáveis nem cópias dos dados.
6. Preencha quadro/listas `APROVA_TRELLO_*` e canal de onboarding.
   Associe Trello ao onboarding `10`/`13` e Slack ao `17`.
   Só habilite as flags de onboarding após conferir destinos e credenciais.
   Confirme o nome/App ID do Check dessa instalação. No ambiente local, foram
   configurados na Fase 6 com o nome `aPRova` do código e o App autenticado;
   os exports não incluem o ID real.
7. Recrie o n8n para carregar variáveis e configure as URLs públicas.
   Valide antes de publicar; subworkflows devem estar disponíveis antes do Core.
   No SLA, associe `slackApi` ao nó `T13`. Mantenha
   `APROVA_SLA_HABILITADO=false` até acordar prazos e configurar políticas,
   canal e responsáveis. A importação não autoriza enviar avisos reais.

## Slack e consultas

### Validação da instalação local — Fase 6

Em 2026-11-15, API e worker locais estavam em execução. Auditoria a partir
do n8n confirmou saúde/prontidão, PostgreSQL/Redis disponíveis, chave pública
e relatório semanal assinado. A semana consultada estava vazia, com taxa nula;
isso não comprova uma análise completa do worker. Uma consulta assinada do
ambiente local atravessou Core → API: foram conferidos o 202 de entrada e
o resultado final com API 200. Entradas sem assinatura no Core e em ações
remotas receberam 401. O bot Slack autenticou e o workspace correspondeu.

`APROVA_GITHUB_CHECK_NOME` e `APROVA_GITHUB_CHECK_APP_ID` foram preenchidos
no `.env` local com evidência do código/App e carregados recriando somente
o n8n, com volume preservado. A leitura administrativa da proteção continua limitada:
o App recebeu 403 e a API informou requisito `administration=read` ausente
nos grants retornados. Consultas de colaboradores no repositório correto
responderam 200; nenhum vínculo humano foi inferido desses papéis.

A 6A concluiu análise real do PR 1 de `PI-GRUPO/aPRova-TESTES`: sandbox em
`go/`, 10 testes passaram, risco BAIXO, Check `success` e comentário do bot.
O evento automático `analise_concluida` chegou autenticado ao Core e seguiu
ao Trello, que encerrou em `sem_card_vinculado`, sem escrita ou réplica Slack.
Documento e `/verify` do recibo responderam 200; hash dos bytes, identidade
e cinco verificações `ok` foram conferidos. Uma auditoria manual somente de
leitura no n8n reutilizou V03/V04/V06/V07 e classificou o recibo como `valido`.
A preparação usa o evento real capturado, sem alterar seu risco ou executar
o alerta da Central. A entrada GitHub foi manual/assinada; sua entrega
automática pelo GitHub ainda não foi comprovada.

A 6B validou o PR 3 com `github.com/google/uuid v1.6.0` declarado: 10 testes
passaram, gate ALTO por dependência nova, Check `action_required`, comentário
e espera por aprovação. Os três eventos reais chegaram autenticados ao Core;
OSV respondeu 200 e persistiu consulta concluída sem avisos conhecidos.
A Central verificou o recibo válido e o Slack confirmou o alerta automático.

O contexto OSV real foi entregue por relay manual ao teste da Central, com
tabelas isoladas: publicação identificada, edição no mesmo canal/ts,
persistência conferida e repetição `osv_ja_aplicado`, sem publicação/edição.
Não se ativou a correlação global. O workflow manual foi restaurado ao modo
isolado, inativo e sem rede/credenciais. Passaram 113 testes em seis arquivos.
Autores de PR com sufixo `[bot]` ainda são rejeitados pelo contrato de eventos;
o cenário completo usou autoria humana autenticada, sem alterar identidade.

Faltam vínculos adicionais confirmados, proteção, interação humana, política
SLA e os demais cenários integrados. Digest, SLA e correlação continuam
desligados; a Fase 6 permanece parcial. Nenhum workflow funcional precisou
de alteração/exportação. As duas auditorias manuais ficam inativas somente
na instância, no projeto pessoal. Detalhes e limites estão no checklist.

### Correlação dos alertas da Central

A Central possui um caminho de publicação pelo bot que guarda canal e
timestamp para futuras atualizações da mensagem. Por padrão,
`APROVA_SLACK_CORRELACAO_HABILITADA=false` mantém o Incoming Webhook anterior.
Os retornos dos formulários continuam no caminho existente. Com a correlação
habilitada, o evento `aprovacao_registrada` atualiza o alerta da mesma análise;
o OSV também pode atualizar o contexto quando a consulta termina depois.

| Variável | Função |
| --- | --- |
| `APROVA_SLACK_CORRELACAO_HABILITADA` | `true` habilita publicação correlacionada e atualizações após decisão/OSV; padrão `false`. |
| `APROVA_SLACK_CANAL_ALERTAS_ID` | Canal confirmado dos alertas da Central, sem fallback para outros canais. |
| `APROVA_SLACK_MENSAGENS_TABELA_ID` | ID da Data Table `aprova_slack_mensagens`. |
| `APROVA_SLACK_CONTROLE_TABELA_ID` | ID da Data Table `aprova_slack_controle`. |

Crie as tabelas conforme [mensagens](recursos/tabela-slack-mensagens.json) e
[controle](recursos/tabela-slack-controle.json), incluindo exatamente uma
linha inicial de controle conforme `initialRows`. Associe `slackApi` ao nó
`C09 | Publicar alerta pelo bot` e `D04 | Atualizar alerta após decisão`,
com permissão para publicar e editar as mensagens do próprio bot no canal.
Carregue as variáveis no container antes de habilitar a flag e valide o
envio em destino autorizado. O export permanece inativo e sem credenciais.

A chave combina instalação, análise, repositório, PR e tipo de alerta;
SHA incompatível impede reutilização. A publicação usa controle persistente
de 15 minutos e reserva antes da chamada Slack. Confirmação exige canal e
timestamp válidos e persistência conferida. Estados `reservado`, `incerto`
e `falha` impedem retry automático; confira Slack e execução antes de recuperar.
Controle ocupado gera erro explícito sem envio e exige reprocessamento
controlado do subworkflow; não há fila automática. Não apague registros ou
reservas para repetir um envio cujo resultado seja desconhecido.

O Core chama a Central também pela saída `aprovacao_registrada`, usando
`APROVA_SLACK_WORKFLOW_ID`, por evento e sem aguardar. A decisão registrada
remove os botões de aprovação/revisão da mensagem, preservando risco, OSV e
links do recibo. Um formulário enviado não substitui esse evento do núcleo.
Somente decisões de risco original ALTO participam desse caminho.

Uma decisão recebida antes do alerta fica na mesma tabela com estado
`aguardando_alerta`; quando o alerta chega, já é publicado com a decisão
exibida. Não são necessárias novas tabelas ou colunas. A decisão e seu estado
de atualização ficam no contexto da associação existente. Decisões repetidas
confirmadas não executam outro `chat.update`; conflitos ou atualizações
reservadas/incertas/falhas exigem conferência manual. A edição só é confirmada
para o mesmo canal e timestamp, com persistência do resultado conferida.

Na Fase 4, OSV encaminha à Central o resultado já persistido de
`dependencia_nova`, via `APROVA_SLACK_WORKFLOW_ID`, por evento e sem aguardar.
Consultas síncronas de contexto da aprovação não fazem esse encaminhamento,
evitando ciclo Central → OSV → Central. O alerta independente de dependência
continua no caminho existente. `O01` valida estrutura, identidade, SHA quando
disponível, validade de 24h e até 5 minutos de relógio futuro.

A atualização reutiliza a mesma associação e reserva de decisão, preservando
risco, recibos, links e decisões já exibidas. Não reintroduz botões removidos;
atualiza o resumo dos botões ainda disponíveis. OSV anterior ao alerta fica
como contexto mínimo em `aguardando_alerta`, junto da eventual decisão, e é
incorporado no primeiro envio. Não cria tabelas, colunas ou variáveis.
Resultado repetido/antigo ou menos completo não substitui contexto recente.
Conflito no mesmo instante e atualização reservada/incerta/falha exigem
conferência manual, impedindo outra edição automática até recuperação.
Completo sem avisos, parcial e indisponível são estados distintos.

Central e OSV estão publicados e exportados. A Fase 4 passou em testes locais
e na instância com tabelas isoladas. O teste Slack real autorizado confirmou
um envio, atualização da mesma mensagem e reprocessamento sem duplicata,
usando contexto OSV fictício. Os adaptadores temporários foram removidos.
A 6B confirmou origem worker → Core → OSV real e resultado persistido. Seu
relay manual ao teste da Central confirmou a edição da mesma mensagem e a
supressão de repetição com contexto real. A correlação global continua
desligada; esse ensaio não habilita o callback automático em produção.

Com a flag desligada, eventos de decisão são ignorados e OSV não é encaminhado;
o Incoming Webhook legado não recebe esses eventos. Não há backfill automático
ao ativar a flag nem localização automática de alertas antigos sem correlação.

### Links e interatividade

Na Central, o alerta de aprovação preserva os botões de decisão e oferece
links para a PR, o recibo e sua verificação. `url_recibo` emitida pelo worker
já termina em `/verify`; o alerta aceita também a URL do recibo sem esse
sufixo. A base vem do evento, por `APROVA_URL_PUBLICA` no worker, e precisa
ser acessível a quem usa o Slack. Não usar a URL interna da API como base
pública. Sem URL de recibo compatível, o alerta informa indisponibilidade.
Antes do alerta, `V01–V07` consultam documento e verificação em
`APROVA_API_URL`, extraindo somente o UUID da URL pública. As rotas são públicas,
sem credenciais; cada GET tem timeout de 5 s, sem redirects ou repetição
automática. O fluxo confere os bytes de `payload_base64` contra tenant, análise,
repositório, PR, SHA, risco e hash do evento autenticado. Ignora o `payload`
legível e limita o base64 a 262144 caracteres. `/verify` exige booleano `valido`,
identidade/hash/chave e cinco conferências coerentes; HTTP 200 sozinho não basta.
O núcleo realiza a verificação criptográfica. A Central distingue válido,
inválido e indisponível, sem alterar risco, autorização ou botões de decisão.
Ausência, incompatibilidade ou falha de consulta permitem preparar o alerta
com estado indisponível. Não há novas tabelas ou variáveis; isso independe
da flag de correlação. A Fase 2 preserva o bloco do recibo no snapshot original.
O recibo histórico retornou 404. A 6A confirmou releitura real de um recibo
válido no n8n; a 6B confirmou o percurso automático `aguardando_aprovacao` →
Central V01–V07 → alerta Slack com recibo válido, encerrando a pendência
integrada da Fase 3. Nenhuma decisão humana foi enviada.

O envio assinado a `/acoes-remotas` não
segue redirects e mantém a confirmação restrita a `202 + status=aceito`.

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

Na Fase 5, erros do agente, falta de cota, timeout retornado pelo provedor e
saída vazia recebem um aviso fixo de indisponibilidade, pelo mesmo destino
autenticado. Erro e sucesso convergem em `06`: não há dois ramos capazes de
duplicar o aviso. Erros/headers/stack internos não entram na resposta.
O retry do n8n está desligado; `alwaysOutputData` permite tratar saída vazia.
O nó Gemini instalado não expõe timeout de requisição; o tratamento cobre
falhas devolvidas pelo agente, sem garantir prazo global. O assistente foi
publicado e exportado, com ferramentas e memória preservadas. Os testes
foram isolados, sem chamadas ao provedor ou envio Slack real nesta fase.

Falta validar os endpoints de negócio com métricas reais e o aviso no percurso
real. Para múltiplas equipes, falta autorização por
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

Na Fase 5, erros de rede e HTTP chegam aos validadores, sem redirects ou
retry do n8n. Relatório inválido/indisponível para antes do envio. A confirmação
exige HTTP 200 e texto `ok`, sem erro de transporte; mantém `status=enviado`
e acrescenta `estado_envio=confirmado`. Recusa HTTP 3xx/4xx (exceto 408) é
`falha`; timeout, 5xx e resposta inesperada são `incerto`. Falha/incerteza
exigem conferência manual e nunca autorizam reenvio automático.

O rascunho foi salvo e exportado; continua **desativado**, sem ativar o cron.
Não há controle persistente de disparos por instalação/período nem recuperação
de horários perdidos. Essa parte depende das operações de reserva/confirmação
do núcleo previstas na Etapa S, ainda ausentes na API; não foi criada tabela
paralela. Faltam relatório real e primeiro disparo autorizado. Execução manual
ainda envia ao canal; confira antes de repetir, sobretudo após envio incerto.

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
Gate/calibração, recibos e publicação vieram da `dev`; a 6A validou gate
BAIXO, recibo e publicação reais. Faltam o evento automático de instalação e
os demais cenários integrados. O SLA com dois limites está publicado, com
envios desligados até configurar e validar políticas.

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

### Extensão SLA — 2026-11-14

O workflow `aPRova | SLA | Aprovação Pendente` foi criado e publicado na
instância, ligado às rotas de espera e decisão do Core pelo nó `07M`.
O JSON funcional é `sla-aprovacao.json`; o Core exportado inclui as duas
conexões de `07M`, sem mudar as rotas anteriores. Os arquivos são inativos,
sem credenciais ou IDs locais. O workflow manual de testes permanece
somente na instância, com tabelas próprias e Slack simulado sem rede.

O SLA usa `APROVA_SLA_WORKFLOW_ID`, os dois IDs explícitos das tabelas,
`APROVA_SLA_HABILITADO`, `APROVA_SLA_CANAL_ID` e
`APROVA_SLA_POLITICAS_JSON`, conforme a tabela abaixo. A política é uma lista
de objetos com `installation_id`, `repositorio`, `habilitado`,
`primeiro_limite_minutos`, `segundo_limite_minutos`, `canal_slack` e
`responsavel_slack`. Os limites são inteiros positivos e o segundo deve ser
maior; ambos contam desde o início da espera, em minutos corridos.

Sem valores acordados, a instalação mantém envios desabilitados e políticas
vazias, mas registra os eventos. O timer avalia a cada minuto e não perde a
contagem após reinício. Antes de habilitar, preencher a política e os IDs
confirmados, recriar somente o n8n e validar os avisos em canal de teste.
Credencial Slack e segredos existentes foram reaproveitados/preservados.
Implementação, tabelas, testes e limites de confiabilidade estão na
[documentação do SLA](../../docs/central-slack-implementacao.md#sla--aprovação-pendente).

Os testes do export usam o código dos próprios nós, sem rede:

```sh
node --test workflows/n8n/tests/sla-aprovacao.test.cjs
```

### Configuração comum

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
| `APROVA_SLA_WORKFLOW_ID` | Core → SLA, espera e decisão | vazio; necessário na instância com a rota SLA | ID da importação SLA | n8n |
| `APROVA_SLA_EVENTOS_TABELA_ID` | SLA: histórico persistente | vazio; obrigatório | ID da Data Table `aprova_sla_eventos` | n8n |
| `APROVA_SLA_CONTROLE_TABELA_ID` | SLA: reserva do temporizador | vazio; obrigatório | ID da Data Table `aprova_sla_controle` | n8n |
| `APROVA_SLA_HABILITADO` | SLA: habilitar avisos | `false`; libera somente políticas habilitadas | Decisão da equipe | n8n |
| `APROVA_SLA_CANAL_ID` | SLA: canal padrão | vazio; política pode definir canal próprio | Canal autorizado | n8n |
| `APROVA_SLA_POLITICAS_JSON` | SLA: limites e responsável por instalação/repositório | `[]`; sem política não há envio | Política e identidades confirmadas | n8n |
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
| `SANDBOX_IMAGEM_JAVA` | worker; imagem construída para executar projetos Maven, só no executor local | worker |
| `SANDBOX_LINGUAGENS` | worker; pares `dono/repo=go`, `dono/repo=python` ou `dono/repo=java`; padrão Go | worker |
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

### Validação real do assistente — Fase 6D

Concluída no canal autorizado: pergunta humana assinada, modelo e ferramenta
semanal reais, dados da API e uma resposta Slack confirmada. Zero execuções
na última semana completa e taxa nula foram preservados; outra consulta
controlada confirmou dados não vazios da semana em curso. A repetição do
mesmo evento não chamou o modelo nem enviou outra resposta. Erro local
controlado confirmou um único aviso seguro no Slack. Passaram 30 testes em
três arquivos. O workflow manual foi restaurado ao modo isolado/inativo;
assistente, consulta e exports funcionais foram preservados. DM permanece
sem validação real específica. A 6C continua com seu caso adiado e a 6E
aguarda política da equipe; nenhum envio operacional foi habilitado.

### Ampliação de consultas após a 6D

Assistente publicado com 21 nós e seis ferramentas somente leitura: relatório
semanal, comparação de duas semanas, listagem de execuções, detalhe por UUID,
saúde e prontidão. Os dois subworkflows novos foram criados no projeto pessoal
existente. As três definições afetadas foram exportadas após validação real.

Importe as duas consultas novas e configure suas variáveis de ID no ambiente,
recarregando o n8n antes de publicar o assistente importado. Instalação continua
fixada por `APROVA_CHAT_INSTALLATION_ID`. API precisa conter as novas rotas;
use o mesmo `N8N_SEGREDO_ACOES` nos dois serviços. As consultas não usam
credenciais HTTP: o HMAC é calculado no nó Code com `crypto` permitido.
Associe as credenciais já existentes ao modelo e aos envios Slack do assistente.

Exemplos: “quais análises estão em andamento em proprietário/repositório?”,
“quais aguardam aprovação?”, “qual o estado da análise UUID?” e “compare
as últimas duas semanas completas”. Para PR com várias análises, preserve ID
junto ao SHA e esclareça a escolha. As métricas semanais não são filtradas por
repositório; o detalhe não retorna código, logs nem motivo de falha. A lista
não prova worker ativo; a ferramenta marca `atividade_worker=nao_consultada`.

Uma comparação executa duas leituras validadas numa única chamada, evitando
depender de encadeamento pelo modelo. O tratamento de resposta também impede
publicar como dado um texto iniciado por `Calling ... with input:` de chamada
malformada. Eventos controlados com modelo/API/Slack reais validaram listagem,
detalhe, ressalva sobre worker e comparação de semana parcial com semana vazia.
Passaram 46 testes de consultas/resiliência em cinco arquivos, quatro pacotes
Go e integração PostgreSQL descartável. Contratos e HMAC das rotas estão em
[eventos-n8n.md](../../docs/eventos-n8n.md#consultas-de-análises-e-comparação-semanal--ampliação-após-a-6d).
O caso adiado da 6C, Digest e SLA permanecem nas condições anteriores.

### Apresentação das respostas do assistente

Respostas usam texto simples, sem `**`; o nó final também remove essa marcação.
Datas são exibidas como `DD/MM/AAAA`; instantes ISO são convertidos para
`DD/MM/AAAA às HHhMM (Brasília)` por `Intl` com `America/Sao_Paulo`, sem segundos
ou frações. Links, IDs e SHA são preservados. Períodos semanais continuam em UTC,
com fim não incluído e datas legíveis; a apresentação não altera as consultas.
Somente o assistente foi publicado/exportado nessa revisão. Passaram 48 testes
de consultas/resiliência em cinco arquivos.

### Correção de chamadas em linguagem natural

Após falhas humanas `MALFORMED_FUNCTION_CALL` antes do acesso à API, o prompt
removeu o exemplo textual de chamada interna e a memória passou ao namespace
`consultas-reais-v2`. Essa mudança reinicia o contexto curto anterior, mantendo
isolamento e cinco interações. A proteção que rejeita texto de chamada como
resposta foi preservada. Não foram alterados o modelo nem as ferramentas.

A pergunta “tem alguma PR esperando alguem aprovar?” foi validada em teste
manual e no canal por evento assinado/controlado, com ferramenta/API reais e
uma publicação Slack confirmada. Também foi validado acompanhamento no teste
manual sem Slack; seu workflow fica inativo no projeto pessoal. O assistente
foi publicado/exportado e os 48 testes de consultas/resiliência passaram.
Listar análises não confirma estado aberto/fechado do PR no GitHub.

### Links GitHub e seleção atual do modelo

A consulta de análises gera `url_repositorio` e `url_pr` depois de validar os
dados, com domínio fixo GitHub e segmentos codificados; não repassa uma URL
arbitrária vinda da resposta. O assistente inclui URLs completas, reconhecidas
como links pelo Slack, sem usar sintaxe Markdown de links.

O usuário escolheu `models/gemini-3.5-flash` no nó do modelo. O rascunho foi
publicado e o export atualizado; credenciais foram preservadas. Um evento
assinado/controlado com modelo/API/Slack reais confirmou análise do PR 2 e
URLs de PR/repositório. Texto de chamada malformada, inclusive variante sem
“with input”, gera aviso seguro; não foi acrescentado retry automático.
Passaram 50 testes de consultas/resiliência em cinco arquivos.

### Recuperação de chamada malformada recorrente

O assistente passou a 23 nós. `05R` detecta texto de chamada inválida e
encaminha uma única vez ao agente `05S`, que recebe somente a pergunta original
autenticada, sem histórico. Ele compartilha o modelo escolhido e as seis
ferramentas de leitura; não interpreta a chamada malformada nem recebe
ferramentas de ação. A recuperação não possui memória, loop ou retry nativo.
Respostas normais seguem direto para `06`; ambos os caminhos preservam um
único envio. Cota, rede e erros comuns não disparam essa nova tentativa.

O agente principal mantém sua memória isolada. Se uma pergunta depender do
histórico e ele falhar, a recuperação pode pedir esclarecimento. Comparações
entre semana atual e anterior usam datas explícitas UTC; a atual é parcial.
Teste manual inativo no projeto pessoal confirmou recuperação com modelo/API
reais após erro inicial controlado. Passaram 52 testes em cinco arquivos.
O teste anterior com o novo modelo não eliminou a recorrência do provedor;
a mudança atual acrescenta recuperação limitada e mantém o aviso em falha.

No reteste integrado da mesma pergunta, o agente principal consultou as duas
semanas corretas e enviou uma resposta Slack confirmada, sem duplicata. Esse
reteste não acionou a recuperação; o teste manual forçou o primeiro erro e
validou o segundo agente com modelo/API reais. Não confundir as evidências.

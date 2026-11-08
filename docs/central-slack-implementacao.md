# Central de operação aPRova no Slack

Atualizado em **2026-11-07**, horário de São Paulo. Este documento reúne as
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

Publicação local não comprova funcionamento integral com GitHub real ou
produção. A API não estava acessível pelo n8n nesta revisão: GETs em
`/health` e `/ready` retornaram `ECONNREFUSED`. Indisponibilidade não
deve ser apresentada como métricas zeradas.

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
| `APROVA_API_URL` | Configurada, mas API indisponível no endereço conferido. |
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

Os oito JSONs em `workflows/n8n/` são portáteis e inativos. O feedback e as
atualizações de Core, Central e Digest estão sincronizados com a instância,
com remoção de credenciais/referências locais e preservação dos bloqueios:

| Diferença | Situação |
| --- | --- |
| Feedback | `feedback-motivo-aprovacao.json`, sem credenciais ou IDs locais. |
| Core e Central | Chamadas ao feedback por `CONFIGURE_WORKFLOW_FEEDBACK`; selecionar a importação nos dois workflows. |
| Digest | Instância e export leem `APROVA_CHAT_INSTALLATION_ID`, sem valor real no JSON. |
| Trello | Export tem cursor `before` para comentários; instância ainda usa `page`. |

Os exports desta revisão foram autorizados. Novas sincronizações exigem
autorização; preservar a correção Trello e a portabilidade. Na importação,
associar credenciais Slack aos nós de publicação e aviso do feedback, selecionar
o subworkflow no Core/Central e manter a API de feedback desabilitada até
existir endpoint real. Esta exportação não alterou a instância.

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
| Demais saídas | Pontos de extensão e registro sem integração. |

O tenant autenticado define a instalação do evento, sem substituição pelo
ID global de consultas. `relatorio_periodico` é pedido da automação,
não um sétimo evento emitido pelo worker, e não publica Digest.

Falta validar worker → Core com dados reais; implementar emissão de
`dependencia_nova` e `instalacao`; e construir SLA com limites
configuráveis, encerramento ao receber decisão e atualização do alerta.

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
O código aceita também `SLACK_GITHUB_VINCULOS`, mas o Compose ainda não
repassa essa variável ao n8n; configurar na API não a disponibiliza no workflow.

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

Esta revisão conferiu configuração, código e estado da instância. Não iniciou
API/worker, aprovou PRs, enviou mensagens ou validou métricas reais.
Testes anteriores com serviços externos/dados fictícios não substituem as
integrações reais ainda pendentes.

# Workflows do n8n

Guia de execução e configuração. Estado local e pendências atualizados em
**2026-11-07**; detalhes dos nós e contratos em
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

JSONs são portáteis e inativos, sem segredos, credenciais pessoais ou dados
fixados. `../exemplos/` contém payloads de eventos, não workflows importáveis.

Feedback, suas ligações no Core/Central e Digest com instalação pelo ambiente
estão nos exports. As chamadas de feedback usam `CONFIGURE_WORKFLOW_FEEDBACK`,
para selecionar a importação daquele ambiente. A correção de paginação
`before` dos comentários Trello continua no repositório, mas não na instância,
que ainda usa `page`; foi preservada nesta sincronização.

**Não reimportar/exportar indiscriminadamente:** preservar essas diferenças.
Novos exports dependem de autorização explícita do usuário.

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

`SLACK_GITHUB_VINCULOS` ainda não está configurado na API. O Compose não
repassa essa variável ao n8n; o feedback também pode ter vínculo no próprio
nó de configuração. Não presumir que ambiente da API é ambiente do n8n.

Ao alterar variáveis, recrie somente o serviço:

```sh
docker compose --profile automacao up -d --no-deps --force-recreate n8n-aprova
```

O volume é preservado. Alterar uma credencial no editor não exige recriação.
API e worker precisam ser iniciados/recarregados separadamente.
Nesta revisão, a API retornou conexão recusada a partir do n8n.

## Importar em outra instalação

1. Importe os JSONs pelo menu **Import from File**, incluindo
   `feedback-motivo-aprovacao.json` antes de configurar as chamadas.
2. Associe Slack aos formulários da Central e aos envios do assistente;
   Gemini ao modelo; Trello aos nós `04`, `07`, `10`, `13`.
3. Configure o ambiente acima. As credenciais GitHub são da API/worker;
   não copiá-las aos workflows.
4. Copie os IDs das importações para as variáveis de chamadas. No Core,
   selecione Trello no nó `09`, substituindo `CONFIGURE_WORKFLOW_TRELLO`;
   execute por item e sem aguardar. Selecione também o feedback no Core
   (`07I | Solicitar Motivo da Aprovação`) e na Central (`07G`), substituindo
   `CONFIGURE_WORKFLOW_FEEDBACK`. Associe `slackApi` aos nós `08`/`15` do
   feedback; mantenha envio à API bloqueado enquanto o endpoint não existir.
5. Crie no projeto a Data Table `aprova_osv_por_analise`, com colunas String
   `chave`/`contexto`. A definição está em
   `recursos/tabela-contexto-osv.json`, que não é workflow importável.
6. No Trello `02`, substitua `CONFIGURE_ID_*` pelos IDs do quadro/listas
   daquele ambiente; adapte o mapeamento se o Kanban for diferente.
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

Falta API acessível, teste com métricas reais e tratamento de falhas/cota do
modelo com aviso ao usuário. Para múltiplas equipes, falta autorização por
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
Falta aplicar o cursor `before` na instância e definir recuperação de avisos.

Feedback recebe decisão aprovada de risco original
ALTO e mostra três motivos. Core o chama; Central encaminha cliques após
assinatura. Sem vínculo revisor Slack/GitHub, não pergunta. API de feedback
está desabilitada e não existe endpoint correspondente no núcleo.
Faltam cadastro de identidade, endpoint autorizado/idempotente, integração
com `approvals.feedback_reason` e calibrador real. Desenvolvimento adiado.
Inventário e contrato proposto estão na
[documentação central](../../docs/central-slack-implementacao.md#feedback--motivo-da-aprovação).

Ações da Central passam pela API com vínculo Slack/GitHub e permissão real.
Conferir/aplicar migrations, incluindo a 15, antes dos testes integrados.
A reexecução ainda não tem executor. O worker já emite análise, espera de
aprovação, decisão e falha recuperável; faltam eventos de dependência e
instalação, gate/calibração definitivos, recibos e publicação definitivos.
SLA com dois limites configuráveis ainda não tem workflow.

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
Preservar correções locais ainda ausentes da instância, como paginação Trello.
Nunca incluir `.env`, credenciais ou conteúdo de `.n8n/`.

Os exports desta revisão foram autorizados; futuras sincronizações também
exigem autorização. Não sobrescrever o Trello corrigido com a versão local.

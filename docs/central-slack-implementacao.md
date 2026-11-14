# Central de operação aPRova no Slack

Atualizado em **2026-11-14**, horário de São Paulo. Este documento reúne as
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

O [plano local de atualização](plano-atualizacao-workflows.md) define as
fases, dependências e critérios de aceite. Ao executar uma fase solicitada
desse plano, sua exportação validada faz parte do escopo; tarefas apenas
documentais não autorizam sincronizações por si só. Este documento, o plano
e `AGENTS.md` são contexto local e devem permanecer fora dos próximos commits.

## Checklist do plano de atualizacao

Auditoria da Fase 0 concluída em 2026-11-14. A Fase 1 foi implementada,
testada sem rede externa, publicada com a correlação desabilitada e exportada.
Sua validação com Slack real continua pendente, portanto a fase ainda não
está concluída. Os links de PR/recibo já existentes não equivalem à
verificação automática prevista na Fase 3.

O plano agora inclui a **Etapa S — Simplificação da persistência do n8n**:
transferir estado e controle dos avisos para o núcleo, reduzir a associação
das mensagens ao necessário e retirar tabelas somente após migração e
validação. Essa diretriz não altera a implementação atual nem conclui as
pendências existentes. Confira suas dependências antes de ampliar a
persistência nas próximas fases.

Marque somente com evidência e registre bloqueios na seção da integração
afetada. Não confunda publicação com validação real. Quando uma etapa não
se aplicar à fase, justifique essa condição em vez de marcar execução.

### Fase 0 — Auditoria e checklist

- [x] Conferir código, contratos, instância e versões publicadas.
- [x] Comparar instância e exports; registrar diferenças e configuração.
- [x] Executar os testes pertinentes e registrar evidências.
- [x] Confirmar a sequência e as dependências das fases.
- [x] Atualizar documentação e checklist com os resultados da auditoria.

### Fase 1 — Correlação persistente das mensagens

- [x] Implementado no n8n.
- [x] Testado de forma isolada, incluindo duplicatas e resultados incertos.
- [x] Publicado na instância, com a correlação desabilitada por configuração.
- [ ] Validado com integração real.
- [x] Exportado e revisado no repositório, incluindo definições de tabelas.
- [x] Documentação atualizada com configuração, evidências e pendências.

### Fase 2 — Atualização após decisão

- [ ] Implementado no n8n.
- [ ] Testado de forma isolada, incluindo ordem invertida e análises distintas.
- [ ] Publicado na instância.
- [ ] Validado com integração real.
- [ ] Exportado e revisado no repositório.
- [ ] Documentação atualizada com configuração, evidências e pendências.

### Fase 3 — Verificação do recibo

- [ ] Implementado no n8n.
- [ ] Testado de forma isolada: válido, inválido, indisponível e correlação.
- [ ] Publicado na instância.
- [ ] Validado com integração real.
- [ ] Exportado e revisado no repositório.
- [ ] Documentação atualizada com configuração, evidências e pendências.

### Fase 4 — Atualização posterior com OSV

- [ ] Implementado no n8n.
- [ ] Testado de forma isolada, incluindo concorrência com decisões.
- [ ] Publicado na instância.
- [ ] Validado com integração real.
- [ ] Exportado e revisado no repositório.
- [ ] Documentação atualizada com configuração, evidências e pendências.

### Fase 5 — Resiliência das consultas e do Digest

- [ ] Implementado no n8n, preservando o Digest desativado.
- [ ] Testado de forma isolada: erros do assistente e controle de envios.
- [ ] Publicado na instância onde aplicável, sem ativação automática do Digest.
- [ ] Validado com integração real dentro das autorizações disponíveis.
- [ ] Exportado e revisado no repositório, incluindo definições de tabelas.
- [ ] Documentação atualizada com configuração, evidências e pendências.

### Fase 6 — Configuração e validação integrada

- [x] Conferir pré-requisitos locais de API, worker e migrations: banco na versão 21, processos iniciados e saúde validada a partir do n8n em 2026-11-14.
- [ ] Conferir identidades Slack/GitHub, instalação e permissões reais.
- [ ] Validar ações remotas no escopo autorizado.
- [ ] Validar worker → Core → integrações existentes.
- [ ] Configurar e validar SLA com política e destinos definidos pela equipe.
- [ ] Exportar e revisar eventuais workflows alterados nesta fase.
- [ ] Documentar evidências reais e bloqueios de configuração/backend.

### Etapa S — Simplificação da persistência do n8n

- [ ] Definir contratos e implementar suporte no núcleo (S1 do plano).
- [ ] Adaptar e testar os workflows com os contratos reais (S2).
- [ ] Migrar os registros necessários, com reversão e controle de duplicatas.
- [ ] Validar integração real, reinício, concorrência e resultados incertos.
- [ ] Retirar somente as tabelas e configurações efetivamente substituídas (S3).
- [ ] Revisar publicação, exports e documentação das integrações afetadas.

### Evidências da auditoria e dependências

Foram inspecionados os dez workflows funcionais, suas versões atuais e
publicadas e os JSONs correspondentes. Nove funcionais estão ativos; o
Digest permanece inativo. Core, Central, OSV, feedback, SLA, assistente e
consulta semanal estavam com a versão atual publicada no início da auditoria.
O onboarding tem um rascunho com `batchSize` omitido no nó `03`, enquanto o
export especifica `1`; essa diferença foi preservada, sem publicar seu rascunho.
Os dois nós manuais de inspeção Trello existem somente na instância e também
foram preservados. IDs de webhook, credenciais e configurações de instância
são diferenças esperadas dos arquivos portáteis. Não foi feita sincronização
dos workflows alheios à Fase 1.

A base anterior passou em 123 testes locais. Com os 14 testes novos de
correlação, passaram 137 testes em 12 arquivos; o Compose foi validado sem
imprimir sua configuração expandida. `.env` e container confirmaram presença
dos segredos de eventos/ações, configuração da API e instalação de consultas;
vínculos Slack/GitHub continuam vazios, SLA desabilitado e políticas vazias.
Isso não valida permissões GitHub, migrations, disponibilidade atual da API
ou dados reais de relatórios. A auditoria das rotas foi feita no código.

A sequência permanece 0 → 1 → 2. A Fase 4 depende da correlação e da
preservação das decisões da Fase 2. A Fase 3 depende dos endpoints de recibos
existentes e de sua validação de identidade; não deve ser confundida com os
links já publicados. Resiliência de métricas/Digest pode ser trabalhada
independentemente, mantendo o Digest desativado. Não iniciar as atualizações
dependentes da Central antes de encerrar a validação pendente da Fase 1.

Próximo passo da Fase 1: teste autorizado de uma mensagem no canal já usado
pela Central, confirmação de canal/timestamp e reprocessamento que não publique
outra mensagem. Não habilitar globalmente a correlação antes dessa validação.

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
| SLA — Aprovação Pendente | Publicado, envios aguardam política | Timer persistente com dois limites e escalonamento ao responsável configurado. |

Os estados locais da tabela foram conferidos em 2026-11-08; esta integração
de código em 2026-11-13 não altera nem revalida a instância n8n.
Exceções em 2026-11-14: o SLA e suas conexões no Core foram implementados e
publicados; a Central recebeu os links de PR/recibo e a restrição de redirects
descritos abaixo, também publicados. Os demais fluxos não foram revalidados
funcionalmente nessa etapa.
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
| `APROVA_URL_PUBLICA` | Configuração local ainda precisa ser conferida; os endpoints `/receipts/{id}`, `/receipts/{id}/verify` e `/receipts/keys` existem no código integrado. |

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

Os dez JSONs de workflows em `workflows/n8n/` são portáteis e inativos. O feedback e as
atualizações de Core, Central e Digest estão sincronizados com a instância,
com remoção de credenciais/referências locais e preservação dos bloqueios:

| Diferença | Situação |
| --- | --- |
| Feedback | `feedback-motivo-aprovacao.json`, sem credenciais ou IDs locais. |
| Core e Central | Chamadas ao feedback por `APROVA_FEEDBACK_WORKFLOW_ID`; configurar o ID da importação no ambiente. |
| Digest | Instância e export leem `APROVA_CHAT_INSTALLATION_ID`, sem valor real no JSON. |
| Trello | Export e instância usam cursor `before` para comentários. |
| Onboarding | `onboarding-repositorio.json`, baseado na versão publicada, sem credenciais ou destinos locais; Trello/Slack desabilitados até configurar a importação. |
| Core → onboarding | Nó `07L`, por item e aguardando retorno; configurar `APROVA_ONBOARDING_WORKFLOW_ID`. Demais rotas preservadas. |
| SLA | `sla-aprovacao.json`, baseado na versão publicada, sem credenciais, dados fixados ou IDs locais; tabelas selecionadas por ambiente. |
| Core → SLA | Nó `07M` nas rotas de espera e decisão, por item e sem aguardar; configurar `APROVA_SLA_WORKFLOW_ID`. Central e feedback preservados. |

Os exports desta revisão foram autorizados. Novas sincronizações exigem
autorização; preservar a correção Trello e a portabilidade. Na importação,
associar credenciais Slack aos nós de publicação e aviso do feedback, selecionar
o subworkflow no Core/Central e manter a API de feedback desabilitada até
existir endpoint real. Esta exportação não alterou a instância.
O workflow manual `aPRova | Testes | Onboarding de Repositório` permanece
somente na instância, fora da rota de eventos e fora dos exports.
O SLA funcional e as duas ligações novas do Core foram exportados após
autorização, preservando as demais configurações dos arquivos existentes.
As definições de suas tabelas estão em `recursos/tabela-sla-eventos.json` e
`recursos/tabela-sla-controle.json`, sem dados locais; a segunda inclui a
linha inicial de controle necessária em uma instalação nova. O workflow
manual de SLA permanece somente na instância e não foi exportado.
Esta sincronização não alterou workflows, tabelas ou envios na instância.
Novas sincronizações também exigem autorização.

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
| `07A`, `07H`, `07M` | Espera e decisão também seguem ao SLA, modo `each`, sem aguardar; Central e feedback preservados. |
| Demais saídas | Pontos de extensão e registro sem integração. |

O tenant autenticado define a instalação do evento, sem substituição pelo
ID global de consultas. `relatorio_periodico` é pedido da automação,
não um sétimo evento emitido pelo worker, e não publica Digest.

Falta validar worker → Core com dados reais; implementar emissão de
`instalacao` (a de `dependencia_nova` já existe); configurar e validar os
avisos reais do SLA e implementar a atualização do alerta original.

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
O publicador real usa o nome de Check `aPRova` (`internal/publicacao`).
O onboarding exige esse nome e o App ID do publicador, confirmados no
ambiente; os checks de CI não substituem a identidade do Check do produto.

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

O export mantém integrações desabilitadas por padrão e lê quadro/nome/canal das variáveis.
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

## SLA — Aprovação Pendente

O workflow `aPRova | SLA | Aprovação Pendente` está publicado na instância.
Política de prazo pertence à equipe e não muda risco, aprovação ou estado do
núcleo. Usa os eventos existentes, sem `/metrics` ou acesso ao banco do aPRova.

### Entrada, timer e persistência

O Core chama `07M | Acompanhar SLA de Aprovação` a partir de `07A`
(`aguardando_aprovacao`) e `07H` (`aprovacao_registrada`). A referência vem de
`APROVA_SLA_WORKFLOW_ID`, modo `each`, sem aguardar; Central e feedback mantêm
suas conexões. O SLA valida a origem, o contrato normalizado, a instalação,
`meta.execution_id` (task_run), repositório, PR e horário do evento. Aceita
início de risco ALTO e encerramento por `aprovada`/`alteracoes_solicitadas`.
Eventos não escolhem políticas, canal ou responsável.

Data Table `aprova_sla_eventos`: `chave`, `evento_id`, `tipo`, `contexto`
(String) e `ocorrido_em` (Date). A chave combina instalação, task_run,
repositório e PR. Guarda fatos de início, encerramento, reserva e resultado
dos avisos. Encerramento recebido antes do início continua encerrado;
reenvio não reinicia a contagem. Uma análise mais recente da mesma PR
substitui a antiga na seleção do temporizador.

O timer verifica a cada minuto. Os dois limites contam desde
`meta.ocorrido_em`, em minutos corridos; o segundo não soma outro prazo ao
primeiro. A persistência substitui execuções estacionadas em Wait: após
reinício, o próximo ciclo retoma os prazos vencidos. Se ambos vencerem na
interrupção, o primeiro é tratado antes do segundo, que exige confirmação
do primeiro envio.

Data Table `aprova_sla_controle`: `chave`/`executor` (String) e
`bloqueado_ate` (Number, timestamp em milissegundos). Possui uma única linha
`chave=temporizador`. Reserva condicional do ciclo, com validade de 15
minutos e liberação ao terminar, evita ciclos simultâneos. Até dez avisos
são processados por ciclo, um por vez; os demais ficam para o próximo.

| Nós | Função |
| --- | --- |
| `E01–E04` | Entrada normalizada, validação, persistência e resultado do acompanhamento. |
| `T01–T04` | Timer, reserva do ciclo, histórico e planejamento dos prazos/pendências. |
| `T05–T10` | Seleção dos avisos, lote unitário e releitura de decisões/configuração. |
| `T11–T12` | Reserva persistente do aviso antes da chamada ao Slack. |
| `T13–T15` | Envio por `aPRova | Slack`, confirmação de canal/timestamp e resultado persistente. |
| `T16`, `T90`, `T99` | Cancelamento na execução, liberação do ciclo e resumo das pendências. |

### Configuração segundo a padronização

`APROVA_SLA_HABILITADO=true` libera envios apenas para políticas habilitadas
e válidas. `APROVA_SLA_CANAL_ID` é um canal padrão opcional.
`APROVA_SLA_POLITICAS_JSON` contém uma lista com os campos abaixo:

| Campo | Uso |
| --- | --- |
| `installation_id`, `repositorio` | Instalação e `owner/repo`, sem inferir vínculos. |
| `habilitado` | Booleano `true`, junto com a flag global, autoriza a política. |
| `primeiro_limite_minutos` | Inteiro positivo para renotificar o canal. |
| `segundo_limite_minutos` | Inteiro maior que o primeiro, também desde o início da espera. |
| `canal_slack` | Canal próprio da política; pode usar o padrão configurado no ambiente. |
| `responsavel_slack` | ID de usuário confirmado para mencionar no segundo aviso. |

Política ausente/ambígua, prazos inválidos ou canal ausente geram
`configuracao_pendente`, sem publicação. Sem responsável, o primeiro aviso
pode funcionar e o segundo fica pendente. Nenhum prazo ou pessoa foi
escolhido em nome da equipe: a instalação atual tem flag `false`, políticas
vazias e canal não preenchido; o recebimento dos eventos permanece ativo.

As tabelas são selecionadas pelos IDs explícitos de
`APROVA_SLA_EVENTOS_TABELA_ID` e `APROVA_SLA_CONTROLE_TABELA_ID`, evitando
resolução ambígua entre nomes funcionais e de teste.
Compose, `.env.example` e `.env` receberam as seis variáveis do SLA; só os
IDs do workflow e das tabelas foram preenchidos no ambiente local. O container foi
recriado preservando volume, credenciais e segredos existentes. Alterar
políticas exige recarregar o ambiente, como nas demais integrações.

### Confirmação, validação e pendências

Reserva impede repetir o mesmo estágio. Só canal/timestamp confirmados
marcam envio concluído. Erro/timeout gera `incerto_1` ou `incerto_2`, sem
retry automático. Falha entre reserva e confirmação exige conferir Slack e
histórico antes de recuperar manualmente; não remover reservas às cegas.

A releitura impede avisar após decisão conhecida, mas há corrida entre
decisão e publicação. Eventos perdidos, fechamento de PR sem decisão ou
nova análise não recebida podem deixar o acompanhamento desatualizado.
Ainda não existe endpoint de estado atual: o adaptador retorna
`revalidacao_nucleo=indisponivel`, e a mensagem informa que a pendência se
baseia nos eventos recebidos. Uma leitura futura deverá entrar antes do
envio, distinguindo ausência de dados de pendência confirmada.

Passaram 31 cenários isolados da lógica: contratos, dois prazos, políticas,
decisões, ordem invertida, duplicatas, instalações, nova análise, reservas,
responsável ausente e confirmação/falha Slack. Na instância, o workflow
manual `aPRova | Testes | SLA de Aprovação` usa tabelas próprias com sufixo
`_teste`, política fictícia e Code sem rede no lugar do Slack. Duas PRs
percorreram ambos os avisos, sem repetição no ciclo seguinte; decisão
anterior não reabriu espera. O funcional não contém esse nó de simulação.

Também foram testados persistência/configuração pendente no funcional,
carregamento das variáveis e evento assinado de pedido de alterações pela
entrada publicada do Core. Esses testes não validam envio Slack real do
SLA, GitHub App, horário útil ou pontualidade sob carga.

Falta acordar os dois limites, canal e responsáveis; habilitar políticas;
validar notificações reais em canal de teste; disponibilizar revalidação do
estado pelo núcleo; e definir horário útil, retenção, recuperação de envios
incertos e atualização da mensagem original. Não excluir fatos de
encerramento/reserva enquanto puderem chegar eventos antigos, pois isso
pode reabrir acompanhamentos ou repetir avisos.

## Slack — Central de Operação

Alertas usam Incoming Webhook enquanto a correlação estiver desabilitada.
Com a Fase 1 habilitada, são publicados pela credencial do bot e correlacionados
em Data Table. Botões abrem formulários pela credencial do bot. A decisão
volta por `POST /acoes-remotas`; não chama GitHub diretamente.

| Nós | Função |
| --- | --- |
| `01–02`, `03A–03B`, `HTTP Request` | Recebem evento, separam e publicam alertas; `03A` inclui acesso à PR, recibo e verificação. |
| `02A–02F` | Buscam/validam contexto OSV e fazem consulta complementar quando possível. |
| `05–07`, `07E` | Interações, assinatura Slack, confirmação e recusa. |
| `07F–07G` | Encaminham ações `aprova_feedback_*` ao feedback, preservando as demais rotas. |
| `08–09`, `10`, `10B` | Separam botões/formulários e abrem aprovação/revisão. |
| `11`, `11R` | Validam decisão/justificativa ou preparam reexecução. |
| `12–14`, `HTTP Request1` | Assinam pedido, consultam núcleo e publicam retorno. |
| `C01–C02` | Validam a configuração/correlação e escolhem publicação rastreável ou envio anterior. |
| `C03–C07` | Reservam o controle de publicação, consultam correlação e bloqueiam duplicatas/ambiguidades. |
| `C08–C08B`, `C09–C11B` | Persistem e conferem reserva; publicam pelo bot; classificam e conferem persistência do resultado. |
| `C12–C13` | Liberam o controle e retornam o resultado da correlação. |

### Correlação persistente — Fase 1

`APROVA_SLACK_CORRELACAO_HABILITADA=true` seleciona os nós `C03–C13`.
Ausente ou `false`, os alertas seguem pelo Incoming Webhook existente.
`APROVA_SLACK_CANAL_ALERTAS_ID`, `APROVA_SLACK_MENSAGENS_TABELA_ID` e
`APROVA_SLACK_CONTROLE_TABELA_ID` precisam estar preenchidos antes de habilitar.
São variáveis novas e próprias da Central; não há fallback para canal de
consulta/onboarding. O canal local foi reaproveitado da Central a partir
de uma interação Slack autenticada, conferido contra os destinos já existentes;
nenhum destino novo foi escolhido. Não registrar os valores neste documento.

Data Table `aprova_slack_mensagens`: String `chave`, `estado`, `contexto`,
`canal`, `ts`, `executor`; Date `atualizado_em`. A chave combina instalação,
análise, repositório normalizado, PR e tipo de alerta. O contexto guarda o
SHA e o payload original, preservando risco, recibos, OSV e ações. SHA
incompatível, mudança de canal, contexto inválido ou mais de uma linha para
a mesma chave impedem novo envio. Alertas de falha e espera são distintos.

Data Table `aprova_slack_controle`: String `chave`/`executor`, Number
`bloqueado_ate`. Deve conter exatamente uma linha inicial
`chave=publicacao`, `bloqueado_ate=0`, `executor=''`. A atualização condicional
reserva a publicação por até 15 minutos. Controle ocupado/interrompido gera
erro explícito antes do envio. Não há fila ou retry automático: uma execução
recusada por controle ocupado precisa ser conferida e reprocessada no
subworkflow, pois o Core pode já ter deduplicado o evento. Não afirmar garantia
de entrega exatamente uma vez ou cobertura de carga concorrente em produção.

Antes do Slack, grava `reservado` e confere linha, proprietário e validade
do controle. Só canal correspondente e timestamp válido confirmam a mensagem.
Resposta explícita `ok=false` gera `falha`; timeout ou retorno incompleto gera
`incerto`. Os três estados `reservado`, `incerto` e `falha` bloqueiam repetição
automática e exigem recuperação manual. Uma publicação confirmada é reutilizada
como `ja_publicado`. O resultado só é concluído após conferir a escrita e
a liberação do controle. Falha de persistência após envio exige consultar
Slack, execução e reserva; não apagar linhas para tentar novamente às cegas.

Os 15 nós foram adicionados no n8n, sem modificar os parâmetros dos 27
nós anteriores. As duas conexões dos alertas passaram pelo seletor da
correlação; as demais rotas, formulários, feedback e retorno das ações foram
preservados. A versão foi publicada com a flag desabilitada. O export é inativo
e sem credenciais, IDs locais ou dados fixados. As duas definições de tabelas
estão em `workflows/n8n/recursos/tabela-slack-*.json`, sem dados locais;
somente a definição do controle inclui a linha inicial.

O workflow manual `aPRova | Testes | Correlação de Alertas` permanece só na
instância. Usa tabelas com sufixo `_teste` e Code no lugar do Slack, sem rede
externa. Foram confirmados no nó final: persistência, releitura, supressão de
duplicata e recusa de reserva ocupada, nos cenários confirmado/incerto/falha.
A primeira execução foi interrompida antes do nó final por saída vazia e não
conta como validação; as configurações de execução foram corrigidas e os três
cenários chegaram ao nó final. O nó temporário de recuperação dessa reserva
de teste ficou desabilitado. O funcional não contém nós de simulação.

O `.env` local recebeu as quatro variáveis, com flag `false`; Compose e
`.env.example` receberam o contrato portátil. O container não foi recriado
nesta fase, portanto as variáveis novas ainda precisam ser carregadas.
Sem configuração carregada, o padrão também mantém o caminho anterior.
Falta o teste Slack real e a decisão de habilitar a correlação; atualizar
mensagem após decisão, OSV posterior e verificação automática do recibo
continuam nas fases seguintes.

### Ações e recibos

Quem envia o formulário deve ser quem o abriu. Aprovação/revisão exigem
justificativa também no backend. O pedido leva ID Slack e, quando disponível,
`task_run_id` original. O núcleo resolve identidade por
`SLACK_GITHUB_VINCULOS`, verifica permissão atual e impede autoaprovação.

`Pedido`, `Servico.Processar` e `ExecutorNoGitHub` ficam em
`internal/acaoremota/`. `RepositorioPostgres` reserva `id_acao`
e registra tentativas; migration 21 inclui
`remote_action_attempts.justification`. A correlação confere análise,
tenant, repositório e PR; justificativa também segue na review GitHub.
Somente `202 + status=aceito` confirma a execução nesse contrato.
Timeout/conexão recusada não confirmam ação.

O alerta de aprovação oferece os links **Abrir PR**, **Ver recibo** e
**Verificar recibo**, preservando os botões de decisão e seu contexto.
O worker já emite `url_recibo` terminada em `/receipts/{id}/verify`; o nó
`03A` aceita esse formato ou `/receipts/{id}` e deriva os dois destinos sem
duplicar `/verify`. Preserva a base pública recebida, inclusive prefixo de
caminho; não a troca por `APROVA_API_URL`, que pode ser interna ao n8n.
URL ausente ou incompatível deixa o recibo indisponível no alerta, sem
inventar ID/link. URLs com credenciais, query ou fragmento não são usadas.
Exibir o link não comprova validade do recibo: a consulta só ocorre quando
a pessoa o abre. O resultado de `/verify` exige conferir `valido` e suas
verificações, pois HTTP 200 também pode trazer recibo inválido.

O nó `13` não segue redirects ao enviar o pedido assinado para
`/acoes-remotas`. A resposta original segue para a classificação existente,
sem reenviar o corpo a outro destino ou confirmar ação por HTTP 3xx.
Esses dois ajustes estão no arquivo portátil e na versão publicada da
Central. A atualização da instância preservou os outros 25 nós, conexões
e credenciais, sem avisos de validação do n8n. Passaram 123 testes locais
dos workflows, incluindo os formatos de URL do worker e os casos de recibo
ausente/incompatível. Não foram enviados eventos, mensagens Slack ou
pedidos de ação; a navegação e a verificação de recibo real continuam
pendentes de validação integrada.

Falta configurar vínculos na API, conferir/aplicar migrations e validar em
PR real: autorizado, sem permissão, autoaprovação, repetição, erro e proteção
de branch. Reexecução ainda usa `ReenfileiradorIndisponivel`, sem executor
real. A Fase 1 prepara o armazenamento/correlação da mensagem; a atualização
do alerta após decisão ainda pertence à Fase 2. O retorno do formulário não
faz essa atualização.

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

O worker já emite `dependencia_nova` quando o gate identifica dependências
novas; falta validar worker → Core → OSV com uma PR real. Permanecem
pendentes contexto na review GitHub e atualização do alerta quando OSV chega
depois. Definir retenção/limpeza do cache e validar cobertura além dos limites.

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

Cursor `before` aplicado na instância e coberto pelos testes isolados; falta
validar histórico extenso real e definir recuperação de avisos, concorrência e mapeamento
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

No nó `02`, os vínculos vêm exclusivamente de `SLACK_GITHUB_VINCULOS`,
`api_feedback_habilitada=false`, `caminho_api_feedback=''`. Sem vínculo
único, termina em `configuracao_pendente` sem pergunta.
O código lê `SLACK_GITHUB_VINCULOS`; o Compose atual repassa essa
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

O núcleo já tem gate determinístico e calibração opcional por histórico
([calibracao.md](calibracao.md)); o feedback posterior ainda não está integrado
a esse fluxo. Falta consumir o sinal rotulado e registrar no artigo a diferença entre
aprender com resultado observado e com motivo declarado pelo humano.
Não afirmar ganho quantitativo antes de medir. O desenvolvimento posterior
deste ramo foi adiado pelo usuário.

## Endpoints e limites do núcleo

| Rota atual | Uso |
| --- | --- |
| `GET /health` | Saúde do processo, sem verificar dependências. |
| `GET /ready` | PostgreSQL/Redis, até 2s; 200 ou 503 com estados. |
| `GET /reports/weekly` | Agregado por instalação/sete dias UTC; HMAC da query bruta. |
| `GET /receipts/{id}` | Conteúdo público do recibo, incluindo payload e assinatura. |
| `GET /receipts/{id}/verify` | Verificação do recibo pelo núcleo; conferir `valido`, não somente HTTP 200. |
| `GET /receipts/keys` | Chaves públicas aceitas para verificação de recibos. |
| `POST /webhooks/github` | Entrada de eventos GitHub autenticados. |
| `POST /acoes-remotas` | Ação humana autenticada/autorizada no núcleo. |

`Servidor.manipularRelatorioSemanal`, `inicioDoRelatorio`,
`domain.RelatorioSemanal` e query `GetWeeklyReport` definem o relatório.
Retorna instalação, período, total, estados, aprovação humana, risco médio,
reversões e taxa. Taxa = concluídas / (concluídas + rejeitadas + falhas),
nula sem finalizadas. `em_andamento` considera execuções recebidas na janela,
não todas as atuais; reversões podem corresponder a análises antigas.
Falha da API não representa semana vazia.

Não existem endpoints para listar PRs pendentes, detalhar execução ou gravar
feedback posterior. Os endpoints públicos de recibos já existem.
O worker emite análise, espera de aprovação, decisão, dependências novas
identificadas e falha recuperável. Gate, recibo, publicação e calibração
opcional estão implementados; testes locais não comprovam integração externa.

## Pendências compartilhadas e implantação

- API/worker locais iniciados em 2026-11-14 com ambiente atualizado. O banco
  foi reconciliado da antiga migration 15 para a 21, sem reset/force e com os
  três registros de ações remotas preservados. Ensaio em cópia e comparação
  com instalação limpa confirmaram esquema idêntico; `migrate up` retorna
  `no change`. `/health` e `/ready` responderam 200, inclusive a partir do
  container do n8n. Falta validar worker → Core no repositório de teste.
- `.env` usa as variáveis atuais de recibos. Como não havia chave configurada
  nem recibos no banco, foi gerado um par Ed25519 local e sua chave pública
  foi conferida em `/receipts/keys`. A imagem `aprova-sandbox-go:local` foi
  construída do código atual e configurada. `APROVA_URL_PUBLICA` aponta para
  `http://localhost:8080`; acesso externo ainda exige endereço confirmado.
  Backups privados, evidências e comandos dos processos ficam no
  [relatório local](../tmp/atualizacao-local-20261114/RELATORIO.md).
- Confirmar instalação, permissões e credenciais no GitHub real. As credenciais
  já estão disponíveis; falta validação integrada.
- Configurar vínculos Slack/GitHub para ações e feedback, sem aceitar identidade
  declarada pela IA como autoridade.
- Conferir URLs HTTPS do Slack/GitHub ao mudar túnel; produção exige domínio
  estável e rede/configuração próprias.
- Validar gate/calibração, recibos e publicação reais com as integrações;
  implementar o evento de instalação, respeitando a separação núcleo/periferia.
- Configurar prazos, canal e responsáveis do SLA publicado; validar envios
  reais e futura revalidação de estado, recuperação e retenção.
- Configurar credenciais/chamadas após importar os exports; preservar
  portabilidade e correção Trello nas próximas sincronizações autorizadas.
- Definir implantação da API/worker, versões de imagens, backups do n8n/chave
  de criptografia, retenção e recuperação de falhas para produção.

A preparação local iniciou API/worker e validou a saúde das dependências;
aprovações, métricas reais e o fluxo completo continuam sem validação nesta
etapa. O onboarding criou uma lista no quadro de teste e enviou resumos ao canal
usado nos testes do projeto. Esses testes não substituem a emissão real pelo
backend nem a confirmação de proteção autenticada ainda pendentes.

## Padronização aplicada — 2026-11-08

Nove workflows sincronizados após comparação e backup fora do Git.
Referências, quadro/listas, canal e flags do onboarding leem ambiente
`APROVA_*`. A paginação Trello `before` está aplicada também na instância.
Oito workflows continuam ativos; Digest continua inativo. As três credenciais
usadas foram renomeadas para `aPRova | Serviço`, preservando IDs e dados
cifrados; exports removem associações pessoais. O `.env` recebeu apenas
configuração reaproveitada da instância, sem novos vínculos de pessoas.

O [contrato de configuração](../workflows/n8n/README.md#contrato-de-configuração--2026-11-08)
substitui as instruções anteriores de editar IDs nos nós e de manter vínculos
locais no feedback. As evidências funcionais acima são históricas; preencher
variáveis não valida GitHub, Slack, Trello ou Gemini. Nesta etapa só foram
executados testes isolados e inspeção estrutural, sem disparos externos.
Permanecem pendentes identidades Slack/GitHub, validação de permissões reais,
endpoint de feedback, executor de reexecução e validação integrada do Check,
gate, calibração, recibos e publicação incorporados da `dev`.

# Central de operação aPRova no Slack

## Instrução permanente para a IA

Este documento é a referência central para registrar todas as informações
sobre os workflows do aPRova que construirmos a partir de agora, incluindo
workflows de outras integrações além do Slack.

Antes de construir ou alterar qualquer workflow, leia obrigatoriamente este
documento e [eventos-n8n.md](eventos-n8n.md). Siga as orientações, contratos e
regras definidos em `eventos-n8n.md` e considere as decisões e pendências já
registradas neste documento. Ao concluir o trabalho,
atualize este mesmo arquivo seguindo a estrutura existente: o que foi
implementado, inventário de nós e suas funções, funções e contratos do
backend, configuração e estado do ambiente, validações realizadas e o que
ainda falta para concluir a task.

Para cada novo workflow, identifique seu nome e finalidade e acrescente as
informações correspondentes sem apagar o conteúdo dos demais workflows.
Mantenha as mudanças dos workflows existentes nas respectivas seções,
registre a data das atualizações e diferencie implementação, configuração,
simulação e validação real. Não registre segredos nem valores de credenciais.

Planeje todos os workflows considerando desenvolvimento e futura produção.
Mantenha contratos e regras de autorização independentes do ambiente; use
configuração externa para endereços, IDs de workflows e segredos. Documente
quais escolhas são locais e o que precisa mudar na implantação. Testes locais
e publicação na instância local do n8n não comprovam prontidão para produção.

Estado da implementação em 2026-11-01. Esta entrega prepara a interface de
operação local e mantém a autorização no núcleo. Não comprova aprovação em
um repositório GitHub real nem calibração automática do gate.

## O que foi implementado

- Eventos do núcleo são autenticados com HMAC-SHA256 sobre o corpo bruto,
  usando `N8N_SEGREDO_EVENTOS`, com janela de cinco minutos em ambas as direções
  do relógio. A entrada marca a origem como verificada somente após conferir.
- Eventos inválidos recebem 400 e eventos não autenticados recebem 401.
  Eventos válidos recebem 202 antes das integrações.
- Deduplicação por `tenant_id:id_evento`, persistida pelo n8n para as últimas
  10.000 chaves. Tipos desconhecidos não acionam integrações.
- `aguardando_aprovacao` e `falha_recuperavel` chamam a Central do Slack por
  subworkflow, sem esperar sua conclusão.
- Aprovação e revisão recebem justificativa em formulário. O usuário do envio
  precisa ser o mesmo que abriu o formulário. O núcleo recebe o ID Slack,
  sem aceitar do chat um login GitHub como autoridade.
- Decisões e reexecução encaminham o `task_run_id` original quando disponível.
- A API é chamada por endereço configurável, com HMAC independente, corpo
  bruto idêntico ao assinado, timeout e captura da resposta HTTP completa.
- O retorno usa o Incoming Webhook das notificações e informa autor, ação,
  justificativa, repositório, número e link da PR, além do resultado do núcleo.
- A API usa PostgreSQL para o histórico e a reserva que impede repetição de
  `id_acao`. A justificativa fica em `remote_action_attempts.justification`.
- Aprovação e revisão exigem justificativa no backend, inclusive se alguém
  chamar a API diretamente. O texto também aparece na review do GitHub.
- Os JSONs importáveis e os testes foram incluídos no repositório. Os exports
  não contêm o Incoming Webhook real, credenciais ou `pinData`.

## Nós do workflow de entrada

| Nó | Função |
| --- | --- |
| `01 | Receber Evento` | Webhook POST com Raw Body ligado. |
| `01B | Verificar Assinatura do Núcleo` | Confere HMAC, timestamp e corpo bruto; só libera eventos autênticos. |
| `01E | Recusar Evento Não Autenticado` | Responde 401 na saída de erro da autenticação. |
| `02 | Capturar Payload Original` | Preserva o evento e a marca de origem verificada. |
| `04 | Normalizar Evento aPRova` | Adapta o envelope v1 ao contrato interno, preservando IDs e dados do núcleo. |
| `03`, `04b` e `05E` | Validam o envelope e respondem 400 quando inválido. |
| `05 | Responder Recebido` | Confirma o recebimento com 202. |
| `05B | Descartar Eventos Repetidos` | Retém a primeira ocorrência de cada chave tenant/evento. |
| `06 | Rotear tipo_evento` | Encaminha pelo tipo, mantendo saída para desconhecidos. |
| `07A` e `07J` | Pontos das rotas de aprovação pendente e falha recuperável, agora ligados à Central. |
| `08 | Executar Central do Slack` | Executa a Central identificada por `APROVA_SLACK_WORKFLOW_ID`, sem aguardar. |

## Nós da Central do Slack

| Nó | Função |
| --- | --- |
| `01` e `02` | Recebem o evento normalizado e separam aprovação pendente de falha recuperável. |
| `03A | Preparar Aprovação` | Mostra PR, autor, motivo do risco, contexto OSV disponível e botões Aprovar/Pedir revisão. |
| `03B | Preparar Falha` | Mostra diagnóstico informado pelo núcleo e botão Reexecutar. |
| `HTTP Request` | Publica a notificação por `SLACK_INCOMING_WEBHOOK_URL`. |
| `05 | Receber Interação do Slack` | Recebe cliques e envios dos formulários com corpo bruto. |
| `06 | Verificar Assinatura do Slack` | Autentica a interação e recusa timestamp fora da janela. |
| `07E | Recusar Interação Não Autenticada` | Responde 401 para a interação inválida, sem executar ação. |
| `07`, `08` e `09` | Confirmam rapidamente o recebimento e separam cliques, formulários e ações. |
| `10` e `10B` | Abrem os formulários de aprovação/revisão, preservando usuário, PR e canal. |
| `11 | Preparar Pedido de Decisão.` | Valida formulário, usuário e justificativa e prepara o contrato da API. |
| `11R | Preparar Pedido de Reexecução` | Valida o clique e prepara reexecução sem exigir justificativa. |
| `12 | Assinar Pedido ao Núcleo` | Assina o JSON com `N8N_SEGREDO_ACOES` e timestamp. |
| `13 | Enviar Ação ao Núcleo` | POST em `APROVA_API_URL/acoes-remotas`, com timeout de 15 segundos e resposta completa. |
| `14 | Preparar Retorno ao Usuário` | Traduz o resultado do núcleo para a mensagem de operação. |
| `HTTP Request1` | Publica o retorno pelo mesmo Incoming Webhook da notificação. |

## Funções e contratos do backend

| Função/estrutura | Responsabilidade e alteração |
| --- | --- |
| `montarAcoesRemotas` (`cmd/api/main.go`) | Recebe o registro persistente construído com o pool PostgreSQL, em vez de criar registro em memória. |
| `Pedido` (`internal/acaoremota/contrato.go`) | Exige justificativa para aprovação/revisão e aceita `task_run_id` opcional em UUID. |
| `interpretarPedido` | Mantém validação estrita, recusa campos desconhecidos e remove espaços nas pontas da justificativa. |
| `Servico.Processar` / `autorizar` / `levantarSituacao` | Mantêm a resolução de identidade, consulta de permissão no GitHub e impedimento de autoaprovação antes da execução. |
| `Servico.tentativaDe` | Preserva a justificativa e o ID da análise no registro da tentativa. |
| `ExecutorNoGitHub.Executar` | Publica `APPROVE` ou `REQUEST_CHANGES`; a aprovação agora inclui a justificativa. |
| `RepositorioPostgres.Reservar` / `Registrar` | Mantêm o histórico e a proteção contra repetição no banco. |
| `RepositorioPostgres.gravarTentativa` / `CreateRemoteActionAttempt` | Gravam justificativa e correlacionam a análise por ID, tenant, repositório e PR. Sem ID informado, usam a análise mais recente. Um ID incompatível não é associado a outra análise. |
| Migration `000013` | Adiciona a justificativa ao histórico com limite de 4000 caracteres e rollback correspondente. |

`202 + {"status":"aceito"}` significa, neste backend, que a ação passou pela
autorização e pelo executor. `400`, `401`, `403`, `409` e `500` recebem mensagens
próprias. Timeout, conexão recusada, corpo inválido ou outro retorno não
confirmam execução. Os motivos detalhados de uma recusa ficam no núcleo;
o n8n não inventa qual permissão faltou.

## Configuração e estado local

- O PostgreSQL local foi inicializado até a migration 13.
- Um segredo de eventos foi criado no `.env` local, que é ignorado pelo Git.
  Os segredos Slack/ações existentes foram preservados. Nenhum valor aparece
  neste documento nem nos exports.
- O Compose foi ampliado e o container do n8n foi recriado com as novas
  variáveis, preservando seu volume.
- O roteador já ativo foi atualizado e republicado com autenticação,
  deduplicação e ligação à Central.
- A Central também foi publicada na instância local. Seu endpoint de
  produção é `/webhook/aprova/slack/interacoes`; configurar esse endereço
  público no Slack App ainda é necessário. O nó 13 está habilitado para
  tentar a API configurada. O modo `/webhook-test/` continua exigindo escuta
  no editor.
- Os formulários usam a credencial Slack já disponível na instância local;
  em outra instalação, associe sua própria credencial aos nós 10 e 10B.
- O endereço padrão do núcleo no container é
  `http://host.docker.internal:8080`. A API precisa estar em execução com suas
  próprias dependências e configuração.

### Conferência de ambiente e preparação para produção — 2026-11-01

O `docker-compose.yml` atual define PostgreSQL, Redis e n8n. Não define um
serviço da API. Na conferência desta data, esses três containers estavam em
execução; nenhum container da API foi identificado. O README orienta iniciar
a API no host com `make run-api`. O Dockerfile de `imagens/go` é do sandbox,
não da API.

`APROVA_API_URL=http://host.docker.internal:8080` corresponde ao n8n em Docker
chamando uma API no host. Se a API passar a um container na mesma rede, essa
variável deve apontar ao nome do serviço e à porta interna definidos para ele.
Na implantação, use o endereço acessível entre os serviços daquele ambiente.
Não copie o endereço de desenvolvimento automaticamente para produção.

Antes da produção, definir e validar:

- Empacotamento e implantação da API e do worker; rede entre núcleo e n8n.
- Endereços públicos HTTPS para webhooks e endereço interno da API.
- Credenciais, segredos e IDs próprios do ambiente; persistência e backup dos
  dados e da chave de criptografia de credenciais do n8n.
- Versões fixadas das imagens, exposição de portas e configuração de banco e
  Redis. O Compose local publica essas portas e usa credenciais de exemplo;
  não é um arquivo de implantação de produção.
- Execução das migrations antes da nova API e revisão das permissões de
  acesso às variáveis/segredos pelos nós e pelos editores do n8n.
- Testes integrados, recuperação de falhas e procedimento de implantação.

Esses itens são requisitos de preparação, não funcionalidades já implantadas.

## Validação realizada

- Suíte Go de `internal`, `cmd`, `scripts` e testes funcionais, com `-race`.
- Testes PostgreSQL com `-race` em container temporário: migrations completas
  de ida e volta, justificativa, correlação da análise, isolamento de tenant
  e proteção contra repetição após criar outra instância do repositório.
- `go vet` nos pacotes alterados e geração dos arquivos pelo sqlc 1.30.0.
- Testes Node dos exports: formulários, justificativa máxima, usuário trocado,
  respostas HTTP, falha de transporte, reexecução, HMAC, replay e exemplos de
  eventos do repositório. A CI executa esses testes sem dependências npm.
- Execuções isoladas no n8n: formulário de aprovação com retorno 202 e de
  revisão com retorno 403. Assinatura Slack, preparação e assinatura do pedido
  executaram de verdade; API e envio ao Slack foram simulados para não alterar
  PRs nem publicar mensagens de teste.
- Webhook local publicado: evento de tipo desconhecido e assinado recebeu
  202; sua repetição recebeu 202 e foi descartada antes do roteador; corpo
  adulterado recebeu 401 e não acionou nenhuma integração.

## O que falta para concluir a task

1. Preencher as credenciais/configuração do GitHub App e os vínculos
   `SLACK_GITHUB_VINCULOS`, iniciar a API atualizada e validar em PR de teste.
   A instalação local ainda não tem esses valores configurados.
2. Configurar no Slack App a URL pública de produção
   `/webhook/aprova/slack/interacoes`, com HTTPS acessível pelo Slack.
3. Conectar a emissão automática dos eventos ao processamento do núcleo.
   Hoje o contrato e o script de envio existem, mas o fluxo de análise ainda
   não emite automaticamente esses eventos.
4. Implementar o consumidor/orquestrador de análises e a reexecução real.
   `ReenfileiradorIndisponivel` permanece intencionalmente ligado; clicar em
   Reexecutar não pode anunciar sucesso enquanto não houver executor.
5. Consumir a justificativa persistida na calibração do gate. O texto já chega
   à API, ao histórico e à review; ainda faltam a composição da memória com
   embedding e o consumidor que usa isso na decisão do gate. Não foram criados
   embeddings ou regras de calibração fictícios.
6. Validar a sequência completa com o GitHub real: usuário autorizado,
   sem permissão, autoaprovação, repetição, erro e indisponibilidade. Confirmar
   também o comportamento com as proteções de branch do repositório de teste.

A atualização do alerta original após `aprovacao_registrada`, prevista no
guia geral de eventos, continua sendo uma evolução separada: exige identificar
e guardar a mensagem original e tratar o evento de decisão. Não faz parte dos
retornos de formulários implementados aqui.

## Workflow aPRova | Dependências | Consulta OSV — revisão de 2026-11-02

### Finalidade e regra crítica

Quando o núcleo em Go emite `dependencia_nova`, consultar o OSV e enriquecer
o texto do alerta com os avisos, CVEs e severidades conhecidas da versão.
O resultado é informativo. O workflow não altera `risk_gates`, não recalcula
`risco.nivel`, não acessa PostgreSQL e não chama uma API do núcleo para alterar
risco. A decisão do gate continua exclusivamente no núcleo.

### O que foi implementado e conferido

- A entrada aceita o evento já normalizado pelo Core. Apenas o tipo
  `dependencia_nova`, com lista de dependências não vazia, segue para consulta.
- Cada dependência é consultada com `package.name`, `package.ecosystem` e
  `version`, sem autenticação, em `https://api.osv.dev/v1/query`.
- As respostas são consolidadas por tenant, evento, ecossistema, nome e versão.
  O contexto da PR e o risco recebidos são preservados nos resultados.
- A paginação envia `page_token`, para quando não existe `next_page_token` ou
  quando o HTTP não é 200 e limita a dez requisições, com intervalo de 200 ms.
  O timeout de cada requisição é de 30 segundos.
- Avisos retirados (`withdrawn`) não entram na contagem. Avisos OSV são
  contados por ID; CVEs são extraídas de IDs/aliases e deduplicadas. Dois
  avisos de fontes diferentes sobre uma mesma CVE não viram duas CVEs.
- A consulta é classificada como `concluida`, `incompleta` ou `falha`.
  Totais definitivos só existem para consultas concluídas. Falha HTTP,
  timeout, resposta inválida ou paginação pendente não significam ausência
  de vulnerabilidades.
- Severidades textuais das fontes são apresentadas como crítica, alta,
  média ou baixa. Vetores CVSS são preservados nos dados, mas não são
  convertidos em severidade pelo workflow. Se não houver classificação
  textual reconhecida, o alerta informa que ela não foi fornecida.
- Um alerta em texto simples é enviado por dependência ao Incoming Webhook
  configurado. No máximo dez CVEs aparecem no texto; a saída da execução
  mantém os demais IDs. Esse histórico está sujeito à retenção do n8n.

### Inventário de nós e funções

| Nó | Função |
| --- | --- |
| `01 | Receber Evento Normalizado` | Trigger de subworkflow com `inputSource=passthrough`, preservando o item do Core. |
| `02 | Validar Evento de Dependências` | IF por tipo do evento e lista não vazia; a saída falsa encerra sem integração. |
| `03 | Separar Dependências` | Split Out em `dependencias`, preservando os demais campos e nomeando o pacote atual como `dependencia`. |
| `04 | Consultar OSV` | POST público por pacote/versão, com resposta completa, paginação, timeout e continuação em caso de falha de transporte. |
| `05 | Consolidar Consulta por Dependência` | Código em todos os itens; `Map` de grupos e avisos, extração de CVEs, descarte de avisos retirados e status de completude. Usa `itemMatching` e `pairedItem` para manter o vínculo com a entrada correta. |
| `06 | Preparar Alerta de Dependência` | Código por item; monta `texto_alerta` e `slack_payload`, traduz somente severidades informadas e preserva o objeto de risco. |
| `07 | Enviar Alerta Enriquecido` | POST em `SLACK_INCOMING_WEBHOOK_URL`, sem token embutido no export. |

Não houve alteração no Go, na API, nas tabelas ou nas migrations para esta
revisão. Os códigos JavaScript dos nós 05 e 06 ficam dentro do export
`workflows/n8n/dependencias-osv.json`.

### Ligação com o Core e preparação do repositório

- No Core, `06 | Rotear tipo_evento` encaminha `dependencia_nova` para
  `07I | dependencia_nova | Conectar workflow` e depois para
  `Call 'aPRova | Dependências | Consulta OSV'`.
- A chamada usa execução por item e `waitForSubWorkflow=false`. O núcleo
  recebe a confirmação de entrada antes das consultas e do envio do alerta.
- Na instância revisada, Core e OSV estão ativos, com o rascunho atual
  publicado. A seleção direta do ID do workflow na chamada ativa foi mantida.
- O novo export OSV foi adicionado e o export do Core foi atualizado com a
  rota criada no editor. O export usa `APROVA_OSV_WORKFLOW_ID` para evitar
  depender do ID local ao importar em outra instalação.
- A variável foi documentada em `.env.example` e adicionada ao Compose.
  Após importar os JSONs, preencher o ID no ambiente e recriar o serviço do
  n8n para carregar a variável. Não houve reinício de container nesta revisão.
- Os exports estão inativos, sem credenciais, segredos ou dados fixados.
  O webhook Slack e os IDs são configurados por ambiente. Não houve commit,
  push ou implantação de produção durante esta revisão.

### Validações realizadas

- Os sete nós passaram em `validate_node_config` do n8n.
- A execução existente `259`, iniciada pelo Core `258` em 2026-11-02,
  comprova a integração real: evento normalizado com origem verificada,
  consulta OSV HTTP 200 e envio ao Slack com resposta `ok`. Foram 10 avisos
  OSV e cinco CVEs distintas naquela consulta; esses números são históricos
  e não devem ser usados como dados atuais da base OSV.
- A execução isolada `260` no n8n testou duas dependências e três respostas:
  duas páginas de um pacote e HTTP 503 no outro. A consolidação manteve os
  pacotes separados, uma CVE na consulta concluída, total nulo na falha e
  `risco.nivel=BAIXO` em ambos. HTTP OSV e envio Slack foram simulados nesse
  teste; nenhuma mensagem adicional foi publicada para a revisão.
- Treze testes em `workflows/n8n/tests/dependencias-osv.test.cjs` passaram:
  contrato da consulta, destinos HTTP permitidos, preservação do risco,
  CVEs repetidas, paginação, resultados parciais, HTTP 429/500, timeout,
  resposta inválida, consulta vazia, avisos retirados, severidade e CVSS,
  isolamento de contexto e configuração portátil da rota.
- Os doze testes existentes da Central do Slack também passaram, incluindo
  a normalização dos exemplos de eventos. A CI já executa todos os arquivos
  `workflows/n8n/tests/*.test.cjs`; o teste novo está incluído nessa regra.

### Parecer e limites para concluir a task

O comportamento revisado atende à task de consulta e enriquecimento de alerta
Slack, respeitando a exclusividade da decisão de risco no núcleo. Os exports,
testes e documentação podem ser versionados em uma branch para revisão do
time. Isso não significa que a implantação de produção esteja concluída.

Para outra instalação/produção, configurar o ID importado, o webhook Slack,
a saída de rede para o OSV, a retenção do histórico e validar o percurso com
as credenciais e endereços do ambiente. A emissão automática do evento pelo
núcleo continua pendente no projeto; a execução real acima foi provocada
por um evento enviado ao Core, não comprova emissão automática pelo gate.

Melhorias posteriores, sem mudar a regra de risco: validar os campos de cada
dependência antes da chamada HTTP, definir política de retry/backoff para
falhas transitórias e conferir limites de volume por evento. Atualmente o IF
valida tipo e lista; campos ausentes são detectados no consolidado. Não há
comentário em PR/Trello implementado neste workflow: o destino atual é o
alerta Slack. Incluir outro destino deve reutilizar o enriquecimento e manter
a decisão de risco no núcleo.

Referências técnicas conferidas:

- [Contrato POST /v1/query e paginação do OSV](https://google.github.io/osv.dev/post-v1-query/).
- [Esquema OSV: IDs, aliases, withdrawn e severidade](https://ossf.github.io/osv-schema/).
- [Paginação no HTTP Request do n8n](https://docs.n8n.io/build/code-in-n8n/cookbook/http-request-node/pagination.md).

## OSV no alerta e nos formulários de decisão — integração de 2026-11-02

Este complemento atualiza a revisão inicial do workflow OSV acima. O escopo
autorizado é apresentar o contexto ao revisor antes da decisão. A ampliação
da publicação de reviews/comentários no núcleo fica para outra entrega.

### O que foi implementado

- O OSV guarda um resumo informativo na Data Table nativa do n8n
  `aprova_osv_por_analise`, no mesmo projeto dos workflows, com duas colunas
  String: `chave` e `contexto`. Não há acesso direto ao PostgreSQL do núcleo,
  nem alteração de `risk_gates` ou uso desse contexto para autorizar ações.
- A chave inclui tenant, `task_run_id`, repositório e número da PR. A Central
  confere esses campos e rejeita SHA incompatível quando presente nos dois
  eventos. A consulta de uma análise nunca é reaproveitada apenas porque
  pertence à mesma PR.
- A Central busca o resultado antes de montar a mensagem com os botões.
  Contextos de até 24 horas são reutilizados. Resultados vencidos são
  consultados novamente usando somente os pacotes da mesma análise.
- Se `aguardando_aprovacao` já trouxer `dados.dependencias`, a normalização
  existente preserva a lista e a Central pode consultar o OSV antes do alerta
  sem depender de um evento anterior de dependências. A lista continua
  opcional no contrato; o emissor Go não foi alterado nesta entrega.
- A consulta complementar é limitada a dez pacotes, duas páginas e cinco
  segundos por requisição. O prazo total depende da quantidade de pacotes e
  páginas; esses limites não são um timeout global de cinco segundos. Os
  limites atingidos são apresentados como resultados parciais.
- Sem resultado ou pacotes conhecidos, com contexto incompatível ou em
  falha, o alerta declara que o resultado está indisponível. Isso não altera
  o risco nem vira uma afirmação de ausência de vulnerabilidades.
- O resumo inclui o instante da consulta e até três pacotes, em no máximo
  1000 caracteres. Ele aparece antes dos botões e acompanha o valor de ambos
  os botões, permitindo mostrar o mesmo contexto nos dois formulários.
- Nenhuma consulta OSV é feita depois do clique. Os formulários continuam
  pedindo a justificativa humana; o resumo não é enviado em `pedido_acao`,
  não é anexado automaticamente à justificativa e não muda a review publicada
  no GitHub nesta fase.
- O modo `contexto_aprovacao` devolve os resultados à Central sem publicar
  outro alerta OSV. O evento `dependencia_nova` mantém seu alerta separado
  por pacote e também alimenta o contexto persistido.

### Nós adicionados e funções

| Workflow / nó | Função |
| --- | --- |
| Central `02A | Identificar Contexto OSV` | Confere a identificação da aprovação e monta a chave da análise. |
| Central `02B | Buscar Contexto OSV` | Lê a Data Table pelo nome e chave; retorna o registro mais recentemente atualizado. Saída vazia ou falha segue para tratamento explícito. |
| Central `02C | Selecionar Contexto OSV` | Confere correlação, SHA, pacotes e validade temporal; seleciona resumo ou prepara consulta complementar. |
| Central `02D | Consultar OSV Antes do Alerta` | Separa reutilização/indisponibilidade de uma consulta necessária. |
| Central `02E | Consultar Contexto OSV` | Chama o subworkflow OSV por item e espera o resultado antes da mensagem com botões. |
| Central `02F | Receber Contexto OSV` | Confere novamente o resultado contra a aprovação original; erros não impedem o alerta com aviso de indisponibilidade. |
| OSV `06A | Agrupar Contexto OSV da Análise` | Agrupa resultados pela análise e monta um snapshot sem risco ou decisão. Persiste até 50 pacotes e sinaliza cobertura parcial quando limitada. |
| OSV `06B | Guardar Contexto OSV` | Faz upsert pela chave na Data Table; uma falha de cache não autoriza ação nem altera risco. |
| OSV `06C | Retornar Contexto OSV` | Devolve o snapshot ao chamador, preservando o contexto de execução. |
| OSV `06D | Publicar Alerta de Dependência` | Só permite o envio do alerta separado quando a chamada não é para contextualizar aprovação. |

Os nós `03A`, `10` e `10B` foram ajustados para mostrar o resumo. A validação
do OSV passou a aceitar aprovação apenas no modo de consulta de contexto, e
o HTTP usa os limites correspondentes a cada modo. As funções JavaScript
`compativel` e `apresentar`, nos nós 02C/02F, fazem a validação e a apresentação
do snapshot. Não houve mudança no Go, na API, em migrations ou na assinatura
do pedido de ação nesta entrega.

### Configuração, exports e estado local

- A tabela foi criada na instância local; não foi criado nem reiniciado
  container. O esquema sem dados está em
  `workflows/n8n/recursos/tabela-contexto-osv.json`.
- Em outra instalação, criar a tabela com o mesmo nome e colunas no projeto
  dos workflows. Os nós usam o nome da tabela, sem ID local embutido.
- O export da Central referencia o OSV por `APROVA_OSV_WORKFLOW_ID`. A chamada
  ativa usa a seleção do workflow já existente na instância local.
- OSV e Central foram publicados após a validação. O Core não precisou ser
  alterado para esta integração: ele continua encaminhando os dois eventos
  por seus caminhos atuais.
- Os exports foram atualizados sem credenciais, segredos ou dados fixados.
  Não houve commit, push ou implantação de produção.

### Validação realizada

- Dez testes novos em `tests/osv-aprovacao.test.cjs` verificaram correlação,
  rejeição de outro tenant/análise/PR/repositório/SHA, ausência de resultado,
  consulta complementar, expiração, falhas, presença nos dois formulários,
  limite dos botões/metadados, preservação do risco e contrato da API.
- Os treze testes do OSV e os doze da Central também passaram: 35 testes
  locais no total. Os novos nós e as configurações alteradas passaram na
  validação de parâmetros do n8n.
- Execução `261`: resposta OSV simulada, escrita real na Data Table e retorno
  do snapshot, sem envio ao Slack. Execução `262`: leitura real do mesmo
  contexto pela Central e geração do alerta enriquecido, com envio simulado.
- Execução `263`: nenhuma linha para a análise consultada; o alerta continuou
  com indisponibilidade explícita, sem reutilizar dados da análise anterior.
- Execuções `264 → 265`: a Central chamou o subworkflow, que fez consulta
  pública real ao OSV para um pacote de teste; o contexto retornou antes de
  montar a aprovação. O envio ao Slack foi simulado e o modo de contexto
  não publicou o alerta OSV separado.
- Execução `266`: o evento de dependência continuou alcançando seu envio de
  alerta, simulado no teste. Execução `267`: os dois registros sintéticos
  criados pelos testes foram removidos por suas chaves exatas; o nó temporário
  de limpeza foi removido antes da publicação final.

### Limites e trabalho posterior

Se a aprovação chegar antes do evento de dependências e não trouxer a lista
no próprio payload, não existe informação suficiente para consultar o OSV
para aquela análise. O alerta declara indisponibilidade. Um evento posterior
atualiza a tabela e publica o alerta de dependências, mas não edita a mensagem
de aprovação já publicada nem seu resumo nos formulários. Para garantir que
toda aprovação tenha OSV disponível desde a primeira mensagem, o emissor deve
fornecer os pacotes junto do evento ou será necessária uma evolução de
atualização/correlação de mensagens. Não se presume uma ordem de chegada.

A validade de 24 horas controla reutilização, não remove registros antigos.
Definir retenção/limpeza da Data Table e validar capacidade é parte da
preparação para produção. O cache é um apoio à apresentação, não a fonte da
decisão. A indisponibilidade do cache ou do OSV não cria autorização paralela.

Continuam pendentes a emissão automática no processamento do núcleo, a
validação de ponta a ponta com Slack/GitHub reais e a futura ampliação do
núcleo para publicar o contexto OSV na review, com contrato e idempotência
próprios. Essa publicação foi explicitamente deixada para depois.

### Diagnóstico dos botões no ambiente local — 2026-11-02

Após enviar os eventos de teste, a execução `272` da Central confirmou a
publicação da notificação no Slack (`ok`). Na conferência seguinte, não havia
nova execução de entrada para os cliques. O workflow estava ativo e com a
versão atual publicada. O container expunha o n8n em `127.0.0.1:5978`.
A inspeção inicial não identificou o ngrok, mas essa conclusão foi corrigida
pela consulta à API local do túnel fora do sandbox: ele estava ativo,
encaminhando HTTPS público para `http://localhost:5978`.

O histórico HTTP do ngrok confirmou três cliques em 16:32:18, 16:32:29 e
16:32:30 (UTC-3), todos com `POST /webhook-test/aprova/slack/interacoes`
e resposta `404 Not Found`. Os cliques chegaram ao túnel, mas usaram o
endpoint de teste sem escuta manual e não iniciaram o workflow publicado.
O envio de notificações por Incoming Webhook não
configura o caminho de volta. Para testar localmente, disponibilizar um
endpoint HTTPS público que encaminhe ao n8n e configurar a Request URL da
interatividade do Slack App como
`https://<endereco-publico>/webhook/aprova/slack/interacoes`.
O prefixo `/webhook-test/` exige escuta manual; `/webhook/` usa o workflow
publicado. Corrigir a Request URL no Slack App para o domínio atual do ngrok
com `/webhook/aprova/slack/interacoes`, salvar e clicar novamente. O painel
do Slack App não foi acessado nem alterado nesta inspeção; o caminho chamado
foi comprovado pelo histórico do ngrok. A abertura do formulário após a
correção ainda precisa ser validada.

Validação posterior no mesmo dia: a execução `273` recebeu o clique pelo
webhook publicado e abriu o formulário de aprovação (`views.open` com
`ok: true`). A execução `274` recebeu o envio, validou a assinatura do Slack,
preparou a decisão com justificativa e assinou a chamada ao núcleo.
O nó `13 | Enviar Ação ao Núcleo` encontrou `ECONNREFUSED` na porta `8080`
do host configurado. Portanto, não houve confirmação de autorização nem
registro de aprovação no GitHub por esse pedido. O nó
`14 | Preparar Retorno ao Usuário` informou que a ação não estava confirmada,
e `HTTP Request1` publicou esse retorno no Slack (`ok`). A execução n8n
terminou em `success` porque a falha de transporte foi tratada; isso não
significa sucesso da ação no núcleo. Continua pendente disponibilizar a API
ao container e validar permissões e execução real no GitHub. Esse teste
validou o formulário de aprovação; o de revisão ainda precisa de teste real.

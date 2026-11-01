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
| `03A | Preparar Aprovação` | Mostra PR, autor, motivo do risco e botões Aprovar/Pedir revisão. |
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

# Cards do dashboard — leitura F0

Leitura via MCP em 16/11/2026. Nenhum card alterado. Estado no quadro não comprova integração.

## [Dashboard: login com GitHub e isolamento por instalação](https://trello.com/c/WkpKt0do/65-dashboard-login-com-github-e-isolamento-por-instala%C3%A7%C3%A3o)

Lista: A Fazer.

O painel mostra dados de todos os clientes, então não pode ficar aberto.

- Login pelo próprio GitHub App (autorização de usuário), sem senha nossa
- Ao entrar, o backend consulta no GitHub as instalações e repositórios a que a pessoa tem acesso. Ela só vê dados desses, pelo tenant_id, o mesmo isolamento do banco
- Sessão em cookie HttpOnly, Secure e SameSite; token do GitHub nunca vai para o navegador
- Permissões do usuário no repositório consultadas no momento da ação, não guardadas em cache (permissão revogada vale na hora)
- Logout

Testes: usuário sem acesso a uma instalação não vê nada dela, nem por id direto na URL; sessão expirada volta para o login.

DEPENDENCIA: após "Dashboard: fundação do front-end".
BLOQUEIA: API de leitura e todas as telas com dados.

## [Dashboard: API de leitura e integração com o sistema](https://trello.com/c/siKOANdi/66-dashboard-api-de-leitura-e-integra%C3%A7%C3%A3o-com-o-sistema)

Lista: A Fazer.

Endpoints que o painel consome, todos filtrados pelas instalações do usuário logado.

- GET /api/analises: lista paginada com filtros (repositório, nível, estado, período) e busca por PR
- GET /api/analises/{id}: detalhe com task_steps, sandbox_results, risk_gates (motivos e evidências), approvals e o recibo
- GET /api/metricas: números da visão geral (por período e por nível, fila aguardando aprovação com idade, falhas de infraestrutura, tempo médio)
- Reaproveitar o que já existe: /reports/weekly e /receipts/*
- Somente leitura: nenhum endpoint deste card altera dado
- Respostas com contrato documentado (para a especificação OpenAPI)

Testes de integração com Postgres real: filtro por tenant em todos os endpoints, paginação estável, id de outra instalação devolve 404.

DEPENDENCIA: após "Dashboard: login com GitHub" e "Modelagem de dados".
BLOQUEIA: todas as telas com dados.

## [Dashboard: visão geral](https://trello.com/c/GOtZY0KC/67-dashboard-vis%C3%A3o-geral)

Lista: A Fazer.

Tela inicial do painel.

- Números do período: análises, bloqueadas, aprovadas por humano, falhas de infraestrutura, tempo médio
- Gráfico de análises por dia, separado por nível de risco
- Fila do que está aguardando aprovação agora, com há quanto tempo cada item espera
- Cada item leva ao detalhe da análise

Consome GET /api/metricas.

DEPENDENCIA: após "Dashboard: API de leitura".

## [Dashboard: lista de análises](https://trello.com/c/jgmvp6MJ/68-dashboard-lista-de-an%C3%A1lises)

Lista: A Fazer.

Tabela de análises com repositório, PR, nível, estado e data.

- Filtros por repositório, nível, estado e período; busca por número de PR
- Filtros refletidos na URL (link compartilhável)
- Paginação
- Linha leva ao detalhe

Consome GET /api/analises.

DEPENDENCIA: após "Dashboard: API de leitura".

## [Dashboard: detalhe da análise](https://trello.com/c/G7vjt1ha/69-dashboard-detalhe-da-an%C3%A1lise)

Lista: A Fazer.

Tudo sobre uma análise numa tela só.

- Linha do tempo dos estados (task_steps)
- Sandbox: tentativas, duração, compilou, testes, tipo de falha e a saída bruta
- Motivos do gate com evidência e arquivos alterados
- Aprovações: quem decidiu, por onde (GitHub ou Slack ou painel) e a justificativa
- Recibo com hash, key_id e o botão Verificar, que chama /receipts/{id}/verify e mostra cada verificação passando
- Links para o PR e para o Check Run

O botão Verificar é o momento central da apresentação.

DEPENDENCIA: após "Dashboard: API de leitura".

## [Dashboard: recibos, cadeia e exportação para auditoria](https://trello.com/c/4zNO67HS/70-dashboard-recibos-cadeia-e-exporta%C3%A7%C3%A3o-para-auditoria)

Lista: A Fazer.

Tela de auditoria.

- Lista de recibos por repositório
- Verificar um recibo (mesmo botão do detalhe)
- Verificar a cadeia inteira de uma instalação: confere cada elo e aponta o primeiro quebrado (detecta recibo apagado no meio)
- Exportar a trilha de auditoria (JSON e CSV) para entregar a um auditor, com instrução de como verificar sem o aPRova (docs/verificacao-recibo.md)
- Chaves públicas em uso

DEPENDENCIA: após "Dashboard: API de leitura" e "Recibo verificável".

## [Dashboard: repositórios e saúde da instalação](https://trello.com/c/YxkXI0uv/71-dashboard-reposit%C3%B3rios-e-sa%C3%BAde-da-instala%C3%A7%C3%A3o)

Lista: A Fazer.

Mostra se o aPRova está de fato protegendo cada repositório.

- Repositórios ativos da instalação
- ALERTA quando o check aPRova não está marcado como obrigatório na proteção de branch. É o modo de falha mais enganoso do projeto: tudo parece funcionar e nada é bloqueado
- Última análise recebida por repositório: muitas horas sem nenhuma costumam indicar webhook quebrado
- Permissões do GitHub App e se falta alguma

O painel só mostra e explica como corrigir; não altera a proteção de branch.

DEPENDENCIA: após "Dashboard: API de leitura".

## [Dashboard: regras do gate e estatísticas por regra](https://trello.com/c/DS8pVoCV/72-dashboard-regras-do-gate-e-estat%C3%ADsticas-por-regra)

Lista: A Fazer.

Somente leitura.

- Regras em vigor, com identificador, nível, descrição e limites
- Quantas vezes cada regra disparou no período
- Taxa de aprovação humana apesar do alto, por regra: mostra quais regras estão rígidas demais
- Esse dado alimenta a calibração por histórico e a sugestão de regras por IA

DEPENDENCIA: após "Dashboard: API de leitura" e "Gate de risco: regras determinísticas".

## [Dashboard: auditoria de ações remotas](https://trello.com/c/HFJuG4Pl/73-dashboard-auditoria-de-a%C3%A7%C3%B5es-remotas)

Lista: A Fazer.

Tentativas de ação vindas do Slack e do painel, aceitas e recusadas.

- Quem tentou, o quê, em qual PR, por onde, decisão e motivo da recusa
- Filtro para ver só as recusadas: quem tentou aprovar sem ter permissão
- Somente leitura

Consome remote_action_attempts.

DEPENDENCIA: após "Dashboard: API de leitura".

## [Dashboard: aprovar, pedir alteração e reexecutar](https://trello.com/c/qzCbNMo5/74-dashboard-aprovar-pedir-altera%C3%A7%C3%A3o-e-reexecutar)

Lista: A Fazer.

As únicas ações do painel sobre análises. Funcionam exatamente como os botões do Slack.

- Aprovar ou pedir alteração num PR bloqueado, com justificativa OBRIGATÓRIA quando o risco é alto (alimenta a calibração)
- Reexecutar uma análise que falhou por infraestrutura
- Toda ação passa pela mesma verificação de permissão do POST /acoes-remotas: o núcleo consulta o GitHub na hora
- Registro em remote_action_attempts com origem painel
- O autor do PR não aprova o próprio PR

O painel NUNCA muda nível de risco, NUNCA faz merge, NUNCA apaga recibo.

DEPENDENCIA: após "Dashboard: detalhe da análise", "Verificação de permissão em ações remotas" e "Publicar resultado no PR".

## [Dashboard: configuração do gate por repositório (versionada)](https://trello.com/c/nCmx3DPf/75-dashboard-configura%C3%A7%C3%A3o-do-gate-por-reposit%C3%B3rio-versionada)

Lista: A Fazer.

EVOLUÇÃO. O card mais delicado do painel, porque mexe no que o gate decide.

- Admin do repositório (conferido no GitHub) ajusta limites e áreas sensíveis do seu repositório
- Toda configuração é versionada e append-only; a versão usada entra no recibo. Sem isso, a mesma análise refeita poderia dar outro resultado e o recibo perderia valor de prova
- Toda mudança é registrada com quem, quando e o antes e depois
- Mudança que AFROUXA uma proteção (ex.: tirar uma área sensível) exige confirmação explícita e aparece destacada na auditoria

Até este card existir, a configuração continua por variável de ambiente.

DEPENDENCIA: após "Gate de risco: regras determinísticas", "Recibo verificável" e "Dashboard: regras do gate".

## [Dashboard: revisão das regras sugeridas pela IA](https://trello.com/c/zG4VlGAR/76-dashboard-revis%C3%A3o-das-regras-sugeridas-pela-ia)

Lista: A Fazer.

EVOLUÇÃO.

- Lista das regras sugeridas pela IA com estado pendente, com os dados que levaram à sugestão
- Admin aprova ou recusa. Aprovada, vira regra determinística como qualquer outra (passa pela configuração versionada)
- A IA propõe, o humano aprova, o gate executa. Nada é aplicado automaticamente

DEPENDENCIA: após "Sugestão de regras por IA" e "Dashboard: configuração do gate por repositório".

## [Dashboard: vínculo Slack ↔ GitHub](https://trello.com/c/ci4pR8Ay/77-dashboard-v%C3%ADnculo-slack-%E2%86%94-github)

Lista: A Fazer.

EVOLUÇÃO, mas destrava a Central no Slack em produção.

- Cada pessoa liga a própria conta do Slack à conta do GitHub com que fez login no painel
- Grava em identity_links, que hoje não tem como ser preenchido fora do banco
- A pessoa pode desfazer o vínculo
- Nunca permite ligar a conta de outra pessoa: o login do GitHub vem da sessão, não de campo digitado

DEPENDENCIA: após "Dashboard: login com GitHub".
BLOQUEIA: Central no Slack funcionando com usuários reais.

## [Dashboard: limites de SLA](https://trello.com/c/cZ6GTYWL/78-dashboard-limites-de-sla)

Lista: A Fazer.

EVOLUÇÃO.

- Admin define, por repositório, os limites de tempo em aguardando_aprovacao (primeiro aviso e escalonamento)
- O workflow de SLA do n8n passa a ler esses limites pela API, em vez de ter o valor fixo
- O n8n continua sem acesso ao banco: só consome a API

DEPENDENCIA: após "Dashboard: login com GitHub" e "Workflow n8n: escalonamento por SLA".

## [Dashboard: fundação do front-end](https://trello.com/c/aqBohAcP/64-dashboard-funda%C3%A7%C3%A3o-do-front-end)

Lista: Em andamento.

Estrutura do painel, sem telas de negócio ainda.

- React + Vite + TypeScript em web/
- O build vira arquivos estáticos embutidos no binário Go (embed) e servidos pelo próprio backend: um deploy só, mesmo domínio, sem CORS
- Padrão visual do protótipo implementado como base: cores (com as de risco reservadas), tipografia, componentes de status de risco e de estado do task_run
- Roteamento das telas previstas, com páginas vazias
- Cliente HTTP único para a API do painel, com tratamento de 401 (volta para o login)
- Lint, testes de componente e build do front no CI

DEPENDENCIA: após "Dashboard: protótipo de design".
BLOQUEIA: todas as telas do painel.

## [Dashboard: protótipo de design](https://trello.com/c/XYKsCB75/63-dashboard-prot%C3%B3tipo-de-design)

Lista: Pronto.

Protótipo visual das telas do painel ANTES de qualquer código, para validar com a professora e servir de referência para quem implementa.

**Telas no protótipo**
- Visão geral
- Lista de análises
- Detalhe da análise (com o botão Verificar recibo)
- Recibos e cadeia
- Repositórios e saúde da instalação

**Direção visual** (a partir da análise do Greptile e do CodeRabbit)
- Tema escuro, denso mas organizado, como painel de operação
- Cores de risco reservadas: verde (baixo), âmbar (médio), vermelho (alto). A cor da marca NÃO pode ser nenhuma das três, para não confundir botão com risco. Usar azul ou violeta
- Fonte monoespaçada só para dado técnico (SHA, hash, ids); o resto em fonte comum
- Inspiração na estrutura, nunca nas frases, formas ou identidade dos concorrentes

**Entrega**
- Telas com dados realistas do aPRova-TESTES (PR #1, bug do frete, recibo real)
- Padrão visual (cores, tipografia, componentes de status) que o card de fundação do front-end vai implementar

**Regras do painel que o protótipo precisa respeitar**
1. Toda ação passa pelo núcleo com verificação de permissão no GitHub
2. O painel nunca muda nível de risco, nunca faz merge, nunca apaga recibo
3. Tudo que o painel faz é registrado

BLOQUEIA: Dashboard: fundação do front-end.

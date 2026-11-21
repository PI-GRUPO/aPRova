# App Home do aPRova no Slack

A aba Home publica uma view individual ao abrir, permite trocar de seção, aplicar um período à lista de análises do dashboard e atualizar a consulta. Mostra a saúde do processo e a disponibilidade do login web a partir de HTTP. O dashboard resolve a identidade GitHub e os repositórios acessíveis no navegador. O núcleo ainda precisa oferecer essa resolução para uma pessoa identificada por workspace e usuário Slack.

## Entrada e apresentação

| Arquivo | Responsabilidade |
| --- | --- |
| `workflows/n8n/slack-consulta-metricas.json` | Recebe eventos Slack em `/webhook/aprova/slack/consultas`. Valida a assinatura, responde ao challenge e confirma o evento. Encaminha `app_home_opened` da aba Home ao adaptador. Mensagens diretas e menções continuam no assistente. |
| `workflows/n8n/central-slack.json` | Recebe interatividade em `/webhook/aprova/slack/interacoes`. Confere assinatura, workspace e app antes do ACK. Separa a superfície Home e seus callbacks dos formulários da Central e dos botões de feedback dos alertas. |
| `cmd/slack-home/main.go` | Inicia o processo HTTP independente. Confere bot e workspace com `auth.test`. O comando substitui o piloto de publicação manual com usuário e instalação fixos. |
| `internal/slackhome/eventos.go` | Valida novamente a assinatura dos bytes originais, extrai o titular do evento, reserva a interação e processa uma view por pessoa de cada vez. |
| `internal/slackhome/home.go` | Constrói a Home, os links de navegação e o formulário de seção/período. `AcessoIndividual.Consultar` recebe `Pessoa{Workspace, Usuario}` e um `Recurso`. O adapter de produção responde `SemContrato` para dados individuais. |
| `internal/slackhome/http.go` | Consome as leituras públicas do núcleo e os métodos do Slack. Confere HTTP, o campo `ok`, o hash da publicação e o ID do modal. |
| `workflows/n8n/feedback-motivo-aprovacao.json` | Ignora o pedido de uma segunda pergunta para aprovações de origem Slack. Preserva o tratamento de callbacks de feedback existentes. |

O n8n confirma o recebimento ao Slack antes de chamar o processo da Home. Os nós H01/H02 encaminham o corpo assinado e os dois cabeçalhos originais para `POST /slack/home` **do adaptador**. Os webhooks do núcleo e `entrada-eventos` continuam recebendo os eventos de PR pelo caminho existente. Uma falha da Home é registrada na execução do ramo n8n e não bloqueia o worker.

### Navegação e concorrência

- `aprova_home_secao` troca entre visão geral, pendências, detalhes, histórico e relatórios, ou integrações e configuração. `aprova_home_atualizar` repete as leituras públicas. Na primeira abertura, o Slack recebe uma view de carregamento e depois o resultado.
- `aprova_home_navegar` usa o `trigger_id` para abrir o formulário. `aprova_home_navegacao` recebe a seção e o período de 7, 30, 90 dias ou todo o histórico. O período compõe o link para a lista de análises no navegador. Pendências inclui o filtro `estado=aguardando_aprovacao`. O relatório semanal mantém sua janela própria. A seção e o período voltam na view individual e são validados a cada interação.
- `views.publish` recebe o usuário do evento autenticado. A substituição envia o hash da versão. Um conflito descarta a atualização antiga. O envio de modal também exige correspondência entre titular, ID do modal aberto e revisão da Home. Depois de uma decisão confirmada pelo núcleo e do retorno privado entregue, H03 encaminha o formulário assinado para reler a Home recente do mesmo usuário. Esse ramo aproveita o hash e o filtro já conhecidos e não reenvia a decisão. Uma nova abertura permite recuperar a view após resposta perdida.
- A assinatura cobre o corpo bruto com timestamp na janela de cinco minutos. A deduplicação em memória retém IDs de eventos e interações por 11 minutos, até 10 mil reservas por processo. Cliques próximos para abrir o mesmo formulário são agrupados. Um HTTP 429 pausa o método pelo `Retry-After`. Timeout, recusa e publicação de resultado incerto não disparam outra escrita. O usuário pode reabrir ou atualizar a Home.
- A execução local prevê **uma réplica** por bot/workspace. Um reinício perde as reservas transitórias e invalida modais em andamento. A proteção entre réplicas ou reinícios exigiria coordenação adicional. Nenhum vínculo, permissão ou preferência de equipe fica nesse estado transitório.

Referências do Slack para o evento, controle de revisão, modais e limitação de chamadas. [Abertura da Home](https://docs.slack.dev/reference/events/app_home_opened/), [publicação](https://docs.slack.dev/reference/methods/views.publish/), [modais](https://docs.slack.dev/surfaces/modals/), [rate limits](https://docs.slack.dev/apis/web-api/rate-limits/).

## Contratos e autenticação

| Consumidor | Contrato utilizado |
| --- | --- |
| Adaptador da Home → núcleo | `GET /health` exige HTTP 200 e `status=ok`. `GET /api/sessao` anônimo verifica os booleanos `autenticado=false` e `login_disponivel`. O cliente envia essas leituras sem cookies, credenciais GitHub ou assinatura de serviço. Resposta malformada, sessão autenticada inesperada ou erro de rede tornam a leitura indisponível. |
| Adaptador → Slack | Token do bot em `Authorization: Bearer` para `auth.test`, `views.publish` e `views.open`. Cliente com timeout e redirects desabilitados. O segredo de assinatura é usado apenas para verificar requisições recebidas. |
| Pessoa → dashboard | Links apontam para a origem pública configurada. O navegador usa `/auth/github`, callback e cookie próprios do dashboard. A sessão autoriza `/api/repositorios`, `/api/analises`, `/api/analises/{id}` e `/api/web/relatorio-semanal` por instalação e repositórios. O login web não cria um vínculo Slack. |
| Assistente → núcleo | Somente `GET /health` e `GET /ready`. Nos dois agentes, inclusive no de recuperação, as ferramentas privadas foram desconectadas. A nova chave de memória inclui escopo público sem equipe/repo, workspace, tipo de conversa, canal e usuário. As respostas continuam na DM ou na menção que originou a pergunta. |
| Central → núcleo | Aprovar e solicitar revisão pelos alertas continuam usando `POST /acoes-remotas` assinado entre servidores. O núcleo revalida o pedido e executa a review. O ACK ao Slack confirma recebimento. O retorno `202` com `status=aceito` confirma a execução segundo esse contrato. A resposta com a justificativa é efêmera para o usuário que enviou o formulário. O alerta operacional continua no canal. Uma aprovação é uma review. Falha ambígua exige conferência. |

As variantes HMAC de listagem e detalhe em `/api/analises` assinam método, caminho e query. Devolvem uma projeção de execução por instalação. `GET /reports/weekly` assina a query e agrega a instalação por sete dias UTC. Nenhuma dessas assinaturas identifica quem pode ler quais repositórios pelo Slack. A Home não envia essas chamadas. Conhecer a URL pública de um recibo também não estabelece a associação autorizada entre pessoa, análise, PR e SHA.

## Recursos e dependências

| Recurso | Estado local e condição para avançar |
| --- | --- |
| Abertura, navegação, modal, atualização e saúde | **Funcionais em teste isolado.** Os exports portáteis também estão publicados no n8n local de teste. A renderização, a entrega dentro do prazo do trigger e os links no cliente Slack ainda precisam de validação integrada. |
| Identidade, equipes e catálogo | **Preparados** em `Vinculo` e `Catalogo`. Falta operação do núcleo que resolva o par workspace/usuário para um vínculo persistente verificado, equipes, instalações, repositórios e capacidades atuais, incluindo revogação. O mapa de ambiente usado hoje pela Central não atende a esse contrato. O seletor privado depende dessa lista. |
| Pendências, detalhes, histórico e evidências | **Bloqueados no Slack**, com acesso pelo dashboard quando o login está disponível. `Fila`, `Evidencias` e `Metricas` reservam os pontos de consulta. A listagem precisa de paginação autorizada e filtros revalidados. Cada detalhe exige instalação, repositório, PR, análise e SHA correspondentes. Risco, motivo, testes, decisões, recibo e CVEs exigem dados correlacionados. O dashboard já oferece parte desses dados por sessão. |
| SLA, onboarding e integrações | **Bloqueados na Home** em `Preferencias` e `Integracoes`. Faltam leitura das políticas, início da espera, prazos e pendências por repositório, confirmação do Check obrigatório e permissão para persistir alterações de canal, responsável, SLA e quadro. Faltam instalação OAuth Slack e confirmação de vínculo pelo titular. O Trello é opcional. O formulário atual salva apenas a apresentação da Home. |
| Aprovação, revisão e feedback | **Funcionais nos alertas no nível do adapter**. O formulário novo de aprovação reúne motivo e justificativa no campo textual aceito pelo núcleo. Formulários antigos permanecem reconhecidos. O evento de origem Slack dispensa a pergunta posterior. O registro estruturado de `feedback_reason` requer contrato próprio. A Home mantém `Decisoes` bloqueado até poder consultar o contexto individual e a elegibilidade atual e então encaminhar a ação ao núcleo. |
| Reexecução | **Indisponível.** O núcleo monta `ReenfileiradorIndisponivel`. Novos alertas de falha preservam o diagnóstico. Cliques no botão de alertas antigos abrem um aviso privado, sem chamar a operação. Reabilitar exige falha recuperável da análise atual e executor efetivo. |
| Métricas, consultas e alertas | **Parciais.** O assistente responde sobre saúde e prontidão. O Digest e as consultas auxiliares mantêm os contratos de serviço existentes. Reativar consultas individuais exige autorização por workspace, usuário, equipe e repositório, memória nova para esse escopo e resposta privada. Anomalias da equipe dependem da associação autorizada, e as globais exigem confirmação de papel administrativo. |
| OSV, Trello, SLA e canais | Os workflows `dependencias-osv`, `trello-sugestao-movimentacao`, `sla-aprovacao`, `digest-semanal`, `onboarding-repositorio` e `entrada-eventos` mantêm os fluxos anteriores. A Home não recalcula risco, agenda avisos ou move cards. Para exibir seus resultados, o núcleo precisa disponibilizar a leitura autorizada e correlacionada. |

A apresentação distingue carregamento, capacidade sem contrato, ausência de vínculo, falta de permissão, falha e lista vazia confirmada. Nesta versão o adapter individual informa capacidade sem contrato. Os demais estados individuais têm testes controlados. Uma lista vazia requer uma leitura explícita para o titular e recurso solicitados.

## Configuração e operação

1. No ambiente do adaptador, forneça `APROVA_SLACK_BOT_TOKEN`, `SLACK_SIGNING_SECRET`, `APROVA_SLACK_TEAM_ID`, `APROVA_SLACK_APP_ID` e `APROVA_API_URL`. Use o token e o segredo do mesmo Slack App. `WEB_URL_PUBLICA` é opcional e indica a origem HTTPS do dashboard. O programa carrega o `.env` local sem sobrescrever o ambiente. Ele dispensa PostgreSQL, Redis e credenciais de serviço do núcleo.
2. Para operação em ambiente de integração, execute `go run ./cmd/slack-home`. O processo escuta por padrão em `127.0.0.1:8091`. `APROVA_SLACK_HOME_LISTEN` seleciona a interface privada alcançável pelo n8n. Configure `APROVA_SLACK_HOME_URL` no n8n com a URL completa desse processo, terminada em `/slack/home`. Em contêiner, o loopback é o do próprio contêiner. Use uma rota privada apropriada entre os processos.
3. No Slack App de integração, habilite Home, mantenha Mensagens e acrescente `app_home_opened` às assinaturas existentes. Preserve os destinos HTTPS de Events e Interactivity da tabela acima. As credenciais de publicação dos alertas e respostas continuam nas integrações existentes do n8n. A configuração técnica é feita uma vez pelo operador.
4. A mudança de roteamento e a restrição de ferramentas do assistente só entram em vigor na instância que usar os exports revisados. Os três workflows foram publicados na instância separada de teste da porta 5979. A revisão também foi aplicada à instância de teste da porta 5978 para usar o túnel HTTPS existente.

## Verificação deste checkout

- `go test -race ./internal/slackhome ./cmd/slack-home` e `go vet ./cmd/slack-home ./internal/slackhome`. Foram exercitados assinatura e timestamp, origem e titular, adulteração de estado, reabertura, filtros, modal, atualização, cliques duplicados, concorrência, hash antigo, limite HTTP e falhas. A API foi exercitada pelos handlers reais de saúde e sessão anônima sem login configurado e por respostas HTTP controladas nos demais cenários.
- `node --test --test-isolation=none workflows/n8n/tests/*.test.cjs` passou nos 257 testes dos 19 arquivos. A suíte cobre encaminhamento dos bytes assinados depois do ACK, caminhos legados, motivo e justificativa, indisponibilidade de reexecução, segregação das ferramentas do agente e preservação dos workflows. A operação da Home foi testada sem Trello configurado.
- Os parâmetros de 15 nós de assinatura, roteamento, encaminhamento, feedback e interatividade passaram no validador de configuração do n8n. Os testes Node exercitam os nós e conexões. As verificações adicionais no runtime n8n estão descritas abaixo.
- Vínculo persistente, revogação real, isolamento entre equipes GitHub, dados privados e operações pela Home continuam bloqueados pelo backend. Testes com estados controlados verificam como a interface trata cada resposta. Nenhuma mensagem ou decisão real foi enviada. As verificações posteriores com o cliente Slack estão registradas na seção seguinte.

### Conferência do ambiente em 20/11/2026

O operador confirmou o app já utilizado no workspace aPRova e a API em `127.0.0.1:8080`. O adaptador foi iniciado em `127.0.0.1:8091` com o token do bot e o segredo de assinatura configurados no checkout vizinho. Na inicialização, `auth.test` confirmou o workspace e a presença de uma identidade de bot. A API respondeu HTTP 200 a `/health` e `/api/sessao`, com `status=ok`, `autenticado=false` e `login_disponivel=true`.

A instância de teste `aprova-slack-home-integracao` está na porta 5979, com volume próprio. Central, Consulta de Métricas e Feedback foram publicados nessa instância. Os nós Slack aplicáveis usam uma credencial local do bot. As referências à Central e ao Feedback usam os IDs das cópias de teste. De dentro do n8n, `GET http://host.docker.internal:8080/health` retornou 200. A conexão a `APROVA_SLACK_HOME_URL=http://host.docker.internal:8091/slack/home` retornou 405 para GET, conforme o contrato POST do adaptador.

Na primeira importação por CLI, o n8n 2.28.6 ativou os workflows mas retornou 404 para os caminhos curtos. Sem `webhookId`, essa versão prefixa a rota com o workflow e o nome do nó. Os dois exports agora contêm identificadores de webhook estáveis, independentes da instalação. Após reimportação, publicação e registro dos caminhos, o webhook de Events devolveu o challenge assinado gerado localmente com HTTP 200 em 308 ms. Interactivity rejeitou uma requisição sem assinatura com HTTP 401. Esses ensaios exercitaram o runtime com entradas controladas. Ainda falta medir o ACK e observar a publicação de uma Home originada no cliente Slack.

O operador confirmou que a instância na porta 5978 também é de teste. Ela recebe os callbacks públicos pelo túnel ngrok já existente. Antes da atualização, foram conferidos o rascunho e a versão publicada dos três workflows, os parâmetros e as conexões da base. As cópias de recuperação ficaram fora do repositório. A atualização preservou os IDs, caminhos, credenciais do bot e do Gemini e referências da instância. Apenas o contêiner n8n da 5978 foi recriado com a URL do adaptador. O override de teste do Compose está em `/tmp/aprova-home-5978-opq6iedr/compose.yml`. Para recriações futuras, a configuração do serviço precisa incluir `APROVA_SLACK_HOME_URL=http://host.docker.internal:8091/slack/home`.

O challenge assinado pelo cliente de teste voltou pelo túnel em 651 ms. Um envio controlado de modal sem sessão revelou uma falha posterior ao ACK. O nó HTTP H02 tentava converter em texto a resposta vazia do adaptador. Nos dois workflows de entrada, H02 agora lê o corpo como binário e inclui o status HTTP. Na Central, a execução 2414 confirmou ACK público em 1,35 s e conclusão do H02 com HTTP 204 do adaptador. O descarte do modal antigo não publica uma view.

Na sequência pelo cliente Slack, as execuções 2444 e 2447 receberam `app_home_opened` com `tab=home` e H02 recebeu HTTP 204 do adaptador. A execução 2446 recebeu `tab=messages` e seguiu fora de H02. A pessoa confirmou que viu o texto da Visão geral na Home. Os cliques em **Seção e período…** e **Atualizar** também chegaram à Central (2450–2454, entre as execuções verificadas). O envio do formulário com o período escolhido e a resposta a uma conversa em Mensagens ainda precisam de confirmação. O texto sobre acesso na Visão geral é a descrição fixa da seção. A listagem privada continua dependente de autorização individual no núcleo.

O dashboard configurado usa HTTP em loopback e o adaptador iniciou sem `WEB_URL_PUBLICA`. A navegação por links e o login no navegador dependem de uma origem HTTPS do dashboard de teste.

### Configuração do app e roteiro para o cliente Slack

No app de teste confirmado, usar a origem HTTPS do túnel que encaminha para a porta 5978 no lugar de `<origem-n8n>`.

| Tela e campo no Slack | Valor |
| --- | --- |
| App Home → Show Tabs | Home Tab e Messages Tab habilitadas, com envio de mensagens na aba Messages permitido |
| Event Subscriptions → Enable Events → Request URL | `<origem-n8n>/webhook/aprova/slack/consultas`. Aguardar **Verified** antes de salvar. |
| Event Subscriptions → Subscribe to bot events | Adicionar `app_home_opened` e preservar `message.im` e `app_mention` para os fluxos de conversa |
| Interactivity & Shortcuts → Interactivity → Request URL | Habilitar e usar `<origem-n8n>/webhook/aprova/slack/interacoes` |

Referências do painel. [App Home](https://docs.slack.dev/surfaces/app-home/), [Events](https://docs.slack.dev/apis/events-api/using-http-request-urls/) e [Interactivity](https://docs.slack.dev/interactivity/handling-user-interaction/).

Se o cliente exibir “este trabalho ainda está em andamento”, conferir primeiro se chegou um `event_callback` de tipo `app_home_opened` com `tab=home` ao túnel e ao workflow de Events. Na abertura relatada em 20/11, o histórico observado continha o challenge de teste e nenhuma entrega dessa abertura. A API do Slack confirmou o token do bot no workspace esperado. O valor **Verified** confirma o challenge da URL. Para receber aberturas, conferir também a inscrição em **Subscribe to bot events**, salvar a configuração e sair e voltar à aba Home. Se o evento chegar, acompanhar H02 até o adaptador e conferir a publicação no cliente. [Evento de abertura da Home](https://docs.slack.dev/reference/events/app_home_opened/)

1. Abrir a Home do app com a primeira pessoa. Conferir a execução de Consulta de Métricas, o ACK e o retorno do adaptador. Na primeira publicação, observar carregamento e resultado. A tela intermediária pode passar rápido. Conferir saúde e login com as duas leituras públicas.
2. Escolher **Pendências**, abrir **Seção e período…**, escolher **Histórico e relatórios** e **Últimos 7 dias**. Atualizar e reabrir a Home. Conferir o filtro e o link **Ver análises do período**. No navegador, verificar o acesso com a própria conta GitHub. Registrar separadamente abertura do link e conclusão do login.
3. Clicar duas vezes para abrir o formulário. Se houver dois clientes para o mesmo usuário, mudar a seção em um e tentar enviar o modal antigo no outro. Conferir a revisão final e o número de publicações. Se houver segunda pessoa, escolher uma seção diferente e reabrir as duas Homes. Conferir a navegação de cada titular.
4. Com Trello ausente no teste, visitar **Integrações e configuração**. Na aba **Mensagens**, perguntar pela saúde da API e conferir a resposta na mesma DM. Para uma menção, usar somente o canal de teste confirmado. Correlacionar horário, execução e ramo com o resultado visível. Registrar falhas e estados indisponíveis.

Para o ensaio de decisão, identificar primeiro a instalação de teste, repositório, PR descartável aberta, SHA atual, análise aguardando decisão, revisor elegível e alerta no canal de teste. Informar esses destinos e a ação escolhida antes de pedir autorização específica. O envio pelo alerta pode gravar uma decisão no núcleo, publicar uma review no GitHub, atualizar o Check e o alerta e entregar uma confirmação efêmera. Para a aprovação de origem Slack, conferir a ausência de uma segunda pergunta de motivo. O ensaio de navegação termina antes de qualquer envio de decisão.

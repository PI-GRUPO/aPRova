# Implementação do dashboard aPRova

Documentação das fases: manter este plano e todos os novos documentos de
levantamento, execução e acompanhamento do dashboard em `design/docs/`.
O [índice desta pasta](README.md) reúne os documentos. Usar links relativos
para as referências técnicas do núcleo que permanecem em `docs/` na raiz.

> Execução F3 em 16/11/2026: [sessão web e relatório semanal](dashboard-f3.md), F3A–C implementados; OAuth real e F3D pendentes.
> Execução F2 em 16/11/2026: consulte [recibos por UUID e validação](dashboard-f2.md); aceite com recibo real pendente.
> Execução F1 em 16/11/2026: consulte [implementação e validação da fundação](dashboard-f1.md).
> Retomada F0 em 16/11/2026: veja [levantamento atual](dashboard-f0.md),
> [inventário Figma](dashboard-inventario-figma.md) e
> [critérios dos cards consultados](dashboard-cards-f0.md).
> Base atual: `feat/web-dashboard`, SHA `a28725d`, a partir de `dev`, sem incorporar
> `feat/slack-app-home`. Há dez rotas: as duas consultas de análises já existem
> como serviço HMAC, ainda sem sessão web nem detalhe agregado. Afirmações de
> oito rotas nas auditorias abaixo são históricas. Por orientação da usuária,
> o Figma é a referência visual; o protótipo local não é fiel e só fornece código
> candidato a reaproveitamento. F0 aguarda confirmação de homologação/recibo real.

## 1. Objetivo, revisão e resultado esperado

Plano revisado em 15/11/2026 para transformar todas as telas do Figma em uma
aplicação utilizável, fiel ao protótipo e integrada ao núcleo. A entrega atual é
documental: não inicia a implementação do front ou do backend.

Base do núcleo/protótipo: branch `dev`, commit
`d2a99d873616fcdc01f1cc12de4302b3aef758d8`. A auditoria foi ampliada para
`feat/atualizacao-workflows-slack`, commit
`9c0fd55676895ea7f2506a4b1ce788dd761d7d5c`, que incorpora essa `dev` e
acrescenta 20 arquivos alterados de automação, documentação e configuração.
Não há alterações adicionais em `internal/`, `cmd/`, queries ou migrations.
A `main` está em
`9753cf05bb35c2589a120b92aa897ee83562012e` e não contém o mesmo avanço.
Revalidar esta base antes de cada fase. O agente de construção adicional ainda
pode ser complementado depois; as orientações locais de n8n fornecidas pela
usuária já foram preservadas no AGENTS.md unificado.

Fluxo de produto a preservar: entrar no escopo autorizado → localizar pendências
→ abrir análise e evidências → tomar decisão humana → acompanhar resultado
→ consultar e verificar o recibo. Gestão do gate, integrações e SLA são fluxos
complementares. O dashboard não faz merge automático e não é a autoridade do gate.

**Prioridade prática:** fundação reaproveitando o protótipo → recibo por UUID,
verificação, chaves e exportação individual → sessão e mediação segura do relatório
semanal → APIs de leitura e telas operacionais → ações humanas → gestão.
O vínculo Slack–GitHub pode entrar logo após a sessão, em F3D/F7A; F7B também
não depende de toda a configuração de regras F6, apenas de seus contratos próprios.

Uma tela pode ter layout concluído e integração bloqueada. Registrar essas duas
dimensões separadamente. Não declarar o dashboard completo enquanto houver
funcionalidades do escopo sem comportamento real.

## 2. Fontes e evidência

| Fonte | Uso neste plano |
| --- | --- |
| [Figma aPRova](https://www.figma.com/design/VtHNXer6wzJj37QdpN3Lb9/aPRova?node-id=9-22) | Metadados das páginas desktop/mobile e contexto estruturado com capturas das telas principais; fluxos, temas, componentes e estados |
| [PRODUCT.md](../prototipo/PRODUCT.md), [DESIGN.md](../prototipo/DESIGN.md), [package.json](../prototipo/package.json), [App.tsx](../prototipo/src/App.tsx) | Stack, tokens, componentes reaproveitáveis e limite do protótipo codificado |
| [servidor.go](../../internal/api/servidor.go), [main.go](../../cmd/api/main.go) | Rotas realmente registradas e dependências realmente conectadas |
| [recibos.go](../../internal/api/recibos.go), [exportacao.go](../../internal/receipt/exportacao.go), [verificador.go](../../internal/receipt/verificador.go) | DTO, erros e alcance da verificação de recibos |
| [relatorio_semanal.go](../../internal/api/relatorio_semanal.go), [reports.sql](../../queries/reports.sql) | Autenticação, janela, denominadores e fontes das métricas |
| [acoes_remotas.go](../../internal/api/acoes_remotas.go), [contrato.go](../../internal/acaoremota/contrato.go), [executor_github.go](../../internal/acaoremota/executor_github.go) | Identidade Slack, justificativa, autorização e reexecução indisponível |
| [persistencia.go](../../internal/domain/persistencia.go), [calibracao.go](../../internal/domain/calibracao.go), queries e migrations | Dados existentes no núcleo que ainda não são uma API de dashboard |
| [Fluxo do produto](../../docs/fluxograma-produto.mmd), [estados](../../docs/maquina-estados.mmd), [recibos](../../docs/verificacao-recibo.md), [calibração](../../docs/calibracao.md), [sandbox](../../docs/sandbox-em-duas-etapas.md) | Semântica do produto e limites de evidência |
| [Central Slack na branch dos workflows](https://github.com/PI-GRUPO/aPRova/blob/9c0fd55676895ea7f2506a4b1ce788dd761d7d5c/docs/central-slack-implementacao.md), [eventos](https://github.com/PI-GRUPO/aPRova/blob/9c0fd55676895ea7f2506a4b1ce788dd761d7d5c/docs/eventos-n8n.md) e [guia n8n](https://github.com/PI-GRUPO/aPRova/blob/9c0fd55676895ea7f2506a4b1ce788dd761d7d5c/workflows/n8n/README.md) | Estado mais recente da automação, validações documentadas e Etapa S |
| [Quadro Trello aPRova](https://trello.com/b/BLUfesZg/aprova), cards #63–#78 e dependências | Escopo, critérios de aceite e decisões de implementação do dashboard |
| Documento de Requisitos_aPRova.docx e diagramas anexados ao projeto | Requisitos de autoridade do núcleo, isolamento, revisão humana, rastreabilidade e ausência de merge automático |

Limite desta auditoria: inspeção de código e testes existentes, sem chamada a uma
API implantada ou ao ambiente local da usuária. “Implementado” abaixo significa
rota e handler no commit auditado; disponibilidade operacional exige smoke test.
Não foi encontrado um contrato OpenAPI do dashboard nesta árvore.
Na revisão da branch dos workflows foram executados os **228 testes locais de
16 arquivos**, todos aprovados. Os ensaios reais de Slack/GitHub/API descritos
nessa branch são evidências documentadas, não novas chamadas feitas nesta revisão.

Há documentação histórica com exemplos que não são o contrato atual. Por exemplo,
a query atual de relatório conta reversões em `pull_request_reverts`; um trecho do
README ainda cita `code_memory`. O handler e a query têm precedência para o DTO e
seu cálculo. Os tipos JSON locais do protótipo são fixtures, não promessa de API.

### 2.1. O que a branch dos workflows acrescenta ao plano

| Capacidade disponível | Evidência/estado | Consequência para o dashboard |
| --- | --- | --- |
| Correlação persistente do alerta Slack | Central com associação por instalação/análise/repositório/PR, canal/ts e controle de duplicatas; código/export disponível; flag global documentada como desligada | Reusar a correlação, sem criar um histórico concorrente no front; flag desligada não significa recurso inexistente |
| Atualização após decisão | Core encaminha `aprovacao_registrada`; Central atualiza a mesma mensagem e remove botões de decisão; validação Slack documentada com decisão fictícia | F5 precisa preservar emissão/consumo do evento e origem painel; o navegador não chama `chat.update` nem fabrica o evento |
| Consulta automática do recibo | Central lê documento e `/verify`, confere identidade, bytes, hash e cinco conferências; preserva o bloco nas atualizações | F2 reaproveita as regras de consistência como referência de contrato e casos de teste, sem depender do n8n para verificar o recibo |
| Contexto OSV posterior | OSV retorna snapshot à Central, que atualiza o alerta sem mudar risco/decisão; origem real worker → OSV ainda pendente na documentação | Não refazer o workflow no front. Se a interface apresentar CVEs, criar leitura autorizada do contexto pelo núcleo; as oito rotas atuais não o expõem |
| Resiliência de chat/Digest | Aviso seguro para erro/saída vazia; Digest distingue confirmado/falha/incerto sem retry automático | Preservar semântica de indisponibilidade; chat e webhook do Core não substituem API de dashboard |
| Ensaio real documentado — 6A dos workflows | PR 1 de `PI-GRUPO/aPRova-TESTES`: 10 testes passaram, risco baixo, Check/comentário, evento worker → Core e recibo válido; entrada GitHub controlada/manual | Usar como cenário candidato de homologação, reconsultando IDs/dados atuais; não usar o antigo fork como alvo da instalação |
| SLA já implementado na automação | Timer periódico e estado persistente em Data Tables; políticas por ambiente; envios desligados e políticas não definidas na validação documentada | F7B entrega gestão/API e adaptação do consumidor existente, não a construção do timer do zero |
| Etapa S — simplificação da persistência | Diretriz documentada para transferir estado atual/controle dos avisos ao núcleo; operações ainda ausentes | Projetar leitura de pendência, reserva/confirmação e recuperação como serviço compartilhado; migrar antes de retirar tabelas |

As oito rotas do núcleo permanecem as mesmas. A branch tem avanço funcional de
automação e mais evidência integrada, mas não implementa login web, listagem de
análises, CRUD de vínculo, configuração de SLA ou o reenfileirador de reanálise.
Persistência no n8n não é API consumível pelo navegador e não deve ser consultada
diretamente pelo front.

Preservar as fronteiras da validação documentada: correlação/SLA desligados;
Digest inativo; vínculo humano não configurado; proteção do alvo correto com
403 por falta de `administration=read`; alto risco/decisão humana/OSV real ainda
pendentes. 403 de proteção significa “não confirmado”. Trechos históricos da
mesma documentação ainda dizem que o worker não foi validado: o checklist e a
seção 6A mais recentes registram a validação limitada acima.

As fases “1–6A” dos workflows e a “Etapa S” são do plano de automação. Não são
as fases F0–F8 deste plano de dashboard e não as tornam automaticamente concluídas.

### 2.2. Rastreabilidade com os cards do Trello

Leitura do quadro em 15/11/2026: **15 cards com Front-end e Desenvolvimento**,
#64–#78, mais o protótipo #63. Fundação está em “Em andamento”; os outros 14
cards de implementação estão em “A Fazer”; protótipo está em “Pronto”. São
estados do quadro, não comprovação de integração na branch auditada. Não há
checklists ou comentários adicionais nos cards de dashboard retornados nessa leitura.

| Card | Fases/lotes | Critério que deve ser preservado |
| --- | --- | --- |
| [#63 — Protótipo](https://trello.com/c/XYKsCB75) | F0 | Referência visual; usar Figma/DESIGN atuais, sem restaurar a antiga indicação de azul/violeta do texto do card |
| [#64 — Fundação do front-end](https://trello.com/c/aqBohAcP) | F1 | `web/`, build estático embutido em Go, um domínio/deploy, rotas, cliente HTTP, 401 e CI com testes de componentes |
| [#65 — Login e isolamento](https://trello.com/c/WkpKt0do) | F3A/B | Autorização de usuário pelo GitHub App, sessão segura, escopo e negação de acesso por ID direto |
| [#66 — API de leitura](https://trello.com/c/siKOANdi) | F3/F4A/B/C | `/api/analises`, `/api/analises/{id}`, `/api/metricas`, reaproveitar relatório/recibos; OpenAPI, Postgres e paginação estável |
| [#67 — Visão geral](https://trello.com/c/GOtZY0KC) | F4B | Métricas por período, série por nível, tempo médio e fila atual com idade |
| [#68 — Lista de análises](https://trello.com/c/jgmvp6MJ) | F4A | Filtros na URL, busca por PR e paginação real |
| [#69 — Detalhe da análise](https://trello.com/c/G7vjt1ha) | F4A + F2 | Timeline, tentativas/saída, motivos, decisão/origem e verificação do recibo; essa verificação é central na apresentação |
| [#70 — Recibos, cadeia e exportação](https://trello.com/c/4zNO67HS) | F2 parcial + F4C | Coleção, cadeia inteira/primeiro elo quebrado, JSON/CSV e guia de verificação independente; F2 sozinha não conclui o card |
| [#71 — Repositórios e saúde](https://trello.com/c/YxkXI0uv) | F4A/C | Check obrigatório, permissões e última análise; orientar correção sem alterar proteção |
| [#72 — Regras e estatísticas](https://trello.com/c/DS8pVoCV) | F4C | Leitura dos limites efetivos, acionamentos e aprovação apesar do alto por regra |
| [#73 — Auditoria](https://trello.com/c/HFJuG4Pl) | F4C + F5A | Tentativas aceitas/recusadas, filtros e origem real Slack/painel |
| [#74 — Aprovar, pedir alteração e reexecutar](https://trello.com/c/qzCbNMo5) | F5A/B | Mesma política do núcleo, justificativa, origem painel e reexecução elegível por falha de infraestrutura |
| [#75 — Configuração versionada](https://trello.com/c/nCmx3DPf) | F6A | Admin conferido no GitHub, append-only, antes/depois e versão usada dentro do recibo assinado |
| [#76 — Revisão das sugestões](https://trello.com/c/zG4VlGAR) | F6B | Aprovação humana por configuração versionada; depende da geração de propostas |
| [#77 — Vínculo Slack–GitHub](https://trello.com/c/ci4pR8Ay) | F3D, executando F7A antecipadamente | Prova de identidade, persistência e consumo pelo resolvedor; desbloqueia usuários reais na Central |
| [#78 — Limites de SLA](https://trello.com/c/cZ6GTYWL) | F7B, independente de F6 | Admin configura por repositório e o workflow existente lê pela API, sem acesso ao banco |

Dependências de núcleo já existentes a revalidar: [modelagem de dados](https://trello.com/c/0hLa3YAf),
[gate determinístico](https://trello.com/c/coWITCr2), [recibo](https://trello.com/c/WmuvNVr3),
[permissões](https://trello.com/c/0iprODgS) e [publicação no PR](https://trello.com/c/soHggtDN).
Dependências compartilhadas: [Central Slack](https://trello.com/c/ZYMuIFyh),
[SLA](https://trello.com/c/8qtCNgBJ), [App Home](https://trello.com/c/UMmYHNeK)
e [geração de sugestões](https://trello.com/c/66KTzOk5). A App Home é outro canal;
reusar identidade, escopo e serviços de leitura/configuração, sem somar toda sua
implementação ao escopo de telas web.

Reconciliações obrigatórias:

- Os caminhos do card #66 prevalecem sobre os nomes em inglês propostos na revisão
  inicial do plano; continuam planejados, não implementados.
- O layout atual de quatro métricas não tem todos os slots citados no card #67,
  como tempo médio. F0 define sua apresentação contextual sem eliminar o requisito
  do contrato nem redesenhar os cards arbitrariamente.
- O card #74 relaciona justificativa à calibração, mas guardá-la não implementa o
  endpoint de feedback posterior ou seu consumo pelo calibrador. Registrar essa
  dependência; não marcar aprendizado como entregue só por persistir a review.
- Cards antigos de apresentação que dizem que o aPRova não tem telas próprias
  descrevem o escopo anterior. O pedido atual, Figma e cards #63–#78 autorizam o
  dashboard e são a referência deste trabalho.
- A rota pública de um recibo por UUID permite a antecipação técnica F2; acesso a
  coleções/instalações e às demais telas privadas continua bloqueado por #65/#66.
  Nenhum card é movido, alterado ou concluído automaticamente por este plano.

## 3. Stack e organização da aplicação

Manter as tecnologias declaradas em PRODUCT.md e instaladas no package.json do
protótipo. Reutilizar suas versões e lockfile como base, verificando a versão de
Node exigida pelas dependências; não presumir que o Node 22 usado nos testes de
workflows do CI já atende ao Vite do front.

| Papel | Tecnologia |
| --- | --- |
| Aplicação e build | React, TypeScript, Vite |
| Estilo e composição | Tailwind CSS, tokens CSS, clsx, tailwind-merge, class-variance-authority |
| Rotas | React Router |
| Componentes acessíveis | Radix; código de componentes permitido em PRODUCT.md, com licença registrada |
| Tabelas e gráficos | TanStack Table; Recharts apenas onde necessário; mapa de calor existente pode continuar em CSS |
| Ícones, datas, fontes | Lucide, date-fns pt-BR, Fontsource Inter Variable e IBM Plex Mono |
| Qualidade | Biome, TypeScript e Playwright |
| HTTP | `fetch` centralizado, AbortController, DTOs e adapters; nenhuma nova biblioteca de estado é requisito |

Estrutura proposta para o front real, criada na fase 1:

```text
web/
  src/app/                rotas, composição, sessão e fronteiras de erro
  src/componentes/        primitivas, casca, tabelas, estados e modais comuns
  src/funcionalidades/    visao-geral, analises, recibos, repositorios, regras,
                         auditoria, configuracao, sugestoes, conta e sla
  src/servicos/           cliente HTTP, DTOs, adapters e configuração pública
  src/estilos/            tokens e temas
  public/                ativos locais necessários
  e2e/                   contratos HTTP simulados, fluxos e capturas
```

`design/prototipo/` permanece referência de validação. Reaproveitar, após revisão,
Casca, Pagina, Esqueleto, Painel, Tabela, SeloDeRisco, CampoDeBusca, Selecao,
MapaDeCalor, FaixaDeMetricas e formatação. O App atual só registra Visão geral e
404; FilaDeAprovacao é parte da Visão geral, não uma tela completa de análises.

Bibliotecas não devem duplicar primitivas Radix. Preservar LICENSES-TERCEIROS e
os avisos aplicáveis ao portar componentes. Não copiar fixtures para o runtime
real nem portar `?estado=` como mecanismo de simulação acessível em produção.

Entrega definida pelo card #64: o build gera ativos estáticos que o Go incorpora
com `embed`, mantendo um binário/deploy e mesma origem. F1 define um pacote Go
que consiga embutir o diretório gerado, a ordem build Node → build Go e o
comportamento de um checkout limpo. O fallback da SPA vale só para navegação
HTML; APIs e ativos ausentes mantêm status/tipo de conteúdo corretos. O ambiente
de desenvolvimento pode usar Vite/proxy, sem torná-lo o servidor de produção.

## 4. Inventário visual e rotas de interface propostas

IDs consultados no arquivo Figma `VtHNXer6wzJj37QdpN3Lb9`. Os caminhos abaixo
são **rotas do navegador propostas**, não endpoints já existentes no núcleo.
Variantes de PR devem reutilizar a mesma tela com dados e UUID diferentes.

| Tela | Nó escuro / claro | Rota proposta | Primeira integração |
| --- | --- | --- | --- |
| Entrar com GitHub | `9:2` / `15:1769` | `/entrar` | F3: sessão real |
| Visão geral | `9:22` / `15:1789` | `/` | F3: semanal parcial; F4B: composição completa |
| Análises e filtros | `9:385` / `15:2152` | `/analises` | F4A |
| Detalhe da análise | `9:608` / `15:2375` | `/analises/:id` | F4A |
| Recibos e chaves | `9:829` / `15:2596` | `/recibos` | F2: chaves; F4C: lista |
| Detalhe do recibo | `9:1024` / `15:2791` | `/recibos/:id` | F2 |
| Repositórios e instalação | `9:1185` / `15:2952` | `/repositorios` | F4A: catálogo; F4C: saúde |
| Análises de um repositório | `45:3566`, `45:3759`, `45:3952` | `/analises?repositorio=...` | F4A |
| Regras do gate e calibração | `9:1351` / `15:3118` | `/regras` | F4C |
| Auditoria de ações | `9:1565` / `15:3332` | `/auditoria` | F4C |
| Configuração do gate | `10:616` / `15:3523` | `/configuracao` | F6A |
| Sugestões da IA | `10:791` / `15:3698` | `/sugestoes` | F6B |
| Conta e integrações | `10:943` / `15:3850` | `/conta` | F3: identidade e antecipação de F7A |
| Limites de SLA | `10:1078` / `15:3985` | `/sla` | F7B |
| Página não encontrada | `86:5942` / `86:6451` | `*` | F1 |

Modais, expansões e estados também fazem parte da entrega:

| Grupo | Nós de referência | Comportamento e dependência |
| --- | --- | --- |
| Aprovar / pedir alteração / reexecutar | `10:1235`, `10:1251`, `10:1263`; mobile `16:225`, `16:238` | Contexto da análise, justificativa, confirmação e resultado; F5 |
| Solicitação recebida / alteração / reexecução | `14:2325`, `14:2331`, `14:2337` | Estados de resposta verdadeiros; HTTP aceito não significa análise concluída |
| Verificação válida / falha | `10:1272`, `10:1309`; mobile `16:159` | Conferências reais, inclusive não verificadas; F2 |
| Cadeia / exportação de trilha | `10:1345`, `10:1354` | Dependem de API adicional; F4C |
| Revisão de configuração / confirmação | `10:1361`, `10:1388`, `14:2343` | Diff real antes/depois, redução de proteção e versão; F6A |
| Caminho sensível | `452:8465`, `452:8567` | Entrada editável, validação, adição e remoção da lista; F6A |
| Revisão / recusa de sugestão | `10:1397`, `14:2349` | Proposta não modifica o gate sem aprovação; F6B |
| Conectar / desconectar Slack | `10:1409`, `10:1426`, `14:2361` | Provar identidade e persistir vínculo no servidor; F7A |
| SLA salvo | `14:2355` | Somente depois de persistência confirmada; F7B |
| Filtros / página seguinte | `14:909`, `14:1060` | URL, coleção completa e paginação do servidor; F4A |
| Vazio / carregando / erro | `14:1510`, `14:1626`, `14:1745`; visão geral `86:5562`, `86:5794`, `86:5678` | Implementar desde F1 em cada recurso |
| Sem acesso / sessão expirada / ação negada / autoaprovação | `14:1861`, `14:1977`, `14:2093`, `14:2209` | Não inferir o motivo específico de um 403 genérico; F3/F5 |
| Detalhe com falha de infraestrutura / aprovado / calibrado | `14:1361`, `14:2489`, `45:4970` | Fixtures distintas e valores reais do núcleo; F4A |
| Mapa de calor / escolha de análise | `82:5826`, `82:5846`, `82:5232` | Célula abre apenas análises do mesmo período/risco/escopo; F4B |
| Testes executados / casos semelhantes | filhos de `9:608`, `334:7382`, `434:9384` | Distinguir sumário, saída bruta, suítes estruturadas e histórico; F4A |

Mobile: `16:2` (visão geral), `16:58` (detalhe), `16:119` (recibos),
`16:159` (verificação), `16:215` (menu), `88:653` (análises), `88:741`
(regras), com variantes claras na página `5:3`. Testar em 390×844 e em largura
menor de 360px; sem desenho específico, adaptar os mesmos componentes, sem criar
outro produto. Desktop de referência: 1440×900, incluindo conteúdo abaixo da dobra.

O Figma contém modais soltos de histórico (`447:8332` e `447:8355`), mas a tela
atual de configuração já apresenta a tabela de versões e não tem botão “Ver
histórico”. Não reintroduzir esse botão por causa dos modais remanescentes.
O guia `17:2676` é apoio ao protótipo; não é uma funcionalidade operacional
obrigatória nem deve navegar o produto real por PRs fictícios.

## 5. Matriz de disponibilidade real do núcleo

**A:** HTTP implementado, consumível após verificar ambiente e origem.
**B:** HTTP implementado, mas exige adaptação segura para uso web.
**C:** há lógica/dados internos, falta contrato HTTP.
**D:** falta também comportamento/persistência necessário ao produto.

| Recurso | Classe | Evidência e limite | Decisão |
| --- | --- | --- | --- |
| Saúde e prontidão | A | `GET /health`, `GET /ready` | F1; `/ready` pode responder 503; não prova autorização ou saúde de cada repositório |
| Buscar recibo por UUID | A | `GET /receipts/{id}` | F2; não existe listagem de recibos |
| Verificar recibo | A | `GET /receipts/{id}/verify` | F2; verifica recibo e elo anterior, não toda a instalação |
| Chaves públicas | A | `GET /receipts/keys` | F2; chaves públicas podem ser entregues ao navegador |
| Exportar recibo individual | A | Corpo de `GET /receipts/{id}` já é documento exportável | F2; download no cliente não exige inventar endpoint `/export` |
| Relatório semanal | B | `GET /reports/weekly`, HMAC da query bruta | F3, por servidor; janela de sete dias, sem heatmap/fila/tentativas por tipo |
| Aprovar / pedir alteração | B | `POST /acoes-remotas`, HMAC e `usuario_slack`; reviews publicadas no GitHub | F5, após identidade web e origem próprias; não chamar diretamente do React |
| Reexecutar | D | Tipo aceito, mas `cmd/api/main.go` injeta `ReenfileiradorIndisponivel` | F5B só após implementação e validação de reanálise; hoje falha na execução |
| Resumo/lista de análises | B | `GET /api/analises` e `GET /api/analises/{id}` existem na base F0 atual, com HMAC n8n, filtros de estado/repo/PR e paginação | F3/F4A; falta autorização por sessão e filtros de risco/período |
| Fila enriquecida, detalhe e timeline | C | Consultas atuais retornam só resumo; `task_steps`, resultados e gate não entram no DTO | F4A; falta contrato agregado |
| Calibração e casos semelhantes | C | `risk_gates.calibration_*`, domínio e memória | F4A/F4C; não estão no DTO do recibo |
| Sandbox e saídas | C/D | Sumário e `raw_output` persistidos; lista de casos/suítes não é contrato exposto | F4A; estrutura de casos e comprovação das etapas exigem contrato/extração no núcleo |
| Visão geral de 30 dias | C | Dados de execução existem; não há agregação HTTP para o desenho | F4B; não renomear o relatório semanal como visão de 30 dias |
| Lista de recibos / exportação de trilha / cadeia completa | C | Persistência de recibos e elos | F4C; faltam coleção paginada, escopo da exportação e verificação completa |
| Repositórios e check obrigatório | C/D | Cliente GitHub e workflows de onboarding não são API de dashboard | F4A/F4C; distinguir configurado, não configurado e não confirmado |
| Regras e estatísticas | C | Catálogo determinístico em Go; decisões persistidas | F4C; catálogo/limites efetivos e contagens precisam de API |
| Auditoria de ações | C | `remote_action_attempts` e justificativa persistida | F4C; falta leitura; origem painel exige extensão do modelo/contrato |
| Login, sessão e escopo autorizado | D | Sem rotas OAuth/sessão/middleware web em `Rotas()` | F3; GitHub App instalado não equivale a usuário autenticado |
| Gate por repositório e histórico de versões | D | Limites/env e regras existem; gestão versionada por repositório não | F6A; formulário local não altera worker |
| Sugestões de regras pela IA | D | Fluxo previsto não é API de proposta/aprovação persistida | F6B; não confundir com calibração automática existente |
| Vincular/desvincular Slack pela conta | D | Tabela de vínculos existe, mas runtime resolve `SLACK_GITHUB_VINCULOS` em memória | F7A; tabela sozinha não entrega vínculo funcional |
| SLA por repositório | D | Timer/persistência no n8n já existem; políticas vêm de `APROVA_SLA_POLITICAS_JSON`; não há CRUD HTTP do desenho | F7B: API/gestão e adaptação do workflow existente, coordenadas com Etapa S |

`POST /webhooks/github` existe, mas é entrada assinada de eventos GitHub, não
endpoint para um formulário do dashboard iniciar/forjar uma análise.

### 5.1. DTOs existentes: integrar sem completar com suposições

Recibo exportado: `id`, `task_run_id`, `key_id`, `hash_sha256`,
`payload_base64`, `assinatura_base64`, `recibo_anterior_id`, `payload`.
Data de emissão e instalação estão dentro do conteúdo assinado (`emitido_em`,
`tenant_id`); o envelope não contém `signed_at` nem `tenant_id` de topo.
`payload` pode ser omitido se o conteúdo não for JSON válido; manter uma tela
diagnóstica e o documento bruto, em vez de quebrar ou inventar conteúdo.

Verificação: `id`, `valido`, `falha?`, `detalhe?`, `verificacoes[]`,
`hash_sha256`, `key_id`. Cada conferência usa `nome` e `resultado` com `ok`,
`falhou` ou `nao_verificada`. As cinco conferências são hash, chave, assinatura,
conteúdo e elo. **HTTP 200 pode conter `valido: false`.** ID inválido/inexistente
responde 404; há 500 e 503 conforme a falha/dependência.

Chaves: `{ chaves: [{ key_id, algoritmo, chave_publica_base64 }] }`.
Lista vazia não comprova a validade de recibos nem chave conhecida.

Relatório: `installation_id`, `inicio`, `fim`, `total_execucoes`, `concluidas`,
`rejeitadas`, `falhas`, `em_andamento`, `taxa_sucesso`,
`exigiram_aprovacao_humana`, `risco_medio`, `revertidas`.
`inicio` é `AAAA-MM-DD`; intervalo `[inicio, fim)` de sete dias UTC; sem início,
última semana completa. `taxa_sucesso` é concluídas / (concluídas + rejeitadas +
falhas), ou `null` sem terminais. “Exigiram aprovação” não é o número aguardando
agora. Falhas de task_run não são a taxa de tentativas de sandbox que falharam.

Ação remota atual: `id_acao`, `tipo`, `usuario_slack`, `repositorio`,
`numero_pr`, `installation_id`, `task_run_id?`, `justificativa`.
Tipos: `aprovar`, `solicitar_alteracao`, `reexecutar_analise`.
Justificativa aparada e não vazia para os dois primeiros, máximo 4000 caracteres
no núcleo; o front deve usar a mesma contagem contratada e validar também no
servidor. O Slack limita seu formulário a 3000. Campos desconhecidos são
recusados. Respostas: 202, 400, 401, 403, 409, 500.
O motivo detalhado da recusa fica na auditoria, não na resposta genérica 403.

### 5.2. Fronteira navegador/servidor

Decisão proposta: hospedar a SPA e uma camada web no servidor sob a mesma origem.
Preferir extensão em Go/chi no próprio núcleo, reaproveitando serviços e políticas,
em vez de introduzir outro backend. Para usar `/reports/weekly`, uma mediação
servidora autentica a sessão, verifica o escopo e assina a query exata com o segredo
servidor. Outra opção de implementação é reutilizar internamente a fonte de
relatórios depois de autorizar a sessão; documentar a opção escolhida em F3.
Essa camada ainda não existe; não descrever proxy do Vite como BFF pronto.

Em F2, um proxy local com alvos fixos pode encaminhar as rotas públicas de recibos.
Em produção, preservar a decisão documentada de recibos verificáveis publicamente;
não transformar UUID em prova de autorização para telas privadas. Dados de
instalação, análise e gestão exigem sessão e autorização no servidor. Se for
escolhida outra origem, configurar CORS restrito e credenciais corretamente;
CORS não substitui autorização.

## 6. Fases e critérios de saída

Cada sublote deve produzir código revisável, um caminho utilizável e evidência.
Todo critério funcional tem de ser validado com contrato HTTP real; testes com
interceptação comprovam estados do cliente, não implantação do núcleo.

### F0 — Preparar a execução e fechar o levantamento

Entrada: este plano, acesso de leitura ao Figma e ao repositório.

1. Confirmar branch, SHA, estado do checkout e instruções já existentes.
2. Revalidar o AGENTS.md unificado com as orientações locais fornecidas: comandos, convenções,
   ferramentas, contexto necessário e formato de entrega. Registrar conflitos.
   Incorporar orientações adicionais de construção caso sejam fornecidas, sem bloquear
   o trabalho independente. Preservar os dois planos e checklists por escopo.
3. Atualizar a matriz tela × endpoint × campos × permissão, a partir de `Rotas()` e
   handlers atuais. Registrar contrato proposto separadamente do implementado.
   Cruzar os cards #63–#78 e a branch dos workflows; registrar o card e os critérios
   de aceite de cada sublote. Não tratar lista “Pronto” como handler existente.
4. Inspecionar tokens, componentes e licenças reaproveitáveis; associar cada rota
   a nós Figma, variantes e estados. Verificar conectividade MCP no ambiente executor.
5. Definir instância de teste, origem da API, recibo real acessível e como serão
   fornecidas credenciais somente ao servidor. Não copiar valores para o plano.

Saída: escopo e dependências explícitos; primeiro lote implementável escolhido;
nenhum endpoint fictício anunciado como pronto. Falha do MCP bloqueia a validação
de fidelidade da tela afetada, não autoriza redesenhar de memória.

### F1 — Fundação da aplicação e componentes compartilhados

Entrada: F0; nenhuma nova API de produto necessária.

- F1A: criar `web/`, scripts e lockfile compatíveis com a stack; portar tokens,
  fontes, ativos/licenças, casca desktop/mobile, tema persistido e 404.
- F1B: registrar as rotas do inventário e seus estados honestos de recurso ainda
  indisponível; criar primitivos reutilizáveis de formulário, modal e tabelas.
- F1C: cliente HTTP/adapters, erros tipados, timeout/cancelamento, configuração
  pública da origem e proxy local de alvos fixos. Smoke de `/health` e `/ready`.
- F1D: fixtures e cenários Playwright isolados da execução real, capturas iniciais
  e comandos de desenvolvimento documentados. Não exibir dados da instalação
  fictícia nem um usuário autenticado no runtime real.
- F1E: build estático embutido em Go e servido pelo backend, conforme #64;
  testes de componentes com comportamento observável (modal/foco, filtros,
  estados e erros) e Biome/build/testes no CI. Reusar Playwright com harness
  isolado quando suficiente; não instalar outro framework só para repetir markup.

Saída: build e Biome passam; navegação por URL/reload funciona; temas e teclado
funcionam em 1440×900/390×844/360px; nenhum segredo no bundle; 503 de prontidão e
erro de rede possuem tratamento. Rotas incompletas não caem silenciosamente em 404.
Um build limpo Node → Go gera binário com a SPA; deep link HTML funciona e uma
API inexistente continua 404, sem devolver login/index como HTTP 200.

### F2 — Primeira entrega integrada: recibos por UUID

Entrada: F1 e endpoints públicos de recibos respondendo no ambiente de teste.

- F2A: `/recibos/:id` fiel a `9:1024`; entrada por link/UUID conhecido e um campo
  acessível para localizar recibo por ID enquanto não existe coleção. Esse campo
  é uma adaptação funcional documentada, sem fabricar uma lista de recibos.
- F2B: buscar chaves e executar verificação real; representar as cinco conferências,
  recusas e estados não verificados; exibir a data da consulta desta sessão.
- F2C: exportar o documento retornado e a chave pública selecionada. Preservar
  os campos base64 byte a byte; não gerar assinatura, hash ou “resultado válido”.
- F2D: representar ausência de elo anterior e dados ausentes; permitir consultar
  o elo conhecido, mas manter a verificação de cadeia completa indisponível.

Saída: um recibo real é aberto, verificado e baixado; o arquivo exportado é
verificável pelo caminho independente em `docs/verificacao-recibo.md` usando uma
chave pública conhecida. Cobrir 404, 500/503, `valido:false`, chave desconhecida,
`anterior:null`, payload ausente e cancelamento ao trocar ID.
Sem endpoint de listagem, a coleção continua pendente de F4C.

### F3 — Sessão web, escopo e relatório semanal real

Entrada: F1. F2 pode continuar independentemente desta extensão do núcleo.

**Dependência do núcleo (lote próprio):** login GitHub, callback validado com state
e proteção adequada ao fluxo escolhido, sessão/cookie HttpOnly, Secure em HTTPS,
SameSite apropriado, expiração/logout, autorização de instalações e repositórios,
proteção CSRF nas mutações e tratamento de identidade/escopo. Nenhuma identidade
recebida do React é confiada sem resolução no servidor.

- F3A: implementar o contrato de sessão e a mediação do relatório; impedir que
  `installation_id` informado pelo cliente escolha outro tenant sem autorização.
- F3B: tela de login, logout e recuperação de sessão, identidade em Conta e
  integrações, casca com escopo real e estados sem instalação/sem acesso.
- F3C: mostrar o agregado semanal real com seu período e denominadores. É uma
  entrega parcial explicitamente semanal; não preencher os cards/heatmap de
  30 dias com esse DTO. Fila e demais agregações continuam indisponíveis.
- F3D: após F3B, antecipar F7A (vínculo Slack–GitHub, card #77) em lote próprio
  quando o contrato/prova de identidade estiverem disponíveis. A branch dos
  workflows já entrega a Central, mas documenta ausência de vínculos humanos.
  Não é preciso aguardar F4–F6 para esse cadastro; F3D referencia F7A e não cria
  uma segunda implementação de vínculo.

Saída: login/logout funcionam e não deixam cache de outro usuário; sessão expirada,
401, 403 e troca de escopo são testados; adulteração de tenant/repositório é negada
no servidor; segredo HMAC fica ausente de bundle/devtools/logs; relatório bate
com a fonte real, incluindo `taxa_sucesso:null` e limites de data UTC.
Nenhuma tela privada é publicada sem essa fronteira.

### F4 — Leitura operacional completa, em três entregas

Entrada: F1 e F3 para dados privados. Contratos de leitura do núcleo precisam
ser implementados e testados antes de anunciar cada sublote como integrado.

**F4A — Análises, detalhe e escopo de repositório (maior valor operacional).**

- Núcleo: coleção paginada de análises e detalhe por UUID, catálogo de repositórios
  autorizados e metadados necessários. Consultar `task_runs`, `task_steps`,
  `sandbox_results`, `risk_gates`, `approvals` e vínculo de recibo sob o mesmo tenant.
  Preservar `/api/analises` e `/api/analises/{id}` previstos no card #66;
  documentar OpenAPI e responder 404 para ID de outra instalação.
- Front: filtros de PR/repositório/nível/estado/período, ordenação e paginação;
  detalhe com autoria/branch quando disponíveis, SHA, links GitHub/Check, timeline,
  motivos, calibração/casos semelhantes, tentativas/saída e decisão humana.
- O modelo persistido de task_run não contém `head_branch`; não fabricar branch.
  Definir se será persistida ou obtida pelo núcleo no GitHub. IDs do Check existem
  internamente, mas falta contrato para URL/visibilidade na tela.
- Casos de teste e suítes por nome não são inferidos dos contadores. Se o núcleo
  só entregar `raw_output`, mostrar saída como texto e indicar a ausência de dados
  estruturados. Especificar captura/parse no núcleo por linguagem para a expansão
  completa, incluindo estados passou/falhou/pulado e saída truncada.
- “Testes sem internet” e tempos das duas etapas exigem evidência/metadata do
  executor ou contrato confiável; não preencher status e duração por ilustração.
- As ações de escrita continuam desabilitadas até F5; links reais ao GitHub
  permitem a revisão pelo mecanismo nativo nesse intervalo.
- A leitura de estado atual pode reaproveitar o serviço do detalhe para o SLA
  antes de publicar aviso, conforme a Etapa S. Dashboard usa sessão de usuário;
  n8n precisa de contrato de serviço autenticado/escopado. Não obrigar o workflow
  a simular login web nem abrir a leitura privada para satisfazer o timer.

Saída F4A: usuário encontra e abre uma análise real pelo UUID; lista e detalhe
concordam; commit novo não reaproveita silenciosamente a decisão anterior;
infraestrutura sem gate mostra “não avaliado”; ausência de branch/saída/histórico
fica explícita. Filtros/total são do escopo completo, não só da página carregada.

**F4B — Visão geral de 30 dias e fila atual.**

- Núcleo: agregação própria com intervalo/fuso/denominadores definidos; fila atual
  de `aguardando_aprovacao`, tempo de espera desde entrada nesse estado, distribuição
  de desfechos, falhas por tipo/tentativa, série diária por risco e último recibo.
  A rota prevista é `/api/metricas` (#66/#67); incluir tempo médio com unidade,
  população e denominador definidos, conciliando sua apresentação com o Figma.
- Distinguir fila atual de estatística no período. Uma pendência antiga não deve
  desaparecer só porque foi recebida antes do intervalo dos gráficos.
- Front: recompor `9:22` com dados reais, busca e filtro da fila, barras, heatmap
  e seleção de análises da célula. Atualizar leituras por polling moderado e
  cancelável, pausado em aba oculta; SSE/WebSocket não é requisito nem rota existente.

Saída F4B: totais, frações e soma das séries reconciliam com o backend. Contar
análises avaliadas, task_runs e tentativas conforme a legenda, sem misturá-los.
Usar janela `[inicio,fim)`, fuso de agrupamento explícito e zero distinto de
ausente. Célula só abre análises compatíveis; atualização não perde filtros/foco.

**F4C — Recibos em coleção, regras, auditoria e saúde de repositórios.**

- Núcleo: lista paginada de recibos; estatísticas/catálogo/limites efetivos de
  regras; trilha de ações com filtros; consulta de saúde e check obrigatório;
  exportação completa autorizada; verificação de toda a cadeia com escopo e
  instante de referência. Reaproveitar lógica de recibo e políticas existentes.
- Front: todas as listas e links cruzados, exportação JSON/CSV de escopo explícito,
  tabela de chaves, conferência de cadeia e detalhe do primeiro elo inválido.
- Verificação de cadeia deve distinguir cadeia inteira, segmento e amostra.
  Apenas caminhar de um recibo até seus ancestrais não prova que ele é o último
  recibo da instalação. Exigir referência da cabeça e completude no servidor.
- A API atual de auditoria não existe e a persistência não fornece um campo de
  origem painel pronto: formalizar evolução, sem rotular ações Slack como painel.
- “Check obrigatório” desconhecido ou sem permissão fica “Não confirmado”; 403 do
  GitHub não vira “Não configurado”. Atualizar verificação consulta o servidor.

Saída F4C: coleções e exportações respeitam tenant/repositório/período; CSV não
executa fórmulas originadas de dados; JSON preserva documentos assinados. Falha
parcial e duração de exportação/verificação são representadas. Dados de motivos,
calibração e estatísticas vêm do núcleo, não de regras duplicadas no navegador.

### F5 — Decisões humanas e reexecução

Entrada: F3 e F4A; sessão, política e contrato de ações web prontos.

**F5A — Aprovar e pedir alteração.** Extensão do núcleo deve resolver a identidade
GitHub pela sessão, reutilizar autorização por repositório e bloqueio de
autoaprovação, vincular análise/tenant/PR/SHA atual, persistir justificativa e origem
web, publicar review e preservar o fluxo de webhook/worker. Não usar identidade
Slack fictícia, nem escrever reviews diretamente do navegador.

Front: modais Radix fiéis, justificativa com contexto e limites, estado enviando,
erros recuperáveis, prevenção de clique duplo e retorno de foco. Usar identificador
de ação estável para repetição da mesma submissão; não retentar mutações em loop.
202 informa a aceitação/publicação conforme o contrato; acompanhar o estado real
por leitura. 409 exige consultar o resultado, sem duplicar efeito. Um 403 genérico
não autoriza informar “você é o autor” se esse motivo não foi devolvido com segurança.

Saída F5A: revisor autorizado decide; autor e leitor são negados; justificativa
vazia/excessiva e commit obsoleto são tratados; repetir a ação não duplica review;
auditoria e resultado no GitHub/núcleo concordam. Nenhum texto diz “merge feito”.
O evento `aprovacao_registrada` continua sendo a confirmação para os consumidores
de automação: a Central correlacionada atualiza a mensagem existente, quando
habilitada. Validar o percurso com decisão real em ambiente de teste; o teste
Slack documentado com decisão fictícia não cobre decisão originada no painel.
Não ativar flags de correlação/SLA como efeito colateral da implementação web.

**F5B — Reexecutar.** Implementar primeiro o reenfileiramento consumido pelo worker,
sem modificar o recibo append-only ou reaproveitar indevidamente uma decisão
antiga. Definir nova execução/tentativa, SHA alvo, idempotência e correlação com a
anterior. Só então ativar o modal e acompanhar até novo resultado. A base auditada
retorna erro nessa operação; não deixar o botão ativo apenas porque o enum existe.
O card #74 prevê reexecução de falha de infraestrutura. Elegibilidade deve ser
informada/revalidada pelo núcleo; não habilitar para qualquer análise concluída
ou falha de teste apenas porque o Figma tem uma variante de botão.

Saída F5B: reexecução real chega ao worker, produz resultado rastreável e não
altera recibo anterior; indisponibilidade/timeout/conflito não geram sucesso falso.

### F6 — Gestão de regras: configuração e propostas

Entrada: F3/F4C; contratos novos e autorização administrativa.

**F6A — Configuração versionada por repositório.** Núcleo: configuração efetiva,
histórico, nova versão imutável com ator/justificativa, controle de concorrência e
aplicação pelo worker em futuras análises. Incluir a versão usada no conteúdo
assinado do recibo, requisito do card #75. Isso exige evolução explícita do
formato e dos verificadores/exportadores, mantendo os recibos antigos verificáveis;
um campo solto na UI não satisfaz o requisito. Não reescrever
recibos antigos nem substituir configuração global inadvertidamente.

Front: limite de arquivos editável, lista de caminhos adicionável/removível,
validação conforme semântica real dos padrões (diretórios e glob de nome), descarte,
estado sem mudanças, revisão antes/depois e aviso quando reduz proteção. A nova
versão só aparece após persistência confirmada; conflito exige reler e reconciliar.
Preservar tabela de histórico já existente e os alinhamentos/botões do Figma atual.

Saída F6A: sem permissão é leitura; input funciona; diff corresponde ao formulário;
duas edições concorrentes não se sobrescrevem; worker usa a nova versão e uma
execução anterior mantém sua decisão. Falha ao salvar conserva a edição.

**F6B — Sugestões da IA.** Núcleo: propostas persistidas com evidência, estado e
versão alvo, revisão/recusa e eventual criação de configuração por aprovação humana.
Front: lista com filtros, detalhe da proposta, evidência e revisão/recusa com
resultado real. Uma sugestão nunca chama diretamente um setter do gate nem muda
classificação passada. Calibração por histórico e sugestão de nova regra são
funcionalidades diferentes.

Saída F6B: proposta aceita cria a alteração prevista sob autorização; recusada
permanece auditável; revisão de proposta desatualizada é detectada. IA não aprova
sua própria proposta nem modifica o gate sozinha.

### F7 — Conta, integrações e limites de SLA

Entrada: F3 e contratos novos correspondentes.

**F7A — Slack.** Implementar prova de propriedade da conta/workspace, conexão e
desconexão persistidas e uso dos vínculos pelo resolvedor real do núcleo. Front
representa identidade da sessão, vínculo, modal e falhas. O simples preenchimento
de um Slack ID não prova propriedade. Não expor tokens Slack/GitHub nem mostrar
que desconectar “desinstala o GitHub App” se só remove um vínculo pessoal.
Este lote pode ser executado em F3D, logo após sessão/escopo (#77), sem esperar
a gestão de regras. Reaproveitar o vínculo no dashboard, Central e futura App Home,
com resolução consistente e escopo de workspace. A fonte em memória
`SLACK_GITHUB_VINCULOS` precisa de transição compatível para a fonte persistida;
não manter duas fontes contraditórias silenciosamente.

Saída F7A: vínculo do usuário certo é efetivamente consumido em uma ação de teste;
desvínculo invalida esse caminho; outra sessão/workspace não pode tomar o vínculo.

**F7B — SLA.** Núcleo: leitura/escrita por repositório, unidade e limites válidos,
versionamento/auditoria e ligação com os timers do n8n. Definir semântica de
mudança para pendências já existentes. Front: primeiro aviso e escalonamento,
revisão do efeito para a equipe, salvar/enviando/erro/salvo real.

Reutilizar o timer periódico e a persistência atuais; ele verifica a cada minuto
e ambos os limites contam desde o início da espera, em minutos corridos. O
segundo limite deve ser maior que o primeiro, sem somar outro prazo ao primeiro.
O card #78 exige que o workflow leia a política pela API; salvar no núcleo sem
adaptar o consumidor não conclui a entrega. Canal/responsável precisam ser
confirmados, sem suposição de nomes ou IDs. Coordenar com a Etapa S: suportar
leitura de estado atual, reserva/confirmação de avisos e recuperação de resultado
incerto no núcleo; migrar os fatos necessários antes de retirar Data Tables.
Isso é extensão do núcleo e adaptação do n8n em lotes próprios, não lógica React.
As tabelas atuais continuam válidas até a migração, sem um terceiro estado
paralelo no navegador. Digest também depende das operações de reserva/confirmação;
sua ativação e interface administrativa nova não fazem parte deste card de front.

Saída F7B: valores inválidos são recusados também no servidor; novo limite afeta
automação em teste com relógio controlado; reload confirma persistência. Um timer
no navegador ou localStorage não substitui a configuração do produto.

### F8 — Homologação e entrega da aplicação completa

Entrada: sublotes integrados ou dependências ainda não entregues explicitamente
registradas. Não empurrar toda a validação visual ou de segurança para esta fase.

- Regressão dos fluxos de produto com API e dados de teste reais: fila → análise
  → decisão → resultado → recibo; reexecução; gestão; vínculo; SLA.
- Cobrir papéis, tenants e repositórios diferentes; expiração de sessão, revogação
  de acesso, erros e dados incompletos. Limpar caches em logout/troca de escopo.
- Revisar todos os nós/variantes e capturas em desktop/mobile, nos dois temas;
  focar conteúdo longo, hashes, tabela/rolagem, botões e rodapés de modal.
- Integrar Biome/build/Playwright ao CI sem remover os checks Go/n8n/Gitleaks;
  garantir dependências e navegadores reprodutíveis. Documentar origem, build e
  fallback de rotas SPA, cache de ativos e ausência de segredos no artefato.
- Documentar execução local, ambiente de homologação, migrações necessárias e
  limitações remanescentes. Preparar procedimento reversível de entrega.

Saída: nenhuma navegação essencial quebrada; nenhum sucesso simulado no runtime;
inventário de telas e ações coberto; checks relevantes passam e evidências estão
registradas. Merge e deploy seguem a autorização do trabalho, não são implícitos
na aprovação de uma tela.

## 7. Contratos a criar — propostas, não endpoints disponíveis

Os caminhos `/api/analises` e `/api/analises/{id}` já têm handlers de serviço na
base revalidada em F0, mas ainda precisam evoluir para o contrato web do card #66.
`/api/metricas` e os demais contratos abaixo continuam propostos. Os demais nomes
serão fechados no lote do núcleo, com testes. Usar a família `/api` para sessão
web e preservar as rotas de automação com autenticação de serviço.
Reusar serviços de domínio não exige expor a mesma credencial
ou permitir que o n8n simule uma sessão de usuário.

| ID | Contrato proposto | Conteúdo mínimo / requisito | Desbloqueia |
| --- | --- | --- | --- |
| C01 | Sessão: início/callback GitHub, `GET /api/sessao`, logout | Usuário resolvido, escopos/instalações/repos autorizados, expiração e proteção de sessão | F3 |
| C02 | `GET /api/relatorios/semanal` | Autorizar instalação e mediar fonte semanal existente sem segredo no cliente | F3 |
| C03 | Evoluir `GET /api/analises` e `GET /api/analises/{id}` (#66), hoje resumos de serviço HMAC | Autorização web, filtros de nível/período e detalhe agregado consistente, UUID, SHA, timeline, gate, tentativas e recibo | F4A |
| C04 | `GET /api/repositorios` e saúde por repositório | Catálogo autorizado, permissões verificadas, check obrigatório com status desconhecido e instante de consulta | F4A/F4C |
| C05 | `GET /api/metricas` (#66/#67) | Intervalo/fuso/denominadores, fila atual, métricas por tentativa, tempo médio, série diária e último recibo | F4B |
| C06 | `GET /api/recibos`, exportação e verificação de cadeia | Coleção autorizada; leitura individual pública permanece em `/receipts/*`; cadeia com cabeça/elos/falhas e completude | F4C |
| C07 | Leitura de regras/estatísticas/calibração e casos semelhantes | Catálogo/limites efetivos, denominadores por motivo e rastro persistido sem embeddings brutos | F4A/F4C |
| C08 | `GET /api/auditoria` | Histórico paginado de tentativas, origem, ator, justificativa e filtros sob autorização | F4C |
| C09 | Ações web: aprovação, alteração e reanálise | Identidade da sessão, política do núcleo, análise/SHA, idempotência, justificativa e origem web | F5 |
| C10 | Configuração efetiva, versões e criação de versão | Repositório, admin, comparação de versão, aplicação pelo worker e versão no recibo assinado | F6A |
| C11 | Propostas de regra, revisão e recusa | Evidência, estado/versão alvo, ator e alteração humana rastreável | F6B |
| C12 | Vínculo Slack pessoal | Prova de propriedade, workspace, status, conectar/desconectar e resolvedor integrado | F3D/F7A |
| C13 | Configuração de SLA | Unidade, limites, repositório, ator e aplicação nos timers | F7B |
| C14 | Estado atual e reserva/confirmação dos avisos (Etapa S) | Serviço escopado ao n8n, idempotência, concorrência, entrega incerta e migração de fatos/controles existentes; sem acesso direto ao banco | Apoio de núcleo/n8n a F4A/F7B; também beneficia Digest |

O feedback posterior à aprovação já tem formulário/workflow, mas seu endpoint
permanece ausente e foi adiado no plano de automação. Considerar a dependência
ao afirmar que justificativas alimentam calibração; não implementar esse ramo
automaticamente como parte de um lote de front.

Cada contrato novo precisa fixar: método/path, DTO e nulabilidade, erros, paginação,
ordenação estável, filtros/intervalo/fuso, limites, isolamento e papéis, identidade
do recurso, correlação, idempotência/concorrência e exemplos reais sanitizados.
Nunca converter um nome dessa tabela em uma chamada ativa sem handler confirmado.

## 8. Validações transversais e definição de concluído

| Tipo | Evidência exigida por lote |
| --- | --- |
| Contrato | Handler e DTO auditados; request/response sanitizados do núcleo; mocks compatíveis, sem campos inventados |
| Visual | Captura Figma do nó + captura local no mesmo viewport/tema/estado; inspeção e diferenças corrigidas, inclusive conteúdo abaixo da dobra |
| Funcional | Fluxo principal e falhas relevantes exercitados no navegador; refresh/deep link/voltar funcionando |
| Acessibilidade | Teclado, foco de entrada/retorno em modal, nomes/erros acessíveis, contraste e alternativa textual a gráfico/risco |
| Segurança de integração | Sem segredo no bundle/logs; escopo validado no servidor; fixtures não entram no modo real; mutações protegidas e sem repetição automática |
| Dados | Nulos/zero/ausência distintos, datas/fuso explícitos, paginação completa e denominadores reconciliados |
| Automação | Comandos exatos e resultado/saída; separar execução com mock de integração real |

As capturas recebidas pelos MCPs e as comparações visuais locais devem ficar
em `.capturas-mcp/`, na raiz do repositório, ignorada pelo Git. Registre os
caminhos locais como evidência, sem adicionar os prints aos commits ou forçar
sua inclusão. Ativos necessários ao produto seguem os diretórios da aplicação.

Comandos de front a preservar na nova pasta, após F1 criar seus scripts:

```sh
npm --prefix web ci
npm --prefix web run verificar
npm --prefix web run build
npm --prefix web run teste
```

Configurar instalação do navegador Playwright e Node compatível no ambiente/CI.
Selecionar specs por lote durante o desenvolvimento; executar regressão pertinente
antes da entrega. Reaproveitar specs úteis do protótipo, retirando pressupostos
de dados fixos e de ações permanentemente desabilitadas. Para verificação visual,
usar fixtures determinísticas separadas; para integração, conferir uma execução
real com IDs do ambiente de teste, sem exigir que ela tenha os números do desenho.

Mudanças Go: testes de handlers/serviços afetados, autorização e concorrência.
Mudanças em persistência: testes PostgreSQL e sqlc; migrations em banco descartável.
A suíte completa do CI continua obrigatória para as mudanças correspondentes.
Não rodar resets destrutivos no banco de desenvolvimento da equipe.

Uma captura local bonita não prova integração; teste com interceptação não prova
API implantada; HTTP 200 não prova recibo válido; 202 não prova conclusão; botão
“Salvar” não prova persistência; GitHub App instalado não prova check obrigatório.

## 9. Protocolo do agente e retomada

Trabalhar por sublotes F1A, F1B etc., dentro do escopo autorizado. Não implementar
o dashboard inteiro em uma única geração de JSX. Ao receber autorização global,
continuar enquanto os critérios e dependências permitirem; não solicitar aprovação
rotineira entre sublotes. Pedir decisão apenas para ambiguidade material sem fonte
ou ação fora da autorização. Um bloqueio de backend não impede componentes e
estados independentes, mas impede declarar a integração concluída.

Antes de cada sublote: ler AGENTS.md, este plano, instruções específicas existentes,
diff do checkout, contrato atual e nós Figma correspondentes. Depois: validar,
revisar o diff, registrar evidência e fazer commit pequeno na branch de trabalho.
Não substituir mudanças de outras pessoas.

Prompt inicial sugerido para o agente executor:

> Leia AGENTS.md e design/docs/implementacao-dashboard.md. Execute a fase/lote indicado
> pela solicitação, começando pela primeira dependência não satisfeita. Consulte
> o Figma via MCP pelos nós do inventário, inspecione os componentes existentes e
> revalide os handlers do núcleo no SHA atual. Reutilize a stack e o design.
> Separe contratos existentes de propostos e mantenha segredos no servidor.
> Implemente, valide visual e funcionalmente, registre evidências e o próximo lote.
> Se um contrato faltar, conclua a parte independente sem simular sucesso no modo
> real. Se a solicitação for de execução completa, avance pelos lotes liberados.

Registro obrigatório por sublote (adicionar entradas abaixo, com links locais de
evidência ou CI e sem valores de credenciais):

```markdown
### Fase/lote — data
- Estado: não iniciado | em andamento | bloqueado | concluído
- Base/commit e escopo:
- Card(s) Trello, critérios cobertos e partes ainda pendentes:
- Nós Figma e temas/viewports:
- Contratos existentes usados / novos ainda pendentes:
- Arquivos alterados:
- Validações e resultados (mock e real separados):
- Evidências visuais e de API:
- Dependências, limitações e próximo lote:
```

Estado deste documento: planejamento revisado com branch dos workflows e cards
Trello; nenhuma implementação de front/backend foi iniciada por esta tarefa.
O card #64 está “Em andamento” no quadro e deve ter seu checkout/progresso
reconciliado antes de F1; não há `web/` nos commits auditados. Os 228 testes locais
da branch dos workflows passaram, com oito rotas do núcleo e nenhum handler novo
de dashboard nessa comparação. As orientações locais de workflows fornecidas pela usuária foram integradas
ao AGENTS.md unificado, preservado fora do Git. O plano de dashboard pode ser
versionado; o plano local de workflows continua ignorado.


## 10. Orientações locais e aplicação deste pacote

Usar um único `AGENTS.md` na raiz, preservando o conteúdo local fornecido pela
usuária e acrescentando a seção de dashboard. Ele e
`docs/plano-atualizacao-workflows.md` permanecem ignorados e fora dos commits.
`design/docs/implementacao-dashboard.md` pode ser versionado. Não usar `git add -f`.

A regra de alterar primeiro a instância n8n e validar antes de exportar aplica-se
a workflows; o front é implementado em `web/`, com a integração Go autorizada.
O progresso dos workflows continua no checklist da Central; o do dashboard fica
neste plano. Ler um plano não inicia sua execução nem autoriza a execução do outro.

O patch do pacote contém somente este documento versionável. O `AGENTS.md`
unificado é aplicado localmente, em separado, preservando eventuais alterações
locais posteriores ao conteúdo enviado. Não usar o patch das revisões anteriores,
que incluía uma proposta de AGENTS.md versionado.

## 11. Registro da execução

### F0 — 16/11/2026

- Estado: em andamento; levantamento técnico concluído, confirmação de ambiente pendente.
- Base/commit e escopo: `feat/web-dashboard`, `a28725daaf15968f45fbcb64ae463bbd9bb14c67`;
  levantamento, sem incorporar App Home ou implementar F1.
- Cards: #63–#78 relidos via Trello; critérios e listas registrados em
  [dashboard-cards-f0.md](dashboard-cards-f0.md). Nenhum card alterado/concluído.
- Nós Figma: três páginas inventariadas; contexto e captura de `5:58`, `9:22`,
  `16:2`; referências de desktop 1440×900 e mobile 390×844, temas no inventário.
- Contratos: dez rotas auditadas; consultas de análises existentes são resumos
  autenticados por serviço; sessão, detalhe agregado e gestão ainda pendentes.
- Arquivos: este plano, `dashboard-f0.md`, `dashboard-inventario-figma.md` e
  `dashboard-cards-f0.md`.
- Validações: testes locais `internal/api` e `internal/receipt` passaram; nenhum
  teste de integração implantada. `/health` local sem conexão na porta 8080.
- Evidências: `.capturas-mcp/f0/` ignorada, com PNGs/contextos/XML; resultados
  e limites descritos em [dashboard-f0.md](dashboard-f0.md).
- Dependências: AGENTS local ausente; homologação e UUID real aguardam confirmação.
  Próximo lote liberado tecnicamente: F1A (#64), sem novas APIs de produto.

### F1A–F1E — 16/11/2026

- Estado: concluído nos sublotes F1A–F1E.
- Base: `c75c7d7` em `feat/web-dashboard`, com a reorganização dos documentos
  autorizada para `design/docs/`. AGENTS local continua ausente.
- Escopo/card: #64, fundação; nenhum aceite de F2–F7 ou mudança no Trello.
- Registro detalhado: [dashboard-f1.md](dashboard-f1.md).

#### F1A

- Card/critérios: #64 — web/, fontes/tokens/ativos locais, casca desktop/mobile, tema e 404.
- Nós/temas/viewports: 9:23, 86:5942, 86:6451, 16:2, 88:826; referências e adaptações no registro F1.
- Contratos: Nenhuma API nova; rodapé sem sessão e dados fictícios.
- Arquivos: web/package*, configs, src/app, src/estilos, src/ativos.
- Validação: Instalação por lockfile, Biome, TypeScript/Vite e capturas em 1440×900/390×844/360×844 nos dois temas.
- Evidências: `.capturas-mcp/f1/`, ignorada; resultados em `dashboard-f1.md`.
- Dependências/próximo lote: F1B; homologação não bloqueia casca.

#### F1B

- Card/critérios: #64 — Rotas reservadas e primitivas de campo, botão, modal, seleção, tabela e erro.
- Nós/temas/viewports: 5:135, 5:144, 10:1263, 15:4170, 16:215; inventário de rotas; referências e adaptações no registro F1.
- Contratos: Telas posteriores ficam indisponíveis; sem mutações simuladas.
- Arquivos: web/src/componentes, src/app/rotas.ts, src/app/App.tsx.
- Validação: Modal/foco/Escape/retorno, validação de formulário, filtros/limpeza e ordenação exercitados no harness.
- Evidências: `.capturas-mcp/f1/`, ignorada; resultados em `dashboard-f1.md`.
- Dependências/próximo lote: F1C; dados e telas operacionais dependem das fases seguintes.

#### F1C

- Card/critérios: #64 — Cliente HTTP, erros tipados, timeout/cancelamento, origem e proxy fixo.
- Nós/temas/viewports: Sem nova tela de produto; padrões de campo/estado da fundação; referências e adaptações no registro F1.
- Contratos: GET /health e /ready reais; 401 privado tratado sem implementar sessão C01.
- Arquivos: web/src/servicos, web/vite.config.ts.
- Validação: DTOs, JSON inválido/HTML, rede, 503, cancelamento e timeout; smoke real dos handlers separado dos mocks.
- Evidências: `.capturas-mcp/f1/`, ignorada; resultados em `dashboard-f1.md`.
- Dependências/próximo lote: F1D; credenciais continuam no servidor.

#### F1D

- Card/critérios: #64 — Harness/fixtures de teste isolados e Playwright.
- Nós/temas/viewports: Mesmos nós da casca/404; capturas locais e comparação inspecionada; referências e adaptações no registro F1.
- Contratos: Interceptações validam cliente; servidor Go isolado usa dependência controlada.
- Arquivos: web/e2e, web/playwright.config.ts.
- Validação: Rotas por URL/reload/voltar, temas, teclado, menu mobile, componentes e erros; /__teste ausente no build normal.
- Evidências: `.capturas-mcp/f1/`, ignorada; resultados em `dashboard-f1.md`.
- Dependências/próximo lote: F1E; sem evidência de API de homologação.

#### F1E

- Card/critérios: #64 — Build estático incorporado em Go, fallback e CI adicional.
- Nós/temas/viewports: Casca/404 conforme F1A; sem novo layout; referências e adaptações no registro F1.
- Contratos: Mesmas dez rotas de API preservadas; fallback só HTML GET/HEAD; APIs/ativos ausentes permanecem erros.
- Arquivos: web/servidor*.go, web/dist/.gitkeep, internal/api/servidor.go, Makefile, .github/workflows/ci.yml, .gitignore.
- Validação: Testes Go de fallback/cache/HEAD/checkout sem build, regressão interna/comandos/web com race, lint sem apontamentos e build do binário; smoke HTTP da SPA.
- Evidências: `.capturas-mcp/f1/`, ignorada; resultados em `dashboard-f1.md`.
- Dependências/próximo lote: Próxima fase: F2, condicionada a API de teste e UUID real.

### F2A–F2D — 16/11/2026

- Estado: implementação local concluída; aceite de homologação pendente.
- Base/escopo: `dbafee3`, branch `feat/web-dashboard`; consulta pública por UUID,
  cinco conferências, chaves e exportação individual. Registro detalhado:
  [recibos F2](dashboard-f2.md).
- Card #70 parcialmente coberto; coleção e cadeia completa permanecem em F4C.
- Nós Figma: `9:1024`, `15:2791`, `16:159`, `10:1272`, `10:1309`;
  dois temas, viewports 1440×900, 390×844 e 360×844.
- Contratos: os três GETs públicos de recibos existentes; nenhuma API nova.
- Arquivos: páginas/serviço/estilo de recibos, rotas, ativos de ajuda/serrilhado,
  cenários Playwright, fixture assinada do servidor exclusivo de testes e documentação.
- Validação: Biome e build aprovados; testes Go de API/receipt/web e go vet web
  aprovados. Regressão F1/servidor aprovada; cenários F2 incluem HTTP local com
  verificador real e assinatura do download conferida independentemente.
- Evidências: `.capturas-mcp/f2/`, ignorada pelo Git. Mock, fixture local e
  homologação são distinguidos no registro detalhado.
- Dependência/próximo lote: URL da API de teste e UUID real ainda não fornecidos;
  validar consulta, verificação, download e conferência independente antes de
  declarar o aceite integrado F2 concluído. F3 é o próximo lote de implementação.

### F3A–F3C — 16/11/2026

- Estado: implementação e validação local concluídas; homologação OAuth real
  pendente. F3D/F7A não iniciado por dependência do contrato de identidade Slack.
- Base/escopo: `9bc2fd6`, branch `feat/web-dashboard`; sessão do GitHub App,
  isolamento de instalações/repositórios, logout e relatório semanal privado.
- Cards: #65/#66 parcialmente cobertos; #67 apenas agregado semanal; #77
  permanece pendente. Nenhum card alterado no Trello.
- Nós Figma: `9:2`, `15:1769`, `10:943`, `9:22`; 1440×900, 390×844,
  360×844 nos dois temas. Referências e adaptações no [registro F3](dashboard-f3.md).
- Contratos novos: início/callback OAuth, consulta de sessão, logout com CSRF e
  GET `/api/web/relatorio-semanal`, documentados no registro F3. Rotas n8n
  continuam exigindo HMAC. Recibos públicos preservados.
- Arquivos: `internal/sessaoweb`, configuração/API, consulta SQL gerada via sqlc,
  páginas/contexto/cliente HTTP web, testes Go/PostgreSQL/Redis/Playwright e CI.
- Validações: build/Biome; regressão Go com race; relatório em PostgreSQL
  descartável; Redis real com cifra, TTL e GETDEL concorrente; linter sem
  apontamentos; dez cenários F3 e regressão F1/F2. Resultado detalhado no registro.
- Evidências: `.capturas-mcp/f3/local-*`, ignoradas. Provedor de teste separado
  do binário de produção; OAuth de conta real não foi simulado como homologado.
- Ambiente local: API atualizada no container `aprova-api-local`, porta 8080,
  reinício automático; `/ready` e recibos da semente aprovados no smoke.
- Dependências/próximo lote: configurar credenciais/callback do GitHub App e
  homologar login/escopo real. F3D exige prova de propriedade Slack e resolvedor
  persistente; F4A é o próximo lote de leitura, condicionado à sessão autorizada.

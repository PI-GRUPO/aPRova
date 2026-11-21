# F4A — Leitura de análises e detalhe autorizado

Execução em 16/11/2026, branch `feat/web-dashboard`, base `bad8b5c`.
AGENTS.md local ausente. Este lote inicia a F4 pela leitura operacional; F4B
(métricas/fila de 30 dias) e F4C (coleções, auditoria e gestão de leitura) seguem
como lotes distintos do plano.

A pendência OAuth foi documentada por orientação explícita da usuária em
[dashboard-f3.md](dashboard-f3.md): as credenciais provavelmente estão no `.env`
da máquina dela, mas não estão neste Codespace. Não foram procuradas fora do
workspace, expostas, fabricadas ou substituídas por uma sessão de demonstração
no binário da API. A homologação de uma conta GitHub real permanece pendente;
a construção e os testes locais da F4A seguem com a fronteira F3 preservada.

## Contratos e autorização

[OpenAPI 3.1 em JSON](dashboard-openapi.json) documenta as três rotas entregues:

- `GET /api/analises`: modo web com total completo, paginação, filtros de
  instalação/repositório/PR/estado/nível/período e ordenação.
- `GET /api/analises/{id}`: detalhe agregado de um único UUID no mesmo tenant
  e nos repositórios autorizados.
- `GET /api/repositorios`: catálogo dos nomes autorizados, consultado no GitHub
  com o token de usuário e paginação completa; não é diagnóstico de proteção.

As duas rotas de análises preservam o contrato HMAC anterior quando o cabeçalho
`X-Aprova-Assinatura` está presente. Sem o cabeçalho, a sessão configurada é
exigida. Assinatura inválida não cai em fallback para um cookie válido; cookie
não permite passar pela autenticação HMAC. O corpo de serviço anterior não
recebe novos campos nem muda o estado padrão `em_andamento`. O modo web usa
`todos` por padrão, documentado separadamente no OpenAPI.

Todos os acessos web confirmam instalação e repositórios no GitHub, com recusa
fechada em falha de autorização. O filtro de repositório apenas reduz o escopo.
UUID de outro tenant ou de repositório não autorizado retorna 404; instalação
não autorizada retorna 403. Uma lista vazia de acesso nunca vira consulta sem
restrição. Estados/níveis inválidos, PR não positivo, datas inválidas, limites
fora da faixa, parâmetros desconhecidos/repetidos e snapshot futuro são 400.

O backend lê exclusivamente tabelas do núcleo: task_runs, risk_gates, task_steps,
sandbox_results, approvals e receipts. Total e página, assim como os agregados
do detalhe, usam transação REPEATABLE READ READ ONLY. Cada SELECT de filho é
restrito ao UUID e tenant; a raiz também exige o repositório autorizado.
Vizinhos da calibração de repositórios fora do acesso atual são omitidos.

## Paginação e semântica dos dados

Página inicia em 1; limite padrão 20, máximo 50; máximo 201 páginas (deslocamento
até 10.000). Ordem por `received_at`, ascendente ou descendente, com UUID
DESC para desempate determinístico. `snapshot` é o instante de criação do
recorte: `created_at <= snapshot`. A partir da segunda página, o cliente deve
reutilizá-lo com os mesmos filtros. Inserções posteriores não deslocam páginas.

O snapshot não congela estados mutáveis nem permissões entre requisições:
revogações são verificadas novamente, e atualizações de gate/estado podem mudar
o conjunto filtrado. A interface informa esse limite e oferece atualização
manual que remove snapshot/página. O total cobre a consulta inteira autorizada,
não a quantidade recebida numa página. Contagem e linhas da mesma resposta
sempre usam a mesma visão do banco.

Períodos são início inclusivo e fim exclusivo, em UTC, pela data de recebimento.
O padrão inclui hoje e os 29 dias anteriores; o front oferece 7/30/90 dias e
todo o histórico. Datas exatas seguem no contrato da API. Filtros e página
ficam na URL; retorno do detalhe e reload preservam o contexto. Mudanças de
filtro reiniciam paginação. Interações rápidas de busca/ordenação preservam os
parâmetros atuais, inclusive enquanto uma navegação React está em andamento.

`nivel:null` significa nenhum gate registrado, apresentado como “Não avaliado”,
inclusive em falha de infraestrutura. Não se assume risco baixo. `atual:false`
indica análise anterior a outra execução do mesmo PR, com aviso destacado no
detalhe. Aprovações só têm `aplicavel_ao_commit:true` com SHA registrado igual ao
da análise; decisões sem SHA ou de outro commit são mostradas como histórico,
sem aplicar a decisão ao commit atual. Uma nova análise não herda gate,
aprovações nem recibo da anterior.

## Telas e evidências disponíveis

Figma consultado com a skill `figma-design-to-code`: lista `9:385`/`15:2152`,
detalhe `9:608`/`15:2375`, com screenshots/contexto de ambos os temas. Casca,
campos, seleção, botões, tabela ordenável e modal da fundação foram reutilizados.
Indicadores de risco e ícones de conclusão/falha foram baixados do Figma para
ativos locais; ajuda, voltar e relógio reutilizam os ativos existentes.

A lista apresenta repositório/PR, nível, estado, recebimento UTC e commit, com
paginação e total reais. Inclui seleção de instalação, busca explícita e limpeza
para suportar o escopo real. As tabelas possuem rolagem própria e foco por
teclado em telas estreitas; não alargam a página inteira.

O detalhe apresenta identificação, SHA, PR e Check Run quando registrado,
regra/evidências, calibração e casos semelhantes autorizados, transições reais
com detalhe acessível, tentativas, saída bruta, decisões/origem e recibo.
A verificação do recibo ocorre por ação explícita e confere a relação com o
UUID da análise antes de exibir as cinco conferências. HTTP 200 com
`valido:false` permanece uma reprovação. Exportação individual usa a tela F2.

Adaptações/ausências honestas:

- **Branch:** não há `head_branch` persistido e este lote não busca metadados
  atuais do PR para preencher uma análise histórica. `branch:null` é explícito.
  A expansão precisa persistir a branch recebida do webhook e definir migração
  dos registros existentes; não usar a branch atual para fabricar o histórico.
- **Testes individuais:** contadores e raw_output são os dados disponíveis.
  `testes_estruturados_disponiveis:false`; nomes/passou/falhou/pulado por teste
  exigem um contrato novo de captura por linguagem no executor.
- **Rede/etapas:** `rede_isolada:null`; não se promete “sem internet” nem tempos
  de etapas separadas sem evidência do executor. A duração exibida é a da tentativa.
- **Saída:** até 65.536 caracteres por tentativa, com `saida_truncada`; escape
  como texto, sem HTML executável. Saída vazia fica explicitamente indisponível.
- **Timeline:** transições persistidas e horários reais, não marcos ilustrativos
  inferidos do Figma. Histórico vazio é informado.
- **Ações:** aprovar/pedir alterações/reexecutar continuam desabilitados até o
  contrato F5; o link GitHub permite a revisão nativa.

Capturas de lista/detalhe em `.capturas-mcp/f4a/local-*`, ignoradas pelo Git,
nos temas claro/escuro e viewports 1440×900, 390×844 e 360×844.

## Escopo dos cards e validação

Cards #66/#68/#69: leitura por sessão, OpenAPI, filtros e paginação, detalhe,
commit, timeline, tentativas/saída, motivos, calibração, origem e recibo.
A coleção completa de recibos/cadeia, saúde de proteção e configuração dos
cards seguintes não é declarada concluída neste lote. Nenhum card Trello foi
alterado. F4B e F4C permanecem os próximos lotes de leitura.

Testes PostgreSQL em container descartável cobrem total/filtros do escopo,
paginação após inserção, risco não avaliado, UUID de outro tenant/repositório,
commit novo sem decisão herdada, saída Unicode truncada, vizinho fora do acesso
e aprovação sem SHA. O banco de demonstração do Codespace é preservado.

Testes de sessão/handlers cobrem filtros inválidos, limites, repetição de
parâmetros e o escopo entregue ao repositório. O fluxo HTTP de navegador usa
handlers reais com dependências explicitamente isoladas no executável E2E;
as capturas e falhas controladas usam interceptações declaradas nos specs.
Esses testes não substituem homologação OAuth contra o GitHub real.

## Resultado da validação local — 16/11/2026

- Suíte completa Playwright: 45 testes aprovados, incluindo os 10 cenários F4A.
- Build de produção e Biome aprovados.
- Testes Go com detector de corrida e `go vet` aprovados nos pacotes verificados.
- Três testes de integração PostgreSQL do painel aprovados em banco descartável.
- `golangci-lint`: zero ocorrências nos pacotes alterados.
- OpenAPI: JSON e referências internas validados; não foi executado validador
  externo completo de OpenAPI.
- API local atualizada para F4A em container com reinício automático; container
  F3 parado e preservado para recuperação. Banco e Redis mantidos.
- Smoke test aprovado para saúde, prontidão, chaves e verificação do recibo de
  demonstração. Rotas HMAC foram puladas no smoke por falta do segredo; seu
  comportamento é coberto pelos testes automatizados.

OAuth real permanece pendente das credenciais locais do responsável, conforme
[registro da F3](dashboard-f3.md). `/api/sessao` informa corretamente
`login_disponivel:false` neste ambiente. Não há bypass de autenticação.
F4B é o próximo lote; este registro não declara toda a F4 concluída.

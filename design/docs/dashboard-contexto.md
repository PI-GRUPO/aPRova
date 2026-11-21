# Dashboard do aPRova

Front em `web/`, branch `feat/web-dashboard`. React 19, TypeScript, Vite, React Router, Tailwind, Radix e componentes locais. O Go serve a SPA de `web/dist/` na origem da API.

## Executar e validar

Execute em `web/`, com Node 24 ou superior. A suíte de navegador também usa Go 1.26 e Chromium.

| Finalidade | Comando |
|---|---|
| Instalar conforme o lockfile | `npm ci` |
| Desenvolver com a API | `npm run dev`. O proxy aponta para `http://127.0.0.1:8080`. Para outra origem local, defina `APROVA_API_PROXY`. |
| Exercitar leituras pendentes em desenvolvimento | `APROVA_DASHBOARD_MOCKS=1 npm run dev`. A sessão continua obrigatória. |
| Validar e compilar | `npm run verificar` e `npm run build` (TypeScript e build estático) |
| Testar | `npm run teste:instalar` para instalar Chromium. Depois do build, `npm run teste -- --workers=4` inicia Vite e o servidor Go isolado nas portas 5183 e 18080. |

## Onde trabalhar

| Caminho | Responsabilidade |
|---|---|
| `web/src/app/{App.tsx,rotas.ts,sessao.tsx}` | Rotas, navegação e ciclo de vida da sessão. |
| `web/src/paginas/` e `web/src/componentes/` | Telas e apresentação. `ConsultaPrivada.tsx` controla leitura, repetição após falha e descarte de resultados por escopo. `Gestao.tsx` fornece os dados da consulta à tela correspondente. |
| `web/src/servicos/http.ts` | Origem, cookies, timeout, cancelamento, validação de JSON e erros HTTP. `VITE_API_ORIGEM` aceita uma origem HTTP(S) pública. A configuração usual usa a mesma origem e navega para `/auth/github`. |
| `web/src/servicos/{analises,sessao,recibos,saude}.ts` | DTOs, validação de respostas e chamadas confirmadas. |
| `web/src/servicos/fonte.ts` | Tipos de escopo, consulta e operação. Uma operação pendente tem `disponivel: false` e sua execução rejeita com `RecursoIndisponivel`. |
| `web/src/mocks/`, `web/vite.config.ts` | Fixtures de desenvolvimento e seleção da fonte em tempo de compilação. Os testes de navegador têm fixtures HTTP próprias em `web/e2e/`. |
| `internal/api/servidor.go`, `internal/api/consulta_analises.go`, `internal/sessaoweb/{servidor,analises}.go` | Registro das rotas e contratos autorizados por sessão. A rota de análises delega ao handler web quando a requisição não traz assinatura de serviço. |

### Seleção da fonte

A API é o padrão. Vite define `__MOCKS__` somente no servidor de desenvolvimento, nos modos `development`, `test` ou `teste`, com `APROVA_DASHBOARD_MOCKS=1` e ambiente diferente de produção. Playwright habilita essa opção explicitamente em seu servidor Vite. Os adapters carregam as fixtures por importação condicional. O build também inspeciona os módulos de cada chunk e falha se incluir código de `src/mocks/`.

As leituras confirmadas usam HTTP inclusive com mocks ativos. As demais podem ler o conjunto sintético após a verificação da sessão e da instalação. Esses dados servem para exercitar a apresentação local. Aprovação, gravações de gestão, identidade Slack, exportação coletiva e verificação da cadeia permanecem indisponíveis em ambas as fontes. A rota auxiliar `/analises/demonstracao/:id` abre o detalhe das fixtures apenas nesse modo. A produção apresenta página não encontrada para esse caminho.

## Operações confirmadas

Arquivos da coluna do front são relativos a `web/src/`. Cada rota HTTP usa GET salvo indicação expressa.

| Tela | Endpoint | Contrato consumido | Front |
|---|---|---|---|
| `/entrar`, conta e navegação | `/api/sessao`, `/auth/github`, `/auth/github/callback` | Disponibilidade do login, usuário GitHub, instalações, expiração e CSRF. O servidor conduz o OAuth. | `paginas/Acesso.tsx`, `app/sessao.tsx`, `servicos/sessao.ts` |
| Sair | `POST /api/sessao/sair` | Cookie, origem válida e `X-CSRF-Token`. Confirma revogação com 204. 401 indica sessão já expirada. | `app/sessao.tsx`, `servicos/sessao.ts` |
| `/repositorios`, seletor de análises | `/api/repositorios` | `installation_id` e nomes completos autorizados. A consulta entrega o catálogo. Proteção de branch, última análise e permissões detalhadas ficam sem confirmação. | `paginas/{LeiturasGestao,Analises}.tsx`, `servicos/analises.ts` |
| `/analises` | `/api/analises` | Instalação obrigatória. Filtros de repositório, PR, estado, nível e intervalo de datas UTC. Ordem recentes/antigas, página 1–201 e limite 1–50. A resposta fornece total, itens, indicador de continuação e snapshot obrigatório a partir da segunda página. Alterar filtros reinicia a paginação. | `paginas/Analises.tsx`, `servicos/analises.ts` |
| `/analises/:id` | `/api/analises/{id}` | UUID e instalação, com filtro opcional de repositório. Detalhe inclui commit, estado atual ou substituído, gate, evidências, transições, tentativas, decisões e UUID do recibo quando emitido. | `paginas/DetalheAnalise.tsx`, `servicos/analises.ts` |
| `/relatorio-semanal` | `/api/web/relatorio-semanal` | Sete dias UTC, instalação, início e repositório opcionais. Taxa de sucesso pode ser nula. | `paginas/RelatorioSemanal.tsx`, `servicos/sessao.ts` |
| `/recibos`, `/recibos/:id`, recibo de uma análise | `/receipts/keys`, `/receipts/{id}`, `/receipts/{id}/verify` | Consulta pública por UUID. Envelope, chaves e resultado das cinco verificações, incluindo o elo anterior. Exportação individual preserva os bytes assinados em base64. | `paginas/{Recibos,DetalheAnalise}.tsx`, `servicos/recibos.ts` |
| Diagnóstico no ambiente de testes | `/health`, `/ready` | Saúde e prontidão. Dependência indisponível pode produzir 503. | `servicos/saude.ts`, `web/e2e/Harness.tsx` |

## Contratos propostos e dependências

Cada linha abaixo aguarda contrato HTTP de sessão. Os tipos do front descrevem as informações necessárias à tela. Os caminhos das telas são rotas da SPA.

| Tela ou ação pendente | Dados e comportamento necessários no backend | Adapter em `web/src/servicos/` |
|---|---|---|
| `/` | Agregado de 30 dias, atualização, motivos, distribuição de falhas, fila com espera e evidências, último recibo e contagem por dia/nível. O estado indisponível oferece a lista real filtrada por aprovação pendente e o relatório semanal. | `visao-geral.ts`, `mapa.ts` |
| Recorte do mapa | Data UTC, risco, IDs das análises, repositório, PR e instante da avaliação no mesmo escopo do agregado. | `mapa.ts`, `analises-demonstracao.ts` |
| Aprovar, pedir alterações e reexecutar em `/analises/:id` | Revalidar sessão, permissão, instalação, repositório, análise atual e SHA ao executar. Receber justificativa e identificador idempotente. Retornar o estado da solicitação. Reexecução depende de suporte do executor. | `acoes-simuladas.ts` |
| Saúde em `/repositorios` | Situação do Check obrigatório, última análise e permissões verificadas por repositório, com estado desconhecido quando a leitura falhar. | `analises.ts` para o catálogo existente |
| `/regras` e `/auditoria` | Regras e estatísticas por período, calibrações e histórico filtrável de tentativas com ator, origem, resultado, motivo, PR e data. Definir paginação/ordenação das coleções com o núcleo. | `gestao.ts` |
| `/configuracao` | Configuração vigente, limites, caminhos e histórico. Gravação exige permissão e controle de concorrência pela versão esperada. | `gestao.ts` |
| `/sugestoes` | Identidade e escopo da proposta, evidência, estado de revisão e transição autorizada. Aplicação depende da versão da regra. | `gestao.ts` |
| `/sla` | Prazos vigentes por repositório e confirmação de alteração autorizada. A automação de avisos fica no servidor. | `gestao.ts` |
| Coleção em `/recibos` | Metadados por período e repositório, verificação da cadeia no escopo autorizado e arquivo de exportação dos registros permitidos. | `colecao-recibos.ts` |
| Slack em `/conta` | Consulta do vínculo persistido, autorização do titular para conectar ou desvincular e nova leitura do estado confirmado. | `vinculo-slack.ts` |

Ao receber um contrato, implemente a chamada em seu adapter com `consultarJSON`, valide o DTO recebido como `unknown` e confira a identidade da resposta antes de convertê-la para o tipo da tela. Substitua a implementação indisponível daquela operação. Para gestão, `consultasGestao` seleciona a leitura por rota. Para comandos, exponha a capacidade somente depois de implementar a execução autenticada e seu tratamento de falhas. `/acoes-remotas` e `/reports/weekly` usam credenciais entre servidores. O navegador utiliza a sessão web.

## Sessão, escopo e falhas

O navegador envia o cookie HttpOnly com `credentials: include`. O provedor reconsulta a sessão no foco e ao restaurar a página, trata expiração e propaga logout entre abas. Consultas privadas com 401 encerram o acesso local e voltam ao login. O 403 informa falta de acesso. Tokens GitHub, chaves de assinatura e segredos HMAC ficam no servidor.

Instalação e filtros vêm da URL. O servidor verifica os repositórios acessíveis. A mudança de escopo remonta a consulta privada correspondente, cancela a requisição anterior e descarta respostas atrasadas. A busca local na fila filtra a visão da instalação já carregada. As telas distinguem carregamento, resposta vazia, falha de consulta e capacidade indisponível. Erros de rede ou de formato não ativam fixtures.

## Verificações e limites

| Verificação | Evidência |
|---|---|
| Dependências e análise estática | As 27 dependências diretas instaladas coincidiram com o lockfile. Biome passou nos 85 arquivos verificados. |
| Produção | `APROVA_DASHBOARD_MOCKS=1 npm run build` passou, incluindo TypeScript e a inspeção dos módulos de todos os chunks. O Vite emitiu aviso de tamanho para o JavaScript principal acima de 500 kB. |
| Navegador | Os 131 cenários Playwright passaram ao longo da suíte e das reexecuções afetadas. Cobertura de desktop/mobile, temas claro/escuro, fluxos com fixtures explícitas, erros de consulta, ações indisponíveis, escopo divergente e logout sem confirmação. |
| Integração isolada | O servidor Go de `web/e2e/servidor/` serviu o build e exercitou os handlers de login, sessão, CSRF, análises, relatório e recibos com dependências de teste. OAuth com o GitHub externo, dados de uma instalação real e produção ficam fora dessa evidência. |
| Separação | A inspeção do front confirmou que páginas e componentes não importam fixtures e que o único `fetch` dos serviços está em `http.ts`. |

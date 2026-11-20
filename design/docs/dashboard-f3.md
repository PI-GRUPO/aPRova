# F3 — Sessão web e relatório semanal

Execução em 16/11/2026, branch `feat/web-dashboard`, base `9bc2fd6`.
AGENTS.md local ausente. Escopo implementado: F3A, F3B e F3C;
aceite com OAuth real pendente. F3D/F7A permanece dependente do contrato de
prova de identidade Slack e da integração do resolvedor persistente.

## Comportamento e limites

Login pelo GitHub App com `state` aleatório, cookie de correlação e PKCE S256.
A tentativa expira em cinco minutos e é consumida atomicamente uma única vez.
A sessão recebe outro identificador aleatório e expira em até oito horas,
limitada à validade do token. Não há renovação automática de tokens nesta fase;
a expiração exige novo login. A sessão anterior é removida após novo login.

O cookie é HttpOnly, SameSite=Lax, Path=/, sem Domain. Em HTTPS é Secure e
usa o prefixo `__Host-`. A exceção HTTP aceita apenas localhost/127.0.0.1.
Host e cabeçalhos de proxy não escolhem o callback nem a origem de logout.
A API mantém tokens exclusivamente no servidor: sessões e verificadores PKCE
são cifrados com AES-256-GCM no Redis, com TTL, chave de registro derivada de
SHA-256 e chave de cifra exclusiva do ambiente. Rotacionar a chave invalida
as sessões existentes; não usar a chave dos recibos para essa finalidade.

Logout exige POST, origem exata e `X-CSRF-Token` correspondente à sessão.
Falha de Redis ou GitHub não libera acesso. O front descarta dados ao trocar
usuário/instalação/período, cancela requisições anteriores, reconsulta sessão
ao recuperar foco e avisa outras abas no logout. Respostas privadas são no-store.
O log de acesso registra caminho, sem query, cookies, códigos OAuth ou tokens.

Permissões são consultadas no GitHub a cada relatório, incluindo paginação:
`GET /user/installations` e
`GET /user/installations/{installation_id}/repositories`.
A consulta SQL restringe todas as contagens, inclusive reversões, ao tenant E
à lista de repositórios autorizada no servidor. Um parâmetro de instalação não
concede acesso; repositório arbitrário, tenant extra ou parâmetro repetido são
recusados antes de consultar dados. Instalações suspensas são ignoradas.
As rotas HMAC do n8n continuam com a autenticação existente; cookies web não
substituem HMAC nessas rotas. Recibos continuam públicos por UUID.

Referências de contrato consultadas: [autorização do GitHub App e PKCE](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-user-access-token-for-a-github-app),
[instalações e repositórios autorizados](https://docs.github.com/en/rest/apps/installations).

## Contratos entregues

Todos os JSON usam `application/json`, sem segredos, com `Cache-Control: no-store`.
Sem redirecionamento automático em erros das APIs JSON; o cliente trata 401.
Não há mutação de decisões, merge ou ações Slack nesta fase.

| Método/rota | Entrada | Resposta |
| --- | --- | --- |
| `GET /auth/github` | Nenhuma; requer configuração OAuth | 303 para autorização no GitHub, cookie de correlação; 503 se desabilitado/Redis indisponível |
| `GET /auth/github/callback` | `code` único e `state`, cookie de correlação | 303 `/` e cookie de sessão; falha leva a `/entrar?erro=autenticacao`, sem detalhe sensível |
| `GET /api/sessao` | Cookie opcional | 200 anônimo ou identidade/instalações; 401 expirada; 503 Redis/GitHub indisponível |
| `POST /api/sessao/sair` | Cookie, Origin e X-CSRF-Token; sem corpo | 204 e remoção do cookie/registro; 401 sem sessão; 403 origem/CSRF; 503 falha de persistência |
| `GET /api/web/relatorio-semanal` | Cookie, `installation_id` positivo obrigatório; `inicio` AAAA-MM-DD e `repositorio` autorizável opcionais | 200 agregado restrito; 400 consulta inválida; 401 sem sessão; 403 escopo negado; 503 fonte/autorização indisponível; 500 falha de consulta |

Sessão anônima: `autenticado:false`, `login_disponivel:boolean` e
`instalacoes:null`; demais campos ausentes. Autenticada: `autenticado:true`,
`login_disponivel:true`, `usuario:{id:number,login:string}`,
`instalacoes:[{id:number,conta:string}]`, `csrf_token:string` e `expira_em` RFC3339.
Uma lista de instalações vazia é um estado válido, sem dados de relatório.
O token CSRF não é uma credencial GitHub e não substitui o cookie HttpOnly.

Relatório: `installation_id`, `repositorios:string[]`, `inicio`/`fim` RFC3339 UTC,
`total_execucoes`, `concluidas`, `rejeitadas`, `falhas`, `em_andamento`,
`taxa_sucesso:number|null`, `exigiram_aprovacao_humana`, `risco_medio`, `revertidas`.
Contagens são inteiros não negativos. O início é inclusivo e o fim exclusivo,
sete dias depois. Sem início, usa a semana anterior completa (segunda a segunda,
UTC). A taxa é concluídas / (concluídas + rejeitadas + falhas); sem finalizadas,
é null. Reversões seguem a data da reversão, e as demais contagens seguem a
janela de recebimento. O front exibe período e denominadores, sem inventar
fila atual, série diária ou indicadores de 30 dias.
Erros JSON têm `{status:string}` com mensagem sem credenciais.

## Configuração para o Codespace

As quatro variáveis devem ser configuradas juntas no ambiente ou no `.env`
ignorado pelo Git:

```dotenv
WEB_URL_PUBLICA=https://NOME-DO-CODESPACE-8080.app.github.dev
GITHUB_CLIENT_ID=CLIENT_ID_DO_GITHUB_APP
GITHUB_CLIENT_SECRET=SEGREDO_DO_GITHUB_APP
WEB_CHAVE_SESSAO=CHAVE_ALEATORIA_DE_32_BYTES_EM_BASE64
```

Gerar a chave sem exibi-la em logs/conversa; por exemplo, gravar o resultado de
`openssl rand -base64 32` diretamente no arquivo de configuração local.
Cadastrar no GitHub App o callback exato:
`https://NOME-DO-CODESPACE-8080.app.github.dev/auth/github/callback`.
O Client ID é diferente do App ID. Não colocar credenciais no bundle, nos commits
ou em variáveis `VITE_*`. Configuração parcial é recusada ao iniciar.
Com todas vazias, a API e os recibos funcionam e o login fica explicitamente
indisponível. A aplicação não concede uma sessão de demonstração em produção.

Para desenvolvimento com cookies/CSRF, prefira o mesmo domínio na porta 8080
com o build incorporado. O proxy Vite inclui `/auth`; quando usado, a origem de
callback/CSRF precisa ser a origem do front configurada em WEB_URL_PUBLICA.

## Visual e cobertura dos cards

Skill `figma-design-to-code` aplicada. Referências Figma consultadas:
`9:2`, `15:1769` (entrada), `10:943` (conta) e `9:22` (visão geral).
Reutilização de casca, marca, campos, seleção, botões, painéis e tokens;
únicos ativos novos são os ícones GitHub dos dois temas (16×16).
Capturas em `.capturas-mcp/f3/local-*`, ignoradas pelo Git, nos viewports
1440×900, 390×844 e 360×844, temas claro/escuro.

Adaptações documentadas: a entrada permite alternar tema e consultar recibos
públicos; remove a indicação de dados simulados. Conta apresenta identidade e
instalações reais; não inventa workspace/status de vínculo Slack. A visão geral
usa somente os agregados semanais existentes, preservando a hierarquia visual
de cartões/painéis. Fila e métricas de 30 dias continuam em F4.

Cards #65/#66: sessão e mediação escopada, sem concluir a leitura operacional
F4. Card #67: entrega semanal parcial, sem declarar a visão geral completa.
Card #77/F3D: não iniciado, dependente de prova de propriedade Slack e consumo
persistente do vínculo pelo resolvedor; nenhuma conexão fictícia foi criada.

## Validação

- `sqlc` v1.30.0 gerou `GetWeeklyReportForRepositories` a partir de
  `queries/reports.sql`; sem migrations novas.
- PostgreSQL descartável: testes de relatório cobrem janela/tenant e exclusão
  de contagens, motivos e reversões de repositório privado; banco local preservado.
- Sessão: state ausente/trocado/expirado/reutilizado, PKCE, atributos do cookie,
  escopo adulterado, revogação, falta de repositório, expiração e CSRF/logout.
- Redis real: conteúdo cifrado, chave diferente recusada, GETDEL concorrente
  com apenas um vencedor e expiração por TTL. Teste opcional exige
  REDIS_TEST_URL; o job dashboard do CI disponibiliza Redis e executa o teste.
- Provedor GitHub: transporte HTTP controlado testa PKCE, segredo no corpo,
  paginação de 101 instalações/repositórios e recusas 401/403/429/500.
- Navegador: os cenários visuais usam interceptações explícitas; o fluxo HTTP
  usa handlers reais com dependências GitHub/armazenamento de teste no executável
  `web/e2e/servidor`, sem rotas ou provedores de teste no binário de produção.

OAuth real: Client ID/Client Secret/callback solicitados, ainda não configurados
nesta execução. Isso impede afirmar que o login de uma conta real foi homologado
ou que os dados locais de demonstração correspondem à instalação GitHub real.
Não usar installation_id=1 da semente para simular autorização de usuário real.

Resultados finais desta execução:

- `npm --prefix web run verificar` e `npm --prefix web run build`: aprovados.
- `go test ./internal/... ./cmd/... ./web/... -race`: aprovado.
- `REDIS_TEST_URL=redis://localhost:6379/15 go test ./internal/sessaoweb ./internal/config -race -count=1`:
  aprovado, incluindo Redis real sem limpar dados de outros usos.
- `go test ./tests/integration/postgres -run 'TestRelatorio' -count=1`:
  aprovado em container descartável.
- `golangci-lint run` nos pacotes sessão/API/config/repository/cmd/api/web:
  **0 apontamentos**.
- Suíte de navegador: 34 cenários aprovados; o restante ainda esperava o título
  anterior da entrada. Após ajustar a expectativa ao título Figma atual,
  `npm --prefix web run teste -- fundacao.spec.ts -g '401 privado'` aprovou
  o cenário restante. Os dez cenários F3 e os treze F2 passaram.
- Capturas de entrada, conta e visão semanal inspecionadas nos temas/viewports
  descritos; artefatos do CI incluem agora todas as pastas de capturas por fase.
- API de produção local atualizada: container `aprova-api-local`, porta 8080,
  política `unless-stopped`, com binário estático e build web incorporado.
  PostgreSQL/Redis e recibos da semente foram preservados; `/ready`,
  `/receipts/keys` e a verificação do recibo local aprovaram o smoke.
  `/api/sessao` confirma `login_disponivel:false` com OAuth não configurado.

A API do Codespace não depende mais de um processo de terminal da conversa.
O container local usa o binário `/tmp/aprova-api-f3-estatica` montado somente para
leitura e as variáveis carregadas do `.env` na criação. Parar/reiniciar o
Codespace suspende o serviço; reconstruir o ambiente pode remover esse binário.
Alterar credenciais no `.env` exige recriar o container para carregar os novos
valores. Não foi realizado login real nem executado CI remoto.

## Pendência confirmada pela usuária — 16/11/2026

A usuária informou que as credenciais OAuth provavelmente estão no `.env` da
máquina dela e não estão disponíveis neste Codespace. A localização e os valores
ainda não foram conferidos. A configuração/homologação real fica adiada por
orientação explícita; o desenvolvimento segue para F4A com os testes isolados e
a autorização F3 preservada.

Ao retomar: conferir o Client ID e Client Secret do GitHub App no ambiente local,
configurar as quatro variáveis descritas acima no Codespace, cadastrar o callback
exato, gerar uma chave de sessão exclusiva e recriar o container da API para
carregar o ambiente. Fazer login com a conta real e validar instalações,
repositórios permitidos, negação de outro escopo e logout. Nenhum segredo deve
ser enviado pela conversa ou versionado. Até isso ocorrer, login indisponível
continua sendo o estado correto no ambiente real; não habilitar bypass local.

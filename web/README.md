# Dashboard aPRova

Front em React/TypeScript. O backend existente permanece responsável por sessão,
autorização e dados. O Vite encaminha as chamadas para `http://127.0.0.1:8080`.
Outro destino pode ser definido por `APROVA_API_PROXY` em `web/.env.local`,
usando `web/.env.example` como referência. Reinicie o Vite após mudar a variável.
Ela é lida somente pelo servidor de desenvolvimento; não use `VITE_*` para isso.
Preparação da API, OAuth e diagnóstico: guia local `design/docs/dashboard-api-desenvolvimento.md`,
ignorado pelo Git.

```sh
npm --prefix web ci
npm --prefix web run dev
```

Abra `http://127.0.0.1:5173`. Com uma sessão autenticada, a fila da visão geral
consulta `GET /api/analises` para a instalação selecionada e abre o detalhe em
`/analises/:id`. O filtro de repositório abrange a fila inteira. A busca textual
por PR, repositório ou commit abrange a página exibida. Os filtros e a paginação
ficam na URL. O relatório semanal usa `GET /api/web/relatorio-semanal` e cobre
sete dias UTC.

Para visualizar as seções simuladas em desenvolvimento, defina
`APROVA_DASHBOARD_MOCKS=1` em `web/.env.local` e reinicie o Vite. As métricas
de 30 dias, distribuições, último recibo e mapa diário usam fixtures em
`src/mocks/`. As células do mapa abrem detalhes de demonstração com o dia e o
nível na URL. A coleção de recibos e as leituras de regras, auditoria,
configuração, sugestões e SLA também identificam a simulação. O build de produção
exclui essas fixtures.

Sessão, repositórios, análises, relatório semanal, recibos por UUID, chaves e
verificação individual usam a API mesmo com as simulações ativadas.
Aprovação, pedido de alterações, reexecução, gravações de gestão, vínculo Slack
e operações sobre a cadeia completa permanecem indisponíveis no painel.

```sh
npm --prefix web run verificar
npm --prefix web run build
npm --prefix web run teste:instalar
npm --prefix web run teste
npm --prefix web run teste:visual
```

A regressão inicia o Vite na porta exclusiva 5183 e o servidor isolado de testes
na 18080. O teste visual inicia o Vite na 5173 e não precisa de backend.
As capturas ficam em `../.capturas-mcp/`, ignoradas pelo Git. Quando o ambiente
não permite gravar o cache padrão do navegador, use `PLAYWRIGHT_BROWSERS_PATH`
com um diretório local gravável na instalação e na execução dos testes.

Não inserir tokens, HMAC ou chaves em variáveis `VITE_*`. A composição visual
vem do Figma; veja o estado por lote em `../design/docs/implementacao-dashboard.md`.

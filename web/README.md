# Dashboard aPRova

Front em React/TypeScript. O backend existente permanece responsável por sessão,
autorização e dados. O Vite encaminha as chamadas para `http://127.0.0.1:8080`.

```sh
npm --prefix web ci
npm --prefix web run dev
```

Abra `http://127.0.0.1:5173`. A visão geral usa dados simulados separados em
`src/mocks/visao-geral.ts`, pois o contrato de métricas de 30 dias ainda não existe.
O serviço substituível fica em `src/servicos/visao-geral.ts`. A indicação de dados
simulados faz parte da tela; falhas da API real nunca ativam mocks automaticamente.
Os links da fila abrem `/analises/demonstracao/:id`, com fixtures de
`src/mocks/analises-demonstracao.ts` e adapter próprio. Esse fluxo funciona
sem sessão e sem consultar a API de análises ou recibos. A consulta real em
`/analises/:id` preserva sua proteção por sessão e tratamento de erros.
O relatório real da API continua em `/relatorio-semanal`, acessível pelo rodapé.
Análises, recibos por UUID, chaves públicas e sessão continuam consumindo seus contratos existentes.
O catálogo de repositórios também consulta a API quando há uma instalação autorizada.

Regras, auditoria, configuração, sugestões e SLA usam `src/mocks/gestao.ts`,
com adapter em `src/servicos/gestao.ts`. Alterações ficam apenas em memória;
recarregar a página restaura os valores da demonstração. A coleção/cadeia de
recibos usa `src/mocks/recibos.ts`; suas exportações não contêm assinaturas ou
valor de prova. O vínculo Slack usa `src/mocks/conta.ts` e um adapter local,
sem OAuth ou escrita na API. As telas identificam essas simulações.

Os botões de decisão no detalhe da análise abrem modais de demonstração.
`src/servicos/acoes-simuladas.ts` usa respostas de `src/mocks/acoes.ts`, sem
HTTP, reviews, merge ou reexecução real. Solicitações ficam em memória e são
isoladas por usuário, instalação, repositório, PR, análise e SHA. O reload restaura
a demonstração; os dados reais da API e os recibos permanecem preservados.

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
